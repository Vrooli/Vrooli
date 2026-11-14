import { useDeferredValue, useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { ArrowLeft, Braces, ChevronDown, ChevronRight, ChevronsDownUp, ChevronsUpDown, Code2, Copy, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { formatBytes } from "../format";
import { formatJsonSource, MAX_STRUCTURE_CHARACTERS, visibleStructure, type StructureNode } from "../structuredData";
import { useStructuredData } from "../useStructuredData";
import { usePreviewClipboard } from "../usePreviewClipboard";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, PreviewButton, PreviewLoading, PreviewToolbar, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";
import { CodeLinePreview } from "./TextRenderers";
import { PreviewNotice } from "./shared";

export function StructuredPreview(props: PreviewRendererProps) { return <StructureInspector key={props.model.previewId} {...props} />; }
function StructureInspector({ model, text }: PreviewRendererProps) {
  const { t } = useTranslation(), c = strings.readerPreview;
  const content = text?.content ?? "", json = /\.json$/i.test(model.resolvedPath);
  const result = useStructuredData(content, json, !!text?.truncated);
  const [source, setSource] = useState(!!model.line), [documentIndex, setDocumentIndex] = useState(0);
  const [query, setQuery] = useState(""), [selectedId, setSelectedId] = useState(0), [details, setDetails] = useState(false);
  const [expanded, setExpanded] = useState(new Set([0]));
  const frame = useRef<HTMLDivElement>(null), focusFrame = useRef(0);
  useEffect(() => () => { cancelAnimationFrame(focusFrame.current); }, []);
  const [formatted, setFormatted] = useState(false);
  const formattedContent = useMemo(() => { try { return formatted ? formatJsonSource(content) : content; } catch { return content; } }, [formatted, content]);
  const deferredQuery = useDeferredValue(query), clipboard = usePreviewClipboard();
  const { ref, rect } = useResizeObserver<HTMLDivElement>(), wide = (rect?.width ?? 0) >= 760;
  useEffect(() => {
    if (details && !wide) frame.current?.querySelector<HTMLButtonElement>(`[data-testid="${selectors.readerPreview.structureDetail}"] button`)?.focus({ preventScroll: true });
  }, [details, wide]);
  const document = result?.documents?.[documentIndex];
  const visible = useMemo(() => document ? visibleStructure(document, expanded, deferredQuery) : { rows: [], matches: new Set<number>() }, [document, expanded, deferredQuery]);
  const selected = document?.nodes[selectedId];
  const selectedVisible = visible.rows.some((row) => row.node.id === selectedId);
  const showSource = source || !!result?.error;
  const toggle = (id: number) => { setExpanded((current) => { const next = new Set(current); if (next.has(id)) next.delete(id); else next.add(id); return next; }); };
  const inspect = (node: StructureNode) => { setSelectedId(node.id); setDetails(true); };
  const keyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    let next = index;
    if (event.key === "ArrowDown") next++;
    else if (event.key === "ArrowUp") next--;
    else if (event.key === "Home") next = 0;
    else if (event.key === "End") next = visible.rows.length - 1;
    else if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
      const node = visible.rows[index]?.node;
      if (!node) return;
      if (event.key === "ArrowRight" && node.children.length && !expanded.has(node.id)) toggle(node.id);
      else if (event.key === "ArrowLeft" && expanded.has(node.id)) toggle(node.id);
      else if (event.key === "ArrowLeft" && node.parent !== null) next = visible.rows.findIndex((row) => row.node.id === node.parent);
      else if (event.key === "ArrowRight" && node.children.length) next++;
    } else return;
    event.preventDefault(); event.stopPropagation();
    event.currentTarget.closest("ul")?.querySelectorAll<HTMLButtonElement>("[data-structure-inspect]")[Math.max(0, Math.min(visible.rows.length - 1, next))]?.focus();
  };
  const countLabel = (node: StructureNode) => node.kind === "object" || node.kind === "array" ? t(node.kind === "array" ? c.items : c.properties, { count: node.children.length }) : node.kind === "string" ? JSON.stringify(node.value) : node.value;
  const value = selected ? selected.kind === "object" || selected.kind === "array" || selected.kind === "reference" ? content.slice(selected.start, selected.end) : selected.value : "";
  const status = clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : strings.artifactPreview.copyFailed) : `${json ? "JSON" : "YAML"} · ${formatBytes(model.sizeBytes)}`;
  return <ArtifactFrame testId="file-preview-structured" frameRef={frame}>
    <span role="status" className="sr-only">{clipboard.status ? status : ""}</span>
    <div ref={ref} className="flex min-h-0 flex-1 flex-col">
      <PreviewToolbar label={t(c.structureControls)}>
        <PreviewButton label={t(showSource ? c.showStructure : c.showSource)} pressed={showSource} disabled={!!result?.error} onClick={() => { setSource(!source); }}>{showSource ? <Braces /> : <Code2 />}</PreviewButton>
        <span className="min-w-0 flex-1 truncate px-2 text-xs text-wc-text-secondary">{t(showSource ? c.source : c.structure)}</span>
        {(result?.documents?.length ?? 0) > 1 && <select aria-label={t(c.yamlDocument)} value={documentIndex} className={`${previewSelectClass} max-w-36`} onChange={(event) => { setDocumentIndex(Number(event.target.value)); setSelectedId(0); setExpanded(new Set([0])); setDetails(false); }}>
          {result?.documents?.map((_, index) => <option key={index} value={index}>{t(c.documentNumber, { number: index + 1 })}</option>)}
        </select>}
        <PreviewButton label={t(c.expandAll)} disabled={showSource || !document || !!query} onClick={() => { setExpanded(new Set(document?.nodes.filter((node) => node.children.length).map((node) => node.id))); }}><ChevronsUpDown /></PreviewButton>
        <PreviewButton label={t(c.collapseAll)} disabled={showSource || !document || !!query} onClick={() => { setExpanded(new Set()); }}><ChevronsDownUp /></PreviewButton>
      </PreviewToolbar>
      {!showSource && <label className="relative block shrink-0 border-b border-wc-default px-3 py-1"><Search className="pointer-events-none absolute start-6 top-[18px] h-4 w-4 text-wc-text-muted" />
        <input type="search" value={query} aria-label={t(c.filterStructure)} placeholder={t(c.filterStructure)} onChange={(event) => { setQuery(event.target.value); setDetails(false); }}
          onKeyDown={(event) => { if (event.key === "Escape" && query) { event.stopPropagation(); setQuery(""); } }}
          className="h-11 w-full min-w-0 rounded-lg bg-wc-surface-input pe-3 ps-9 text-sm outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" /></label>}
      {result?.error && <PreviewNotice message={t(result.error === "invalid" ? c.invalidStructure : result.error === "limit" ? c.structureLimit : c.unsupportedStructure)} tone="info" />}
      {formatted && showSource && <p className="px-3 py-2 text-xs text-wc-text-muted">{t(c.formattedView)}</p>}
      {showSource ? <div className="min-h-0 flex-1">{result?.error === "limit" || content.length > MAX_STRUCTURE_CHARACTERS || text?.truncated
        ? <pre data-testid="file-preview-code" dir="ltr" role="region" aria-label={t(c.source)} tabIndex={0} className="m-0 h-full overflow-auto p-4 font-mono text-xs leading-relaxed">{content}</pre>
        : <CodeLinePreview content={formattedContent} path={model.resolvedPath} highlightLine={formatted ? null : model.line ?? null} truncated={text?.truncated} />}</div>
        : !result ? <div className="min-h-0 flex-1"><PreviewLoading label={t(c.loadingStructure)} /></div>
          : <div className="flex min-h-0 flex-1">
            {(!details || wide) && <div className="min-h-0 min-w-0 flex-1 overflow-auto p-2" data-testid={selectors.readerPreview.structureTree}>
              <ul aria-label={t(c.structure)}>{visible.rows.map(({ node, depth }, index) => <li key={node.id} className={`flex min-h-11 items-start rounded-lg ${selectedId === node.id ? "bg-wc-accent/10" : "hover:bg-wc-surface-input"}`}>
                <button type="button" tabIndex={node.children.length ? 0 : -1} disabled={!node.children.length} aria-label={t(c.toggleNode, { path: node.path })} aria-expanded={node.children.length ? expanded.has(node.id) || !!query : undefined}
                  onClick={() => { toggle(node.id); }} style={{ marginInlineStart: Math.min(depth, wide ? 8 : 4) * 12 }}
                  className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-wc-text-muted outline-none focus-visible:ring-2 focus-visible:ring-wc-accent disabled:invisible">{expanded.has(node.id) || query ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}</button>
                <button type="button" data-structure-inspect={node.id} aria-pressed={selectedId === node.id} tabIndex={selectedId === node.id || (!selectedVisible && index === 0) ? 0 : -1}
                  onFocus={() => { setSelectedId(node.id); }} onKeyDown={(event) => { keyDown(event, index); }} onClick={() => { inspect(node); }} title={node.path}
                  className="min-h-11 min-w-0 flex-1 rounded-lg px-1 py-2 text-start outline-none focus-visible:ring-2 focus-visible:ring-wc-accent">
                  <span className="block break-words text-xs font-semibold text-wc-text-primary">{node.key || '""'}</span>{" "}<span className={`block truncate font-mono text-xs ${node.kind === "string" ? "text-wc-success-text" : "text-wc-text-muted"}`}>{countLabel(node)}</span>
                </button>
              </li>)}</ul>
              {!visible.rows.length && <p role="status" className="p-5 text-sm text-wc-text-muted">{t(c.noProperties)}</p>}
            </div>}
            {(details || wide) && selected && <section aria-label={t(c.selectedValue)} className={`min-h-0 min-w-0 overflow-auto p-4 ${wide ? "w-[45%] border-s border-wc-default" : "flex-1"}`} data-testid={selectors.readerPreview.structureDetail}>
              {!wide && <button type="button" className={`${previewTextButtonClass} mb-4 border-transparent px-0`} onClick={() => { setDetails(false); focusFrame.current = requestAnimationFrame(() => { frame.current?.querySelector<HTMLButtonElement>(`[data-structure-inspect="${String(selectedId)}"]`)?.focus(); }); }}><ArrowLeft className="h-4 w-4" />{t(c.backStructure)}</button>}
              <p className="text-xs font-semibold text-wc-text-muted">{t(c.selectedValue)}</p><p dir="ltr" className="my-2 break-words font-mono text-xs text-wc-accent">{selected.path}</p>
              <p className="mb-3 text-xs text-wc-text-muted">{t(c.kind, { kind: t(c.kinds[selected.kind]) })}</p>
              <div className="mt-3 flex flex-wrap gap-2"><button type="button" className={previewTextButtonClass} onClick={() => { void clipboard.copy(value); }}><Copy className="h-4 w-4" />{t(c.copyValue)}</button><button type="button" className={previewTextButtonClass} onClick={() => { void clipboard.copy(selected.path); }}>{t(c.copyPath)}</button><button type="button" className={previewTextButtonClass} onClick={() => { void clipboard.copy(content.slice(selected.start, selected.end)); }}>{t(c.copySource)}</button></div>
              <pre dir="ltr" className="mt-3 max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-wc-default bg-wc-surface-input p-3 font-mono text-xs">{value === "" ? t(c.emptyString) : value}</pre>
              {selected.kind === "reference" && <p className="mt-3 text-xs text-wc-text-muted">{t(c.referenceHint)}</p>}
            </section>}
          </div>}
      {!showSource && document && <p role="status" className="shrink-0 border-t border-wc-default px-3 py-2 text-xs text-wc-text-muted">{query ? t(c.matchingProperties, { count: visible.matches.size }) : t(c.nodeCount, { count: document.nodes.length })}</p>}
    </div>
    <ArtifactFooter model={model} metadata={status}>
      <button type="button" className={previewTextButtonClass} onClick={() => { void clipboard.copy(content); }}><Copy className="h-4 w-4" />{t(c.copySource)}</button>
      {json && result?.documents && <button type="button" className={previewTextButtonClass} onClick={() => { setFormatted(!formatted); setSource(true); }}>{t(formatted ? c.originalSource : c.formatJson)}</button>}
    </ArtifactFooter>
  </ArtifactFrame>;
}
