import { useEffect, useRef, useState } from "react";
import { Box, Maximize, Minus, Plus, RotateCcw, Rotate3D, Grid3X3 } from "lucide-react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { useTranslation } from "react-i18next";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { DEFAULT_MODEL_OPTIONS, modelFormat, type ModelOptions, type ModelView } from "../modelData";
import { useModelPreview } from "../useModelPreview";
import { formatBytes } from "../format";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, InspectionChoices, InspectionHeader, InspectionOptions, InspectionSetting, PreviewButton, PreviewLoading, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";
import { PreviewNotice } from "./shared";

export function ModelPreview(props: PreviewRendererProps) { return <ModelReader key={props.model.previewId} {...props} />; }
function ModelReader({ model, onReopen }: PreviewRendererProps) {
  const { t, i18n } = useTranslation(), c = strings.modelPreview;
  const container = useRef<HTMLDivElement>(null), { ref, rect } = useResizeObserver<HTMLDivElement>(), wide = (rect?.width ?? 0) >= 640;
  const [attempt, setAttempt] = useState(0), [options, setOptions] = useState<ModelOptions>(DEFAULT_MODEL_OPTIONS), [view, setView] = useState<ModelView>("perspective");
  const format = modelFormat(model.basename), reader = useModelPreview(container, model.blobHref, format, model.sizeBytes, attempt), ready = !!reader.stats && !reader.error;
  const dimension = new Intl.NumberFormat(i18n.resolvedLanguage || "en", { maximumSignificantDigits: 4 });
  const count = new Intl.NumberFormat(i18n.resolvedLanguage || "en", { maximumFractionDigits: 0 });
  useEffect(() => { reader.stage.current?.options(options); }, [options, reader.stats, reader.stage]);
  useEffect(() => { reader.stage.current?.view(view); }, [view, reader.stats, reader.stage]);
  const chooseView = (value: ModelView) => { setView(value); reader.stage.current?.view(value); };
  const update = <K extends keyof ModelOptions>(key: K, value: ModelOptions[K]) => { setOptions(current => ({ ...current, [key]: value })); };
  const views: { value: ModelView; label: string }[] = ["perspective", "front", "right", "top"].map(value => ({ value: value as ModelView, label: t(c[value as ModelView]) }));
  const reset = () => { chooseView("perspective"); update("rotate", false); };
  const settings = <>
    <InspectionSetting label={t(c.view)}><InspectionChoices label={t(c.view)} value={view} options={views} onChange={value => { chooseView(value as ModelView); }} /></InspectionSetting>
    <InspectionSetting label={t(c.background)}><InspectionChoices label={t(c.background)} value={options.background} options={["studio", "light", "dark"].map(value => ({ value, label: t(c[value as "studio" | "light" | "dark"]) }))} onChange={value => { update("background", value as ModelOptions["background"]); }} /></InspectionSetting>
    <InspectionSetting label={t(c.lighting)}><InspectionChoices label={t(c.lighting)} value={options.lighting} options={[{ value: "studio", label: t(c.studio) }, { value: "soft", label: t(c.soft) }]} onChange={value => { update("lighting", value as ModelOptions["lighting"]); }} /></InspectionSetting>
    <div className="space-y-1">{(["wireframe", "grid", "rotate"] as const).map(key => <label key={key} className="flex min-h-11 cursor-pointer items-center justify-between gap-3 text-sm text-wc-text-secondary"><span>{t(c[key])}</span><input type="checkbox" checked={options[key]} onChange={event => { update(key, event.target.checked); }} className="h-5 w-5 accent-wc-accent" /></label>)}</div>
    {reader.stats && <InspectionSetting label={t(c.geometry)}><dl className="space-y-2 text-xs text-wc-text-muted"><div className="flex justify-between"><dt>{t(c.meshes)}</dt><dd className="tabular-nums">{count.format(reader.stats.meshes)}</dd></div><div className="flex justify-between"><dt>{t(c.triangles)}</dt><dd className="tabular-nums">{count.format(reader.stats.triangles)}</dd></div><div className="flex flex-wrap justify-between gap-2"><dt>{t(c.dimensions)}</dt><dd dir="ltr" className="tabular-nums">{reader.stats.dimensions.map(value => dimension.format(value)).join(" × ")}</dd></div></dl><p className="text-xs leading-relaxed text-wc-text-muted">{t(c.units)}</p></InspectionSetting>}
    <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.support)}</p>
    <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.keyboard)}</p>
  </>;
  return <ArtifactFrame testId={selectors.modelPreview.root}>
    <div ref={ref} className="flex min-h-0 min-w-0 flex-1 flex-col">
      <InspectionHeader icon={<Box />} title={t(c.title)} description={ready && reader.stats ? `${format?.toUpperCase() || "3D"} · ${t(c.summary, { count: reader.stats.meshes, triangles: count.format(reader.stats.triangles) })}` : t(c.description)}>
        <PreviewButton label={t(c.wireframe)} pressed={options.wireframe} disabled={!ready} onClick={() => { update("wireframe", !options.wireframe); }}><Grid3X3 /></PreviewButton>
        <InspectionOptions label={t(c.options)}>{() => settings}</InspectionOptions>
      </InspectionHeader>
      {reader.partial && !reader.error && <PreviewNotice message={t(c.partial)} tone="info" />}
      <div className="relative min-h-0 flex-1 overflow-hidden bg-wc-surface-base">
        <div ref={container} tabIndex={ready ? 0 : -1} role="img" aria-label={t(c.canvas, { name: model.basename })} aria-describedby={ready ? "model-preview-instructions" : undefined} data-testid={selectors.modelPreview.stage}
          onKeyDown={event => {
            if (!ready || event.ctrlKey || event.metaKey || event.altKey) return;
            const stage = reader.stage.current;
            if (["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "+", "=", "-", "0", "Home"].includes(event.key)) {
              event.preventDefault(); event.stopPropagation();
              if (event.key === "+" || event.key === "=") stage?.zoom(0.8); else if (event.key === "-") stage?.zoom(1.25); else if (event.key === "0" || event.key === "Home") reset();
              else stage?.orbit(event.key === "ArrowLeft" ? -0.15 : event.key === "ArrowRight" ? 0.15 : 0, event.key === "ArrowUp" ? -0.15 : event.key === "ArrowDown" ? 0.15 : 0);
            }
          }} className="absolute inset-0 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent" />
        {!ready && !reader.error && <div className="pointer-events-none absolute inset-0"><PreviewLoading label={t(c.loading)} /></div>}
        {reader.error && <div className="absolute inset-0 flex flex-col items-center justify-center gap-4 overflow-auto px-6 py-8 text-center"><Box aria-hidden="true" className="h-9 w-9 shrink-0 text-wc-text-muted" /><p role="status" className="max-w-sm text-sm leading-relaxed text-wc-text-secondary">{t(c[reader.error])}</p><button className={previewTextButtonClass} onClick={() => { if (reader.error === "fetch" && onReopen) onReopen(); else { setAttempt(value => value + 1); setView("perspective"); } }}>{t(strings.messagesFileViewer.reopen)}</button></div>}
        {ready && <p id="model-preview-instructions" className="pointer-events-none absolute inset-x-4 bottom-3 text-center text-[11px]" style={{ color: options.background === "light" ? "#506273" : "#a2b2c3" }}>{t(wide ? c.pointerHint : c.touchHint)}</p>}
      </div>
      {ready && <div className="flex shrink-0 items-center justify-between gap-1 border-t border-wc-default bg-wc-surface-raised px-2 py-1" data-testid={selectors.modelPreview.navigation}>
        {wide ? <div className="flex min-w-0 items-center">{views.map(item => <button key={item.value} type="button" aria-pressed={view === item.value} onClick={() => { chooseView(item.value); }} className={`${previewTextButtonClass} border-transparent px-2 aria-pressed:bg-wc-surface-input aria-pressed:text-wc-text-primary`}>{item.label}</button>)}</div> : <label className="min-w-0"><span className="sr-only">{t(c.view)}</span><select aria-label={t(c.view)} value={view} onChange={event => { chooseView(event.target.value as ModelView); }} className={`${previewSelectClass} max-w-36 border-transparent bg-transparent`}>{views.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>}
        <div className="flex shrink-0 items-center"><PreviewButton label={t(c.zoomOut)} onClick={() => reader.stage.current?.zoom(1.25)}><Minus /></PreviewButton><PreviewButton label={t(c.fit)} onClick={() => { chooseView(view); }}><Maximize /></PreviewButton><PreviewButton label={t(c.zoomIn)} onClick={() => reader.stage.current?.zoom(0.8)}><Plus /></PreviewButton>{wide && <PreviewButton label={t(c.reset)} onClick={reset}><RotateCcw /></PreviewButton>}<PreviewButton label={t(c.rotate)} pressed={options.rotate} onClick={() => { update("rotate", !options.rotate); }}><Rotate3D /></PreviewButton></div>
      </div>}
      <ArtifactFooter model={model} metadata={`${format?.toUpperCase() || "3D"} · ${formatBytes(model.sizeBytes)}`} />
    </div>
  </ArtifactFrame>;
}
