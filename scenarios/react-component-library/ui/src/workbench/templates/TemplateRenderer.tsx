/* eslint-disable jsx-a11y/no-static-element-interactions, jsx-a11y/click-events-have-key-events --
 * The artboard is a pointer convenience for selecting a layer. The layer tree
 * beside it is the complete keyboard path, so the canvas itself is not a
 * focusable control and must not pretend to be one. */
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from "react";
import { cx } from "../cx";
import { useTranslation } from "../../i18n";
import { TemplateNodeRenderer, type NodeSlotContext } from "./TemplateNodeRenderer";
import { TemplateSlotOverlay } from "./TemplateSlotOverlay";
import { useTemplateCanvasLayerDrag } from "./useTemplateCanvasLayerDrag";
import type { LayerDropPosition } from "../studio/layerMovement";
import {
  DEFAULT_SLOT_HEIGHT,
  findNode,
  findSlot,
  slotPlacement,
  templateStyle,
  type SlotPlacement,
  type Template,
  type TemplateNode,
} from "./spec";

interface RenderProps {
  template: Template;
  spec: TemplateNode;
  slot?: ReactNode;
  /** Renders per-node catalog bindings in authored slot positions. */
  renderComponentSlot?: (node: TemplateNode) => ReactNode;
  selectedId?: string;
  onSelect?: (id: string) => void;
  /** Reparents a layer when it is dragged onto another layer in the canvas. */
  onMoveLayer?: (sourceId: string, targetId: string, position: LayerDropPosition) => void;
  /** Paints a per-layer outline showing shared RCL primitives vs custom structure. */
  coverageMode?: boolean;
  /** Preview hides selection affordances and edit chrome. */
  mode?: "canvas" | "preview" | "thumb";
  /** Enables moving and resizing the live component inside the page. */
  slotEditable?: boolean;
  /** Controlled placement. Omit to let the renderer place the slot from its authored anchor. */
  slotPlacement?: SlotPlacement;
  onSlotPlacementChange?: (nodeId: string, placement: SlotPlacement) => void;
  /** Choose the slot that receives the live component. Defaults to the first. */
  activeSlotId?: string;
}

type SlotContext = NodeSlotContext;

export function TemplateRenderer({
  template,
  spec,
  slot,
  renderComponentSlot,
  selectedId,
  onSelect,
  onMoveLayer,
  coverageMode = false,
  mode = "canvas",
  slotEditable = false,
  slotPlacement: controlledPlacement,
  onSlotPlacementChange,
  activeSlotId,
}: RenderProps) {
  const { t } = useTranslation();
  const interactive = mode === "canvas" && Boolean(onSelect);
  const { rootRef, canvasDrag, onPointerDown, onPointerMove, onPointerUp, onClick } =
    useTemplateCanvasLayerDrag({ spec, interactive, onSelect, onMoveLayer });
  const anchorRef = useRef<HTMLDivElement>(null);
  const [measured, setMeasured] = useState<SlotPlacement | undefined>(undefined);
  const [active, setActive] = useState<SlotPlacement | undefined>(undefined);
  const [sourceView, setSourceView] = useState(false);
  const slotNode = useMemo(() => {
    const selected = activeSlotId ? findNode(spec, activeSlotId) : undefined;
    return selected?.type === "slot" ? selected : findSlot(spec);
  }, [activeSlotId, spec]);
  const authored = slotPlacement(slotNode);
  const placement = controlledPlacement ?? active ?? authored;
  const placed = Boolean(placement);
  const lastTemplate = useRef(template.id);
  const lastControlled = useRef(controlledPlacement);

  // A local drag owns the placement until the template changes or the owner
  // clears it. Without this reset, switching templates would keep the previous
  // page's coordinates and the reset action would not take effect.
  useEffect(() => {
    if (lastTemplate.current !== template.id) {
      lastTemplate.current = template.id;
      setActive(undefined);
      setMeasured(undefined);
      setSourceView(false);
    }
  }, [template.id]);

  useEffect(() => {
    if (lastControlled.current && !controlledPlacement) setActive(undefined);
    lastControlled.current = controlledPlacement;
  }, [controlledPlacement]);

  const scale = useCallback(() => {
    const root = rootRef.current;
    if (!root) return 1;
    const rect = root.getBoundingClientRect();
    return root.offsetWidth > 0 ? rect.width / root.offsetWidth : 1;
  }, [rootRef]);

  const measure = useCallback(() => {
    const root = rootRef.current;
    const anchor = anchorRef.current;
    if (!root || !anchor) return;
    const rootRect = root.getBoundingClientRect();
    const anchorRect = anchor.getBoundingClientRect();
    const factor = scale() || 1;
    setMeasured({
      x: Math.max(0, (anchorRect.left - rootRect.left) / factor),
      y: Math.max(0, (anchorRect.top - rootRect.top) / factor),
      w: Math.max(80, anchorRect.width / factor),
      h: Math.max(24, anchorRect.height / factor || DEFAULT_SLOT_HEIGHT),
    });
  }, [rootRef, scale]);

  // The anchor is the authored in-flow slot. Measure it only while the
  // component still sits in flow; once it is placed the anchor is a spacer.
  useLayoutEffect(() => {
    if (!placed) measure();
  }, [measure, placed, spec, mode, template.id]);

  useEffect(() => {
    if (placed || typeof ResizeObserver === "undefined") return;
    const anchor = anchorRef.current;
    if (!anchor) return;
    const observer = new ResizeObserver(measure);
    observer.observe(anchor);
    return () => observer.disconnect();
  }, [measure, placed, spec, mode, template.id]);

  const commit = useCallback(
    (next: SlotPlacement) => {
      const clamped: SlotPlacement = {
        x: Math.max(0, Math.round(next.x)),
        y: Math.max(0, Math.round(next.y)),
        w: Math.max(80, Math.round(next.w)),
        h: Math.max(24, Math.round(next.h)),
      };
      setActive(clamped);
      if (slotNode) onSlotPlacementChange?.(slotNode.id, clamped);
    },
    [onSlotPlacementChange, slotNode],
  );

  const beginDrag = useCallback(
    (event: ReactPointerEvent<HTMLElement>) => {
      if (!slotEditable || !slotNode) return;
      if (event.button !== 0 && event.pointerType === "mouse") return;
      const origin = active ?? controlledPlacement ?? authored ?? measured;
      if (!origin) return;
      event.preventDefault();
      event.stopPropagation();
      const startX = event.clientX;
      const startY = event.clientY;
      let committed = false;
      const onMove = (move: PointerEvent) => {
        const factor = scale() || 1;
        const dx = (move.clientX - startX) / factor;
        const dy = (move.clientY - startY) / factor;
        if (!committed && Math.hypot(dx, dy) < 2) return;
        committed = true;
        commit({ ...origin, x: origin.x + dx, y: origin.y + dy });
      };
      const onUp = () => {
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onUp);
        window.removeEventListener("pointercancel", onUp);
      };
      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", onUp);
      window.addEventListener("pointercancel", onUp);
    },
    [active, authored, commit, controlledPlacement, measured, scale, slotEditable, slotNode],
  );

  const beginResize = useCallback(
    (event: ReactPointerEvent<HTMLElement>) => {
      if (!slotEditable || !slotNode || !placement) return;
      if (event.button !== 0 && event.pointerType === "mouse") return;
      event.preventDefault();
      event.stopPropagation();
      const origin = placement;
      const startX = event.clientX;
      const startY = event.clientY;
      const onMove = (move: PointerEvent) => {
        const factor = scale() || 1;
        commit({
          ...origin,
          w: origin.w + (move.clientX - startX) / factor,
          h: origin.h + (move.clientY - startY) / factor,
        });
      };
      const onUp = () => {
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onUp);
        window.removeEventListener("pointercancel", onUp);
      };
      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", onUp);
      window.addEventListener("pointercancel", onUp);
    },
    [commit, placement, scale, slotEditable, slotNode],
  );

  const onKeyMove = useCallback(
    (event: ReactKeyboardEvent<HTMLElement>) => {
      if (!slotEditable || !slotNode) return;
      const deltas: Record<string, [number, number]> = {
        ArrowLeft: [-1, 0],
        ArrowRight: [1, 0],
        ArrowUp: [0, -1],
        ArrowDown: [0, 1],
      };
      const delta = deltas[event.key];
      if (!delta) return;
      event.preventDefault();
      const step = event.shiftKey ? 24 : 8;
      const origin = active ?? controlledPlacement ?? authored ?? measured;
      if (!origin) return;
      commit({ ...origin, x: origin.x + delta[0] * step, y: origin.y + delta[1] * step });
    },
    [active, authored, commit, controlledPlacement, measured, slotEditable, slotNode],
  );

  const slotContext: SlotContext | undefined = slot
    ? { placed, editing: slotEditable, onDragStart: beginDrag, onKeyMove }
    : undefined;
  const showOverlay = Boolean(slot && placement);

  return (
    <div
      ref={rootRef}
      className={cx(
        "tpl",
        mode === "thumb" && "is-thumb",
        mode === "preview" && "is-preview",
        slotEditable && "is-slot-editable",
        showOverlay && "has-floating-slot",
      )}
      style={templateStyle(template)}
      data-template={template.id}
      data-coverage-mode={coverageMode ? "true" : undefined}
      data-layer-dragging={canvasDrag ? "true" : undefined}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerUp}
      onClick={onClick}
    >
      {typeof spec.props.sourceHtml === "string" && mode === "canvas" ? (
        <button
          type="button"
          className="tpl-source-toggle"
          aria-pressed={sourceView}
          onClick={(event) => {
            event.stopPropagation();
            setSourceView((current) => !current);
          }}
        >
          {sourceView
            ? t("workbench.template.sourceViewLayers", { defaultValue: "Editable layers" })
            : t("workbench.template.sourceViewOriginal", { defaultValue: "Original source" })}
        </button>
      ) : null}
      <TemplateNodeRenderer
        node={spec}
        context={{
          template,
          slot,
          renderComponentSlot,
          selectedId,
          canvasDrag,
          coverageMode,
          interactive,
          mode,
          sourceView,
          slotContext,
          activeSlotId: slotNode?.id,
          labels: {
            move: t("workbench.template.move", { defaultValue: "Move" }),
            moveAria: t("workbench.template.moveComponent", {
              defaultValue: "Move component on the page",
            }),
            placed: t("workbench.template.componentPlaced", {
              defaultValue: "Component placed on the page",
            }),
            slot: t("workbench.template.componentSlot", { defaultValue: "Component slot" }),
            unknownType: t("workbench.template.unknownLayer", {
              defaultValue: "Unsupported layer type: {{type}}",
            }),
            imageEmpty: t("workbench.template.imageEmpty", {
              defaultValue: "Image unavailable",
            }),
          },
        }}
        anchorRef={anchorRef}
      />
      {showOverlay && placement ? (
        <TemplateSlotOverlay
          nodeId={slotNode?.id}
          placement={placement}
          slot={slot}
          editable={slotEditable}
          selected={selectedId === slotNode?.id}
          moveLabel={t("workbench.template.move", { defaultValue: "Move" })}
          moveAriaLabel={t("workbench.template.moveComponent", {
            defaultValue: "Move component on the page",
          })}
          resizeAriaLabel={t("workbench.template.resizeComponent", {
            defaultValue: "Resize component",
          })}
          onDragStart={beginDrag}
          onResizeStart={beginResize}
          onKeyMove={onKeyMove}
          onResizeKeyDown={(event) => {
            const deltas: Record<string, number> = { ArrowLeft: -8, ArrowRight: 8 };
            const delta = deltas[event.key];
            if (delta === undefined) return;
            event.preventDefault();
            commit({ ...placement, w: placement.w + delta });
          }}
          onMoveClick={(event) => event.stopPropagation()}
        />
      ) : null}
    </div>
  );
}
