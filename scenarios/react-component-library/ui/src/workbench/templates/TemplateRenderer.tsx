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
  type CSSProperties,
  type KeyboardEvent as ReactKeyboardEvent,
  type MouseEvent,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from "react";
import { Avatar } from "@vrooli/react-component-library/Avatar/1";
import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card } from "@vrooli/react-component-library/Card/1";
import { Chip } from "@vrooli/react-component-library/Chip/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Stat } from "@vrooli/react-component-library/Stat/1";

import { cx } from "../cx";
import { useTranslation } from "../../i18n";
import {
  DEFAULT_SLOT_HEIGHT,
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
  selectedId?: string;
  onSelect?: (id: string) => void;
  /** Preview hides selection affordances and edit chrome. */
  mode?: "canvas" | "preview" | "thumb";
  /** Enables moving and resizing the live component inside the page. */
  slotEditable?: boolean;
  /** Controlled placement. Omit to let the renderer place the slot from its authored anchor. */
  slotPlacement?: SlotPlacement;
  onSlotPlacementChange?: (nodeId: string, placement: SlotPlacement) => void;
}

const toneClass = (tone: unknown): string => (typeof tone === "string" ? `is-${tone}` : "");

function paddingStyle(pad: unknown): CSSProperties | undefined {
  if (typeof pad === "number") return { padding: pad };
  if (typeof pad === "string") return { padding: pad };
  return undefined;
}

function gapStyle(gap: unknown, fallback = 12): CSSProperties {
  return { gap: typeof gap === "number" ? gap : fallback };
}

interface SlotContext {
  placed: boolean;
  editing: boolean;
  onDragStart: (event: ReactPointerEvent<HTMLElement>) => void;
  onKeyMove: (event: ReactKeyboardEvent<HTMLElement>) => void;
}

export function TemplateRenderer({
  template,
  spec,
  slot,
  selectedId,
  onSelect,
  mode = "canvas",
  slotEditable = false,
  slotPlacement: controlledPlacement,
  onSlotPlacementChange,
}: RenderProps) {
  const { t } = useTranslation();
  const interactive = mode === "canvas" && Boolean(onSelect);
  const rootRef = useRef<HTMLDivElement>(null);
  const anchorRef = useRef<HTMLDivElement>(null);
  const [measured, setMeasured] = useState<SlotPlacement | undefined>(undefined);
  const [active, setActive] = useState<SlotPlacement | undefined>(undefined);
  const slotNode = useMemo(() => findSlot(spec), [spec]);
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
    }
  }, [template.id]);

  useEffect(() => {
    if (lastControlled.current && !controlledPlacement) setActive(undefined);
    lastControlled.current = controlledPlacement;
  }, [controlledPlacement]);

  const handleClick = (event: MouseEvent<HTMLDivElement>) => {
    if (!interactive) return;
    const target = (event.target as HTMLElement).closest<HTMLElement>("[data-node-id]");
    if (target?.dataset.nodeId) {
      event.stopPropagation();
      onSelect?.(target.dataset.nodeId);
    }
  };

  const scale = useCallback(() => {
    const root = rootRef.current;
    if (!root) return 1;
    const rect = root.getBoundingClientRect();
    return root.offsetWidth > 0 ? rect.width / root.offsetWidth : 1;
  }, []);

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
  }, [scale]);

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
      onClick={handleClick}
    >
      {renderNode(
        spec,
        {
          template,
          slot,
          selectedId,
          interactive,
          mode,
          slotContext,
          labels: {
            move: t("workbench.template.move", { defaultValue: "Move" }),
            moveAria: t("workbench.template.moveComponent", {
              defaultValue: "Move component on the page",
            }),
            placed: t("workbench.template.componentPlaced", {
              defaultValue: "Component placed on the page",
            }),
            slot: t("workbench.template.componentSlot", { defaultValue: "Component slot" }),
          },
        },
        anchorRef,
      )}
      {showOverlay && placement ? (
        <div
          className={cx(
            "tpl-slot-float",
            slotEditable && "is-editable",
            selectedId === slotNode?.id && "is-selected",
          )}
          data-node-id={slotNode?.id}
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
          {slotEditable ? (
            <>
              <button
                type="button"
                className="tpl-slot-grip"
                aria-label={t("workbench.template.moveComponent", {
                  defaultValue: "Move component on the page",
                })}
                onPointerDown={beginDrag}
                onKeyDown={onKeyMove}
                onClick={(event) => event.stopPropagation()}
              >
                {t("workbench.template.move", { defaultValue: "Move" })}
              </button>
              <button
                type="button"
                className="tpl-slot-resize"
                aria-label={t("workbench.template.resizeComponent", {
                  defaultValue: "Resize component",
                })}
                onPointerDown={beginResize}
                onKeyDown={(event) => {
                  const deltas: Record<string, number> = { ArrowLeft: -8, ArrowRight: 8 };
                  const delta = deltas[event.key];
                  if (delta === undefined) return;
                  event.preventDefault();
                  commit({ ...placement, w: placement.w + delta });
                }}
              />
              <span className="tpl-slot-size">
                {Math.round(placement.w)} × {Math.round(placement.h)}
              </span>
            </>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

interface NodeContext {
  template: Template;
  slot?: ReactNode;
  selectedId?: string;
  interactive: boolean;
  mode: "canvas" | "preview" | "thumb";
  slotContext?: SlotContext;
  labels: { move: string; moveAria: string; placed: string; slot: string };
}

function nodeAttrs(node: TemplateNode, ctx: NodeContext) {
  return {
    "data-node-id": node.id,
    "data-node-type": node.type,
    "data-selected": ctx.selectedId === node.id ? "true" : undefined,
    className: cx(ctx.selectedId === node.id && "is-selected"),
  };
}

function renderNode(
  node: TemplateNode,
  ctx: NodeContext,
  anchorRef?: { current: HTMLDivElement | null },
): ReactNode {
  const props = node.props;
  const children = node.children?.map((child) => renderNode(child, ctx, anchorRef));
  const attrs = nodeAttrs(node, ctx);
  const textAlign =
    typeof props.align === "string" ? (props.align as CSSProperties["textAlign"]) : undefined;

  switch (node.type) {
    case "page":
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx("tpl-page", attrs.className)}
          style={paddingStyle(props.pad)}
        >
          {children}
        </div>
      );

    case "stack":
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx("tpl-stack", attrs.className)}
          style={{
            ...gapStyle(props.gap),
            ...paddingStyle(props.pad),
            alignItems: typeof props.align === "string" ? props.align : undefined,
            justifyItems: typeof props.justify === "string" ? props.justify : undefined,
          }}
        >
          {children}
        </div>
      );

    case "row":
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx(
            "tpl-row",
            props.wrap === true && "is-wrap",
            props.border === true && "is-bordered",
            attrs.className,
          )}
          style={{
            ...gapStyle(props.gap, 10),
            ...paddingStyle(props.pad),
            alignItems: typeof props.align === "string" ? props.align : "center",
            justifyContent: typeof props.justify === "string" ? props.justify : undefined,
            minHeight: typeof props.h === "number" ? props.h : undefined,
          }}
        >
          {children}
        </div>
      );

    case "grid":
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx("tpl-grid", attrs.className)}
          style={
            {
              ...gapStyle(props.gap),
              ...paddingStyle(props.pad),
              "--tpl-cols": typeof props.cols === "number" ? props.cols : 2,
              alignItems: typeof props.align === "string" ? props.align : undefined,
            } as CSSProperties
          }
        >
          {children}
        </div>
      );

    case "surface": {
      const bg =
        props.bg === "accent"
          ? "var(--t-accent)"
          : props.bg === "surfaceAlt"
            ? "var(--t-surface-alt)"
            : "var(--t-surface)";
      const ink = props.bg === "accent" ? "var(--t-accent-ink)" : "var(--t-ink)";
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx("tpl-surface", attrs.className)}
          style={{
            ...paddingStyle(props.pad),
            background: bg,
            color: ink,
            borderRadius: typeof props.radius === "number" ? props.radius : "var(--t-radius)",
            minHeight: typeof props.h === "number" ? props.h : undefined,
            position: "relative",
            overflow: "hidden",
          }}
        >
          {children}
        </div>
      );
    }

    case "text":
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx(
            "tpl-text",
            `is-${String(props.variant ?? "body")}`,
            toneClass(props.tone),
            attrs.className,
          )}
          style={{ textAlign }}
        >
          {typeof props.text === "string" ? props.text : null}
        </div>
      );

    case "button":
      return (
        <span key={node.id} {...attrs} className={cx("tpl-control", attrs.className)}>
          <Button
            variant={
              props.variant === "primary"
                ? "primary"
                : props.variant === "secondary"
                  ? "secondary"
                  : "ghost"
            }
            size={props.size === "sm" ? "sm" : "md"}
            shape="square"
            density="compact"
          >
            {typeof props.text === "string" ? props.text : null}
          </Button>
        </span>
      );

    case "input":
      return (
        <span key={node.id} {...attrs} className={cx("tpl-control", attrs.className)}>
          <Input
            aria-label={typeof props.placeholder === "string" ? props.placeholder : "Field"}
            placeholder={typeof props.placeholder === "string" ? props.placeholder : undefined}
            data-node-id={node.id}
          />
        </span>
      );

    case "card":
      return (
        <Card
          key={node.id}
          {...attrs}
          className={cx("tpl-card", attrs.className)}
          style={{
            ...paddingStyle(props.pad),
            borderRadius: typeof props.radius === "number" ? props.radius : "var(--t-radius)",
            background: props.bg === "surfaceAlt" ? "var(--t-surface-alt)" : "var(--t-surface)",
            boxShadow: props.elevated === true ? "0 18px 50px rgba(0,0,0,.14)" : undefined,
          }}
        >
          {children}
        </Card>
      );

    case "stat":
      return (
        <span key={node.id} {...attrs} className={cx("tpl-stat", attrs.className)}>
          <Stat
            label={typeof props.label === "string" ? props.label : undefined}
            value={typeof props.value === "string" ? props.value : undefined}
            trend={typeof props.detail === "string" ? props.detail : undefined}
            trendTone={
              props.tone === "red"
                ? "negative"
                : props.tone === "amber"
                  ? "neutral"
                  : props.tone === "green"
                    ? "positive"
                    : "neutral"
            }
            scale="card"
            provenance="sample"
          />
        </span>
      );

    case "avatar":
      return (
        <span key={node.id} {...attrs} className={cx("tpl-control", attrs.className)}>
          <Avatar
            name={typeof props.name === "string" ? props.name : "Member"}
            size={props.size === "lg" ? "lg" : props.size === "xs" ? "xs" : "sm"}
            shape="circle"
          />
        </span>
      );

    case "chip":
      return (
        <span key={node.id} {...attrs} className={cx("tpl-control", attrs.className)}>
          <Chip>{typeof props.text === "string" ? props.text : null}</Chip>
        </span>
      );

    case "meter":
      return (
        <span key={node.id} {...attrs} className={cx("tpl-meter", attrs.className)}>
          <BoundedMeter
            label={typeof props.label === "string" ? props.label : undefined}
            ariaLabel={`${typeof props.label === "string" ? props.label : "Composition"} meter · ${node.id}`}
            value={typeof props.value === "number" ? props.value : 0}
            max={100}
            tone={
              props.tone === "green"
                ? "success"
                : props.tone === "amber"
                  ? "warning"
                  : props.tone === "red"
                    ? "danger"
                    : "neutral"
            }
            valueText={`${typeof props.value === "number" ? props.value : 0}%`}
          />
        </span>
      );

    case "divider":
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx("tpl-divider", attrs.className)}
          style={{ marginBlock: typeof props.space === "number" ? props.space : 8 }}
        />
      );

    case "spacer":
      return (
        <div
          key={node.id}
          {...attrs}
          aria-hidden
          style={{ height: typeof props.h === "number" ? props.h : 16 }}
        />
      );

    case "swatch": {
      const tone =
        props.tone === "accent"
          ? "var(--t-accent)"
          : props.tone === "secondary"
            ? "var(--t-secondary)"
            : props.tone === "line"
              ? "var(--t-line)"
              : "var(--t-surface-alt)";
      const shape = typeof props.shape === "string" ? props.shape : undefined;
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx("tpl-swatch", attrs.className)}
          data-shape={shape}
          style={{
            height: typeof props.h === "number" ? props.h : 48,
            width: typeof props.w === "number" ? props.w : undefined,
            flexGrow: props.grow === true ? 1 : undefined,
            backgroundColor: tone,
            borderRadius: typeof props.radius === "number" ? props.radius : 6,
          }}
        >
          {shape ? (
            <span className="tpl-art" aria-hidden>
              <span className={`tpl-art-shape is-${shape}`} />
            </span>
          ) : null}
        </div>
      );
    }

    case "slot": {
      const slotState = ctx.slotContext;
      const placed = slotState?.placed ?? false;
      const editing = slotState?.editing ?? false;
      return (
        <div
          key={node.id}
          ref={anchorRef}
          {...attrs}
          className={cx(
            "tpl-slot",
            ctx.slot ? "is-filled" : "is-empty",
            placed && "is-placed-anchor",
            editing && !placed && "is-editable",
            attrs.className,
          )}
          style={{ minHeight: typeof props.minH === "number" ? props.minH : 44 }}
        >
          {placed ? (
            editing || ctx.mode === "canvas" ? (
              <span className="tpl-slot-hint">{ctx.labels.placed}</span>
            ) : null
          ) : (
            (ctx.slot ?? (
              <span className="tpl-slot-hint">
                {typeof props.hint === "string" ? props.hint : ctx.labels.slot}
              </span>
            ))
          )}
          {editing && !placed && slotState ? (
            <button
              type="button"
              className="tpl-slot-grip is-inline"
              aria-label={ctx.labels.moveAria}
              onPointerDown={slotState.onDragStart}
              onKeyDown={slotState.onKeyMove}
              onClick={(event) => event.stopPropagation()}
            >
              {ctx.labels.move}
            </button>
          ) : null}
        </div>
      );
    }

    default:
      return null;
  }
}
