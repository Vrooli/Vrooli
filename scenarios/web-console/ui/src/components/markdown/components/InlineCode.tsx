import type { ReactNode } from "react";
import { useMemo } from "react";
import { FileText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton/1";
import { strings } from "../../../consts/strings";
import { copyText } from "../../../lib/clipboard";
import { looksLikeInlineFileReference } from "../../../lib/fileReferences";

interface InlineCodeProps {
  children: ReactNode;
  /**
   * When provided and the inline code text looks like a file path, the chip
   * becomes a button that opens the file in the preview dialog. Lets agents
   * reference files with backticks (e.g. `~/.vrooli/plans/foo.md`) the same
   * way as markdown links.
   */
  onFileReferenceClick?: (path: string) => void;
}

/**
 * The chip's copy control: the library's copy button, sized to the line of
 * text it sits in rather than to a toolbar, and revealed on hover or focus.
 */
const CHIP_COPY_STYLE = { inlineSize: "1.25rem", blockSize: "1.25rem" };
const CHIP_COPY_CLASS = "shrink-0 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100";

/** Styled inline code with hover-reveal copy button. */
export function InlineCode({ children, onFileReferenceClick }: InlineCodeProps) {
  const { t } = useTranslation();
  const textContent = useMemo(() => extractTextContent(children), [children]);

  const isFileRef =
    onFileReferenceClick !== undefined &&
    textContent.length > 0 &&
    looksLikeInlineFileReference(textContent);

  const copyButton = (label: string) => (
    <CopyIconButton
      size="xs"
      denseTapTarget
      style={CHIP_COPY_STYLE}
      className={CHIP_COPY_CLASS}
      value={textContent}
      writeText={copyText}
      aria-label={label}
      copiedLabel={t(strings.messageActions.copied)}
      failedLabel={t(strings.messageActions.copyFailed)}
    />
  );

  if (isFileRef) {
    return (
      <span className="group inline-flex max-w-full items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/80 px-2 py-0.5 text-xs font-mono align-middle">
        <button
          type="button"
          onClick={() => onFileReferenceClick?.(textContent)}
          className="inline-flex min-w-0 items-center gap-1 text-wc-accent hover:text-wc-accent/80 underline underline-offset-2"
          title={`Open ${textContent}`}
        >
          <FileText className="h-3 w-3 shrink-0" aria-hidden="true" />
          <code className="min-w-0 break-all [overflow-wrap:anywhere] leading-relaxed">{children}</code>
        </button>
        {copyButton(t(strings.messageActions.copyPath))}
      </span>
    );
  }

  return (
    <span className="group inline-flex max-w-full items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/80 px-2 py-0.5 text-xs font-mono text-wc-text-primary align-middle">
      <code className="leading-relaxed min-w-0 break-all [overflow-wrap:anywhere]">{children}</code>
      {textContent ? copyButton(t(strings.messageActions.copyInlineCode)) : null}
    </span>
  );
}

function extractTextContent(children: ReactNode): string {
  if (typeof children === "string") return children;
  if (Array.isArray(children)) return children.map(extractTextContent).join("");
  if (children && typeof children === "object" && "props" in children) {
    return extractTextContent((children as { props: { children?: ReactNode } }).props.children);
  }
  return "";
}
