import type { ReactNode, Ref } from "react";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { ChevronDown, Download, ExternalLink, Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import type { PreviewModel } from "../types";

export const previewSelectClass = "h-11 min-w-0 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-wc-accent disabled:opacity-40";
export const previewTextButtonClass = "inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-wc-default px-3 text-xs font-medium text-wc-text-secondary hover:bg-wc-surface-input focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent disabled:opacity-40";

export function ArtifactFrame({ children, testId, frameRef }: { children: ReactNode; testId: string; frameRef?: Ref<HTMLDivElement> }) {
  return <div ref={frameRef} data-testid={testId} className="wc-artifact-preview flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-wc-surface-base text-wc-text-primary">{children}</div>;
}

export function PreviewToolbar({ children, label }: { children: ReactNode; label: string }) {
  return <div role="group" aria-label={label} className="flex shrink-0 items-center justify-between gap-1 border-b border-wc-default bg-wc-surface-raised px-2 py-1">{children}</div>;
}

export function PreviewButton({ label, children, onClick, disabled, pressed, primary, testId }: {
  label: string; children: ReactNode; onClick: () => void; disabled?: boolean; pressed?: boolean; primary?: boolean; testId?: string;
}) {
  return <IconButton type="button" surface={primary ? "solid" : "ghost"} shape={primary ? "circle" : "rounded"}
    size={primary ? "lg" : "md"} aria-label={label} title={label} aria-pressed={pressed} disabled={disabled}
    onClick={onClick} data-testid={testId} style={{ inlineSize: primary ? "3.5rem" : "2.75rem", blockSize: primary ? "3.5rem" : "2.75rem" }} className="shrink-0">
    {children}
  </IconButton>;
}

export function PreviewLoading({ label }: { label: string }) {
  return <div role="status" className="flex h-full min-h-32 items-center justify-center gap-2 p-5 text-sm text-wc-text-muted"><Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden="true" />{label}</div>;
}

export function ArtifactFooter({ model, metadata, children }: { model: PreviewModel; metadata: string; children?: ReactNode }) {
  const { t } = useTranslation();
  return <div className="shrink-0 border-t border-wc-default bg-wc-surface-raised px-3 py-1">
    <div className="flex min-w-0 flex-wrap items-center justify-between gap-x-2">
      <p className="min-w-0 flex-1 truncate text-[11px] text-wc-text-muted" title={metadata}>{metadata}</p>
      <div className="flex shrink-0 items-start gap-1">
        {model.blobHref && model.canDownload && <a href={model.blobHref} download={model.basename} className={cn(previewTextButtonClass, "border-transparent px-2")} data-testid="file-preview-download"><Download className="h-4 w-4" aria-hidden="true" /><span className="sr-only">{t(strings.messagesFileViewer.download)}</span></a>}
        <details className="group">
          <summary className={cn(previewTextButtonClass, "cursor-pointer list-none border-transparent px-2 [&::-webkit-details-marker]:hidden")}>
            {t(strings.artifactPreview.actions)}<ChevronDown className="h-3.5 w-3.5 transition group-open:rotate-180" aria-hidden="true" />
          </summary>
          <div className="flex max-w-48 flex-col items-start gap-1 pb-2">
            {model.blobHref && <a href={model.blobHref} target="_blank" rel="noreferrer" className={previewTextButtonClass} data-testid="file-preview-open-new-tab"><ExternalLink className="h-4 w-4 shrink-0" aria-hidden="true" />{t(strings.messagesFileViewer.openInNewTab)}</a>}
            {children}
          </div>
        </details>
      </div>
    </div>
  </div>;
}
