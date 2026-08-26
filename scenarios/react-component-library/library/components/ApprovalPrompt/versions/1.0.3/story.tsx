import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useState, type ReactNode } from "react";
import { ApprovalPrompt, type ApprovalPromptStatus } from "./ApprovalPrompt";

const shell = {
  display: "grid",
  alignContent: "start",
  gap: "var(--space-md)",
  width: "min(100%, 680px)",
  minHeight: 420,
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background:
    "linear-gradient(145deg, var(--color-surface-raised), color-mix(in srgb, var(--color-primary) 5%, var(--color-surface-raised)))",
  boxShadow: "var(--elev-raised)",
} as const;

function Showcase({
  title,
  detail,
  children,
}: {
  title: string;
  detail: string;
  children: ReactNode;
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
          {useStrings("ai.approval-prompt.human-in-the-loop", "Human in the loop")}
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

const common = {
  action: "publish the release",
  target: "Acme production workspace",
  scope: "the current release only",
  description: "A human review is required before this external side effect can proceed.",
  consequences:
    "The release will become visible to workspace members and cannot be silently rolled back.",
  alternatives: "Save as a draft and ask a workspace owner to review it later.",
  expiresLabel: "Expires in 8 minutes",
};

export function Default() {
  return (
    <Showcase
      title={useStrings("ai.approval-prompt.title", "Consent should feel deliberate")}
      detail="The exact scope sits directly above the approval action, with target, consequence, and expiry in view."
    >
      <ApprovalPrompt {...common} />
    </Showcase>
  );
}

export function Submitting() {
  return (
    <Showcase
      title={useStrings(
        "ai.approval-prompt.title.the-decision-has-a-stable-pending-state",
        "The decision has a stable pending state",
      )}
      detail="The action remains legible while the request is being recorded."
    >
      <ApprovalPrompt {...common} status="submitting" />
    </Showcase>
  );
}

export function Success() {
  return (
    <Showcase
      title={useStrings(
        "ai.approval-prompt.title.a-successful-decision-closes-the-loop",
        "A successful decision closes the loop",
      )}
      detail="The surface confirms what was authorized instead of disappearing without context."
    >
      <ApprovalPrompt {...common} status="success" />
    </Showcase>
  );
}

export function RequestError() {
  return (
    <Showcase
      title={useStrings(
        "ai.approval-prompt.title.failures-preserve-the-decision-context",
        "Failures preserve the decision context",
      )}
      detail="A failed request explains that nothing was authorized and offers a truthful retry."
    >
      <ApprovalPrompt {...common} status="request-error" />
    </Showcase>
  );
}

export function PermissionDenied() {
  return (
    <Showcase
      title={useStrings(
        "ai.approval-prompt.title.permission-boundaries-stay-explainable",
        "Permission boundaries stay explainable",
      )}
      detail="The user gets a recovery path without being invited to retry an action their role cannot perform."
    >
      <ApprovalPrompt {...common} status="permission-denied" />
    </Showcase>
  );
}

export function Retry() {
  return (
    <Showcase
      title={useStrings(
        "ai.approval-prompt.title.retry-keeps-scope-stable",
        "Retry keeps scope stable",
      )}
      detail="The same consent line remains in place when transport recovery is needed."
    >
      <ApprovalPrompt {...common} status="retry" />
    </Showcase>
  );
}

export function Interactive() {
  const [status, setStatus] = useState<ApprovalPromptStatus>("default");
  return (
    <Showcase
      title={useStrings(
        "ai.approval-prompt.title.a-real-decision-not-a-decorative-card",
        "A real decision, not a decorative card",
      )}
      detail="Approve transitions through the same submitting and success states a consumer receives from its request handler."
    >
      <ApprovalPrompt
        {...common}
        status={status}
        onApprove={async () => {
          setStatus("submitting");
          await new Promise((resolve) => window.setTimeout(resolve, 320));
          setStatus("success");
        }}
        onDeny={() => setStatus("permission-denied")}
        onRetry={() => setStatus("default")}
      />
    </Showcase>
  );
}
