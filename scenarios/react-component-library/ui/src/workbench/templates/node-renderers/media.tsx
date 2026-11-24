import type { ReactNode } from "react";
import { API_BASE } from "../../../api/client";
import { cx } from "../../cx";
import type { NodeRenderState } from "./shared";
export function renderMediaNode(state: NodeRenderState): ReactNode {
  const { node, context: ctx, children, attrs, sourceStyle, anchorRef } = state;
  const props = node.props;
  switch (node.type) {
    case "divider":
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx("tpl-divider", attrs.className)}
          style={{ marginBlock: typeof props.space === "number" ? props.space : 8, ...sourceStyle }}
        />
      );
    case "spacer":
      return (
        <div
          key={node.id}
          {...attrs}
          aria-hidden
          style={{ height: typeof props.h === "number" ? props.h : 16, ...sourceStyle }}
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
            ...sourceStyle,
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
      const activeSlot = !ctx.activeSlotId || ctx.activeSlotId === node.id;
      const activePlaced = activeSlot && placed;
      const hasFallback = Boolean(children && children.length > 0);
      const boundComponent = ctx.renderComponentSlot?.(node);
      return (
        <div
          key={node.id}
          ref={activeSlot ? anchorRef : undefined}
          {...attrs}
          className={cx(
            "tpl-slot",
            boundComponent || (activeSlot && ctx.slot)
              ? "is-filled"
              : hasFallback
                ? "is-fallback"
                : "is-empty",
            activePlaced && "is-placed-anchor",
            editing && !placed && "is-editable",
            attrs.className,
          )}
          style={{ minHeight: typeof props.minH === "number" ? props.minH : 44, ...sourceStyle }}
        >
          {activePlaced ? (
            editing || ctx.mode === "canvas" ? (
              <span className="tpl-slot-hint">{ctx.labels.placed}</span>
            ) : null
          ) : boundComponent ? (
            boundComponent
          ) : activeSlot ? (
            (ctx.slot ?? (hasFallback ? children : null))
          ) : hasFallback ? (
            children
          ) : null}
          {editing && activeSlot && !placed && slotState ? (
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
    case "image": {
      const raw = typeof props.src === "string" ? props.src : "";
      const src = raw.startsWith("/") ? `${API_BASE.replace(/\/$/, "")}${raw}` : raw;
      const fit = props.fit === "contain" ? "contain" : "cover";
      return (
        <figure
          key={node.id}
          {...attrs}
          className={cx("tpl-image", attrs.className)}
          style={{
            height: typeof props.h === "number" ? props.h : 220,
            // Imported artwork often sits in a vertically constrained flex
            // stack. Preserve its authored height so the illustration cannot
            // collapse and leave a blank gap in the composition.
            flexShrink: sourceStyle?.height !== undefined ? 0 : undefined,
            // Imported source images often represent transparent SVG artwork,
            // not a framed RCL image card. The source declarations win when
            // the authored layer explicitly supplies a background or radius.
            ...(sourceStyle ? { background: "transparent", borderRadius: 0 } : {}),
            ...sourceStyle,
          }}
        >
          {src ? (
            <img
              src={src}
              alt={typeof props.alt === "string" ? props.alt : ""}
              loading="lazy"
              style={{ objectFit: fit }}
            />
          ) : (
            <span className="tpl-image-empty">{ctx.labels.imageEmpty}</span>
          )}
        </figure>
      );
    }
    default: {
      // A typo or an imported node type must be visible, not silently dropped:
      // a blank region is much harder to diagnose than a labelled placeholder.
      const copy = typeof props.text === "string" ? props.text : "";
      return (
        <div key={node.id} {...attrs} className={cx("tpl-unknown", attrs.className)} role="note">
          <span>{ctx.labels.unknownType.replace("{{type}}", node.type)}</span>
          {copy ? <span className="tpl-unknown-copy">{copy}</span> : null}
        </div>
      );
    }
  }
  return null;
}
