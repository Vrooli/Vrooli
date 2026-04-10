/**
 * CapturePanel - Floating panel for quick capture input.
 *
 * Opens via the capture FAB. Uses FloatingPanel for draggable desktop /
 * bottom-sheet mobile behavior (same pattern as SettingsDrawer).
 */

import { FloatingPanel } from "../../../components/ui/floating-panel";
import { QuickCaptureInput } from "../../../components/capture/quick-capture-input";

interface CapturePanelProps {
  isOpen: boolean;
  onClose: () => void;
}

const INITIAL_POSITION = { x: Math.max(8, window.innerWidth - 520), y: Math.max(8, window.innerHeight - 320) };

export function CapturePanel({ isOpen, onClose }: CapturePanelProps) {
  return (
    <FloatingPanel
      isOpen={isOpen}
      onClose={onClose}
      title="Quick Capture"
      initialPosition={INITIAL_POSITION}
      className="max-w-lg"
      testId="capture-panel"
    >
      <QuickCaptureInput />
    </FloatingPanel>
  );
}
