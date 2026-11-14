import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { OverlayActionBar } from "../../../library/components/OverlayActionBar/versions/1.0.0/OverlayActionBar";
import { fiveGermanOverlayActions, missingIDOverlayActions } from "../../../library/usage-contracts/adversarial-fixtures";

describe("OverlayActionBar usage contract", () => {
  afterEach(() => vi.restoreAllMocks());

  it("keeps five long German actions in one row and moves the extra actions to overflow", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    render(<OverlayActionBar actions={fiveGermanOverlayActions} />);

    const group = screen.getByRole("group", { name: "Actions" });
    expect(within(group).getAllByRole("button")).toHaveLength(3);
    expect(within(group).getByRole("button", { name: "More 3 actions" })).toBeInTheDocument();
    expect(within(group).getByRole("button", { name: "Änderungen dauerhaft speichern" })).toHaveTextContent("…");
    expect(group).toHaveStyle({ flexWrap: "nowrap" });
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("maximum is 4"), expect.any(Object));
  });

  it("opens a menu for the remaining actions without stacking the row", () => {
    render(<OverlayActionBar actions={fiveGermanOverlayActions} />);

    fireEvent.click(screen.getByRole("button", { name: "More 3 actions" }));

    expect(screen.getAllByRole("menuitem")).toHaveLength(3);
    expect(screen.getByRole("menuitem", { name: "Einstellungen zurücksetzen" })).toBeInTheDocument();
  });

  it("does not let tuning props raise the contract's row or label limits", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    render(<OverlayActionBar actions={fiveGermanOverlayActions} maxVisible={5} maxLabelLength={80} />);

    const group = screen.getByRole("group", { name: "Actions" });
    expect(within(group).getAllByRole("button")).toHaveLength(3);
    expect(within(group).getByRole("button", { name: "More 3 actions" })).toBeInTheDocument();
    expect(within(group).getByRole("button", { name: "Änderungen dauerhaft speichern" })).toHaveTextContent("…");
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("maxVisible 5 exceeds overflow limit 2"), expect.any(Object));
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("maxLabelLength 80 exceeds 28"), expect.any(Object));
  });

  it("reports malformed inputs without crashing the rendered action bar", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    render(<OverlayActionBar actions={missingIDOverlayActions} />);

    expect(screen.getByRole("group", { name: "Actions" })).toBeInTheDocument();
    expect(warning).toHaveBeenCalledWith(expect.stringContaining("has no id"), expect.any(Object));
  });
});
