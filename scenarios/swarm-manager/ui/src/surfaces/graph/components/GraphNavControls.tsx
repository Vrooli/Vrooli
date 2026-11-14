/**
 * GraphNavControls - On-screen pan/zoom buttons for TV and accessibility.
 *
 * Designed for environments where only a D-pad (up/down/left/right/OK) is
 * available (e.g. Google TV with a remote). All buttons are native <button>
 * elements so the TV browser's spatial navigation moves focus between them
 * via the D-pad, and OK activates the focused button.
 *
 * Toggled on/off via the "Show Nav Controls" setting in SettingsDrawer.
 * Off by default to avoid clutter on phone/desktop.
 */

import { useCallback } from "react";
import {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  Maximize2,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { selectors } from "../../../consts/selectors";

/** Pixels the viewport shifts per pan button click. */
export const PAN_AMOUNT = 100;

const NAV_BUTTON_CLASS =
  "rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50";

export function GraphNavControls() {
  const flowInstance = useGraphUIStore((s) => s.flowInstance);

  // React Flow viewport coordinates: increasing x shifts the visible area
  // to the LEFT (reveals content to the left in graph space). So "pan left"
  // = increase vp.x, "pan up" = increase vp.y.
  const panBy = useCallback(
    (dx: number, dy: number) => {
      if (!flowInstance) return;
      const vp = flowInstance.getViewport();
      void flowInstance.setViewport(
        { x: vp.x + dx, y: vp.y + dy, zoom: vp.zoom },
        { duration: 200 },
      );
    },
    [flowInstance],
  );

  const handleZoomIn = useCallback(() => {
    void flowInstance?.zoomIn({ duration: 200 });
  }, [flowInstance]);

  const handleZoomOut = useCallback(() => {
    void flowInstance?.zoomOut({ duration: 200 });
  }, [flowInstance]);

  const handleFitView = useCallback(() => {
    void flowInstance?.fitView({ padding: 0.2, maxZoom: 1.2, duration: 300 });
  }, [flowInstance]);

  return (
    <div
      className="flex items-center gap-1"
      data-testid={selectors.graphNavControls.container}
    >
      {/* D-pad cluster: left, stacked up/down, right */}
      <button
        type="button"
        onClick={() => panBy(PAN_AMOUNT, 0)}
        className={NAV_BUTTON_CLASS}
        aria-label="Pan left"
        data-testid={selectors.graphNavControls.panLeft}
      >
        <ArrowLeft className="h-4 w-4" />
      </button>
      <div className="flex flex-col gap-1">
        <button
          type="button"
          onClick={() => panBy(0, PAN_AMOUNT)}
          className={NAV_BUTTON_CLASS}
          aria-label="Pan up"
          data-testid={selectors.graphNavControls.panUp}
        >
          <ArrowUp className="h-4 w-4" />
        </button>
        <button
          type="button"
          onClick={() => panBy(0, -PAN_AMOUNT)}
          className={NAV_BUTTON_CLASS}
          aria-label="Pan down"
          data-testid={selectors.graphNavControls.panDown}
        >
          <ArrowDown className="h-4 w-4" />
        </button>
      </div>
      <button
        type="button"
        onClick={() => panBy(-PAN_AMOUNT, 0)}
        className={NAV_BUTTON_CLASS}
        aria-label="Pan right"
        data-testid={selectors.graphNavControls.panRight}
      >
        <ArrowRight className="h-4 w-4" />
      </button>

      {/* Separator */}
      <div className="mx-1 h-6 w-px bg-slate-700/50" />

      {/* Zoom cluster */}
      <button
        type="button"
        onClick={handleZoomOut}
        className={NAV_BUTTON_CLASS}
        aria-label="Zoom out"
        data-testid={selectors.graphNavControls.zoomOut}
      >
        <ZoomOut className="h-4 w-4" />
      </button>
      <button
        type="button"
        onClick={handleZoomIn}
        className={NAV_BUTTON_CLASS}
        aria-label="Zoom in"
        data-testid={selectors.graphNavControls.zoomIn}
      >
        <ZoomIn className="h-4 w-4" />
      </button>
      <button
        type="button"
        onClick={handleFitView}
        className={NAV_BUTTON_CLASS}
        aria-label="Fit to view"
        data-testid={selectors.graphNavControls.fitView}
      >
        <Maximize2 className="h-4 w-4" />
      </button>
    </div>
  );
}
