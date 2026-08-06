import { useState } from "react";
import { MorphingIcon, type MorphingIconProps } from "./MorphingIcon";

const panelStyle = {
  display: "grid",
  gap: "var(--space-sm, 12px)",
  minWidth: "min(100%, 22rem)",
  padding: "var(--space-lg, 24px)",
  border: "1px solid var(--color-border, #d8dee9)",
  borderRadius: "var(--radius-panel, 20px)",
  background:
    "linear-gradient(145deg, var(--color-surface, #fff), var(--color-surface-muted, #f4f7fb))",
  color: "var(--color-foreground, #142033)",
  boxShadow: "var(--elev-raised, 0 14px 40px rgb(20 32 51 / 12%))",
  fontFamily: "var(--font-sans, ui-sans-serif, system-ui, sans-serif)",
};

export function MorphingIconShowcase({
  args,
}: {
  args?: Partial<MorphingIconProps>;
}) {
  const icon = args?.icon ?? "check";
  const from = args?.from;
  const strategy = args?.strategy ?? "auto";
  const label = args?.label ?? "Current state";
  return (
    <div style={panelStyle}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: "24px",
        }}
      >
        <div style={{ minWidth: 0 }}>
          <span
            style={{
              display: "block",
              color: "var(--color-muted-foreground, #667085)",
              fontSize: "11px",
              fontWeight: 800,
              letterSpacing: "0.12em",
              textTransform: "uppercase",
            }}
          >
            State transition
          </span>
          <strong
            style={{
              display: "block",
              marginTop: "6px",
              fontSize: "20px",
              letterSpacing: "-0.03em",
            }}
          >
            {label}
          </strong>
          <span
            style={{
              display: "block",
              marginTop: "6px",
              color: "var(--color-muted-foreground, #667085)",
              fontSize: "13px",
              lineHeight: 1.5,
            }}
          >
            The icon changes meaning in place without shifting its control.
          </span>
        </div>
        <div
          style={{
            display: "grid",
            placeItems: "center",
            minWidth: "72px",
            minHeight: "72px",
            borderRadius: "18px",
            background: "var(--color-accent-subtle, #e8efff)",
            color: "var(--color-accent, #315efb)",
          }}
        >
          <MorphingIcon
            icon={icon}
            from={from}
            strategy={strategy}
            label={label}
            size="lg"
          />
        </div>
      </div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: "12px",
          paddingTop: "12px",
          borderTop: "1px solid var(--color-border, #d8dee9)",
          color: "var(--color-muted-foreground, #667085)",
          fontSize: "12px",
        }}
      >
        <span>Strategy</span>
        <strong
          style={{ color: "var(--color-foreground, #142033)", fontWeight: 700 }}
        >
          {strategy}
        </strong>
      </div>
    </div>
  );
}

export function ToggleMorphingIcon() {
  const [sent, setSent] = useState(false);
  return (
    <div style={panelStyle}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: "16px",
        }}
      >
        <div>
          <strong
            style={{
              display: "block",
              fontSize: "16px",
              letterSpacing: "-0.01em",
            }}
          >
            Delivery state
          </strong>
          <span
            style={{
              display: "block",
              marginTop: "4px",
              color: "var(--color-muted-foreground, #667085)",
              fontSize: "13px",
            }}
          >
            Meaning changes in place.
          </span>
        </div>
        <MorphingIcon
          icon={sent ? "check" : "send"}
          from="send"
          label={sent ? "Sent" : "Send"}
          size="lg"
        />
      </div>
      <button
        type="button"
        onClick={() => setSent((value) => !value)}
        style={{
          border: 0,
          borderRadius: "var(--radius-control, 10px)",
          padding: "10px 14px",
          background: "var(--color-accent, #315efb)",
          color: "white",
          fontWeight: 700,
          cursor: "pointer",
        }}
      >
        {sent ? "Reset state" : "Send message"}
      </button>
    </div>
  );
}
