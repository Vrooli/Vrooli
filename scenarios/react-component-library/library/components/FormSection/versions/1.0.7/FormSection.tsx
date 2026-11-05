/**
 * @libraryId react-component-library:FormSection
 * @displayName FormSection
 * @description The semantic grouping component with heading, description, responsive layout, optional collapse, summary, error badge, and compatibility with sticky section navigation.
 * @version 1.0.7
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:FormSection
 * @vrooliComponentSourceSlot forms.form-section */
import { useId, useState, type CSSProperties, type ReactNode } from "react";
import { CollapsibleRegion } from "@vrooli/react-component-library/CollapsibleRegion/1";

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
  [data-rcl-form-section] { container: rcl-form-section / inline-size; min-inline-size: 0; overflow: clip; border: 1px solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-raised); }
  [data-rcl-form-section-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-md); padding: var(--space-lg); }
  [data-rcl-form-section-heading] { display: grid; gap: var(--space-3xs); min-inline-size: 0; }
  [data-rcl-form-section-title] { color: var(--color-foreground); font: var(--text-subtitle); letter-spacing: var(--text-subtitle-tracking); }
  [data-rcl-form-section-description] { max-inline-size: 62ch; color: var(--color-muted-foreground); font: var(--text-body); }
  [data-rcl-form-section-summary] { color: var(--color-muted-foreground); font: var(--text-caption); }
  [data-rcl-form-section-error] { display: inline-flex; align-items: center; gap: var(--space-3xs); color: var(--color-danger); font: var(--text-caption); }
  [data-rcl-form-section-error-mark] { display: inline-grid; place-items: center; inline-size: 1.125rem; block-size: 1.125rem; border: 1px solid currentColor; border-radius: 50%; font-size: .6875rem; }
  [data-rcl-form-section-toggle] { display: inline-flex; align-items: center; justify-content: center; flex: 0 0 auto; min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); margin: -.25rem -.25rem 0 0; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); font: 700 1.125rem/1 system-ui, sans-serif; cursor: pointer; transition: background var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
  [data-rcl-form-section-toggle]:hover { background: color-mix(in srgb, var(--color-primary) 8%, transparent); color: var(--color-primary); }
  [data-rcl-form-section-toggle][aria-expanded="true"] { color: var(--color-primary); }
  [data-rcl-form-section-content] { display: grid; gap: var(--space-md); padding: 0 var(--space-lg) var(--space-lg); }
  [data-rcl-form-section-content]::before { content: ""; block-size: 1px; background: var(--color-border); }
  @container rcl-form-section (max-width: 30rem) { [data-rcl-form-section-header] { gap: var(--space-sm); padding: var(--space-md); } [data-rcl-form-section-content] { padding: 0 var(--space-md) var(--space-md); } [data-rcl-form-section-title] { font-size: .9375rem; } }

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
    <section className={className} style={style} data-rcl-form-section data-open={isOpen}>
      <StyleSheet libraryId="react-component-library:FormSection" version="1.0.6" css={styles} />
      <header data-rcl-form-section-header>
        <div data-rcl-form-section-heading>
          <div data-rcl-form-section-title>{title}</div>
          {description && <div data-rcl-form-section-description>{description}</div>}
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
