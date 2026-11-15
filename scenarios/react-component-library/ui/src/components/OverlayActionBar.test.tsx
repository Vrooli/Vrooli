import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  OverlayActionBar,
  type OverlayActionList,
} from "../../../library/components/OverlayActionBar/versions/1.0.1/OverlayActionBar";
import { LibraryStringsProvider } from "../../../library/hooks/useLocale/versions/1.1.2/useLocale";
import {
  fiveGermanOverlayActions,
  missingIDOverlayActions,
} from "../../../library/usage-contracts/adversarial-fixtures";

const unsafeFiveGermanActions = fiveGermanOverlayActions as unknown as OverlayActionList;
const unsafeMissingIDActions = missingIDOverlayActions as unknown as OverlayActionList;
const typePrimary = {
  id: "primary",
  label: "Save",
  onSelect: () => undefined,
  primary: true,
} as const;
const typeSecondary = { id: "secondary", label: "Cancel", onSelect: () => undefined } as const;
const typedFourActions = [
  typePrimary,
  typeSecondary,
  typeSecondary,
  typeSecondary,
] as const satisfies OverlayActionList;
// @ts-expect-error the published action contract permits at most four actions
const typedFiveActions: OverlayActionList = [
  typePrimary,
  typeSecondary,
  typeSecondary,
  typeSecondary,
  typeSecondary,
];

describe("OverlayActionBar usage contract", () => {
  afterEach(() => vi.restoreAllMocks());

  it("types the published maximum of four actions", () => {
    expect(typedFourActions).toHaveLength(4);
    expect(typedFiveActions).toHaveLength(5);
  });

  it("keeps five long German actions in one row and moves the extra actions to overflow", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    render(<OverlayActionBar actions={unsafeFiveGermanActions} />);

    const group = screen.getByRole("group", { name: "Actions" });
    expect(within(group).getAllByRole("button")).toHaveLength(3);
    expect(within(group).getByRole("button", { name: "More 3 actions" })).toBeInTheDocument();
    expect(
      within(group).getByRole("button", { name: "Änderungen dauerhaft speichern" }),
    ).toHaveTextContent("…");
    expect(group).toHaveStyle({ flexWrap: "nowrap" });
    expect(warning).toHaveBeenCalledWith(
      expect.stringContaining("maximum is 4"),
      expect.any(Object),
    );
  });

  it("opens a menu for the remaining actions without stacking the row", () => {
    render(<OverlayActionBar actions={unsafeFiveGermanActions} />);

    fireEvent.click(screen.getByRole("button", { name: "More 3 actions" }));

    expect(screen.getAllByRole("menuitem")).toHaveLength(3);
    expect(
      screen.getByRole("menuitem", { name: "Einstellungen zurücksetzen" }),
    ).toBeInTheDocument();
  });

  it("does not let tuning props raise the contract's row or label limits", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    render(
      <OverlayActionBar actions={unsafeFiveGermanActions} maxVisible={5} maxLabelLength={80} />,
    );

    const group = screen.getByRole("group", { name: "Actions" });
    expect(within(group).getAllByRole("button")).toHaveLength(3);
    expect(within(group).getByRole("button", { name: "More 3 actions" })).toBeInTheDocument();
    expect(
      within(group).getByRole("button", { name: "Änderungen dauerhaft speichern" }),
    ).toHaveTextContent("…");
    expect(warning).toHaveBeenCalledWith(
      expect.stringContaining("maxVisible 5 exceeds overflow limit 2"),
      expect.any(Object),
    );
    expect(warning).toHaveBeenCalledWith(
      expect.stringContaining("maxLabelLength 80 exceeds 28"),
      expect.any(Object),
    );
  });

  it("reports malformed inputs without crashing the rendered action bar", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    render(<OverlayActionBar actions={unsafeMissingIDActions} />);

    expect(screen.getByRole("group", { name: "Actions" })).toBeInTheDocument();
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("has no id"), expect.any(Object));
  });
  describe("localised chrome", () => {
    const spanish: Record<string, string> = {
      "overlay-action-bar.label": "Acciones",
      "overlay-action-bar.more": "Más",
      "overlay-action-bar.actions": "acciones",
    };
    const translate = (key: string, fallback: string) => spanish[key] ?? fallback;

    it("translates the group label, the More trigger and its screen-reader count through the host's strings", () => {
      vi.spyOn(console, "warn").mockImplementation(() => undefined);
      render(
        <LibraryStringsProvider translate={translate}>
          <OverlayActionBar actions={unsafeFiveGermanActions} />
        </LibraryStringsProvider>,
      );

      const group = screen.getByRole("group", { name: "Acciones" });
      expect(within(group).getByRole("button", { name: "Más 3 acciones" })).toBeInTheDocument();
      expect(screen.queryByRole("group", { name: "Actions" })).toBeNull();
      expect(screen.queryByText("More")).toBeNull();
    });

    it("lets a caller's explicit label win over the translated default", () => {
      vi.spyOn(console, "warn").mockImplementation(() => undefined);
      render(
        <LibraryStringsProvider translate={translate}>
          <OverlayActionBar actions={unsafeFiveGermanActions} label="Archivo" />
        </LibraryStringsProvider>,
      );

      expect(screen.getByRole("group", { name: "Archivo" })).toBeInTheDocument();
    });

    it("falls back to English for keys the host does not translate", () => {
      vi.spyOn(console, "warn").mockImplementation(() => undefined);
      render(
        <LibraryStringsProvider translate={(_key, fallback) => fallback}>
          <OverlayActionBar actions={unsafeFiveGermanActions} />
        </LibraryStringsProvider>,
      );

      expect(screen.getByRole("group", { name: "Actions" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "More 3 actions" })).toBeInTheDocument();
    });
  });
});
