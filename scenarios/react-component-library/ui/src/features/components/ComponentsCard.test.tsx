import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  makeComponent,
  makeGetComponentContentResponse,
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
            tags: ["form", "primary"],
          }),
          makeComponent({
            id: "b",
            libraryId: "react-component-library:Card",
            displayName: "Card",
            version: "0.4.0",
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

  it("opens the editor when an item's Edit button is clicked", async () => {
    const { componentsClient } = await import("../../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValue(
      makeListComponentsResponse({
        components: [
          makeComponent({ id: "cmp-1", libraryId: "lib:Button", displayName: "Button" }),
        ],
      }),
    );
    vi.mocked(componentsClient.getComponentContent).mockResolvedValueOnce(
      makeGetComponentContentResponse({ content: "// hello\n" }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ComponentsCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.itemEditButton)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.components.itemEditButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.components.editor.panel)).toBeInTheDocument();
    });
    expect(componentsClient.getComponentContent).toHaveBeenCalledWith({ id: "cmp-1" });
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
