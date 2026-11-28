import { useId, useRef, useState, type ReactNode, type Ref } from "react";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { ArrowRight, ChevronDown, Download, ExternalLink, Loader2, Settings2 } from "lucide-react";
import { useFileViewerHost, useFileViewerText } from "../../i18n/context.js";
import { strings } from "../../i18n/strings.js";
import { cn } from "../../lib/classnames.js";
import type { PreviewModel } from "../types.js";

export const previewSelectClass = "h-11 min-w-0 rounded-lg border border-fv-default bg-fv-surface-input px-2 text-xs text-fv-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-fv-accent disabled:opacity-40";
export const previewTextButtonClass = "inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-fv-default px-3 text-xs font-medium text-fv-text-secondary hover:bg-fv-surface-input focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-fv-accent disabled:opacity-40";

export function ArtifactFrame({ children, testId, frameRef }: { children: ReactNode; testId: string; frameRef?: Ref<HTMLDivElement> }) {
  return <div ref={frameRef} data-testid={testId} className="fv-artifact-preview flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-fv-surface-base text-fv-text-primary">{children}</div>;
}

export function PreviewToolbar({ children, label }: { children: ReactNode; label: string }) {
  return <div role="group" aria-label={label} className="flex shrink-0 items-center justify-between gap-1 border-b border-fv-default bg-fv-surface-raised px-2 py-1">{children}</div>;
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
  return <div role="status" className="flex h-full min-h-32 items-center justify-center gap-2 p-5 text-sm text-fv-text-muted"><Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden="true" />{label}</div>;
}

export function ArtifactFooter({ model, metadata, children }: { model: PreviewModel; metadata: string; children?: ReactNode }) {
  const { t } = useFileViewerText();
  // A host that shows size, format and download itself owns this bar entirely.
  if (useFileViewerHost().fileInfo === "host") return null;
  return <div className="shrink-0 border-t border-fv-default bg-fv-surface-raised px-3 py-1">
    <div className="flex min-w-0 flex-wrap items-center justify-between gap-x-2">
      <p className="min-w-0 flex-1 truncate text-[11px] text-fv-text-muted" title={metadata}>{metadata}</p>
      <div className="flex shrink-0 items-start gap-1">
        {model.blobHref && model.canDownload && <a href={model.blobHref} download={model.basename} className={cn(previewTextButtonClass, "border-transparent px-2")} data-testid="file-preview-download"><Download className="h-4 w-4" aria-hidden="true" /><span className="sr-only">{t(strings.viewer.download)}</span></a>}
        <details className="group">
          <summary className={cn(previewTextButtonClass, "cursor-pointer list-none border-transparent px-2 [&::-webkit-details-marker]:hidden")}>
            {t(strings.artifactPreview.actions)}<ChevronDown className="h-3.5 w-3.5 transition group-open:rotate-180" aria-hidden="true" />
          </summary>
          <div className="flex max-w-48 flex-col items-start gap-1 pb-2">
            {model.blobHref && <a href={model.blobHref} target="_blank" rel="noreferrer" className={previewTextButtonClass} data-testid="file-preview-open-new-tab"><ExternalLink className="h-4 w-4 shrink-0" aria-hidden="true" />{t(strings.viewer.openInNewTab)}</a>}
            {children}
          </div>
        </details>
      </div>
    </div>
  </div>;
}

// Composed identity and tools for document inspectors. Media playback keeps its
// transport chrome; readers need a quieter header above their working surface.
export function InspectionHeader({ icon, title, description, children }: { icon: ReactNode; title: string; description: ReactNode; children: ReactNode }) {
  return <div className="flex shrink-0 items-center gap-3 border-b border-fv-default bg-fv-surface-raised px-3 py-3 sm:px-4">
    <span aria-hidden="true" className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-fv-accent/15 bg-fv-accent/10 text-fv-accent [&>svg]:h-5 [&>svg]:w-5">{icon}</span>
    <div className="min-w-0 flex-1"><p className="truncate text-sm font-semibold tracking-tight text-fv-text-primary">{title}</p><div className="mt-0.5 truncate text-[11px] text-fv-text-muted">{description}</div></div>
    <div className="flex shrink-0 items-center gap-1">{children}</div>
  </div>;
}

// Settings are portaled through the existing overlay owner. They cannot stretch
// the toolbar, be clipped by its canvas, or leave controls behind the keyboard.
export function InspectionOptions({ label, children }: { label: string; children: (close: () => void) => ReactNode }) {
  const { t } = useFileViewerText();
  const [open, setOpen] = useState(false), trigger = useRef<HTMLButtonElement>(null);
  return <>
    <IconButton ref={trigger} type="button" surface="ghost" size="md" aria-label={label} title={label} aria-haspopup="dialog" aria-expanded={open}
      onClick={() => { setOpen(true); }} style={{ inlineSize: "2.75rem", blockSize: "2.75rem" }}><Settings2 aria-hidden="true" className="h-4 w-4" /></IconButton>
    <ResponsiveDialog open={open} onClose={() => { setOpen(false); }} returnFocusRef={trigger} avoidKeyboard dismissAffordance="close" size="sm"
      title={<span className="text-base font-semibold tracking-tight">{label}</span>} ariaLabel={label} closeLabel={t(strings.inspectionPreview.closeOptions)} testId="inspection-options">
      <div className="fv-inspection-settings space-y-6 p-5">{children(() => { setOpen(false); })}</div>
    </ResponsiveDialog>
  </>;
}

export function InspectionSetting({ label, children }: { label: string; children: ReactNode }) {
  return <div className="space-y-2.5"><p className="text-xs font-medium text-fv-text-muted">{label}</p>{children}</div>;
}

export function InspectionChoices({ label, value, options, onChange }: { label: string; value: string; options: { value: string; label: string }[]; onChange: (value: string) => void }) {
  const name = useId();
  return <fieldset className="fv-inspection-choices"><legend className="sr-only">{label}</legend>
    {options.map((option) => <label key={option.value} className="min-w-0 flex-1">
      <input type="radio" name={name} value={option.value} checked={value === option.value} onChange={() => { onChange(option.value); }} className="peer sr-only" />
      <span className="flex min-h-11 items-center justify-center rounded-lg px-2 py-2 text-center text-xs font-medium text-fv-text-muted transition peer-checked:bg-fv-surface-raised peer-checked:text-fv-text-primary peer-checked:shadow-sm peer-focus-visible:ring-2 peer-focus-visible:ring-fv-accent">{option.label}</span>
    </label>)}
  </fieldset>;
}

export function InspectionJump({ label, value, invalid, placeholder, numeric, onChange, onGo }: { label: string; value: string; invalid: boolean; placeholder: string; numeric?: boolean; onChange: (value: string) => void; onGo: () => void }) {
  const { t } = useFileViewerText(), id = useId();
  return <form onSubmit={(event) => { event.preventDefault(); onGo(); }} className="space-y-2.5">
    <label htmlFor={id} className="block text-xs font-medium text-fv-text-muted">{label}</label>
    <div className="fv-inspection-jump flex min-w-0 items-center rounded-xl border border-fv-default bg-fv-surface-base p-1 focus-within:border-fv-accent focus-within:ring-1 focus-within:ring-fv-accent">
      <input id={id} type="text" inputMode={numeric ? "numeric" : "text"} autoCapitalize="characters" autoComplete="off" spellCheck={false} aria-invalid={invalid} value={value} onChange={(event) => { onChange(event.target.value); }} placeholder={placeholder} dir="ltr" className="h-11 min-w-0 flex-1 border-0 bg-transparent px-3 font-mono text-base text-fv-text-primary outline-none" />
      <button type="submit" className="inline-flex min-h-11 shrink-0 items-center justify-center gap-2 rounded-lg bg-fv-accent px-4 text-sm font-semibold text-fv-accent-fg outline-none hover:opacity-90 focus-visible:ring-2 focus-visible:ring-fv-accent focus-visible:ring-offset-2">{t(strings.inspectionPreview.go)}<ArrowRight aria-hidden="true" className="h-4 w-4 rtl:rotate-180" /></button>
    </div>
  </form>;
}
