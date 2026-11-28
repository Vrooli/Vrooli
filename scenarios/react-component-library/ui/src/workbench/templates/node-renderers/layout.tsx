import type { CSSProperties, ReactNode } from "react";
import { cx } from "../../cx";
import { gapStyle, paddingStyle, type NodeRenderState } from "./shared";
export function renderLayoutNode(state: NodeRenderState): ReactNode {
  const { node, context: ctx, children, attrs, sourceStyle } = state;
  const props = node.props;
  switch (node.type) {
    case "page":
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx(
            "tpl-page",
            typeof props.sourceHtml === "string" && ctx.sourceView && "has-source",
            attrs.className,
          )}
          style={
            typeof props.sourceHtml === "string" && ctx.sourceView
              ? undefined
              : paddingStyle(props.pad)
                ? { ...paddingStyle(props.pad), ...sourceStyle }
                : sourceStyle
          }
        >
          {typeof props.sourceHtml === "string" && ctx.sourceView ? (
            <iframe
              className="tpl-source-frame"
              title={`${ctx.template.name} · original source`}
              sandbox=""
              referrerPolicy="no-referrer"
              srcDoc={props.sourceHtml}
            />
          ) : (
            children
          )}
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
            flexGrow: props.grow === true ? 1 : undefined,
            ...sourceStyle,
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
            ...sourceStyle,
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
              ...sourceStyle,
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
            backgroundImage: "none",
            border: "none",
            boxShadow: "none",
            ...sourceStyle,
          }}
        >
          {children}
        </div>
      );
    }
  }
  return null;
}
