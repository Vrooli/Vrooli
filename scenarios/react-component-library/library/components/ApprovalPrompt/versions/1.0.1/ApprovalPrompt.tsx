/**
 * @libraryId react-component-library:ApprovalPrompt
 * @displayName ApprovalPrompt
 * @description A human-in-the-loop decision surface that makes action scope, consequences, expiry, and recovery explicit.
 * @version 1.0.1
 * @tags ["ai","approval","permission","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource ai.approval-prompt */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { useEffect, useId, useMemo, useState, type CSSProperties, type ReactNode } from "react";
import { Alert } from "../../../Alert/versions/1.0.0/Alert";
import { Button } from "../../../Button/versions/2.0.0/Button";
import { ButtonGroup } from "../../../ButtonGroup/versions/1.0.0/ButtonGroup";
import {
  StatusIndicator,
  type StatusCertainty,
  type StatusUrgency,
} from "../../../StatusIndicator/versions/1.0.0/StatusIndicator";

export type ApprovalPromptStatus =
  | "default"
  | "submitting"
  | "success"
  | "request-error"
  | "permission-denied"
  | "retry";

export interface ApprovalPromptProps {
  action: string;
  target: string;
  scope: string;
  title?: string;
  description?: ReactNode;
  consequences?: ReactNode;
  alternatives?: ReactNode;
  expiresAt?: Date | number | string;
  expiresLabel?: string;
  status?: ApprovalPromptStatus;
  defaultStatus?: ApprovalPromptStatus;
  certainty?: StatusCertainty;
  urgency?: StatusUrgency;
  approveLabel?: string;
  denyLabel?: string;
  retryLabel?: string;
  pendingLabel?: string;
  permissionLabel?: string;
  onApprove?: () => void | Promise<void>;
  onDeny?: () => void;
  onRetry?: () => void | Promise<void>;
  disabled?: boolean;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-approval-prompt] { display: grid; gap: var(--space-md); min-inline-size: 0; padding: var(--space-lg); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-raised); }
[data-rcl-approval-header] { display: grid; gap: var(--space-2xs); min-inline-size: 0; }
[data-rcl-approval-eyebrow] { color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .1em; text-transform: uppercase; }
[data-rcl-approval-title] { margin: 0; color: var(--color-foreground); font: var(--text-title); letter-spacing: -.015em; }
[data-rcl-approval-description] { margin: 0; color: var(--color-muted-foreground); font: var(--text-body); line-height: 1.55; }
[data-rcl-approval-facts] { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--space-2xs); margin: 0; }
[data-rcl-approval-fact] { display: grid; gap: var(--space-3xs); min-inline-size: 0; padding: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); }
[data-rcl-approval-fact] dt { color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-approval-fact] dd { min-inline-size: 0; margin: 0; color: var(--color-foreground); font: var(--text-body-sm); overflow-wrap: anywhere; }
[data-rcl-approval-section] { display: grid; gap: var(--space-2xs); }
[data-rcl-approval-section] h3 { margin: 0; color: var(--color-foreground); font: var(--text-label); letter-spacing: .02em; }
[data-rcl-approval-section] p, [data-rcl-approval-section] ul { margin: 0; color: var(--color-muted-foreground); font: var(--text-body-sm); line-height: 1.5; }
[data-rcl-approval-section] ul { display: grid; gap: var(--space-3xs); padding-inline-start: var(--space-md); }
[data-rcl-approval-consent] { display: grid; gap: var(--space-3xs); padding: var(--space-sm); border: var(--border-strong) solid color-mix(in srgb, var(--color-primary) 36%, var(--color-border)); border-radius: var(--radius-panel); background: color-mix(in srgb, var(--color-primary) 7%, var(--color-surface)); }
[data-rcl-approval-consent-label] { color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-approval-consent-line] { margin: 0; color: var(--color-foreground); font: var(--text-body); line-height: 1.5; }
[data-rcl-approval-consent-line] strong { color: var(--color-primary); font-weight: 750; }
[data-rcl-approval-expiry] { display: inline-flex; align-items: center; gap: var(--space-2xs); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-approval-expiry][data-expired="true"] { color: var(--color-danger); }
[data-rcl-approval-actions] { justify-content: end; }
[data-rcl-approval-actions] [data-control-slot="label"] { overflow: visible; text-overflow: clip; white-space: nowrap; }
[data-rcl-approval-prompt][data-status="submitting"] [data-rcl-approval-consent], [data-rcl-approval-prompt][data-status="success"] [data-rcl-approval-consent] { border-color: var(--color-border); }
@media (max-width: 42rem) { [data-rcl-approval-prompt] { padding: var(--space-md); } [data-rcl-approval-facts] { grid-template-columns: 1fr; } [data-rcl-approval-actions] { justify-content: stretch; } [data-rcl-approval-actions] > * { flex: 1 1 100%; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-approval-prompt] *, [data-rcl-approval-prompt] *::before, [data-rcl-approval-prompt] *::after { scroll-behavior: auto; transition-duration: 0.01ms; animation-duration: 0.01ms; } }
@media (forced-colors: active) { [data-rcl-approval-prompt] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-approval-fact], [data-rcl-approval-consent] { border-color: CanvasText; background: Canvas; } [data-rcl-approval-consent-line] strong { color: Highlight; } [data-rcl-approval-expiry][data-expired="true"] { color: Mark; } }
`;

function expiryTimestamp(value: Date | number | string) {
  if (value instanceof Date) return value.getTime();
  if (typeof value === "number") return value;
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function formatExpiry(timestamp: number, now: number) {
  const remaining = timestamp - now;
  if (remaining <= 0) return "Expired";
  const minutes = Math.ceil(remaining / 60_000);
  if (minutes < 2) return "Expires in under a minute";
  if (minutes < 60) return `Expires in ${minutes} minutes`;
  const hours = Math.ceil(minutes / 60);
  if (hours < 24) return `Expires in ${hours} hours`;
  return `Expires in ${Math.ceil(hours / 24)} days`;
}

function statusSignal(status: ApprovalPromptStatus) {
  if (status === "submitting") return "pending" as const;
  if (status === "success") return "success" as const;
  if (status === "request-error" || status === "permission-denied") return "error" as const;
  return "idle" as const;
}

function statusLabel(status: ApprovalPromptStatus) {
  if (status === "submitting") return "Awaiting approval";
  if (status === "success") return "Approved";
  if (status === "request-error") return "Approval needs attention";
  if (status === "permission-denied") return "Permission denied";
  return status === "retry" ? "Ready to retry" : "Awaiting decision";
}

export function ApprovalPrompt({
  action,
  target,
  scope,
  title = translate("ai.approval-prompt.title.1", "Review this action"),
  description,
  consequences,
  alternatives,
  expiresAt,
  expiresLabel,
  status,
  defaultStatus = "default",
  certainty = "estimated",
  urgency = "actionable",
  approveLabel = `Approve ${action}`,
  denyLabel = "Not now",
  retryLabel = "Retry approval",
  pendingLabel = "Waiting for confirmation…",
  permissionLabel = "Request access",
  onApprove,
  onDeny,
  onRetry,
  disabled = false,
  className,
  style,
}: ApprovalPromptProps) {
  const titleId = useId();
  const descriptionId = useId();
  const [localStatus, setLocalStatus] = useState(defaultStatus);
  const [now, setNow] = useState<number>();
  const resolvedStatus = status ?? localStatus;
  const expiry = useMemo(
    () => (expiresAt === undefined ? undefined : expiryTimestamp(expiresAt)),
    [expiresAt],
  );

  useEffect(() => {
    if (expiry === undefined) return;
    setNow(Date.now());
    const timer = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(timer);
  }, [expiry]);

  const expiryText = expiresLabel ?? (expiry && now ? formatExpiry(expiry, now) : undefined);
  const expired = expiry !== undefined && now !== undefined && expiry <= now;
  const busy = disabled || resolvedStatus === "submitting" || expired;
  const canApprove =
    !busy &&
    (resolvedStatus === "default" ||
      resolvedStatus === "retry" ||
      resolvedStatus === "request-error");

  const run = async (kind: "approve" | "retry") => {
    if (busy) return;
    if (!status) setLocalStatus("submitting");
    try {
      if (kind === "approve") await onApprove?.();
      else await onRetry?.();
      if (!status) setLocalStatus("success");
    } catch {
      if (!status) setLocalStatus("request-error");
    }
  };

  const stateAlert =
    resolvedStatus === "success" ? (
      <Alert
        tone="success"
        title={translate("ai.approval-prompt.title.2", "Approval recorded")}
        description={translate(
          "ai.approval-prompt.description.3",
          "The requested action is now authorized within the scope above.",
        )}
      />
    ) : resolvedStatus === "request-error" ? (
      <Alert
        tone="danger"
        title={translate("ai.approval-prompt.title.4", "Approval could not be recorded")}
        description={translate(
          "ai.approval-prompt.description.5",
          "Nothing was authorized. You can retry without changing the requested scope.",
        )}
      />
    ) : resolvedStatus === "permission-denied" ? (
      <Alert
        tone="warning"
        title={translate("ai.approval-prompt.title.6", "Permission was not granted")}
        description={translate(
          "ai.approval-prompt.description.7",
          "Your role cannot approve this action. Request access or return to the conversation.",
        )}
      />
    ) : null;

  return (
    <section
      data-rcl-approval-prompt
      data-status={resolvedStatus}
      data-expired={expired || undefined}
      aria-labelledby={titleId}
      aria-describedby={description ? descriptionId : undefined}
      aria-busy={resolvedStatus === "submitting"}
      className={className}
      style={style}
    >
      <style data-rcl-approval-prompt-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <header data-rcl-approval-header>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            gap: "var(--space-sm)",
            flexWrap: "wrap",
          }}
        >
          <span data-rcl-approval-eyebrow>
            {translate("ai.approval-prompt.text.1", "Permission request")}
          </span>
          <StatusIndicator
            status={statusSignal(resolvedStatus)}
            label={statusLabel(resolvedStatus)}
            certainty={certainty}
            urgency={urgency}
          />
        </div>
        <h2 id={titleId} data-rcl-approval-title>
          {title}
        </h2>
        {description ? (
          <p id={descriptionId} data-rcl-approval-description>
            {description}
          </p>
        ) : null}
      </header>

      <dl data-rcl-approval-facts>
        <div data-rcl-approval-fact>
          <dt>{translate("ai.approval-prompt.text.2", "Action")}</dt>
          <dd>{action}</dd>
        </div>
        <div data-rcl-approval-fact>
          <dt>{translate("ai.approval-prompt.text.3", "Target")}</dt>
          <dd>{target}</dd>
        </div>
        <div data-rcl-approval-fact>
          <dt>{translate("ai.approval-prompt.text.4", "Expiry")}</dt>
          <dd data-rcl-approval-expiry data-expired={expired || undefined}>
            {expiryText ?? "No expiry set"}
          </dd>
        </div>
      </dl>

      {consequences ? (
        <section data-rcl-approval-section aria-labelledby={`${titleId}-consequences`}>
          <h3 id={`${titleId}-consequences`}>
            {translate("ai.approval-prompt.text.5", "What happens next")}
          </h3>
          {typeof consequences === "string" ? <p>{consequences}</p> : consequences}
        </section>
      ) : null}
      {alternatives ? (
        <section data-rcl-approval-section aria-labelledby={`${titleId}-alternatives`}>
          <h3 id={`${titleId}-alternatives`}>
            {translate("ai.approval-prompt.text.6", "Alternatives")}
          </h3>
          {typeof alternatives === "string" ? <p>{alternatives}</p> : alternatives}
        </section>
      ) : null}

      <div data-rcl-approval-consent>
        <span data-rcl-approval-consent-label>
          {translate("ai.approval-prompt.text.7", "Confirm the exact scope")}
        </span>
        <p data-rcl-approval-consent-line>
          {translate("ai.approval-prompt.text.8", "Approve")}
          <strong>{action}</strong>
          {translate("ai.approval-prompt.text.9", "for")}
          <strong>{target}</strong> within <strong>{scope}</strong>.
        </p>
      </div>

      {stateAlert}

      {resolvedStatus !== "success" ? (
        <ButtonGroup
          label={translate("ai.approval-prompt.label.8", "Approval actions")}
          data-rcl-approval-actions
        >
          <Button
            variant={resolvedStatus === "permission-denied" ? "secondary" : "primary"}
            pending={resolvedStatus === "submitting"}
            pendingLabel={pendingLabel}
            disabled={!canApprove && resolvedStatus !== "permission-denied"}
            onClick={() => {
              if (resolvedStatus === "permission-denied") void run("retry");
              else
                void run(
                  resolvedStatus === "request-error" || resolvedStatus === "retry"
                    ? "retry"
                    : "approve",
                );
            }}
          >
            {resolvedStatus === "permission-denied"
              ? permissionLabel
              : resolvedStatus === "request-error" || resolvedStatus === "retry"
                ? retryLabel
                : approveLabel}
          </Button>
          {resolvedStatus !== "submitting" && resolvedStatus !== "permission-denied" ? (
            <Button variant="ghost" disabled={busy} onClick={onDeny}>
              {denyLabel}
            </Button>
          ) : null}
        </ButtonGroup>
      ) : null}
    </section>
  );
}
