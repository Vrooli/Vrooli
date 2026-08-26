/** @vrooliComponentSource react-component-library:Alert */
import { useId, type CSSProperties, type ReactNode } from "react";
import { Icon } from "../../../../primitives/Icon/versions/1.0.0/Icon";
import { Stack } from "../../../../primitives/Stack/versions/1.0.0/Stack";
import { Text } from "../../../../primitives/Text/versions/1.0.0/Text";

export type AlertTone = "info" | "success" | "warning" | "danger";

export interface AlertProps {
  tone?: AlertTone;
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
  dismissible?: boolean;
  dismissLabel?: string;
  onDismiss?: () => void;
  className?: string;
  style?: CSSProperties;
}

const toneCopy: Record<
  AlertTone,
  { label: string; icon: "chevronRight" | "check" | "plus" | "close" }
> = {
  info: { label: "Information", icon: "chevronRight" },
  success: { label: "Success", icon: "check" },
  warning: { label: "Warning", icon: "plus" },
  danger: { label: "Error", icon: "close" },
};

const styles = `
[data-rcl-alert] { --rcl-alert-accent: var(--color-primary, #2563eb); --rcl-alert-surface: color-mix(in srgb, var(--rcl-alert-accent) 8%, var(--color-surface, #fff)); --rcl-alert-border: color-mix(in srgb, var(--rcl-alert-accent) 32%, var(--color-border, #cbd5e1)); min-inline-size: 0; box-sizing: border-box; display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: start; gap: var(--space-sm, 12px); padding: var(--space-md, 16px); border: 1px solid var(--rcl-alert-border); border-inline-start: 4px solid var(--rcl-alert-accent); border-radius: var(--radius-panel, 12px); background: var(--rcl-alert-surface); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-flat, 0 1px 2px rgb(15 23 42 / .08)); }
[data-rcl-alert][data-tone="success"] { --rcl-alert-accent: var(--color-success, #15803d); }
[data-rcl-alert][data-tone="warning"] { --rcl-alert-accent: var(--color-warning, #b45309); }
[data-rcl-alert][data-tone="danger"] { --rcl-alert-accent: var(--color-danger, #dc2626); }
[data-rcl-alert-icon] { display: grid; place-items: center; inline-size: var(--space-xl, 40px); block-size: var(--space-xl, 40px); flex: 0 0 auto; border-radius: var(--radius-control, 8px); background: color-mix(in srgb, var(--rcl-alert-accent) 16%, transparent); color: var(--rcl-alert-accent); }
[data-rcl-alert-icon] svg { inline-size: var(--icon-size-md, 20px); block-size: var(--icon-size-md, 20px); stroke-width: 2.5; }
[data-rcl-alert-title] { color: var(--color-foreground, #0f172a); font-weight: 750; }
[data-rcl-alert-description] { color: var(--color-muted-foreground, #64748b); line-height: 1.5; }
[data-rcl-alert-actions] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2xs, 8px); margin-block-start: var(--space-xs, 12px); }
[data-rcl-alert-actions] > * { min-block-size: var(--tap-target-min, 44px); }
[data-rcl-alert-close] { display: grid; place-items: center; inline-size: var(--tap-target-min, 44px); block-size: var(--tap-target-min, 44px); margin: calc(var(--space-2xs, 8px) * -1) calc(var(--space-2xs, 8px) * -1) 0 0; border: 0; border-radius: var(--radius-control, 8px); background: transparent; color: var(--color-muted-foreground, #64748b); cursor: pointer; }
[data-rcl-alert-close]:hover { background: color-mix(in srgb, currentColor 10%, transparent); color: var(--color-foreground, #0f172a); }
[data-rcl-alert-close]:focus-visible { outline: 3px solid var(--color-focus-ring, #2563eb); outline-offset: 2px; }
[data-rcl-alert-close] svg { inline-size: var(--icon-size-sm, 16px); block-size: var(--icon-size-sm, 16px); }
@media (max-width: 30rem) { [data-rcl-alert] { grid-template-columns: auto minmax(0, 1fr); padding: var(--space-sm, 12px); } [data-rcl-alert-close] { grid-column: 2; grid-row: 1; justify-self: end; } [data-rcl-alert-content] { padding-inline-end: var(--space-2xs, 8px); } [data-rcl-alert-actions] { grid-column: 2; } }
@media (forced-colors: active) { [data-rcl-alert] { border-color: CanvasText; border-inline-start-color: Highlight; background: Canvas; } [data-rcl-alert-icon] { border: 1px solid CanvasText; background: Canvas; color: Highlight; } [data-rcl-alert-close] { color: CanvasText; } }
`;

export function Alert({
  tone = "info",
  title,
  description,
  actions,
  dismissible = false,
  dismissLabel = "Dismiss message",
  onDismiss,
  className,
  style,
}: AlertProps) {
  const descriptionId = useId();
  const copy = toneCopy[tone];
  const role = tone === "danger" ? "alert" : "status";

  return (
    <div
      data-rcl-alert
      data-tone={tone}
      data-rcl-alert-role={role}
      role={role}
      aria-live={tone === "danger" ? "assertive" : "polite"}
      aria-describedby={description ? descriptionId : undefined}
      className={className}
      style={style}
    >
      <style data-rcl-alert-styles>{styles}</style>
      <div data-rcl-alert-icon aria-label={copy.label} role="img">
        <Icon name={copy.icon} />
      </div>
      <Stack data-rcl-alert-content gap="3xs">
        <Text as="strong" data-rcl-alert-title>
          {title}
        </Text>
        {description && (
          <Text id={descriptionId} data-rcl-alert-description>
            {description}
          </Text>
        )}
        {actions && <div data-rcl-alert-actions>{actions}</div>}
      </Stack>
      {dismissible && (
        <button
          type="button"
          data-rcl-alert-close
          aria-label={dismissLabel}
          onClick={onDismiss}
        >
          <Icon name="close" />
        </button>
      )}
    </div>
  );
}
