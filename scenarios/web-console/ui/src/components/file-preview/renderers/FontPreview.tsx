import { useId, useMemo, useState, type CSSProperties } from "react";
import { ChevronLeft, ChevronRight, Minus, Plus, RotateCcw, Type } from "lucide-react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { useTranslation } from "react-i18next";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { FONT_PAGE_SIZE, filterFontCharacters, fontFormat, fontHero, fontSample, unicodeLabel, visibleFontCharacter } from "../fontData";
import { useFontPreview } from "../useFontPreview";
import { formatBytes } from "../format";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, InspectionChoices, InspectionHeader, InspectionOptions, InspectionSetting, PreviewButton, PreviewLoading, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";

type FontMode = "specimen" | "characters";
type Appearance = "automatic" | "light" | "dark";
const rangeClass = "h-11 min-w-0 flex-1 accent-wc-accent";
export function FontPreview(props: PreviewRendererProps) { return <FontSpecimen key={props.model.previewId} {...props} />; }

function FontSpecimen({ model, onReopen }: PreviewRendererProps) {
  const { t, i18n } = useTranslation(), c = strings.fontPreview;
  const sampleId = useId(), searchId = useId();
  const { ref, rect } = useResizeObserver<HTMLDivElement>(), wide = (rect?.width ?? 0) >= 640;
  const [attempt, setAttempt] = useState(0), [mode, setMode] = useState<FontMode>("specimen");
  const [sample, setSample] = useState<string | null>(null), [size, setSize] = useState(48);
  const [spacing, setSpacing] = useState(0), [lineHeight, setLineHeight] = useState(1.4);
  const [appearance, setAppearance] = useState<Appearance>("automatic"), [align, setAlign] = useState<"start" | "center">("start");
  const [kerning, setKerning] = useState(true), [ligatures, setLigatures] = useState(true), [axisValues, setAxisValues] = useState<Record<string, number>>({});
  const [query, setQuery] = useState(""), [page, setPage] = useState(0), [selected, setSelected] = useState<number | null>(null);
  const reader = useFontPreview(model.blobHref, model.sizeBytes, attempt), info = reader.loaded?.info;
  const defaultSample = useMemo(() => fontSample(info?.characters ?? [], [t(c.sampleDefault), ...["en", "ar", "ja"].map(lng => t(c.sampleDefault, { lng }))]), [info, t, c.sampleDefault]);
  const waterfallSample = useMemo(() => fontSample(info?.characters ?? [], [t(c.waterfallText), ...["en", "ar", "ja"].map(lng => t(c.waterfallText, { lng }))]), [info, t, c.waterfallText]);
  const sampleText = sample ?? defaultSample;
  const allCharacters = useMemo(() => info?.characters.filter(visibleFontCharacter) ?? [], [info]);
  const filtered = useMemo(() => filterFontCharacters(allCharacters, query), [allCharacters, query]);
  const pages = Math.max(1, Math.ceil(filtered.length / FONT_PAGE_SIZE)), currentPage = Math.min(page, pages - 1);
  const characters = filtered.slice(currentPage * FONT_PAGE_SIZE, (currentPage + 1) * FONT_PAGE_SIZE);
  const number = new Intl.NumberFormat(i18n.resolvedLanguage === "cimode" ? "en" : i18n.resolvedLanguage);
  const fontStyle: CSSProperties = {
    fontFamily: reader.loaded ? `"${reader.loaded.family}", sans-serif` : undefined,
    fontSynthesis: "none", fontWeight: 400, fontStyle: "normal",
    fontKerning: kerning ? "normal" : "none", fontVariantLigatures: ligatures ? "common-ligatures" : "no-common-ligatures",
    fontVariationSettings: info?.axes.length ? info.axes.map(axis => `"${axis.tag}" ${String(axisValues[axis.tag] ?? axis.default)}`).join(", ") : "normal",
  };
  const specimenColors: CSSProperties = appearance === "light" ? { background: "#faf9f6", color: "#20262d" } : appearance === "dark" ? { background: "#111923", color: "#e7edf3" } : {};
  const reset = () => { setSample(null); setSize(48); setSpacing(0); setLineHeight(1.4); setAlign("start"); setKerning(true); setLigatures(true); setAxisValues({}); setAppearance("automatic"); };
  const changeSize = (value: number) => { setSize(Math.min(144, Math.max(16, value))); };
  const sizeRange = <input type="range" aria-label={t(c.size)} min={16} max={144} step={1} value={size} onChange={event => { changeSize(Number(event.target.value)); }} className={rangeClass} />;
  const settings = <>
    <InspectionSetting label={t(c.size)}><div className="flex items-center gap-3">{sizeRange}<output className="w-14 text-end text-xs tabular-nums text-wc-text-secondary">{size} px</output></div></InspectionSetting>
    <InspectionSetting label={t(c.spacing)}><div className="flex items-center gap-3"><input type="range" aria-label={t(c.spacing)} min={-0.05} max={0.2} step={0.01} value={spacing} onChange={event => { setSpacing(Number(event.target.value)); }} className={rangeClass} /><output className="w-16 text-end text-xs tabular-nums text-wc-text-secondary">{spacing.toFixed(2)} em</output></div></InspectionSetting>
    <InspectionSetting label={t(c.lineHeight)}><div className="flex items-center gap-3"><input type="range" aria-label={t(c.lineHeight)} min={1} max={2} step={0.05} value={lineHeight} onChange={event => { setLineHeight(Number(event.target.value)); }} className={rangeClass} /><output className="w-14 text-end text-xs tabular-nums text-wc-text-secondary">{lineHeight.toFixed(2)}</output></div></InspectionSetting>
    <InspectionSetting label={t(c.appearance)}><InspectionChoices label={t(c.appearance)} value={appearance} options={(["automatic", "light", "dark"] as const).map(value => ({ value, label: t(c[value]) }))} onChange={value => { setAppearance(value as Appearance); }} /></InspectionSetting>
    <InspectionSetting label={t(c.alignment)}><InspectionChoices label={t(c.alignment)} value={align} options={(["start", "center"] as const).map(value => ({ value, label: t(c[value]) }))} onChange={value => { setAlign(value as "start" | "center"); }} /></InspectionSetting>
    <div className="space-y-1">{([{ key: "kerning", value: kerning, change: setKerning }, { key: "ligatures", value: ligatures, change: setLigatures }] as const).map(item => <label key={item.key} className="flex min-h-11 cursor-pointer items-center justify-between gap-3 text-sm text-wc-text-secondary"><span>{t(c[item.key])}</span><input type="checkbox" checked={item.value} onChange={event => { item.change(event.target.checked); }} className="h-5 w-5 accent-wc-accent" /></label>)}</div>
    {!!info?.axes.length && <InspectionSetting label={t(c.axes)}>{info.axes.map(axis => <label key={axis.tag} className="block space-y-1 text-xs text-wc-text-secondary"><span className="flex justify-between gap-2"><span dir="auto">{axis.name}</span><span className="tabular-nums">{number.format(axisValues[axis.tag] ?? axis.default)}</span></span><input type="range" aria-label={axis.name} min={axis.min} max={axis.max} step={(axis.max - axis.min) / 100 || 1} value={axisValues[axis.tag] ?? axis.default} onChange={event => { const value = Number(event.target.value); setAxisValues(previous => ({ ...previous, [axis.tag]: value })); }} className="h-11 w-full accent-wc-accent" /></label>)}</InspectionSetting>}
    {info && <InspectionSetting label={t(c.details)}><dl className="space-y-2 text-xs text-wc-text-muted">{([
      [t(c.family), info.family || model.basename], [t(c.style), info.style || "—"], [t(c.format), info.format.toUpperCase()], [t(c.glyphs), number.format(info.glyphs)], [t(c.mapped), number.format(info.characters.length)], [t(c.units), number.format(info.unitsPerEm)],
    ]).map(([label, value]) => <div key={label} className="flex justify-between gap-4"><dt className="shrink-0">{label}</dt><dd dir="auto" className="min-w-0 break-words text-end">{value}</dd></div>)}</dl></InspectionSetting>}
    <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.fallback)}</p>
    <button type="button" className={`${previewTextButtonClass} w-full`} onClick={reset}><RotateCcw className="h-4 w-4" aria-hidden="true" />{t(c.reset)}</button>
  </>;
  return <ArtifactFrame testId={selectors.fontPreview.root} frameRef={ref}>
    <InspectionHeader icon={<Type />} title={t(c.title)} description={info ? `${info.family || model.basename} · ${info.style || t(c.fixed)}${info.axes.length ? ` · ${t(c.variable)}` : ""}` : t(c.description)}>
      {wide && <PreviewButton label={t(c.reset)} onClick={reset}><RotateCcw /></PreviewButton>}
      <InspectionOptions label={t(c.options)}>{() => settings}</InspectionOptions>
    </InspectionHeader>
    {!info && !reader.error && <div className="min-h-0 flex-1"><PreviewLoading label={t(c.loading)} /></div>}
    {reader.error && <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 overflow-auto px-6 py-8 text-center"><Type aria-hidden="true" className="h-9 w-9 shrink-0 text-wc-text-muted" /><p role="status" className="max-w-sm text-sm leading-relaxed text-wc-text-secondary">{t(c[reader.error])}</p><button type="button" className={previewTextButtonClass} onClick={() => { if (reader.error === "fetch" && onReopen) onReopen(); else setAttempt(value => value + 1); }}>{t(strings.messagesFileViewer.reopen)}</button></div>}
    {info && !reader.error && <>
      {mode === "specimen" ? <div data-testid={selectors.fontPreview.specimen} className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 py-6 sm:px-8 sm:py-8" style={specimenColors}>
        <div style={{ ...fontStyle, fontSize: wide ? 160 : 128, lineHeight: 1.15, textAlign: align }} aria-hidden="true">{fontHero(info.characters, defaultSample)}</div>
        <label htmlFor={sampleId} className="mb-3 mt-6 block text-[11px] font-medium tracking-wide opacity-60">{t(c.sample)}</label>
        <textarea id={sampleId} data-testid={selectors.fontPreview.sample} dir="auto" value={sampleText} maxLength={1000} spellCheck={false} onChange={event => { setSample(event.target.value); }} rows={3}
          style={{ ...fontStyle, fontSize: size, letterSpacing: `${String(spacing)}em`, lineHeight, textAlign: align, minHeight: 160, maxHeight: 400 }}
          className="block w-full resize-y rounded-none border-0 bg-transparent p-0 outline-none [overflow-wrap:anywhere] placeholder:opacity-40 focus-visible:ring-2 focus-visible:ring-wc-accent" />
        <div className="mt-8 border-t border-wc-default pt-5"><p className="mb-4 text-[11px] font-medium tracking-wide opacity-60">{t(c.waterfall)}</p><div className="space-y-5">{[14, 24, 36].map(value => <div key={value} className="flex items-baseline gap-4"><span className="w-5 shrink-0 text-[10px] tabular-nums opacity-50">{value}</span><p dir="auto" className="min-w-0 [overflow-wrap:anywhere]" style={{ ...fontStyle, fontSize: value, lineHeight: 1.4 }}>{waterfallSample}</p></div>)}</div></div>
      </div> : <div className="flex min-h-0 flex-1 flex-col" data-testid={selectors.fontPreview.characters}>
        <div className="shrink-0 space-y-2 px-4 py-3"><label className="sr-only" htmlFor={searchId}>{t(c.search)}</label><input id={searchId} type="search" value={query} maxLength={64} onChange={event => { setQuery(event.target.value); setPage(0); }} placeholder={t(c.searchPlaceholder)} className="h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 text-base text-wc-text-primary outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" /><p className="text-[11px] leading-relaxed text-wc-text-muted">{t(c.coverage)}</p></div>
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4" style={specimenColors}>
          {characters.length ? <div className="grid" style={{ gridTemplateColumns: "repeat(auto-fill, minmax(64px, 1fr))" }}>{characters.map(code => <button key={code} type="button" aria-label={t(c.character, { code: unicodeLabel(code) })} aria-pressed={selected === code} onClick={() => { setSelected(code); }} className="flex min-h-24 min-w-0 flex-col items-center justify-center gap-2 rounded-lg px-1 py-3 outline-none hover:bg-wc-accent/10 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent aria-pressed:bg-wc-accent/15"><span dir="auto" className="max-w-full overflow-hidden text-3xl" style={fontStyle}>{String.fromCodePoint(code)}</span><span className="text-[10px] tabular-nums opacity-70">{unicodeLabel(code)}</span></button>)}</div> : <p role="status" className="py-12 text-center text-sm opacity-60">{t(c.empty)}</p>}
        </div>
        <div className="flex shrink-0 items-center justify-between gap-1 border-t border-wc-default px-2 py-1"><PreviewButton label={t(c.previous)} disabled={!currentPage} onClick={() => { setPage(currentPage - 1); }}><ChevronLeft className="rtl:rotate-180" /></PreviewButton><p role="status" className="text-xs tabular-nums text-wc-text-muted">{t(c.page, { current: number.format(currentPage + 1), total: number.format(pages) })}</p><PreviewButton label={t(c.next)} disabled={currentPage + 1 >= pages} onClick={() => { setPage(currentPage + 1); }}><ChevronRight className="rtl:rotate-180" /></PreviewButton></div>
        <div className="flex min-h-16 shrink-0 items-center gap-3 border-t border-wc-default px-4 py-2">{selected === null ? <p className="text-xs text-wc-text-muted">{t(c.selectionHint)}</p> : <><span className="w-12 shrink-0 overflow-hidden text-center text-4xl" style={fontStyle}>{String.fromCodePoint(selected)}</span><p className="min-w-0 flex-1 text-xs text-wc-text-muted" role="status">{t(c.selected, { code: unicodeLabel(selected) })}</p><button type="button" disabled={sampleText.length + String.fromCodePoint(selected).length > 1000} className={previewTextButtonClass} onClick={() => { setSample(current => (current ?? defaultSample) + String.fromCodePoint(selected)); setMode("specimen"); }}>{t(c.useCharacter)}</button></>}</div>
      </div>}
      <div className="flex shrink-0 items-center justify-between gap-2 border-t border-wc-default bg-wc-surface-raised px-2 py-1" data-testid={selectors.fontPreview.navigation}>
        {wide ? <div className="flex shrink-0 items-center" role="group" aria-label={t(c.mode)}>{(["specimen", "characters"] as const).map(value => <button key={value} type="button" aria-pressed={mode === value} onClick={() => { setMode(value); }} className={`${previewTextButtonClass} border-transparent px-3 aria-pressed:bg-wc-surface-input aria-pressed:text-wc-text-primary`}>{t(c[value])}</button>)}</div> : <select aria-label={t(c.mode)} value={mode} onChange={event => { setMode(event.target.value as FontMode); }} className={`${previewSelectClass} max-w-36 border-transparent bg-transparent`}><option value="specimen">{t(c.specimen)}</option><option value="characters">{t(c.characters)}</option></select>}
        {mode === "specimen" ? <div className="flex min-w-0 items-center gap-1"><PreviewButton label={t(c.smaller)} disabled={size <= 16} onClick={() => { changeSize(size - 4); }}><Minus /></PreviewButton>{wide && <div className="flex w-28">{sizeRange}</div>}<span className="w-12 text-center text-xs tabular-nums text-wc-text-secondary" aria-label={`${t(c.size)}: ${String(size)} px`}>{size} px</span><PreviewButton label={t(c.larger)} disabled={size >= 144} onClick={() => { changeSize(size + 4); }}><Plus /></PreviewButton></div> : <span className="px-2 text-xs tabular-nums text-wc-text-muted">{t(c.characterCount, { total: number.format(filtered.length) })}</span>}
      </div>
    </>}
    <ArtifactFooter model={model} metadata={`${(info?.format || fontFormat(model.basename) || "font").toUpperCase()} · ${formatBytes(model.sizeBytes)}`} />
  </ArtifactFrame>;
}
