/**
 * TitlePopover
 *
 * Header title that wraps to `clampLines` lines (two by default — enough for
 * nearly every entity title) and reveals the full text plus a copy button on
 * click. The popover is the overflow escape hatch, not the only way to read a
 * title: a single-line clamp made long titles unreadable everywhere.
 */

import { useCallback, useRef, useState } from "react";
import { Check, Copy } from "lucide-react";
import { cn } from "../../lib/utils";
import { Popover } from "../ui/popover";

const CLAMP_CLASS: Record<number, string> = {
  1: "line-clamp-1",
  2: "line-clamp-2",
  3: "line-clamp-3",
};

interface TitlePopoverProps {
  title: string;
  /** Optional className applied to the inline title button. */
  className?: string;
  /** How many lines the title may wrap to before clamping. Defaults to 2. */
  clampLines?: 1 | 2 | 3;
}

export function TitlePopover({ title, className, clampLines = 2 }: TitlePopoverProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [pos, setPos] = useState({ x: 0, y: 0 });
  const [copied, setCopied] = useState(false);
  const buttonRef = useRef<HTMLButtonElement>(null);

  const handleOpen = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    const rect = buttonRef.current?.getBoundingClientRect();
    if (rect) {
      setPos({ x: rect.left, y: rect.bottom + 4 });
    }
    setCopied(false);
    setIsOpen(true);
  }, []);

  const handleClose = useCallback(() => {
    setIsOpen(false);
  }, []);

  const handleCopy = useCallback(async (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(title);
        setCopied(true);
        return;
      } catch {
        // Fall through to legacy copy.
      }
    }
    try {
      const textarea = document.createElement("textarea");
      textarea.value = title;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.focus();
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setCopied(true);
    } catch {
      // Ignore copy errors.
    }
  }, [title]);

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        onClick={handleOpen}
        title={title}
        aria-label={`${title} — show full title`}
        className={className}
        data-testid="detail-title-button"
      >
        <span className={cn("block text-left", CLAMP_CLASS[clampLines] ?? CLAMP_CLASS[2])}>
          {title}
        </span>
      </button>

      <Popover
        isOpen={isOpen}
        onClose={handleClose}
        x={pos.x}
        y={pos.y}
        className="max-w-[min(90vw,32rem)] p-3"
        testId="detail-title-popover"
      >
        <div className="flex items-start gap-2">
          <span
            className="flex-1 select-all break-words text-sm text-slate-100"
            data-testid="detail-title-popover-text"
          >
            {title}
          </span>
          <button
            type="button"
            onClick={handleCopy}
            title={copied ? "Copied" : "Copy title"}
            aria-label={copied ? "Copied" : "Copy title"}
            className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-white/20 text-slate-300 transition-colors hover:bg-white/10 active:bg-white/20"
            data-testid="detail-title-copy-button"
          >
            {copied ? (
              <Check className="h-3.5 w-3.5 text-emerald-300" />
            ) : (
              <Copy className="h-3.5 w-3.5" />
            )}
          </button>
        </div>
      </Popover>
    </>
  );
}
