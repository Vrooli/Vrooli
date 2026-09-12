/**
 * @libraryId react-component-library:CopyIconButton
 * @displayName CopyIconButton
 * @description An icon button that copies text, confirms by morphing its icon to a check in the success colour, and announces the result.
 * @version 0.1.0
 * @tags ["controls","clipboard","feedback"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:CopyIconButton
 * @vrooliComponentSourceSlot controls.copy-icon-button */
import {
  forwardRef,
  useEffect,
  useRef,
  useState,
  type MouseEvent as ReactMouseEvent,
} from "react";
import { IconButton, type IconButtonProps } from "@vrooli/react-component-library/IconButton/3";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/**
 * Why a copy control is its own asset.
 *
 * Every surface that copies had rebuilt the same four things by hand — a
 * clipboard write with a fallback for insecure contexts, a timer that flips a
 * "copied" flag back, a second icon, and a colour for it — and each got a
 * different subset. The result was copy buttons that differed in surface,
 * size, colour, and whether they announced anything at all. This is IconButton
 * with those four things decided once: the icon change animates through
 * IconButton's own morph, success turns the glyph the success colour, failure
 * the danger colour, and both are announced.
 */
export const copyIconButtonStyles = `
[data-rcl-icon-button][data-rcl-copy-button][data-rcl-copy-state="copied"],
[data-rcl-icon-button][data-rcl-copy-button][data-rcl-copy-state="copied"]:hover:not(:disabled) { color: var(--color-success); }
[data-rcl-icon-button][data-rcl-copy-button][data-rcl-copy-state="failed"],
[data-rcl-icon-button][data-rcl-copy-button][data-rcl-copy-state="failed"]:hover:not(:disabled) { color: var(--color-danger); }
`;

export type CopyIconButtonState = "idle" | "copied" | "failed";

/**
 * The clipboard write, with the gesture-bound fallback an insecure context
 * (a LAN address over plain HTTP) needs: there `navigator.clipboard` is absent.
 */
export async function writeClipboardText(text: string): Promise<boolean> {
  if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fall through to the legacy path, which works where the API is denied.
    }
  }
  if (typeof document === "undefined") return false;
  const area = document.createElement("textarea");
  area.value = text;
  area.setAttribute("readonly", "");
  area.style.position = "fixed";
  area.style.opacity = "0";
  document.body.appendChild(area);
  area.select();
  try {
    return document.execCommand("copy");
  } catch {
    return false;
  } finally {
    area.remove();
  }
}

/*
 * Three distinct module-scope glyphs. IconButton keys its morph on component
 * identity, so declaring them here — not inline — is what makes the swap
 * animate.
 */
function CopyGlyph() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect width="13" height="13" x="9" y="9" rx="2" ry="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  );
}

function CopiedGlyph() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M20 6 9 17l-5-5" />
    </svg>
  );
}

function FailedGlyph() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M18 6 6 18" />
      <path d="m6 6 12 12" />
    </svg>
  );
}

export interface CopyIconButtonProps
  extends Omit<IconButtonProps, "children" | "aria-label" | "selected" | "iconKey"> {
  /** The text to copy, or a function that reads it at the moment of the press. */
  value: string | (() => string);
  /** Names the action at rest, e.g. "Copy message". */
  "aria-label": string;
  /** Tooltip and announcement after a successful copy. */
  copiedLabel?: string;
  /** Tooltip and announcement after a failed copy. */
  failedLabel?: string;
  /** How long the result stays on the button, in milliseconds. */
  resetAfterMs?: number;
  /**
   * Replaces the built-in clipboard write. Returning or resolving `false`, or
   * throwing, reports a failure.
   */
  writeText?: (text: string) => boolean | void | Promise<boolean | void>;
  /**
   * Shows the copied state without a press here — for a surface that can also
   * copy the same text another way, such as a menu row.
   */
  copied?: boolean;
  /** Called with the text after it reached the clipboard. */
  onCopied?: (text: string) => void;
}

export const CopyIconButton = forwardRef<HTMLButtonElement, CopyIconButtonProps>(function CopyIconButton(
  {
    value,
    "aria-label": ariaLabel,
    copiedLabel = "Copied",
    failedLabel = "Copy failed",
    resetAfterMs = 2000,
    writeText = writeClipboardText,
    copied = false,
    onCopied,
    onClick,
    title,
    ...props
  },
  ref,
) {
  useLibraryStyleSheet("react-component-library:CopyIconButton", "1.0.0", copyIconButtonStyles);
  const [result, setResult] = useState<CopyIconButtonState>("idle");
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => () => {
    if (timerRef.current) clearTimeout(timerRef.current);
  }, []);

  const settle = (next: CopyIconButtonState) => {
    setResult(next);
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => { setResult("idle"); }, resetAfterMs);
  };

  const handleClick = (event: ReactMouseEvent<HTMLButtonElement>) => {
    onClick?.(event);
    if (event.defaultPrevented) return;
    const text = typeof value === "function" ? value() : value;
    // The write starts inside the press itself: the legacy fallback only
    // works during a user gesture.
    let pending: ReturnType<NonNullable<CopyIconButtonProps["writeText"]>>;
    try {
      pending = writeText(text);
    } catch {
      settle("failed");
      return;
    }
    Promise.resolve(pending).then(
      (ok) => {
        if (ok === false) {
          settle("failed");
          return;
        }
        settle("copied");
        onCopied?.(text);
      },
      () => { settle("failed"); },
    );
  };

  const state: CopyIconButtonState = result === "idle" && copied ? "copied" : result;
  const announcement = state === "copied" ? copiedLabel : state === "failed" ? failedLabel : "";

  return (
    <>
      <IconButton
        {...props}
        ref={ref}
        aria-label={ariaLabel}
        title={announcement || title}
        onClick={handleClick}
        data-rcl-copy-button=""
        data-rcl-copy-state={state}
      >
        {state === "copied" ? <CopiedGlyph /> : state === "failed" ? <FailedGlyph /> : <CopyGlyph />}
      </IconButton>
      <span className="rcl-visually-hidden" role="status" aria-live="polite">
        {announcement}
      </span>
    </>
  );
});