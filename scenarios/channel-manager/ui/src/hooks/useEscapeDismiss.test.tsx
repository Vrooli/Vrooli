import { describe, expect, it, vi } from "vitest";
import { fireEvent } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { useEscapeDismiss } from "./useEscapeDismiss";

const emitShortcutIntent = vi.fn();

vi.mock("@vrooli/iframe-bridge", () => ({ emitShortcutIntent: (...args: unknown[]) => emitShortcutIntent(...args) }));

function Harness({ open, onDismiss }: { open: boolean; onDismiss: () => void }) {
  useEscapeDismiss(open, onDismiss);
  return <div>ready</div>;
}

describe("useEscapeDismiss", () => {
  it("dismisses and relays a handled Escape intent", () => {
    const onDismiss = vi.fn();
    renderWithProviders(<Harness open={true} onDismiss={onDismiss} />);

    fireEvent.keyDown(window, { key: "Escape" });

    expect(onDismiss).toHaveBeenCalledTimes(1);
    expect(emitShortcutIntent).toHaveBeenCalledWith({ action: "channel-manager.dialog.close", outcome: "handled", chord: "Escape", source: "keyboard" });
  });
});
