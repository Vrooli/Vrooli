import type { ReactNode } from "react";
import { Avatar } from "@vrooli/react-component-library/Avatar/1.3.16";
import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1.0.10";
import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { Card } from "@vrooli/react-component-library/Card/1.3.1";
import { Chip } from "@vrooli/react-component-library/Chip/1.0.7";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { Stat } from "@vrooli/react-component-library/Stat/1.0.8";
import { TemplateIcon } from "../icons";
import { cx } from "../../cx";
import {
  controlContentStyle,
  importedButtonStyle,
  importedButtonUsesDenseTapTarget,
  importedControlStyle,
  paddingStyle,
  toneClass,
  type NodeRenderState,
} from "./shared";
export function renderContentNode(state: NodeRenderState): ReactNode {
  const { node, children, attrs, sourceStyle, textAlign } = state;
  const props = node.props;
  switch (node.type) {
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
          style={{
            textAlign,
            flexGrow: props.grow === true ? 1 : undefined,
            minWidth: 0,
            ...sourceStyle,
          }}
        >
          {typeof props.text === "string" ? props.text : null}
          {children}
        </div>
      );
    case "button":
      return (
        <span
          key={node.id}
          {...attrs}
          className={cx("tpl-control", attrs.className)}
          style={sourceStyle}
        >
          <Button
            variant={
              props.variant === "primary"
                ? "primary"
                : props.variant === "secondary"
                  ? "secondary"
                  : "ghost"
            }
            size={props.size === "sm" ? "sm" : "md"}
            shape={props.shape === "pill" ? "pill" : "square"}
            density="compact"
            denseTapTarget={importedButtonUsesDenseTapTarget(sourceStyle)}
            aria-pressed={typeof props.pressed === "boolean" ? props.pressed : undefined}
            style={importedButtonStyle(sourceStyle)}
          >
            {typeof props.text === "string" ? props.text : null}
            {children ? (
              <span className="tpl-control-content" style={controlContentStyle(sourceStyle)}>
                {children}
              </span>
            ) : null}
          </Button>
        </span>
      );
    case "input":
      return (
        <span
          key={node.id}
          {...attrs}
          className={cx("tpl-control", attrs.className)}
          style={sourceStyle}
        >
          <Input
            aria-label={typeof props.placeholder === "string" ? props.placeholder : "Field"}
            placeholder={typeof props.placeholder === "string" ? props.placeholder : undefined}
            data-node-id={node.id}
            style={importedControlStyle(sourceStyle)}
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
            ...sourceStyle,
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
    case "metric": {
      const size = typeof props.size === "string" ? props.size : "md";
      return (
        <div
          key={node.id}
          {...attrs}
          className={cx("tpl-metric", `is-${size}`, toneClass(props.tone), attrs.className)}
          style={sourceStyle}
        >
          {typeof props.label === "string" ? (
            <span className="tpl-metric-label">{props.label}</span>
          ) : null}
          <span className="tpl-metric-value">
            {typeof props.prefix === "string" ? (
              <span className="tpl-metric-prefix">{props.prefix}</span>
            ) : null}
            <b>{typeof props.value === "string" ? props.value : ""}</b>
            {typeof props.frac === "string" ? (
              <span className="tpl-metric-frac">{props.frac}</span>
            ) : null}
            {typeof props.suffix === "string" ? (
              <span className="tpl-metric-suffix">{props.suffix}</span>
            ) : null}
          </span>
          {typeof props.detail === "string" ? (
            <span className="tpl-metric-detail">{props.detail}</span>
          ) : null}
        </div>
      );
    }
    case "icon": {
      const size = typeof props.size === "number" ? props.size : 16;
      return (
        <span
          key={node.id}
          {...attrs}
          className={cx(
            "tpl-icon",
            props.box === true && "is-boxed",
            toneClass(props.tone),
            attrs.className,
          )}
          style={{ width: size + 16, height: size + 16, ...sourceStyle }}
        >
          <TemplateIcon name={typeof props.name === "string" ? props.name : "spark"} size={size} />
        </span>
      );
    }
    case "avatar":
      return (
        <span
          key={node.id}
          {...attrs}
          className={cx("tpl-control", attrs.className)}
          style={sourceStyle}
        >
          <Avatar
            name={typeof props.name === "string" ? props.name : "Member"}
            size={props.size === "lg" ? "lg" : props.size === "xs" ? "xs" : "sm"}
            shape="circle"
          />
        </span>
      );
    case "chip":
      return (
        <span
          key={node.id}
          {...attrs}
          className={cx("tpl-control tpl-chip", attrs.className)}
          data-tone={typeof props.tone === "string" ? props.tone : undefined}
          style={sourceStyle}
        >
          <Chip>{typeof props.text === "string" ? props.text : null}</Chip>
        </span>
      );
    case "meter": {
      const value = typeof props.value === "number" ? props.value : 0;
      const clamped = Math.max(0, Math.min(100, value));
      if (props.compact === true) {
        // In a table row or a factor list the meter is a slim value, not a
        // labelled control. The native meter keeps the semantics without the
        // component's label/value chrome.
        return (
          <span
            key={node.id}
            {...attrs}
            className={cx("tpl-meter-inline", attrs.className)}
            role="meter"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={Math.round(clamped)}
            aria-label={typeof props.label === "string" ? props.label : `${node.id} meter`}
            data-tone={typeof props.tone === "string" ? props.tone : undefined}
            style={sourceStyle}
          >
            <span style={{ width: `${clamped}%` }} />
          </span>
        );
      }
      return (
        <span
          key={node.id}
          {...attrs}
          className={cx("tpl-meter", attrs.className)}
          style={sourceStyle}
        >
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
    }
  }
  return null;
}
