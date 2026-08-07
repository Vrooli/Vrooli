import { useState } from "react";
import { AttachmentPreview } from "./AttachmentPreview";

function Showcase({
  children,
  eyebrow,
  title,
  detail,
}: {
  children: React.ReactNode;
  eyebrow: string;
  title: string;
  detail: string;
}) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 640px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          {eyebrow}
        </span>
        <strong
          style={{
            font: "var(--text-title)",
            color: "var(--color-foreground)",
          }}
        >
          {title}
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {detail}
        </span>
      </div>
      {children}
    </section>
  );
}

export function Uploading() {
  return (
    <Showcase
      eyebrow="Secure handoff"
      title="Stay oriented while work continues"
      detail="Progress is per file, cancellation is explicit, and the attachment keeps its identity throughout the transfer."
    >
      <AttachmentPreview
        name="research-notes.pdf"
        mimeType="application/pdf"
        sizeBytes={1843200}
        status="uploading"
        progress={68}
        onCancel={() => undefined}
      />
    </Showcase>
  );
}

export function Success() {
  return (
    <Showcase
      eyebrow="Ready to use"
      title="A completed attachment stays actionable"
      detail="Preview, download, and removal remain available after the transfer finishes."
    >
      <AttachmentPreview
        name="product-brief.pdf"
        mimeType="application/pdf"
        sizeBytes={2457600}
        status="success"
        onOpen={() => undefined}
        onDownload={() => undefined}
        onRemove={() => undefined}
      />
    </Showcase>
  );
}

export function Recovery() {
  const [status, setStatus] = useState<"error" | "uploading">("error");
  return (
    <Showcase
      eyebrow="Recovery path"
      title="A failed transfer explains itself"
      detail="Retry is a real adapter callback, so the preview never claims a network operation happened when it did not."
    >
      <AttachmentPreview
        name="team-recording.mp4"
        mimeType="video/mp4"
        sizeBytes={73400320}
        status={status}
        errorMessage="The connection closed before the file finished uploading."
        onRetry={() => setStatus("uploading")}
        onCancel={() => setStatus("error")}
        progress={status === "uploading" ? 12 : undefined}
      />
    </Showcase>
  );
}
