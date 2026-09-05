import { useRef, useState } from "react";

import { SegmentedControl } from "../../components/ui/segmented-control";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  applyAspect,
  clampRect,
  contentRect,
  displayPointToImage,
  imageRectToDisplay,
  roundRect,
  type Point,
  type Rect,
  type Size,
} from "./cropMath";
import { ASPECT_LABEL } from "./opCatalog";

/** Aspect presets → width/height ratio (0 = free, -1 = original image ratio). */
const ASPECTS = [
  { value: "free", ratio: 0 },
  { value: "square", ratio: 1 },
  { value: "standard", ratio: 4 / 3 },
  { value: "wide", ratio: 16 / 9 },
  { value: "original", ratio: -1 },
] as const;

type AspectValue = (typeof ASPECTS)[number]["value"];

const CORNERS = ["nw", "ne", "sw", "se"] as const;
type Corner = (typeof CORNERS)[number];

const NUDGE = 1;

export interface CropOverlayProps {
  /** The image's natural (pixel) dimensions. */
  natural: Size;
  /** The displayed element's client size (object-contain box). */
  client: Size;
  /** Current crop rect in image pixels. */
  rect: Rect;
  onChange: (rect: Rect) => void;
}

/**
 * A draggable crop box overlaid on the canvas image. The body drags to move,
 * four corner handles resize, and arrow keys nudge the focused box — every
 * gesture also has the numeric Advanced-fields fallback in the Inspector.
 * Aspect presets snap the box. All geometry lives in `cropMath`, so this
 * component is pointer/keyboard plumbing + positioning only.
 */
export function CropOverlay({ natural, client, rect, onChange }: CropOverlayProps) {
  const { t } = useTranslation();
  const [aspect, setAspect] = useState<AspectValue>("free");
  const boxRef = useRef<HTMLButtonElement>(null);
  const content = contentRect(natural, client);
  const display = imageRectToDisplay(clampRect(rect, natural), content);

  const ratioFor = (value: AspectValue): number => {
    const found = ASPECTS.find((a) => a.value === value);
    if (!found || found.ratio === 0) return 0;
    if (found.ratio === -1) {
      return natural.height > 0 ? natural.width / natural.height : 0;
    }
    return found.ratio;
  };

  const commit = (next: Rect, withAspect: AspectValue = aspect) => {
    const ratio = ratioFor(withAspect);
    const shaped = ratio > 0 ? applyAspect(next, ratio, natural) : clampRect(next, natural);
    onChange(roundRect(shaped));
  };

  /** Element-local display point from a viewport pointer event. */
  const localPoint = (event: PointerEvent): Point | null => {
    const parent = boxRef.current?.parentElement;
    if (!parent) return null;
    const box = parent.getBoundingClientRect();
    return { x: event.clientX - box.left, y: event.clientY - box.top };
  };

  const beginDrag = (
    capture: HTMLElement,
    pointerId: number,
    onMove: (image: Point) => void,
  ) => {
    capture.setPointerCapture(pointerId);
    const onPointerMove = (move: PointerEvent) => {
      const local = localPoint(move);
      if (local) {
        onMove(displayPointToImage(local, content));
      }
    };
    const onPointerUp = () => {
      capture.releasePointerCapture(pointerId);
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", onPointerUp);
    };
    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", onPointerUp);
  };

  const startMove = (event: React.PointerEvent<HTMLButtonElement>) => {
    event.preventDefault();
    const origin = { ...rect };
    const grabLocal = { x: event.clientX, y: event.clientY };
    const parent = boxRef.current?.parentElement?.getBoundingClientRect();
    const grabImage = parent
      ? displayPointToImage({ x: grabLocal.x - parent.left, y: grabLocal.y - parent.top }, content)
      : { x: origin.x, y: origin.y };
    beginDrag(event.currentTarget, event.pointerId, (image) => {
      commit({ ...origin, x: origin.x + (image.x - grabImage.x), y: origin.y + (image.y - grabImage.y) });
    });
  };

  const startResize = (corner: Corner) => (event: React.PointerEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    const origin = { ...rect };
    const right = origin.x + origin.width;
    const bottom = origin.y + origin.height;
    beginDrag(event.currentTarget, event.pointerId, (p) => {
      let { x, y, width, height } = origin;
      if (corner === "nw") {
        x = p.x;
        y = p.y;
        width = right - p.x;
        height = bottom - p.y;
      } else if (corner === "ne") {
        y = p.y;
        width = p.x - origin.x;
        height = bottom - p.y;
      } else if (corner === "sw") {
        x = p.x;
        width = right - p.x;
        height = p.y - origin.y;
      } else {
        width = p.x - origin.x;
        height = p.y - origin.y;
      }
      commit({ x, y, width, height });
    });
  };

  const onKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    const deltas: Record<string, { dx: number; dy: number }> = {
      ArrowLeft: { dx: -NUDGE, dy: 0 },
      ArrowRight: { dx: NUDGE, dy: 0 },
      ArrowUp: { dx: 0, dy: -NUDGE },
      ArrowDown: { dx: 0, dy: NUDGE },
    };
    const delta = deltas[event.key];
    if (!delta) return;
    event.preventDefault();
    commit({ ...rect, x: rect.x + delta.dx, y: rect.y + delta.dy });
  };

  return (
    <>
      {/* The box body is a real <button> (interactive: keyboard + focus) so the
          resize handles can sit beside it without nesting interactive elements
          (invalid HTML + jsx-a11y violation). They overlay the box's corners. */}
      <button
        ref={boxRef}
        type="button"
        data-testid={selectors.workspace.crop.box}
        aria-label={t(strings.workspace.crop.boxLabel)}
        onKeyDown={onKeyDown}
        onPointerDown={startMove}
        className="absolute cursor-move border-2 border-app-primary bg-app-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
        style={{
          left: `${display.x}px`,
          top: `${display.y}px`,
          width: `${display.width}px`,
          height: `${display.height}px`,
        }}
      />
      {CORNERS.map((corner) => {
        const atLeft = corner === "nw" || corner === "sw";
        const atTop = corner === "nw" || corner === "ne";
        return (
          <button
            key={corner}
            type="button"
            data-testid={selectors.workspace.crop.handle({ corner })}
            aria-label={t(strings.workspace.crop.boxLabel)}
            onPointerDown={startResize(corner)}
            className="absolute h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-pill border border-app-surface bg-app-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
            style={{
              left: `${atLeft ? display.x : display.x + display.width}px`,
              top: `${atTop ? display.y : display.y + display.height}px`,
              cursor: corner === "nw" || corner === "se" ? "nwse-resize" : "nesw-resize",
            }}
          />
        );
      })}
      <div className="absolute left-0 top-0 z-10 m-2">
        <SegmentedControl<AspectValue>
          label={t(strings.workspace.crop.aspect.label)}
          value={aspect}
          options={ASPECTS.map((a) => ({
            value: a.value,
            label: t(ASPECT_LABEL[a.value]),
          }))}
          onChange={(value) => {
            setAspect(value);
            commit(rect, value);
          }}
          data-testid={selectors.workspace.crop.aspect}
        />
      </div>
    </>
  );
}
