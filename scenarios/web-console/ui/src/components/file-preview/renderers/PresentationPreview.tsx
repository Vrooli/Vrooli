import { useEffect, useRef, useState } from "react";
import { ChevronLeft, ChevronRight, Copy, Download, Grid2X2, Maximize, Minus, Plus, Presentation, StickyNote } from "lucide-react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { useTranslation } from "react-i18next";
import { presentationRatio } from "../publicationData";
import { usePresentation } from "../usePresentation";
import { usePreviewClipboard } from "../usePreviewClipboard";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, InspectionChoices, InspectionHeader, InspectionJump, InspectionOptions, InspectionSetting, PreviewButton, PreviewLoading, previewTextButtonClass } from "./ArtifactChrome";
import { PreviewNotice } from "./shared";

export function PresentationPreview(props: PreviewRendererProps) { return <PresentationReader key={props.model.previewId} {...props} />; }
function PresentationReader({ model, onReopen }: PreviewRendererProps) {
  const { t, i18n } = useTranslation(), c = strings.presentationPreview;
  const [attempt, setAttempt] = useState(0), [index, setIndex] = useState(0), [thumbnails, setThumbnails] = useState(false), [panel, setPanel] = useState<"slide" | "notes" | "text">("slide"), [zoom, setZoom] = useState(1), [jump, setJump] = useState("");
  const { ref, rect } = useResizeObserver<HTMLDivElement>(), wide = (rect?.width ?? 0) >= 720;
  const reader = usePresentation(model.blobHref, model.sizeBytes, /\.ppt$/i.test(model.basename), attempt, value => { setIndex(value); if (!wide) setThumbnails(false); });
  const count = reader.metadata?.slides.length ?? 0, slide = reader.metadata?.slides[index], clipboard = usePreviewClipboard(), controls = useRef<HTMLDivElement>(null);
  const { metadata, frame, post } = reader;
  const navigate = (value: number) => { setIndex(Math.max(0, Math.min(count - 1, value))); controls.current?.focus(); };
  useEffect(() => {
    if (!metadata || !frame.current) return;
    const element = frame.current.parentElement; if (!element) return;
    const styles = getComputedStyle(element);
    const color = (name: string, fallback: string) => { const value = styles.getPropertyValue(name).trim(); return value ? `rgb(${value})` : fallback; };
    post({ type: "view", view: { index, thumbnails, zoom, labels: { slide: t(c.slide), hidden: t(c.hidden) }, rtl: i18n.dir() === "rtl", language: i18n.resolvedLanguage || "en", theme: { background: color("--wc-surface-base", "#111821"), surface: color("--wc-surface-raised", "#151d28"), text: color("--wc-text-primary", "#e5eaf0"), muted: color("--wc-text-muted", "#99a8b9"), border: color("--wc-border-default", "#293442"), accent: color("--wc-accent", "#87b7d7") } } });
  }, [metadata, frame, post, index, thumbnails, zoom, wide, t, i18n, c.slide, c.hidden]);
  const showSlide = () => { setPanel("slide"); setThumbnails(false); };
  return <ArtifactFrame testId={selectors.presentationPreview.root}>
    <div ref={ref} className="flex min-h-0 min-w-0 flex-1 flex-col">
      {!reader.error && <InspectionHeader icon={<Presentation />} title={t(c.title)} description={count ? t(c.summary, { count, ratio: reader.metadata ? presentationRatio(reader.metadata.width, reader.metadata.height) : "" }) : t(c.description)}>
        <PreviewButton label={t(c.slides)} pressed={thumbnails} disabled={!count} onClick={() => { setPanel("slide"); setThumbnails(value => !value); }}><Grid2X2 /></PreviewButton>
        <PreviewButton label={t(c.notes)} pressed={panel === "notes"} disabled={!count} onClick={() => { setPanel(value => value === "notes" ? "slide" : "notes"); setThumbnails(false); }}><StickyNote /></PreviewButton>
        <InspectionOptions label={t(c.options)}>{close => <>
          <InspectionSetting label={t(c.view)}><InspectionChoices label={t(c.view)} value={panel} options={[{ value: "slide", label: t(c.slide) }, { value: "notes", label: t(c.notes) }, { value: "text", label: t(c.text) }]} onChange={value => { setPanel(value as typeof panel); setThumbnails(false); close(); }} /></InspectionSetting>
          <InspectionJump label={t(c.goToSlide)} value={jump} invalid={!!jump && (!/^\d+$/.test(jump) || Number(jump) < 1 || Number(jump) > count)} placeholder={`1–${String(count)}`} numeric onChange={setJump} onGo={() => { if (/^\d+$/.test(jump) && Number(jump) >= 1 && Number(jump) <= count) { navigate(Number(jump) - 1); showSlide(); setJump(""); close(); } }} />
          <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.fidelity)}</p>
          <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.keyboard)}</p>
          {model.canDownload && <a className={previewTextButtonClass} href={model.blobHref} download={model.basename}><Download className="h-4 w-4" aria-hidden="true" />{t(strings.messagesFileViewer.download)}</a>}
        </>}</InspectionOptions>
      </InspectionHeader>}
      {reader.warning && <PreviewNotice message={t(c.partial)} tone="info" />}
      {reader.error ? <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 overflow-auto px-5 py-8 text-center"><Presentation aria-hidden="true" className="h-9 w-9 text-wc-text-muted" /><p role="status" className="max-w-sm text-sm leading-relaxed text-wc-text-secondary">{t(c[reader.error])}</p>{reader.error !== "legacy" && <button className={previewTextButtonClass} onClick={() => { if (reader.error === "fetch" && onReopen) onReopen(); else setAttempt(value => value + 1); }}>{t(strings.messagesFileViewer.reopen)}</button>}</div> : <>
        {!reader.metadata && <PreviewLoading label={t(c.loading)} />}
        {reader.source && <iframe ref={reader.frame} data-testid={selectors.presentationPreview.frame} title={t(c.canvas)} sandbox="allow-scripts" referrerPolicy="no-referrer" srcDoc={reader.source} onLoad={() => { reader.post({ type: "hello" }); }} className={`min-h-0 w-full flex-1 border-0 ${!reader.metadata || panel !== "slide" ? "hidden" : ""}`} />}
        {reader.metadata && panel !== "slide" && <div data-testid={selectors.presentationPreview.notes} className="min-h-0 flex-1 overflow-auto px-5 py-6 sm:px-10 sm:py-8" tabIndex={0} role="region" aria-label={t(panel === "notes" ? c.notes : c.text)}>
          <div className="mx-auto max-w-[72ch]"><p className="text-xs font-medium text-wc-text-muted">{t(c.counter, { current: index + 1, total: count })}</p><h2 className="mb-6 mt-2 text-xl font-semibold tracking-tight">{slide?.title}</h2><p dir="auto" className="whitespace-pre-wrap break-words text-base leading-7 text-wc-text-secondary">{(panel === "notes" ? slide?.notes : slide?.text) || t(panel === "notes" ? c.noNotes : c.noText)}</p>
          <button className={`${previewTextButtonClass} mt-6`} onClick={() => { void clipboard.copy((panel === "notes" ? slide?.notes : slide?.text) || ""); }} disabled={!(panel === "notes" ? slide?.notes : slide?.text)}><Copy className="h-4 w-4" aria-hidden="true" />{t(c.copy)}</button><p role="status" className="mt-2 text-xs text-wc-text-muted">{clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : strings.artifactPreview.copyFailed) : ""}</p></div>
        </div>}
      </>}
      {count > 0 && !reader.error && <div ref={controls} tabIndex={-1} data-testid={selectors.presentationPreview.navigation} role="group" aria-label={t(c.navigation)} onKeyDown={event => { if (event.target instanceof Element && event.target.closest("input,textarea")) return; const next = event.key === "ArrowRight" || event.key === "PageDown", previous = event.key === "ArrowLeft" || event.key === "PageUp"; if (next || previous || ["Home", "End"].includes(event.key)) { event.preventDefault(); navigate(event.key === "Home" ? 0 : event.key === "End" ? count - 1 : index + (next ? 1 : -1)); } }} className="flex shrink-0 items-center justify-between gap-1 border-t border-wc-default px-2 py-1 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent">
        <div className="flex min-w-0 items-center"><PreviewButton label={t(c.previous)} disabled={!index} onClick={() => { navigate(index - 1); }}><ChevronLeft className="rtl:rotate-180" /></PreviewButton><span role="status" className="min-w-12 text-center text-xs tabular-nums text-wc-text-muted">{index + 1} / {count}{slide?.hidden && <span className="sr-only"> · {t(c.hidden)}</span>}</span><PreviewButton label={t(c.next)} disabled={index >= count - 1} onClick={() => { navigate(index + 1); }}><ChevronRight className="rtl:rotate-180" /></PreviewButton></div>
        <div className="flex items-center"><PreviewButton label={t(c.zoomOut)} disabled={zoom <= 1 || panel !== "slide" || thumbnails && !wide} onClick={() => { setZoom(value => Math.max(1, value - 0.25)); }}><Minus /></PreviewButton><PreviewButton label={t(c.fit)} disabled={panel !== "slide" || thumbnails && !wide} onClick={() => { setZoom(1); }}>{zoom === 1 ? <Maximize /> : <span className="text-[11px] tabular-nums">{Math.round(zoom * 100)}%</span>}</PreviewButton><PreviewButton label={t(c.zoomIn)} disabled={zoom >= 3 || panel !== "slide" || thumbnails && !wide} onClick={() => { setZoom(value => Math.min(3, value + 0.25)); }}><Plus /></PreviewButton></div>
      </div>}
      {reader.error && <ArtifactFooter model={model} metadata={model.basename} />}
    </div>
  </ArtifactFrame>;
}
