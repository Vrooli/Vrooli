/**
 * @libraryId react-component-library:Textarea
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 * @category controls
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import * as React from "react";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const styleSheet = `
[data-rcl-textarea] { box-sizing: border-box; display: block; inline-size: 100%; min-block-size: calc(var(--space-xl) * 2); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding: var(--space-2xs) var(--space-sm); font: inherit; line-height: var(--text-body-line); resize: vertical; transition: border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard), background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-textarea]::placeholder { color: var(--color-muted-foreground); opacity: var(--opacity-muted); }
[data-rcl-textarea]:hover:not(:disabled) { border-color: var(--color-primary); background: var(--color-surface-raised); }
[data-rcl-textarea][aria-invalid="true"] { border-color: var(--color-danger); }
[data-rcl-textarea]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); resize: none; }

`;

const joinClasses = (...inputs: Array<string | undefined>) =>
  inputs.filter(Boolean).join(" ");

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <>
        <StyleSheet name="textarea-1-0-0-1" css={styleSheet} />
        <textarea
          data-rcl-textarea="true"
          className={joinClasses(
            "rounded-control border border-app-border",
            className,
          )}
          ref={ref}
          {...props}
        />
      </>
    );
  },
);
Textarea.displayName = "Textarea";

export { Textarea };
