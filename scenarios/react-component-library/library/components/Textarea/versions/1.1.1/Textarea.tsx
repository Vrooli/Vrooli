/**
 * @libraryId react-component-library:Textarea
 * @displayName Textarea
 * @version 1.1.1
 * @tags ["form","interactive"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const styleSheet = `
[data-rcl-textarea] { box-sizing: border-box; display: block; inline-size: 100%; min-block-size: calc(var(--space-xl) * 2); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-field, var(--color-surface)); color: var(--color-foreground); padding: var(--space-2xs) var(--space-sm); font: inherit; line-height: var(--text-body-line); resize: vertical; transition: border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard), background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-textarea]::placeholder { color: var(--color-muted-foreground); opacity: var(--opacity-muted); }
[data-rcl-textarea]:hover:not(:disabled) { border-color: var(--color-primary); background: var(--color-surface-raised); }
[data-rcl-textarea][aria-invalid="true"] { border-color: var(--color-danger); }
[data-rcl-textarea]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); resize: none; }

/* The "mobile-safe font sizing" this component has always advertised, finally
   implemented. iOS zooms the whole viewport when a text control smaller than
   16px receives focus, which on a fixed-position layout is a one-way trip.
   Deliberately at the component's own weight, so a consumer that states a size
   still wins: the promise is that the *default* is safe, not that the library
   overrules a product decision. '1em' resolves against the parent, so this is
   a floor and never an enlargement. */
@media (pointer: coarse) {
  [data-rcl-textarea] { font-size: max(16px, 1em); }
}
`;

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <>
        <StyleSheet name="textarea-1-1-0" css={styleSheet} />
        <textarea
          data-testid="forms.textarea"
          data-rcl-textarea="true"
          className={className}
          ref={ref}
          {...props}
        />
      </>
    );
  },
);
Textarea.displayName = "Textarea";

export { Textarea };
