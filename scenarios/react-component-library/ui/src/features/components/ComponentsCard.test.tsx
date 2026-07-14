import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  makeComponent,
  makeIndexComponentsResponse,
  makeListComponentsResponse,
} from "./mocks/factories";
import { makeComponentsMocks } from "./mocks/components";

vi.mock("@monaco-editor/react", () => ({
  __esModule: true,
  default: (props: {
    value?: string;
    onChange?: (v: string | undefined) => void;
  }) => (
    <textarea
      data-testid="monaco-stub"
      value={props.value ?? ""}
      onChange={(e) => props.onChange?.(e.target.value)}
    />
  ),
}));

vi.mock("../../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/components")>();
  return { ...actual, ...makeComponentsMocks() };
});

import { ComponentsCard } from "./ComponentsCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("ComponentsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when the registry has no components", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValueOnce(makeListComponentsResponse());

    renderWithProviders(<ComponentsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.components.list)).not.toBeInTheDocument();
  });

  it("renders indexed components with libraryId, displayName, version, tags", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValueOnce(
      makeListComponentsResponse({
        components: [
          makeComponent({
            id: "a",
            libraryId: "react-component-library:Button",
            displayName: "Button",
            version: "1.2.3",
            slot: "ui-primitive",
            tags: ["form", "primary"],
            designStyles: [
              { styleId: "vrooli-default", affinity: 1 },
              { styleId: "vrooli-conversion-landing", affinity: 3 },
            ],
          }),
          makeComponent({
            id: "b",
            libraryId: "react-component-library:Card",
            displayName: "Card",
            version: "0.4.0",
            slot: "ui-pattern",
            tags: [],
          }),
        ],
      }),
    );

    renderWithProviders(<ComponentsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.list)).toBeInTheDocument();
    });

    const ids = screen.getAllByTestId(selectors.components.itemLibraryId).map((n) => n.textContent);
    expect(ids).toEqual([
      "react-component-library:Button",
      "react-component-library:Card",
    ]);

    expect(screen.getByTestId(selectors.components.summary).textContent).toContain("2");
    const versions = screen.getAllByTestId(selectors.components.itemVersion).map((n) => n.textContent);
    expect(versions[0]).toContain("1.2.3");
    expect(versions[1]).toContain("0.4.0");

    const slots = screen.getAllByTestId(selectors.components.itemSlot).map((n) => n.textContent);
    expect(slots).toEqual(["Slot: ui-primitive", "Slot: ui-pattern"]);

    const styles = screen.getAllByTestId(selectors.components.itemDesignStyles).map((n) => n.textContent);
    expect(styles[0]).toContain("vrooli-default:native");
    expect(styles[0]).toContain("vrooli-conversion-landing:discouraged");

    const tagLines = screen.getAllByTestId(selectors.components.itemTags).map((n) => n.textContent);
    expect(tagLines[0]).toContain("form");
    expect(tagLines[0]).toContain("primary");
    expect(tagLines[1]).toMatch(/No tags/);
  });

  it("forwards search + tag filters to listComponents", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValue(makeListComponentsResponse());

    const user = userEvent.setup();
    renderWithProviders(<ComponentsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.searchInput)).toBeInTheDocument();
    });

    await user.type(screen.getByTestId(selectors.components.searchInput), "btn");
    await user.type(screen.getByTestId(selectors.components.tagInput), "form");

    await waitFor(() => {
      const calls = vi.mocked(componentsClient.listComponents).mock.calls;
      const last = calls[calls.length - 1]?.[0] ?? {};
      expect(last).toMatchObject({ match: "btn", tag: "form" });
    });
  });

  it("forwards multi-tag (comma-split) + category to listComponents", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValue(makeListComponentsResponse());

    const user = userEvent.setup();
    renderWithProviders(<ComponentsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.tagsInput)).toBeInTheDocument();
    });

    await user.type(screen.getByTestId(selectors.components.tagsInput), "form, layout");
    await user.type(screen.getByTestId(selectors.components.categoryInput), "controls");

    await waitFor(() => {
      const calls = vi.mocked(componentsClient.listComponents).mock.calls;
      const last = calls[calls.length - 1]?.[0] ?? {};
      expect(last).toMatchObject({
        tags: ["form", "layout"],
        category: "controls",
      });
    });
  });

  it("forwards style and affinity filters to listComponents", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValue(makeListComponentsResponse());

    const user = userEvent.setup();
    renderWithProviders(<ComponentsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.styleInput)).toBeInTheDocument();
    });

    await user.type(screen.getByTestId(selectors.components.styleInput), "vrooli-default");
    await user.type(screen.getByTestId(selectors.components.affinityInput), "native");

    await waitFor(() => {
      const calls = vi.mocked(componentsClient.listComponents).mock.calls;
      const last = calls[calls.length - 1]?.[0] ?? {};
      expect(last).toMatchObject({
        styleId: "vrooli-default",
        affinity: "native",
      });
    });
  });

  it("uses the routed editor as the sole item entry point", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValue(
      makeListComponentsResponse({
        components: [
          makeComponent({ id: "cmp-1", libraryId: "lib:Button", displayName: "Button" }),
        ],
      }),
    );
    renderWithProviders(<ComponentsCard />);

    await waitFor(() => {
      expect(screen.getByRole("link", { name: "Open" })).toHaveAttribute("href", "/components/cmp-1");
    });
    expect(screen.queryByTestId(selectors.components.itemEditButton)).not.toBeInTheDocument();
    expect(componentsClient.getComponentContent).not.toHaveBeenCalled();
  });

  it("keeps all server filters inside one collapsed filter control", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValue(makeListComponentsResponse());

    renderWithProviders(<ComponentsCard />);

    const filters = await screen.findByTestId("components-filters");
    expect(filters).not.toHaveAttribute("open");
    expect(filters.querySelectorAll("input")).toHaveLength(6);
  });

  it("invokes indexComponents when re-index is clicked", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValue(makeListComponentsResponse());
    vi.mocked(componentsClient.indexComponents).mockResolvedValueOnce(
      makeIndexComponentsResponse({ scanned: 3, indexed: 2 }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ComponentsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.indexButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.components.indexButton));

    await waitFor(() => {
      expect(componentsClient.indexComponents).toHaveBeenCalledTimes(1);
    });
  });
});
