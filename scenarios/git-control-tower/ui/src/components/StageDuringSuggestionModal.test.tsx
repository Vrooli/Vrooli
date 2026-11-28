import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { setViewport } from "../test-utils";
import { StageDuringSuggestionModal } from "./StageDuringSuggestionModal";

describe("stage queue confirmation", () => {
  it.each([[1280, 800], [390, 844]])("queues a separate selection without starting a suggestion (%s px)", (width, height) => {
    setViewport(width, height);
    const onQueue = vi.fn();
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    renderWithProviders(<StageDuringSuggestionModal
      hold={{ state: "drafting", paths: ["current.go"] }} paths={["next.go", "next_test.go"]}
      queuedSelections={2} onQueue={onQueue} onConfirm={onConfirm} onCancel={onCancel}
    />);
    const dialog = screen.getByRole("dialog", { name: "A commit suggestion is being drafted" });
    expect(within(dialog).getByText(/2 selections are already queued/)).toBeInTheDocument();
    fireEvent.click(within(dialog).getByTestId("stage-during-suggestion-queue"));
    expect(onQueue).toHaveBeenCalledOnce();
    expect(onConfirm).not.toHaveBeenCalled();
    expect(onCancel).not.toHaveBeenCalled();
  });

  it("names queued suggestions that an immediate stage will remove", () => {
    setViewport(1280, 800);
    renderWithProviders(<StageDuringSuggestionModal
      hold={null}
      paths={["src/a.go"]}
      queuedConflicts={[{ id: "queue-1", paths: ["src/a.go", "src/b.go"] }]}
      onConfirm={vi.fn()}
      onCancel={vi.fn()}
    />);
    expect(screen.getByRole("dialog", { name: "Queued suggestions use these files" })).toHaveTextContent("src/a.go, src/b.go");
    expect(screen.getByTestId("stage-during-suggestion-confirm")).toHaveTextContent("Stage + remove 1 queued");
  });

  it("disables queueing when requested files already belong to a queued selection and deduplicates paths", () => {
    setViewport(390, 844);
    renderWithProviders(<StageDuringSuggestionModal
      hold={{ state: "ready", paths: ["current.go"] }}
      paths={["src/a.go", "src/a.go", "src/next.go"]}
      queuedConflicts={[{ id: "queue-1", paths: ["src/a.go"] }]}
      onQueue={vi.fn()}
      onConfirm={vi.fn()}
      onCancel={vi.fn()}
    />);
    const dialog = screen.getByRole("dialog", { name: "A commit suggestion is ready" });
    expect(within(dialog).getByText("Stage 2 more files?")).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Queue files" })).toBeDisabled();
    expect(within(dialog).getByText(/Some files are already in the queued selections above/)).toBeInTheDocument();
    expect(within(dialog).getByLabelText("Files to stage").children).toHaveLength(2);
  });
});
