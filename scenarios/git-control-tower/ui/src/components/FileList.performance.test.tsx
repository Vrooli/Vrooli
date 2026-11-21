import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { FileListProps } from "./FileListTypes";
import { FileList } from "./FileList";
import { renderWithProviders } from "../test-utils/renderWithProviders";

// Keep TanStack's actual range/measurement logic. Only supply the viewport
// geometry and scrolling that jsdom cannot measure.
vi.mock("@tanstack/react-virtual", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-virtual")>();
  const observeElementRect: NonNullable<Parameters<typeof actual.useVirtualizer>[0]["observeElementRect"]> = (_instance, cb) => {
    cb({ width: 400, height: 480 });
  };
  return { ...actual, useVirtualizer: (options: Parameters<typeof actual.useVirtualizer>[0]) => actual.useVirtualizer({ ...options, observeElementRect, measureElement: () => 32 }) };
});
vi.mock("../hooks", () => ({ useIsMobile: () => false, useLongPress: () => ({}) }));

function propsFor(paths: string[], grouped: boolean): FileListProps {
  return {
    files: { staged: [], unstaged: paths, untracked: [], conflicts: [] },
    fileViewMode: grouped ? "grouped" : "flat",
    resolvedGroups: grouped ? [{ key: "large", id: "large", kind: "scenario", source: "contract", label: "Large group", files: paths }] : undefined,
    selectedFiles: [], selectionKey: ({ path, staged }) => `${staged}:${path}`,
    onSelectFile: vi.fn(), onStageFile: vi.fn(), onUnstageFile: vi.fn(),
    onStageAll: vi.fn(), onUnstageAll: vi.fn(), onDiscardFile: vi.fn(), onIgnoreFile: vi.fn(),
    isStaging: false, isDiscarding: false, isIgnoring: false,
    confirmingDiscard: null, confirmingIgnore: null,
    onConfirmDiscard: vi.fn(), onConfirmIgnore: vi.fn(),
  };
}

describe("large Changes view", () => {
  for (const grouped of [false, true]) {
    it(`bounds mounted file rows for 10,000 changes in ${grouped ? "grouped" : "flat"} mode`, async () => {
      const paths = Array.from({ length: 10_000 }, (_, i) => `src/file-${i}.ts`);
      const props = propsFor(paths, grouped);
      renderWithProviders(<FileList {...props} />);
      await waitFor(() => expect(screen.getAllByTestId("file-item-unstaged").length).toBeGreaterThan(0));
      expect(screen.getAllByTestId("file-item-unstaged").length).toBeLessThan(60);
      expect(screen.queryByText("src/file-9999.ts")).not.toBeInTheDocument();
      fireEvent.click(screen.getAllByTestId("file-action-unstaged")[0] as HTMLElement);
      expect(props.onStageFile).toHaveBeenCalledWith(paths[0]);
    });
  }

  it("keeps distant rows reachable and selection intact while scrolling and collapsing a large group", async () => {
    const paths = Array.from({ length: 10_000 }, (_, i) => `src/file-${i}.ts`);
    const props = propsFor(paths, true);
    props.selectedFiles = [{ path: paths[0] ?? "", staged: false }];
    renderWithProviders(<FileList {...props} />);
    const scroller = screen.getByTestId("changes-scroll-region");
    await waitFor(() => expect(screen.getByText("src/file-0.ts")).toBeInTheDocument());
    expect(screen.getByText("src/file-0.ts").closest("li")).toHaveClass("bg-slate-700/50");
    // Fixed geometry: two header rows plus 10,000 file rows at 32px.
    act(() => {
      scroller.scrollTop = 320_000;
      fireEvent.scroll(scroller);
    });
    await waitFor(() => expect(screen.getByText("src/file-9999.ts")).toBeInTheDocument());
    expect(screen.getAllByTestId("file-item-unstaged").length).toBeLessThan(60);
    const last = screen.getByText("src/file-9999.ts").closest("li");
    fireEvent.click(last as HTMLElement);
    expect(props.onSelectFile).toHaveBeenCalledWith("src/file-9999.ts", false, expect.anything());
    act(() => { scroller.scrollTop = 0; fireEvent.scroll(scroller); });
    await waitFor(() => expect(screen.getByTestId("file-group-toggle-large")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("file-group-toggle-large"));
    expect(screen.queryAllByTestId("file-item-unstaged")).toHaveLength(0);
    fireEvent.click(screen.getByTestId("file-group-toggle-large"));
    await waitFor(() => expect(screen.getByText("src/file-0.ts")).toBeInTheDocument());
    expect(screen.getByText("src/file-0.ts").closest("li")).toHaveClass("bg-slate-700/50");
  });

  it("bounds mounted group headers across 1,000 groups", async () => {
    const paths = Array.from({ length: 1000 }, (_, i) => `src/file-${i}.ts`);
    const props = propsFor(paths, true);
    props.resolvedGroups = paths.map((path, i) => ({ key: `group-${i}`, id: String(i), kind: "scenario", source: "contract", label: `Group ${i}`, files: [path] }));
    renderWithProviders(<FileList {...props} />);
    await waitFor(() => expect(screen.getByTestId("file-group-group-0")).toBeInTheDocument());
    expect(document.querySelectorAll('[data-testid^="file-group-group-"]').length).toBeLessThan(30);
    expect(screen.queryByText("Group 999")).not.toBeInTheDocument();
  });
});
