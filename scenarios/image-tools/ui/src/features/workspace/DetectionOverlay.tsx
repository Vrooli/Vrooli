import type { AnalyzeBox } from "../../api/analysis";
import { selectors } from "../../consts/selectors";
import { contentRect, imageRectToDisplay, type Size } from "./cropMath";

/** One labeled region to draw over the canvas image (e.g. an OCR text block). */
export interface DetectionBox {
  id: string;
  /** Short label shown on the box (truncated); the full data lives in the panel. */
  label: string;
  /** The region in the analyzed image's pixel space. */
  box: AnalyzeBox;
}

export interface DetectionOverlayProps {
  /** The image's natural (pixel) dimensions. */
  natural: Size;
  /** The displayed element's client size (object-contain box). */
  client: Size;
  boxes: readonly DetectionBox[];
}

/**
 * Draws read-only labeled rectangles over the canvas image, mapped 1:1 from the
 * analyzed image's pixel space to the on-screen object-contain box (shared
 * geometry with the crop overlay). Decorative + non-interactive: the same
 * regions and their text are listed accessibly in the Analyze panel, so the
 * overlay is `aria-hidden` and `pointer-events-none` (it never steals the
 * canvas's pan/zoom or focus).
 */
export function DetectionOverlay({ natural, client, boxes }: DetectionOverlayProps) {
  const content = contentRect(natural, client);

  return (
    <div
      data-testid={selectors.workspace.analyze.overlay}
      aria-hidden="true"
      className="pointer-events-none absolute inset-0"
    >
      {boxes.map((entry) => {
        const display = imageRectToDisplay(entry.box, content);
        return (
          <div
            key={entry.id}
            className="absolute rounded-sm border-2 border-app-accent/80 bg-app-accent/10"
            style={{
              left: `${display.x}px`,
              top: `${display.y}px`,
              width: `${display.width}px`,
              height: `${display.height}px`,
            }}
          >
            {entry.label && (
              <span className="absolute left-0 top-0 max-w-full -translate-y-full truncate rounded-t-sm bg-app-accent px-1 text-[10px] font-medium leading-tight text-app-surface">
                {entry.label}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
}
