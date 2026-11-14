import { MarkdownRenderer } from "../markdown/MarkdownRenderer";

/** Render hub supplied prose with the same safe, device-friendly links as chat. */
export default function ApprovalLinkedText({
  content,
  className,
}: {
  content: string;
  className?: string;
}) {
  if (!content) return null;
  return (
    <div className={className} data-testid="approval-linked-text">
      <MarkdownRenderer content={content} />
    </div>
  );
}
