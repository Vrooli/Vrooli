import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import MessagesFileViewer from "./MessagesFileViewer";
import { IDLE_PREVIEW_STATE } from "./file-preview/types";

describe("linked file preview recovery", () => {
  it("shows a retry action when resolving a deep-linked path fails", () => {
    const onReopen = vi.fn();
    renderWithProviders(<MessagesFileViewer
      state={{ ...IDLE_PREVIEW_STATE, open: true, status: "error", requestedPath: "/workspace/missing.md", error: "File not found" }}
      onClose={vi.fn()}
      onReopen={onReopen}
      onRendererError={vi.fn()}
      onNavigate={vi.fn()}
      onNavigateBack={vi.fn()}
      onLoadMore={vi.fn()}
      onListOptionsChange={vi.fn()}
    />);

    expect(screen.getByText("File not found")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("file-preview-reopen"));
    expect(onReopen).toHaveBeenCalledOnce();
  });
});
