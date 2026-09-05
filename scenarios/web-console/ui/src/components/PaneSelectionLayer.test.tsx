import { renderWithProviders as render } from "../test-utils";
import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { PaneSelectionLayer } from "./PaneSelectionLayer";

describe("PaneSelectionLayer", () => {
  it("keeps status and selection actions in pane chrome", () => {
    render(
      <PaneSelectionLayer
        contextMenu={{ x: 10, y: 20 }} hasSelection inputError="Rejected" paneStatus={{ kind: "reconnected" }}
        uploadError={null} uploading={false} uploadingLabel="Uploading" ttsSupported
        onCopy={vi.fn()} onPaste={vi.fn(async () => ({ status: "ok" as const }))} onSelectAll={vi.fn()}
        onClear={vi.fn()} onUploadImage={vi.fn()} onSpeak={vi.fn()} onClose={vi.fn()}
      >
        <div data-testid="terminal-content" />
      </PaneSelectionLayer>,
    );
    expect(screen.getByTestId("terminal-content")).toBeInTheDocument();
    expect(screen.getByTestId("input-error")).toHaveTextContent("Rejected");
    expect(screen.getByText("reconnected")).toBeInTheDocument();
  });

  it("offers a pane-only tmux mouse toggle when the backend supports it", () => {
    const onToggle = vi.fn();
    render(
      <PaneSelectionLayer
        contextMenu={{ x: 10, y: 20 }} hasSelection={false} inputError={null} paneStatus={null}
        uploadError={null} uploading={false} uploadingLabel="Uploading" ttsSupported={false}
        mouseMode={false} onToggleMouseMode={onToggle}
        onCopy={vi.fn()} onPaste={vi.fn(async () => ({ status: "ok" as const }))} onSelectAll={vi.fn()}
        onClear={vi.fn()} onUploadImage={vi.fn()} onSpeak={vi.fn()} onClose={vi.fn()}
      />,
    );
    const toggle = screen.getByTestId("ctx-mouse-mode");
    expect(toggle).toHaveTextContent("Enable tmux mouse mode (this pane only)");
    toggle.click();
    expect(onToggle).toHaveBeenCalledWith(true);
  });
});
