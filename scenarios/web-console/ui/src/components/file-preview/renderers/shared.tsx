import { useState } from "react";
import { AlertTriangle, Check, Copy, Download, ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import { formatBytes } from "../format";
import type { PreviewModel } from "../types";

// PreviewActions renders the shared toolbar affordances (download, open in new
// tab, copy path) every renderer can offer. Download/open only show when the
// model exposes a blob href.
export function PreviewActions({ model, className }: { model: PreviewModel; className?: string }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  const copyPath = () => {
    if (!model.resolvedPath) return;
    void navigator.clipboard.writeText(model.resolvedPath);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      {model.blobHref && model.canDownload && (
        <a
          href={model.blobHref}
          download={model.basename}
          data-testid="file-preview-download"
          className="inline-flex items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary"
        >
          <Download className="h-3.5 w-3.5" />
          {t(strings.messagesFileViewer.download)}
        </a>
      )}
      {model.blobHref && (
        <a
          href={model.blobHref}
          target="_blank"
          rel="noreferrer"
          data-testid="file-preview-open-new-tab"
          className="inline-flex items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary"
        >
          <ExternalLink className="h-3.5 w-3.5" />
          {t(strings.messagesFileViewer.openInNewTab)}
        </a>
      )}
      <button
        type="button"
        onClick={copyPath}
        data-testid="file-preview-copy-path"
        className="inline-flex items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary"
      >
        {copied ? <Check className="h-3.5 w-3.5 text-green-400" /> : <Copy className="h-3.5 w-3.5" />}
        {copied ? t(strings.messagesFileViewer.copied) : t(strings.messagesFileViewer.copyPath)}
      </button>
    </div>
  );
}

// PreviewMetaLine renders the kind/mime/size summary shared across renderers.
export function PreviewMetaLine({ model }: { model: PreviewModel }) {
  const size = formatBytes(model.sizeBytes);
  return (
    <p className="text-xs text-wc-text-muted">
      <span className="uppercase tracking-wide">{model.kind}</span>
      {model.mimeType && <span> · {model.mimeType}</span>}
      {size && <span> · {size}</span>}
    </p>
  );
}

// CenteredPreview is the standard full-height centered shell for media/image
// renderers, with a checkerboard backdrop so transparency is visible.
export function CenteredPreview({
  children,
  checkerboard = false,
  testId,
}: {
  children: React.ReactNode;
  checkerboard?: boolean;
  testId?: string;
}) {
  return (
    <div
      data-testid={testId}
      className={cn(
        "flex h-full w-full flex-col items-center justify-center gap-4 overflow-auto p-6",
        checkerboard &&
          "bg-[linear-gradient(45deg,var(--wc-surface-base)_25%,transparent_25%,transparent_75%,var(--wc-surface-base)_75%),linear-gradient(45deg,var(--wc-surface-base)_25%,transparent_25%,transparent_75%,var(--wc-surface-base)_75%)] bg-[length:24px_24px] bg-[position:0_0,12px_12px]",
      )}
    >
      {children}
    </div>
  );
}

// PreviewNotice renders an inline warning/hint block (e.g. media-load failures,
// truncation notices).
export function PreviewNotice({ message, tone = "warn" }: { message: string; tone?: "warn" | "info" }) {
  return (
    <div
      data-testid="file-preview-notice"
      className={cn(
        "flex items-start gap-2 rounded-xl border px-3 py-2 text-sm",
        tone === "warn"
          ? "border-amber-500/30 bg-amber-500/10 text-amber-300"
          : "border-wc-default bg-wc-surface-input text-wc-text-muted",
      )}
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
      <span>{message}</span>
    </div>
  );
}
