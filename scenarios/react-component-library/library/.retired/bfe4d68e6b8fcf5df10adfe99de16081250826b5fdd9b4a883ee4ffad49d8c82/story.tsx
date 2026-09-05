import { useState, type CSSProperties } from "react";
import { FilePreview } from "./FilePreview";

const frame: CSSProperties = {
  display: "grid",
  gap: "var(--space-sm, 12px)",
  width: "min(100%, 40rem)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-lg, 24px)",
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 16px)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  boxShadow: "var(--elev-raised, 0 12px 36px rgb(15 23 42 / .1))",
};

export function Default() {
  const [removed, setRemoved] = useState(false);
  return (
    <div style={frame}>
      {removed ? (
        <p
          style={{ margin: 0, color: "var(--color-muted-foreground, #64748b)" }}
        >
          The attachment was removed safely.
        </p>
      ) : (
        <FilePreview
          name="research-brief.pdf"
          mimeType="application/pdf"
          sizeBytes={2_400_000}
          onOpen={() => undefined}
          onDownload={() => undefined}
          onRemove={() => setRemoved(true)}
        />
      )}
    </div>
  );
}

export function Loading() {
  return (
    <div style={frame}>
      <FilePreview
        name="workspace-export.zip"
        mimeType="application/zip"
        sizeBytes={18_400_000}
        status="loading"
        statusMessage="Scanning for a safe preview"
        onRemove={() => undefined}
      />
    </div>
  );
}

export function ErrorState() {
  return (
    <div style={frame}>
      <FilePreview
        name="prototype.heic"
        mimeType="image/heic"
        sizeBytes={8_100_000}
        status="error"
        statusMessage="This format cannot be previewed here"
        onDownload={() => undefined}
        onRemove={() => undefined}
      />
    </div>
  );
}
