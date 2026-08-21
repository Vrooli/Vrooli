import { useEffect, useState } from "react";
import { Button } from "../../../../components/Button/versions/1.0.0/Button";
import { IconButton } from "../../../../components/IconButton/versions/2.0.0/IconButton";
import { Surface } from "../../../Surface/versions/1.0.0/Surface";
import { MorphingIcon } from "./MorphingIcon";

const stageStyle = {
  display: "grid",
  gap: "var(--space-lg, 24px)",
  width: "100%",
  maxWidth: "36rem",
  boxSizing: "border-box" as const,
  padding: "var(--space-lg, 24px)",
  color: "var(--color-foreground, #142033)",
  fontFamily: "var(--font-sans, ui-sans-serif, system-ui, sans-serif)",
};

export function ToggleMorphingIcon() {
  const [sent, setSent] = useState(false);
  return (
    <Surface elevation="raised" style={stageStyle}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: "var(--space-md, 20px)",
        }}
      >
        <div>
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
            }}
          >
            {sent ? "Message sent" : "Ready to send"}
          </strong>
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
            color: sent
              ? "var(--color-success, #15803d)"
              : "var(--color-accent, #315efb)",
            transition: "color 280ms ease",
          }}
        >
          <MorphingIcon
            icon={sent ? "check" : "send"}
            from="send"
            label={sent ? "Sent" : "Send"}
            size="lg"
          />
        </div>
      </div>
      <Button onClick={() => setSent((value) => !value)}>
        {sent ? "Reset state" : "Send message"}
      </Button>
    </Surface>
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
    <Surface elevation="raised" style={stageStyle}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: "var(--space-md, 20px)",
        }}
      >
        <div>
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
            Clipboard feedback
          </span>
          <strong
            style={{
              display: "block",
              marginTop: "var(--space-2xs, 8px)",
              fontSize: "var(--text-heading-size, 18px)",
            }}
          >
            {copied ? "Copied successfully" : "Copy this value"}
          </strong>
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
    </Surface>
  );
}
