/**
 * @libraryId react-component-library:FormSection
 * @displayName FormSection
 * @description A semantic form grouping with responsive rhythm, optional collapse, summaries, and visible validation counts.
 * @version 1.0.3
 * @tags ["form","layout","validation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";

/** @vrooliComponentSource forms.form-section */
import { useId, useState, type CSSProperties, type ReactNode } from "react";
import { CollapsibleRegion } from "@vrooli/react-component-library/CollapsibleRegion/1.0.0";

export interface FormSectionProps {
  title: ReactNode;
  children: ReactNode;
  description?: ReactNode;
  summary?: ReactNode;
  errorCount?: number;
  actions?: ReactNode;
  collapsible?: boolean;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-form-section] { min-inline-size: 0; overflow: clip; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 0.5rem); background: var(--color-surface-raised, #ffffff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10)); }
  [data-rcl-form-section-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-md, 24px); padding: var(--space-lg, 32px); }
  [data-rcl-form-section-heading] { display: grid; gap: var(--space-3xs, 4px); min-inline-size: 0; }
  [data-rcl-form-section-title] { color: var(--color-foreground, #0f172a); font: var(--text-subtitle, 600 var(--text-subheading-size) / var(--text-subheading-line) var(--font-sans)); letter-spacing: var(--text-subtitle-tracking, 0); }
  [data-rcl-form-section-description] { max-inline-size: 62ch; color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }
  [data-rcl-form-section-summary] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
  [data-rcl-form-section-error] { display: inline-flex; align-items: center; gap: var(--space-3xs, 4px); color: var(--color-danger, #dc2626); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
  [data-rcl-form-section-error-mark] { display: inline-grid; place-items: center; inline-size: 1.125rem; block-size: 1.125rem; border: 1px solid currentColor; border-radius: 50%; font-size: .6875rem; }
  [data-rcl-form-section-toggle] { display: inline-flex; align-items: center; justify-content: center; flex: 0 0 auto; min-block-size: var(--tap-target-min, 44px); min-inline-size: var(--tap-target-min, 44px); margin: -.25rem -.25rem 0 0; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); background: transparent; color: var(--color-muted-foreground, #64748b); font: 700 1.125rem/1 system-ui, sans-serif; cursor: pointer; transition: background var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)); }
  [data-rcl-form-section-toggle]:hover { background: color-mix(in srgb, var(--color-primary, #2563eb) 8%, transparent); color: var(--color-primary, #2563eb); }
  [data-rcl-form-section-toggle][aria-expanded="true"] { color: var(--color-primary, #2563eb); }
  [data-rcl-form-section-content] { display: grid; gap: var(--space-md, 24px); padding: 0 var(--space-lg, 32px) var(--space-lg, 32px); }
  [data-rcl-form-section-content]::before { content: ""; block-size: 1px; background: var(--color-border, #cbd5e1); }
  @media (max-width: 30rem) { [data-rcl-form-section-header] { gap: var(--space-sm, 16px); padding: var(--space-md, 24px); } [data-rcl-form-section-content] { padding: 0 var(--space-md, 24px) var(--space-md, 24px); } [data-rcl-form-section-title] { font-size: .9375rem; } }

`;

export const FormSection = withClassName(function FormSection({
  title,
  children,
  description,
  summary,
  errorCount = 0,
  actions,
  collapsible = false,
  open,
  defaultOpen = true,
  onOpenChange,
  className,
  style,
}: FormSectionProps) {
  const generatedID = useId().replace(/:/g, "");
  const contentID = `form-section-${generatedID}-content`;
  const titleLabel = typeof title === "string" ? title : "form section";
  const [uncontrolledOpen, setUncontrolledOpen] = useState(defaultOpen);
  const isOpen = open ?? uncontrolledOpen;
  const setOpen = (next: boolean) => {
    if (open === undefined) setUncontrolledOpen(next);
    onOpenChange?.(next);
  };

  return (
    <section
      className={className}
      style={style}
      data-rcl-form-section
      data-open={isOpen}
    >
      <StyleSheet name="formsection-1-0-2-1" css={styles} />
      <header data-rcl-form-section-header>
        <div data-rcl-form-section-heading>
          <div data-rcl-form-section-title>{title}</div>
          {description && (
            <div data-rcl-form-section-description>{description}</div>
          )}
          {summary && <div data-rcl-form-section-summary>{summary}</div>}
          {errorCount > 0 && (
            <div data-rcl-form-section-error role="status" aria-live="polite">
              <span data-rcl-form-section-error-mark aria-hidden="true">
                !
              </span>
              {errorCount} {errorCount === 1 ? "issue" : "issues"} to review
            </div>
          )}
        </div>
        {collapsible && (
          <button
            data-testid="forms.form-section"
            type="button"
            data-rcl-form-section-toggle
            aria-expanded={isOpen}
            aria-controls={contentID}
            aria-label={`${isOpen ? "Collapse" : "Expand"} ${titleLabel}`}
            onClick={() => setOpen(!isOpen)}
          >
            {isOpen ? "⌃" : "⌄"}
          </button>
        )}
        {!collapsible && actions}
      </header>
      <CollapsibleRegion open={isOpen}>
        <div
          id={contentID}
          data-rcl-form-section-content
          hidden={!isOpen}
          aria-hidden={!isOpen || undefined}
        >
          {children}
        </div>
      </CollapsibleRegion>
    </section>
  );
});
