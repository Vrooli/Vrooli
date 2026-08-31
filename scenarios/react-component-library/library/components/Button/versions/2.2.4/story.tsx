import { useState } from "react";
import { Button, type ButtonProps } from "./Button";
import { PreviewShowcase } from "../../../../preview-harnesses/showcase/PreviewShowcase";

export function ButtonStory({ args }: StoryHarnessProps) {
  const buttonArgs = args as unknown as ButtonProps;
  const label = typeof buttonArgs.children === "string" ? buttonArgs.children : "Action";
  const detail = buttonArgs.disabled
    ? "Disabled states retain the same geometry and clearly communicate that the action is unavailable."
    : "A clear visual hierarchy, a full touch target, and a small amount of responsive motion make the next action feel inevitable.";
  const Subject = (props: Record<string, unknown>) => (
    <Button
      {...(props as unknown as ButtonProps)}
      aria-label={typeof props["aria-label"] === "string" ? props["aria-label"] : label}
    />
  );
  return (
    <PreviewShowcase
      subject={Subject}
      args={buttonArgs as unknown as Record<string, unknown>}
      label={label}
      description={detail}
      config={{ density: "compact" }}
    />
  );
}

/** A real async boundary: the action stays disabled while the work is pending. */
export function AsyncSaveStory({ args }: StoryHarnessProps) {
  const buttonArgs = args as unknown as ButtonProps;
  const label = typeof buttonArgs.children === "string" ? buttonArgs.children : "Save changes";
  const Subject = (props: Record<string, unknown>) => {
    const [pending, setPending] = useState(false);
    return (
      <div
        style={{
          display: "grid",
          gap: "var(--space-sm)",
          justifyItems: "start",
        }}
      >
        <Button
          {...(props as unknown as ButtonProps)}
          aria-label={label}
          pending={pending}
          pendingLabel="Saving changes…"
          onClick={() => setPending(true)}
        >
          {label}
        </Button>
        <output
          aria-label="Saving changes status"
          aria-live="polite"
          style={{ color: "var(--color-muted-foreground)" }}
        >
          {pending ? "Request in progress" : "Ready to save"}
        </output>
      </div>
    );
  };
  return (
    <PreviewShowcase
      subject={Subject}
      label="Async interaction"
      description="Click the action to expose the pending state used while a save request is in flight."
      config={{ density: "compact", title: "Save with an async boundary" }}
      args={buttonArgs as unknown as Record<string, unknown>}
    />
  );
}

export function LongContentStory({ args }: StoryHarnessProps) {
  void args;
  const Subject = (props: Record<string, unknown>) => (
    <Button
      {...(props as unknown as ButtonProps)}
      aria-label="Review and apply all pending workspace configuration changes"
      size="lg"
      // Long labels are a real adoption case. Keep the control inside its
      // available measure while preserving the full accessible name.
      style={{ inlineSize: "100%", maxInlineSize: "100%", overflow: "hidden" }}
    >
      Review and apply all pending workspace configuration changes
    </Button>
  );
  return (
    <PreviewShowcase
      subject={Subject}
      label="Stress content"
      description="Long labels must remain understandable without breaking the control geometry."
      config={{ density: "compact", title: "A deliberately long action label" }}
    />
  );
}

/** A recoverable request failure: the user gets context and one obvious retry. */
export function ErrorRecoveryStory({ args }: StoryHarnessProps) {
  const buttonArgs = args as unknown as ButtonProps;
  const [recovered, setRecovered] = useState(false);
  const label = recovered ? "Saved" : "Try again";
  return (
    <PreviewShowcase
      label="Recoverable failure"
      description="Failures keep the next action explicit and give the user a safe retry without duplicating the control."
      config={{ density: "compact", title: "Retry after a failed save" }}
      args={buttonArgs as unknown as Record<string, unknown>}
      subject={() => (
        <div style={{ display: "grid", gap: "var(--space-xs)", justifyItems: "start" }}>
          <div role={recovered ? "status" : "alert"} aria-live="polite">
            {recovered ? "Changes saved successfully." : "We couldn’t save your changes."}
          </div>
          <Button {...buttonArgs} aria-label={label} onClick={() => setRecovered(true)}>
            {label}
          </Button>
        </div>
      )}
    />
  );
}
