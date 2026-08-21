import { useEffect, useState } from "react";
import { Copy } from "lucide-react";
import { IconButton } from "../../../../components/IconButton/versions/2.0.0/IconButton";
import { MorphingIcon, type MorphingIconProps } from "./MorphingIcon";

const panelStyle = {
  display: "grid",
  width: "100%",
  minWidth: 0,
  maxWidth: "36rem",
  boxSizing: "border-box" as const,
  gap: "var(--space-sm, 12px)",
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
          flexWrap: "wrap" as const,
          alignItems: "center",
          justifyContent: "space-between",
          gap: "var(--space-md, 24px)",
        }}
      >
        <div style={{ minWidth: 0, flex: "1 1 14rem" }}>
          <span
            style={{
              display: "block",
              color: "var(--color-muted-foreground, #667085)",
              fontSize: "var(--text-label-size, 12px)",
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
              marginTop: "var(--space-2xs, 8px)",
              fontSize: "var(--text-title-size, 24px)",
              lineHeight: "var(--text-title-line, 30px)",
              letterSpacing: "-0.03em",
            }}
          >
            {label}
          </strong>
          <span
            style={{
              display: "block",
              marginTop: "var(--space-2xs, 8px)",
              color: "var(--color-muted-foreground, #667085)",
              fontSize: "var(--text-body-sm-size, 13px)",
              lineHeight: "var(--text-body-sm-line, 20px)",
            }}
          >
            The icon changes meaning in place without shifting its control.
          </span>
        </div>
        <div
          style={{
            display: "grid",
            placeItems: "center",
            minWidth: "var(--space-xl, 40px)",
            minHeight: "var(--space-xl, 40px)",
            padding: "var(--space-xs, 12px)",
            borderRadius: "var(--radius-panel, 20px)",
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
          gap: "var(--space-xs, 12px)",
          paddingTop: "var(--space-xs, 12px)",
          borderTop:
            "var(--border-hairline, 1px) solid var(--color-border, #d8dee9)",
          color: "var(--color-muted-foreground, #667085)",
          fontSize: "var(--text-label-size, 12px)",
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
          flexWrap: "wrap" as const,
          alignItems: "center",
          justifyContent: "space-between",
          gap: "var(--space-sm, 16px)",
        }}
      >
        <div>
          <strong
            style={{
              display: "block",
              fontSize: "var(--text-heading-size, 18px)",
              lineHeight: "var(--text-heading-line, 24px)",
              letterSpacing: "-0.01em",
            }}
          >
            Delivery state
          </strong>
          <span
            style={{
              display: "block",
              marginTop: "var(--space-3xs, 4px)",
              color: "var(--color-muted-foreground, #667085)",
              fontSize: "var(--text-body-sm-size, 13px)",
              lineHeight: "var(--text-body-sm-line, 20px)",
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
          minHeight: "var(--tap-target-min, 44px)",
          paddingInline: "var(--space-sm, 16px)",
          border: "var(--border-hairline, 1px) solid transparent",
          borderRadius: "var(--radius-control, 10px)",
          background: "var(--color-primary, #2563eb)",
          color: "var(--color-primary-foreground, #ffffff)",
          font: "600 var(--text-body-sm-size, 13px)/var(--text-body-sm-line, 20px) var(--font-sans, ui-sans-serif, system-ui, sans-serif)",
          fontWeight: 700,
          cursor: "pointer",
          transition:
            "transform var(--dur-quick, 180ms) var(--ease-standard, ease), filter var(--dur-quick, 180ms) var(--ease-standard, ease)",
        }}
      >
        {sent ? "Reset state" : "Send message"}
      </button>
    </div>
  );
}

export function CopyMorphingIcon() {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return undefined;
    const timeout = window.setTimeout(() => setCopied(false), 1200);
    return () => window.clearTimeout(timeout);
  }, [copied]);

  return (
    <div
      style={{
        display: "grid",
        justifyItems: "center",
        gap: "var(--space-sm, 12px)",
        padding: "var(--space-xl, 32px)",
        color: "var(--color-muted-foreground, #667085)",
        fontFamily: "var(--font-sans, ui-sans-serif, system-ui, sans-serif)",
      }}
    >
      <IconButton
        aria-label={copied ? "Copied" : "Copy value"}
        variant="ghost"
        onClick={() => setCopied(true)}
      >
        {copied ? (
          <MorphingIcon
            icon="check"
            from="send"
            label="Copied"
            size="md"
            strategy="morph"
            duration="deliberate"
            style={{ color: "var(--color-success, #15803d)" }}
          />
        ) : (
          <Copy aria-hidden size={20} />
        )}
      </IconButton>
      <span>{copied ? "Copied" : "Copy value"}</span>
    </div>
  );
}
