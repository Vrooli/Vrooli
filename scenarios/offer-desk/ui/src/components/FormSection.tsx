/**
 * @vrooliComponentSource react-component-library:FormSection
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 386a33cb-0355-4a03-8854-186503c5d655
 * @vrooliComponentAppliedAt 2026-08-16T04:09:29Z
 * @vrooliComponentSourceSha256 1eca4ca10ee3ab7df83a9150e240e0480f0088a88d43f6761b28e5e5cd92a334
 * @vrooliComponentDriftHash fca4823ba48cd1a6fc1e173c1348bf2f3fdafb0c8f209181af703d0dcc4d5b78
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useId, useState, type CSSProperties, type ReactNode } from "react";
import { CollapsibleRegion } from "./CollapsibleRegion";

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
  [data-rcl-form-section] { min-inline-size: 0; overflow: clip; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, .875rem); background: var(--color-surface-raised, #fff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08)); }
  [data-rcl-form-section-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-md, 1rem); padding: var(--space-lg, 1.5rem); }
  [data-rcl-form-section-heading] { display: grid; gap: var(--space-3xs, .25rem); min-inline-size: 0; }
  [data-rcl-form-section-title] { color: var(--color-foreground, #0f172a); font: var(--text-subtitle, 700 1rem/1.35 system-ui, sans-serif); letter-spacing: var(--text-subtitle-tracking, -.01em); }
  [data-rcl-form-section-description] { max-inline-size: 62ch; color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 .875rem/1.375rem system-ui, sans-serif); }
  [data-rcl-form-section-summary] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 500 .75rem/1rem system-ui, sans-serif); }
  [data-rcl-form-section-error] { display: inline-flex; align-items: center; gap: var(--space-3xs, .25rem); color: var(--color-danger, #dc2626); font: var(--text-caption, 650 .75rem/1rem system-ui, sans-serif); }
  [data-rcl-form-section-error-mark] { display: inline-grid; place-items: center; inline-size: 1.125rem; block-size: 1.125rem; border: 1px solid currentColor; border-radius: 50%; font-size: .6875rem; }
  [data-rcl-form-section-toggle] { display: inline-flex; align-items: center; justify-content: center; flex: 0 0 auto; min-block-size: var(--tap-target-min, 44px); min-inline-size: var(--tap-target-min, 44px); margin: -.25rem -.25rem 0 0; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, .625rem); background: transparent; color: var(--color-muted-foreground, #64748b); font: 700 1.125rem/1 system-ui, sans-serif; cursor: pointer; transition: background var(--dur-quick, 160ms) var(--ease-standard, ease), color var(--dur-quick, 160ms) var(--ease-standard, ease); }
  [data-rcl-form-section-toggle]:hover { background: color-mix(in srgb, var(--color-primary, #2563eb) 8%, transparent); color: var(--color-primary, #2563eb); }
  [data-rcl-form-section-toggle]:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-primary, #2563eb) 28%, transparent); outline-offset: 2px; }
  [data-rcl-form-section-toggle][aria-expanded="true"] { color: var(--color-primary, #2563eb); }
  [data-rcl-form-section-content] { display: grid; gap: var(--space-md, 1rem); padding: 0 var(--space-lg, 1.5rem) var(--space-lg, 1.5rem); }
  [data-rcl-form-section-content]::before { content: ""; block-size: 1px; background: var(--color-border, #cbd5e1); }
  @media (max-width: 30rem) { [data-rcl-form-section-header] { gap: var(--space-sm, .75rem); padding: var(--space-md, 1rem); } [data-rcl-form-section-content] { padding: 0 var(--space-md, 1rem) var(--space-md, 1rem); } [data-rcl-form-section-title] { font-size: .9375rem; } }
  @media (prefers-reduced-motion: reduce) { [data-rcl-form-section-toggle] { transition: none; } }
`;

export function FormSection({
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
      <style
        data-rcl-form-section-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
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
              {errorCount}
            </div>
          )}
        </div>
        {collapsible && (
          <button
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
}
