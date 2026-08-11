import { useState } from "react";
import { Message, type MessageProps } from "./Message";

const image =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 120 120'%3E%3Cdefs%3E%3ClinearGradient id='m' x1='0' x2='1'%3E%3Cstop stop-color='%230ea5e9'/%3E%3Cstop offset='1' stop-color='%231d4ed8'/%3E%3C/linearGradient%3E%3C/defs%3E%3Crect width='120' height='120' fill='url(%23m)'/%3E%3Ccircle cx='60' cy='46' r='23' fill='white' fill-opacity='.86'/%3E%3Cpath d='M18 116c4-29 20-42 42-42s38 13 42 42' fill='white' fill-opacity='.86'/%3E%3C/svg%3E";

const shell = {
  display: "grid",
  gap: "var(--space-sm)",
  width: "min(100%, 720px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;

function Showcase({
  title,
  detail,
  children,
}: {
  title: string;
  detail: string;
  children: React.ReactNode;
}) {
  return (
    <section style={shell}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Conversation surface
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
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

const base: MessageProps = {
  actor: {
    name: "Ari Okafor",
    role: "Research assistant",
    src: image,
    presence: "online",
  },
  timestamp: "2026-08-10T14:32:00Z",
  content: (
    <>
      <p>I found three credible sources that support the release brief.</p>
      <p>
        The strongest signal is consistent across the original study and the
        current field report.
      </p>
    </>
  ),
  attachments: [
    {
      id: "brief",
      name: "release-brief.pdf",
      detail: "PDF · 1.8 MB",
      status: "ready",
    },
  ],
  citations: [
    {
      id: "study",
      label: "Human-centered AI in practice",
      source: "Signal Review · 2026",
      excerpt: "Evidence-backed product decisions reduce rework.",
    },
  ],
  actions: [
    { id: "copy", label: "Copy response" },
    { id: "save", label: "Save to brief" },
  ],
  footer: (
    <>
      Response ID · <span>msg_2048</span>
    </>
  ),
};

export function Default() {
  return (
    <Showcase
      title="Evidence stays attached to the answer"
      detail="Actor, content, sources, and safe actions remain one readable unit."
    >
      <Message {...base} state="default" />
    </Showcase>
  );
}

export function Loading() {
  return (
    <Showcase
      title="A considered answer is on its way"
      detail="The identity and reserved message geometry remain stable while work is in flight."
    >
      <Message
        actor={base.actor}
        timestamp="just now"
        state="loading"
        activity={{
          label: "Searching sources",
          detail: "Checking the latest field notes",
          status: "pending",
          certainty: "estimated",
          urgency: "informational",
        }}
      />
    </Showcase>
  );
}

export function Partial() {
  return (
    <Showcase
      title="Streaming without losing context"
      detail="Partial output is explicitly labeled so readers can distinguish a draft from a delivered answer."
    >
      <Message
        {...base}
        state="partial"
        content={
          <p>
            The first pattern is clear: teams move faster when evidence is
            visible beside the decision…
          </p>
        }
        actions={[]}
      />
    </Showcase>
  );
}

export function Success() {
  return (
    <Showcase
      title="A delivered answer can be trusted"
      detail="Success is communicated as a semantic state, not only a green color or a fleeting animation."
    >
      <Message
        {...base}
        state="success"
        activity={{
          label: "Verified against 3 sources",
          status: "success",
          certainty: "confirmed",
          urgency: "ambient",
        }}
      />
    </Showcase>
  );
}

export function RequestError() {
  return (
    <Showcase
      title="Failure keeps the conversation intact"
      detail="The message explains what happened and offers one honest recovery action without discarding the draft context."
    >
      <Message
        {...base}
        actor={{ ...base.actor, src: undefined }}
        state="request-error"
        content={
          <p>
            I prepared a response, but the source verification request timed out
            before it could be delivered.
          </p>
        }
        onRetry={() => undefined}
      />
    </Showcase>
  );
}

export function Retry() {
  return (
    <Showcase
      title="Retry from the same context"
      detail="A recoverable request preserves the actor, draft content, and evidence boundary while waiting for a deliberate retry."
    >
      <Message
        {...base}
        state="retry"
        content={
          <p>
            The draft is ready. Retry verification when the connection is
            stable.
          </p>
        }
        onRetry={() => undefined}
      />
    </Showcase>
  );
}

export function Interactive() {
  const [state, setState] = useState<MessageProps["state"]>("request-error");
  return (
    <Showcase
      title="Recovery is an interaction, not a promise"
      detail="The retry action changes the semantic state and keeps the message in place."
    >
      <Message
        {...base}
        state={state}
        onRetry={() => setState("success")}
        content={
          state === "success" ? (
            <p>Verification completed. The answer is ready to use.</p>
          ) : (
            base.content
          )
        }
        activity={
          state === "success"
            ? {
                label: "Verified against 3 sources",
                status: "success",
                certainty: "confirmed",
              }
            : undefined
        }
      />
    </Showcase>
  );
}
