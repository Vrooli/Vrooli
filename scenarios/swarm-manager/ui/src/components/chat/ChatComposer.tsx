import { useCallback, useRef } from "react";
import type { KeyboardEvent, ReactNode } from "react";
import { Loader2, SendHorizontal } from "lucide-react";
import { Button } from "../ui/button";
import { Textarea } from "../ui/textarea";
import { useAutoResizeTextarea } from "../../hooks/useAutoResizeTextarea";
import { cn } from "../../lib/utils";

const MAX_TEXTAREA_HEIGHT = 104;

interface ChatComposerProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  disabled?: boolean;
  isSubmitting?: boolean;
  placeholder?: string;
  submitLabel?: string;
  testId?: string;
  className?: string;
  footer?: ReactNode;
}

export function ChatComposer({
  value,
  onChange,
  onSubmit,
  disabled = false,
  isSubmitting = false,
  placeholder = "Write a message...",
  submitLabel = "Send",
  testId,
  className,
  footer,
}: ChatComposerProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  useAutoResizeTextarea(textareaRef, value, { maxHeight: MAX_TEXTAREA_HEIGHT });

  const canSubmit = Boolean(value.trim() && !disabled && !isSubmitting);

  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
        event.preventDefault();
        if (canSubmit) onSubmit();
      }
    },
    [canSubmit, onSubmit],
  );

  return (
    <div className={cn("space-y-2", className)}>
      <Textarea
        ref={textareaRef}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        rows={2}
        disabled={disabled || isSubmitting}
        data-testid={testId}
      />
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0 flex-1">{footer}</div>
        <Button size="sm" onClick={onSubmit} disabled={!canSubmit} data-testid={testId ? `${testId}-submit` : undefined}>
          {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <SendHorizontal className="mr-2 h-4 w-4" />}
          {submitLabel}
        </Button>
      </div>
    </div>
  );
}
