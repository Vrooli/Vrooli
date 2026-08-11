/**
 * @vrooliComponentSource react-component-library:Textarea
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 92d58798-3a73-4ac5-9bfe-6eef7c233d59
 * @vrooliComponentAppliedAt 2026-08-11T00:47:59Z
 * @vrooliComponentSourceSha256 d6bfb596a63f62d8ef6ec5daf84512cc633a48ecd0c9f21dd137ef8f3601e3ac
 * @vrooliComponentDriftHash 41be79d56b028bed02939be48ea6ca6d61eef8561f9281ca37e0e0fb4dfe65e5
 * @vrooliComponentTokenTranslation border-app-border->border-app-border
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import * as React from "react";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const styleSheet = `
[data-rcl-textarea] { box-sizing: border-box; display: block; inline-size: 100%; min-block-size: calc(var(--space-xl) * 2); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding: var(--space-2xs) var(--space-sm); font: inherit; line-height: var(--text-body-line); resize: vertical; transition: border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard), background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-textarea]::placeholder { color: var(--color-muted-foreground); opacity: var(--opacity-muted); }
[data-rcl-textarea]:hover:not(:disabled) { border-color: var(--color-primary); background: var(--color-surface-raised); }
[data-rcl-textarea]:focus-visible { border-color: var(--color-focus); outline: var(--border-strong) solid color-mix(in srgb, var(--color-focus) 30%, transparent); outline-offset: var(--space-3xs); }
[data-rcl-textarea][aria-invalid="true"] { border-color: var(--color-danger); }
[data-rcl-textarea]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); resize: none; }
@media (prefers-reduced-motion: reduce) { [data-rcl-textarea] { transition: none; } }
`;

const joinClasses = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <>
        <style data-rcl-textarea-styles dangerouslySetInnerHTML={{ __html: styleSheet }} />
        <textarea
          data-rcl-textarea="true"
          className={joinClasses("rounded-control border border-app-border", className)}
          ref={ref}
          {...props}
        />
      </>
    );
  },
);
Textarea.displayName = "Textarea";

export { Textarea };
