import "@testing-library/jest-dom";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import { DiffViewer } from "./DiffViewer";
import type { DiffResponse, LineChange, ViewMode } from "../lib/api";
import { FileContentConflictError } from "../lib/api";

vi.mock("../lib/highlighter", () => ({
  highlightCode: vi.fn(async (content: string) =>
    content.split("\n").map((line, index) => ({
      lineNumber: index + 1,
      tokens: [{ content: line }]
    }))
  ),
  getLanguageFromPath: vi.fn(() => "typescript")
}));

vi.mock("@monaco-editor/react", () => ({
  default: (props: {
    value?: string;
    onChange?: (value?: string) => void;
  }) => (
    <textarea
      data-testid="monaco-editor"
      value={props.value ?? ""}
      onChange={(event) => props.onChange?.(event.target.value)}
    />
  )
}));

function buildDiffResponse(overrides: Partial<DiffResponse> = {}): DiffResponse {
  return {
    repo_dir: "/repo",
    path: "src/main.ts",
    staged: false,
    has_diff: true,
    stats: { additions: 5, deletions: 2, files: 1 },
    full_content: "",
    annotated_lines: [],
    timestamp: "2026-02-13T00:00:00Z",
    ...overrides
  };
}

function renderViewer(
  diff: DiffResponse,
  viewMode: ViewMode = "source",
  overrides: Partial<ComponentProps<typeof DiffViewer>> = {}
) {
  return render(
    <DiffViewer
      diff={diff}
      selectedFile="src/main.ts"
      isStaged={false}
      isUntracked={false}
      isLoading={false}
      error={null}
      repoDir="/repo"
      viewMode={viewMode}
      onViewModeChange={() => {}}
      isReadOnly={false}
      {...overrides}
    />
  );
}

describe("DiffViewer minimap", () => {
  it("renders minimap for long source files and hides it for short files", async () => {
    const longContent = Array.from({ length: 120 }, (_, i) => `line ${i + 1}`).join("\n");
    const view = renderViewer(buildDiffResponse({ full_content: longContent }), "source");

    await waitFor(() => {
      expect(screen.getByTestId("diff-minimap")).toBeInTheDocument();
    });
    expect(screen.getByTestId("diff-minimap-texture")).toBeInTheDocument();
    expect(screen.getAllByTestId("diff-minimap-texture-line").length).toBeGreaterThan(20);

    const shortContent = Array.from({ length: 20 }, (_, i) => `short ${i + 1}`).join("\n");
    view.rerender(
      <DiffViewer
        diff={buildDiffResponse({ full_content: shortContent })}
        selectedFile="src/main.ts"
        isStaged={false}
        isUntracked={false}
        isLoading={false}
        error={null}
        repoDir="/repo"
        viewMode="source"
        onViewModeChange={() => {}}
        isReadOnly={false}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("diff-minimap")).not.toBeInTheDocument();
    });
  });

  it("jumps to clicked location in minimap rail", async () => {
    const longContent = Array.from({ length: 180 }, (_, i) => `line ${i + 1}`).join("\n");
    renderViewer(buildDiffResponse({ full_content: longContent }), "source");

    const minimapRail = await screen.findByTestId("diff-minimap-rail");
    const scroller = minimapRail.closest("[data-testid='diff-viewer-panel']")?.querySelector(".overflow-auto") as HTMLDivElement;
    expect(scroller).toBeTruthy();

    Object.defineProperty(scroller, "scrollHeight", { configurable: true, value: 4000 });
    Object.defineProperty(scroller, "clientHeight", { configurable: true, value: 400 });
    Object.defineProperty(scroller, "scrollTop", { configurable: true, writable: true, value: 0 });
    scroller.scrollTo = vi.fn(
      (xOrOptions?: number | ScrollToOptions, y?: number) => {
        if (typeof xOrOptions === "number") {
          scroller.scrollTop = typeof y === "number" ? y : 0;
          return;
        }
        scroller.scrollTop = typeof xOrOptions?.top === "number" ? xOrOptions.top : 0;
      }
    ) as typeof scroller.scrollTo;

    vi.spyOn(minimapRail, "getBoundingClientRect").mockReturnValue({
      top: 0,
      left: 0,
      width: 40,
      height: 200,
      right: 40,
      bottom: 200,
      x: 0,
      y: 0,
      toJSON: () => ({})
    });

    fireEvent.pointerDown(minimapRail, { clientY: 100 });

    expect(scroller.scrollTo).toHaveBeenCalled();
    expect(scroller.scrollTop).toBeGreaterThan(1700);
    expect(scroller.scrollTop).toBeLessThan(1900);
  });

  it("renders change markers in full diff mode", async () => {
    const annotatedLines = Array.from({ length: 120 }, (_, index) => ({
      number: index + 1,
      content: `line ${index + 1}`,
      change:
        index === 3 ? "added" : index === 60 ? "modified" : index === 100 ? "deleted" : ("" as const),
      old_number: index === 100 ? 98 : undefined
    })) as Array<{ number: number; content: string; change: LineChange; old_number?: number }>;
    const fullContent = annotatedLines.map((line) => line.content).join("\n");
    renderViewer(buildDiffResponse({ annotated_lines: annotatedLines, full_content: fullContent }), "full_diff");

    await waitFor(() => {
      expect(screen.getByTestId("diff-minimap")).toBeInTheDocument();
    });

    expect(screen.getByTestId("diff-minimap-viewport")).toBeInTheDocument();
    expect(screen.getAllByTestId("diff-minimap-texture-line").length).toBeGreaterThan(20);
    expect(screen.getAllByTestId("diff-minimap-marker").length).toBeGreaterThanOrEqual(3);
  });

  it("allows editing and saving in source mode", async () => {
    const onSaveFileContent = vi.fn(async () => ({
      success: true,
      path: "src/main.ts",
      content_hash: "new-hash",
      bytes_written: 14,
      timestamp: "2026-02-13T00:00:00Z"
    }));
    renderViewer(
      buildDiffResponse({
        full_content: "const a = 1;\n",
        content_hash: "old-hash"
      }),
      "source",
      { onSaveFileContent }
    );

    fireEvent.click(screen.getByTestId("start-editing-button"));
    const editor = screen.getByTestId("monaco-editor");
    fireEvent.change(editor, { target: { value: "const a = 2;\n" } });
    fireEvent.click(screen.getByTestId("save-file-button"));

    await waitFor(() => {
      expect(onSaveFileContent).toHaveBeenCalledWith("src/main.ts", "const a = 2;\n", "old-hash");
    });
  });

  it("shows conflict message when save encounters hash conflict", async () => {
    const onSaveFileContent = vi.fn(async () => {
      throw new FileContentConflictError("conflict", "src/main.ts", "server-hash");
    });
    renderViewer(
      buildDiffResponse({
        full_content: "const a = 1;\n",
        content_hash: "old-hash"
      }),
      "source",
      { onSaveFileContent }
    );

    fireEvent.click(screen.getByTestId("start-editing-button"));
    const editor = screen.getByTestId("monaco-editor");
    fireEvent.change(editor, { target: { value: "const a = 3;\n" } });
    fireEvent.click(screen.getByTestId("save-file-button"));

    await waitFor(() => {
      expect(screen.getByTestId("save-error")).toHaveTextContent("File changed on disk");
      expect(screen.getByTestId("save-error")).toHaveTextContent("server-hash");
    });
  });
});
