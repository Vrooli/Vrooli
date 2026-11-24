import type { KeyboardEvent, MouseEvent, PointerEvent, ReactNode } from "react";
import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { cx } from "../cx";
import type { SlotPlacement } from "./spec";

interface TemplateSlotOverlayProps {
  nodeId?: string;
  placement: SlotPlacement;
  slot: ReactNode;
  editable: boolean;
  selected: boolean;
  moveLabel: string;
  moveAriaLabel: string;
  resizeAriaLabel: string;
  onDragStart: (event: PointerEvent<HTMLElement>) => void;
  onResizeStart: (event: PointerEvent<HTMLElement>) => void;
  onKeyMove: (event: KeyboardEvent<HTMLElement>) => void;
  onResizeKeyDown: (event: KeyboardEvent<HTMLButtonElement>) => void;
  onMoveClick: (event: MouseEvent<HTMLButtonElement>) => void;
}

export function TemplateSlotOverlay({
  nodeId,
  placement,
  slot,
  editable,
  selected,
  moveLabel,
  moveAriaLabel,
  resizeAriaLabel,
  onDragStart,
  onResizeStart,
  onKeyMove,
  onResizeKeyDown,
  onMoveClick,
}: TemplateSlotOverlayProps) {
  return (
    <div
      className={cx("tpl-slot-float", editable && "is-editable", selected && "is-selected")}
      data-node-id={nodeId}
      data-node-type="slot"
      style={{
        left: `min(${Math.round(placement.x)}px, calc(100% - ${Math.round(placement.w)}px))`,
        top: Math.round(placement.y),
        width: `min(${Math.round(placement.w)}px, 100%)`,
      }}
    >
      <div className="tpl-slot-float-body" style={{ height: placement.h }}>
        {slot}
      </div>
      {editable ? (
        <>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            shape="square"
            density="compact"
            denseTapTarget
            className="tpl-slot-grip"
            aria-label={moveAriaLabel}
            onPointerDown={onDragStart}
            onKeyDown={onKeyMove}
            onClick={onMoveClick}
          >
            {moveLabel}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            shape="square"
            density="compact"
            denseTapTarget
            className="tpl-slot-resize"
            aria-label={resizeAriaLabel}
            onPointerDown={onResizeStart}
            onKeyDown={onResizeKeyDown}
          >
            <span aria-hidden="true" />
          </Button>
          <span className="tpl-slot-size">
            {Math.round(placement.w)} × {Math.round(placement.h)}
          </span>
        </>
      ) : null}
    </div>
  );
}
