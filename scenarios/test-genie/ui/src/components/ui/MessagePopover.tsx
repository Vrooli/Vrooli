import { useState, useRef, useEffect } from "react";
import { MessageSquarePlus, X } from "lucide-react";
import { cn } from "../../lib/utils";

interface MessagePopoverProps {
  message: string;
  onChange: (message: string) => void;
  disabled?: boolean;
  placeholder?: string;
  maxLength?: number;
  className?: string;
  "data-testid"?: string;
}

export function MessagePopover({
  message,
  onChange,
  disabled = false,
  placeholder = "Add context for the AI agent...",
  maxLength = 2000,
  className,
  "data-testid": dataTestId
}: MessagePopoverProps) {
  const [isOpen, setIsOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  // Close popover when clicking outside
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (event: MouseEvent) => {
      if (
        popoverRef.current &&
        !popoverRef.current.contains(event.target as Node) &&
        buttonRef.current &&
        !buttonRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isOpen]);

  const hasMessage = message.trim().length > 0;

  return (
    <div className={cn("relative inline-block", className)} data-testid={dataTestId}>
      <button
        ref={buttonRef}
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        disabled={disabled}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-2 py-1 text-xs transition",
          hasMessage
            ? "text-violet-400 hover:text-violet-300"
            : "text-slate-400 hover:text-slate-300",
          disabled && "cursor-not-allowed opacity-50"
        )}
      >
        <MessageSquarePlus className="h-3.5 w-3.5" />
        <span>{hasMessage ? "Note added" : "Add note"}</span>
        {hasMessage && (
          <span className="ml-0.5 h-1.5 w-1.5 rounded-full bg-violet-400" />
        )}
      </button>

      {isOpen && (
        <div
          ref={popoverRef}
          className="absolute right-0 top-full z-50 mt-1 w-80 rounded-lg border border-white/10 bg-slate-900 p-3 shadow-xl"
        >
          <div className="mb-2 flex items-center justify-between">
            <label className="text-xs font-medium text-slate-300">
              Additional Context
            </label>
            <button
              type="button"
              onClick={() => setIsOpen(false)}
              className="text-slate-500 hover:text-slate-300"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <textarea
            value={message}
            onChange={(e) => onChange(e.target.value)}
            placeholder={placeholder}
            maxLength={maxLength}
            disabled={disabled}
            rows={4}
            className={cn(
              "w-full resize-none rounded-md border border-white/10 bg-black/30 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-white/20 focus:outline-none",
              disabled && "cursor-not-allowed opacity-50"
            )}
          />
          <div className="mt-1.5 flex items-center justify-between text-xs text-slate-500">
            <span>Optional context to help the AI</span>
            <span>
              {message.length}/{maxLength}
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
