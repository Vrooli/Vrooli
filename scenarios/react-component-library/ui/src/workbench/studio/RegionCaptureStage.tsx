import type { RefObject } from "react";
import type { PointerEvent as ReactPointerEvent, KeyboardEvent as ReactKeyboardEvent } from "react";

import { HANDLES, percent, type DragState } from "./regionCaptureGeometry";
import type { Region, RegionBounds, ResizeHandle } from "./region";

interface RegionCaptureStageProps {
  asset: { name: string; url: string };
  imageRef: RefObject<HTMLImageElement>;
  stageRef: RefObject<HTMLDivElement>;
  marqueeRef: RefObject<HTMLDivElement>;
  natural: RegionBounds | null;
  region: Region | null;
  imageFailed: boolean;
  marqueeValue: string;
  regionCoversImage: boolean;
  imageAlt: string;
  marqueeLabel: string;
  loadingLabel: string;
  onImageLoad: () => void;
  onImageError: () => void;
  onBeginDrag: (
    event: ReactPointerEvent<HTMLElement>,
    mode: DragState["mode"],
    handle: ResizeHandle,
  ) => void;
  onPointerMove: (event: ReactPointerEvent<HTMLDivElement>) => void;
  onPointerEnd: (event: ReactPointerEvent<HTMLDivElement>) => void;
  onLostPointerCapture: () => void;
  onMarqueeKeyDown: (event: ReactKeyboardEvent<HTMLDivElement>) => void;
}

export function RegionCaptureStage({
  asset,
  imageRef,
  stageRef,
  marqueeRef,
  natural,
  region,
  imageFailed,
  marqueeValue,
  regionCoversImage,
  imageAlt,
  marqueeLabel,
  loadingLabel,
  onImageLoad,
  onImageError,
  onBeginDrag,
  onPointerMove,
  onPointerEnd,
  onLostPointerCapture,
  onMarqueeKeyDown,
}: RegionCaptureStageProps) {
  return (
    <div
      ref={stageRef}
      className="rc-stage"
      data-testid="region-capture-stage"
      onPointerDown={(event) => onBeginDrag(event, "new", "se")}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerEnd}
      onPointerCancel={onPointerEnd}
      onLostPointerCapture={onLostPointerCapture}
    >
      <img
        ref={imageRef}
        className="rc-image"
        src={asset.url}
        alt={imageAlt}
        draggable={false}
        onLoad={onImageLoad}
        onError={onImageError}
        data-testid="region-capture-image"
      />
      {region && natural ? (
        <div
          ref={marqueeRef}
          className="rc-marquee"
          role="slider"
          tabIndex={0}
          aria-label={marqueeLabel}
          aria-valuemin={0}
          aria-valuemax={Math.max(natural.width, natural.height)}
          aria-valuenow={region.w}
          aria-valuetext={marqueeValue}
          data-testid="region-capture-marquee"
          style={{
            left: percent(region.x, natural.width),
            top: percent(region.y, natural.height),
            width: percent(region.w, natural.width),
            height: percent(region.h, natural.height),
          }}
          onKeyDown={onMarqueeKeyDown}
          onPointerDown={(event) => {
            marqueeRef.current?.focus();
            onBeginDrag(event, regionCoversImage ? "new" : "move", "se");
          }}
        >
          {HANDLES.map((handle) => (
            <span
              key={handle}
              className={`rc-handle rc-handle-${handle}`}
              aria-hidden="true"
              data-testid={`region-capture-handle-${handle}`}
              onPointerDown={(event) => onBeginDrag(event, "resize", handle)}
            />
          ))}
        </div>
      ) : null}
      {!natural && !imageFailed ? (
        <span className="rc-loading" role="status">
          {loadingLabel}
        </span>
      ) : null}
    </div>
  );
}
