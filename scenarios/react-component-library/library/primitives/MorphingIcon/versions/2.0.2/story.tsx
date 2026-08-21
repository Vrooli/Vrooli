import { useEffect, useState } from "react";
import { IconButton } from "../../../../components/IconButton/versions/2.0.0/IconButton";
import { MorphingIcon } from "./MorphingIcon";
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
        color: copied
          ? "var(--color-success, #15803d)"
          : "var(--color-foreground, #142033)",
        transition: "color 360ms var(--ease-standard, ease)",
      }}
    >
      <IconButton
        aria-label={copied ? "Copied" : "Copy value"}
        onClick={() => setCopied(true)}
      >
        <MorphingIcon
          icon={copied ? "check" : "copy"}
          from="copy"
          label={copied ? "Copied" : "Copy value"}
          duration={420}
        />
      </IconButton>
      <span>{copied ? "Copied" : "Copy value"}</span>
    </div>
  );
}
export function ToggleMorphingIcon() {
  const [sent, setSent] = useState(false);
  return (
    <button type="button" onClick={() => setSent((value) => !value)}>
      <MorphingIcon
        icon={sent ? "check" : "send"}
        from="send"
        label={sent ? "Sent" : "Send"}
      />
    </button>
  );
}
