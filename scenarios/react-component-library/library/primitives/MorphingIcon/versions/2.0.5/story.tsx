import { useEffect, useState } from "react";
import { IconButton } from "../../../../components/IconButton/versions/2.0.0/IconButton";
import { MorphingIcon } from "./MorphingIcon";

const panelStyle = {
  display: "grid",
  gap: "var(--space-lg, 24px)",
  width: "100%",
  maxWidth: "32rem",
  boxSizing: "border-box" as const,
  padding: "var(--space-xl, 32px)",
  border: "1px solid var(--color-border, #d8dee9)",
  borderRadius: "var(--radius-panel, 20px)",
  background:
    "linear-gradient(145deg, var(--color-surface, #fff), var(--color-surface-muted, #f4f7fb))",
  color: "var(--color-foreground, #142033)",
  boxShadow: "var(--elev-raised, 0 14px 40px rgb(20 32 51 / 12%))",
  fontFamily: "var(--font-sans, ui-sans-serif, system-ui, sans-serif)",
};

export function ToggleMorphingIcon() {
  const [sent, setSent] = useState(false);
  return (
    <div style={panelStyle}>
      <strong>Delivery state</strong>
      <MorphingIcon
        icon={sent ? "check" : "send"}
        from="send"
        label={sent ? "Sent" : "Send"}
        size="lg"
      />
      <button type="button" onClick={() => setSent((value) => !value)}>
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
    <div style={panelStyle}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: "var(--space-md, 20px)",
        }}
      >
        <div>
          <strong
            style={{
              display: "block",
              fontSize: "var(--text-heading-size, 18px)",
            }}
          >
            Copy feedback
          </strong>
          <span style={{ color: "var(--color-muted-foreground, #667085)" }}>
            A compact success transition.
          </span>
        </div>
        <IconButton
          aria-label={copied ? "Copied" : "Copy value"}
          onClick={() => setCopied(true)}
        >
          <MorphingIcon
            icon={copied ? "check" : "copy"}
            from="copy"
            label={copied ? "Copied" : "Copy value"}
            duration={420}
            style={{
              color: copied
                ? "var(--color-success, #15803d)"
                : "var(--color-foreground, #142033)",
              transition: "color 420ms ease",
            }}
          />
        </IconButton>
      </div>
    </div>
  );
}
