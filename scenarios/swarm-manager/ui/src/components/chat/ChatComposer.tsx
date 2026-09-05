import type { ReactNode } from "react";
import { MessageComposer } from "../composer/MessageComposer";

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
  return (
    <MessageComposer
      value={value}
      onChange={onChange}
      onSubmit={onSubmit}
      disabled={disabled}
      isSubmitting={isSubmitting}
      placeholder={placeholder}
      submitLabel={submitLabel}
      testId={testId}
      className={className}
      footer={footer}
      canSubmit={Boolean(value.trim())}
    />
  );
}
