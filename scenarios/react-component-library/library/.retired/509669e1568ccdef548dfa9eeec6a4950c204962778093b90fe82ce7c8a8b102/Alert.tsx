/**
 * @libraryId react-component-library:Alert
 * @displayName Alert
 * @description A semantic local message surface for information, success, warning, or error with resilient actions and dismissal.
 * @version 1.0.4
 * @tags ["feedback","status","recovery","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:Alert */
import { useId, type CSSProperties, type ReactNode } from "react";
import { Icon } from "@vrooli/react-component-library/Icon/1";
import { Stack } from "@vrooli/react-component-library/Stack/1";
import { Text } from "@vrooli/react-component-library/Text/1";

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
[data-rcl-alert] { --rcl-alert-accent: var(--color-primary, #2563eb); --rcl-alert-surface: color-mix(in srgb, var(--rcl-alert-accent) 8%, var(--color-surface, #ffffff)); --rcl-alert-border: color-mix(in srgb, var(--rcl-alert-accent) 32%, var(--color-border, #cbd5e1)); min-inline-size: 0; box-sizing: border-box; display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: start; gap: var(--space-sm, 16px); padding: var(--space-md, 24px); border: 1px solid var(--rcl-alert-border); border-inline-start: 4px solid var(--rcl-alert-accent); border-radius: var(--radius-panel, 0.5rem); background: var(--rcl-alert-surface); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-flat, none); }
[data-rcl-alert][data-tone="success"] { --rcl-alert-accent: var(--color-success, #16a34a); }
[data-rcl-alert][data-tone="warning"] { --rcl-alert-accent: var(--color-warning, #d97706); }
[data-rcl-alert][data-tone="danger"] { --rcl-alert-accent: var(--color-danger, #dc2626); }
[data-rcl-alert-icon] { display: grid; place-items: center; inline-size: var(--space-xl, 40px); block-size: var(--space-xl, 40px); flex: 0 0 auto; border-radius: var(--radius-control, 0.375rem); background: color-mix(in srgb, var(--rcl-alert-accent) 16%, transparent); color: var(--rcl-alert-accent); }
[data-rcl-alert-icon] svg { inline-size: var(--icon-size-md, 20px); block-size: var(--icon-size-md, 20px); stroke-width: 2.5; }
[data-rcl-alert-title] { color: var(--color-foreground, #0f172a); font-weight: 750; }
[data-rcl-alert-description] { color: var(--color-muted-foreground, #64748b); line-height: 1.5; }
[data-rcl-alert-actions] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2xs, 8px); margin-block-start: var(--space-xs, 12px); }
[data-rcl-alert-actions] > * { min-block-size: var(--tap-target-min, 44px); }
[data-rcl-alert-close] { display: grid; place-items: center; inline-size: var(--tap-target-min, 44px); block-size: var(--tap-target-min, 44px); margin: calc(var(--space-2xs, 8px) * -1) calc(var(--space-2xs, 8px) * -1) 0 0; border: 0; border-radius: var(--radius-control, 0.375rem); background: transparent; color: var(--color-muted-foreground, #64748b); cursor: pointer; }
[data-rcl-alert-close]:hover { background: color-mix(in srgb, currentColor 10%, transparent); color: var(--color-foreground, #0f172a); }
[data-rcl-alert-close] svg { inline-size: var(--icon-size-sm, 16px); block-size: var(--icon-size-sm, 16px); }
@media (max-width: 30rem) { [data-rcl-alert] { grid-template-columns: auto minmax(0, 1fr); padding: var(--space-sm, 16px); } [data-rcl-alert-close] { grid-column: 2; grid-row: 1; justify-self: end; } [data-rcl-alert-content] { padding-inline-end: var(--space-2xs, 8px); } [data-rcl-alert-actions] { grid-column: 2; } }

`;

export const Alert = withClassName(function Alert({
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
      <StyleSheet name="alert-1-0-2-1" css={styles} />
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
          data-testid="feedback.alert"
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
});
