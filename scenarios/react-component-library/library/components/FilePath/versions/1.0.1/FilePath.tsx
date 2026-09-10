/**
 * @libraryId react-component-library:FilePath
 * @displayName FilePath
 * @description
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { Check, Copy, FileText } from "lucide-react";
import { useCallback, useState } from "react";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { Popover, PopoverContent, PopoverTrigger } from "@vrooli/react-component-library/Popover/1";

/** A path that stays compact in a header but exposes the complete value on demand. */
export interface FilePathProps {
  path: string;
  /** Show the copy affordance beside the truncated path. The popover always has one. */
  showCopyButton?: boolean;
  copyLabel?: string;
  copiedLabel?: string;
  className?: string;
  testId?: string;
  copyButtonTestId?: string;
}

const styles = `
[data-rcl-file-path] { display: flex; min-inline-size: 0; max-inline-size: 100%; align-items: center; gap: var(--space-2xs); color: var(--color-foreground); }
[data-rcl-file-path-trigger] { display: inline-flex; min-inline-size: 0; max-inline-size: 100%; flex: 1 1 auto; align-items: center; gap: var(--space-2xs); overflow: hidden; border: 0; background: transparent; color: inherit; cursor: pointer; font: var(--text-caption); text-align: start; }
[data-rcl-file-path-trigger] > span { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-file-path-popover] { display: flex; max-inline-size: min(42rem, calc(100vw - 2rem)); align-items: flex-start; gap: var(--space-sm); padding: var(--space-sm); }
[data-rcl-file-path-popover] code { min-inline-size: 0; flex: 1 1 auto; overflow-wrap: anywhere; color: var(--color-foreground); font: var(--text-caption); user-select: text; }
`;

async function copyText(value: string): Promise<void> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value);
      return;
    }
  } catch {
    // Fall through to the legacy path for local shells and restricted contexts.
  }
  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "fixed";
  textarea.style.insetInlineStart = "-9999px";
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand("copy");
  textarea.remove();
}

export function FilePath({
  path,
  showCopyButton = false,
  copyLabel = "Copy path",
  copiedLabel = "Copied",
  className,
  testId = "file-path",
  copyButtonTestId,
}: FilePathProps) {
  const [copied, setCopied] = useState(false);
  const copy = useCallback(() => {
    void copyText(path).then(() => {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    });
  }, [path]);

  return (
    <div data-rcl-file-path data-testid={testId} className={`rcl-file-path ${className ?? ""}`}>
      <style dangerouslySetInnerHTML={{ __html: styles }} />
      <Popover placement="bottom-start" responsive="auto">
        <PopoverTrigger asChild>
          <button type="button" data-rcl-file-path-trigger aria-label={path} title={path}>
            <FileText aria-hidden="true" size={14} />
            <span>{path}</span>
          </button>
        </PopoverTrigger>
        <PopoverContent initialFocus="none" data-rcl-file-path-popover>
          <code>{path}</code>
          <IconButton
            type="button"
            size="sm"
            surface="soft"
            data-testid={`${testId}.popover-copy`}
            aria-label={copied ? copiedLabel : copyLabel}
            title={copied ? copiedLabel : copyLabel}
            onClick={copy}
          >
            {copied ? (
              <Check size={15} aria-hidden="true" />
            ) : (
              <Copy size={15} aria-hidden="true" />
            )}
          </IconButton>
        </PopoverContent>
      </Popover>
      {showCopyButton ? (
        <IconButton
          type="button"
          size="sm"
          surface="soft"
          data-testid={copyButtonTestId}
          aria-label={copied ? copiedLabel : copyLabel}
          title={copied ? copiedLabel : copyLabel}
          onClick={copy}
        >
          {copied ? <Check size={15} aria-hidden="true" /> : <Copy size={15} aria-hidden="true" />}
        </IconButton>
      ) : null}
    </div>
  );
}
