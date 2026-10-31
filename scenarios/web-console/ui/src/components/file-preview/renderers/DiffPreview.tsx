import { Fragment, type CSSProperties, useEffect, useMemo, useRef, useState } from "react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { ChevronDown, ChevronUp, ChevronLeft, ChevronRight, Code2, Copy, GitCompareArrows, WrapText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { cn } from "../../../lib/classnames";
import { parseDiff, splitHunk, type DiffLine } from "../diffData";
import { usePreviewClipboard } from "../usePreviewClipboard";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, PreviewButton, InspectionHeader, InspectionOptions, InspectionSetting, InspectionChoices, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";
import { CodeLinePreview } from "./CodeLinePreview";
import { PreviewNotice } from "./shared";

export function DiffPreview(props: PreviewRendererProps) { return <ChangeReader key={props.model.previewId} {...props} />; }
function ChangeReader({ model, text }: PreviewRendererProps) {
  const { t } = useTranslation(), c = strings.inspectionPreview, content = text?.content ?? "";
  const parsed = useMemo(() => parseDiff(content), [content]);
  const [fileIndex, setFileIndex] = useState(0), [hunkIndex, setHunkIndex] = useState(0), [source, setSource] = useState(false), [layout, setLayout] = useState("auto"), [wrap, setWrap] = useState(true), [page, setPage] = useState(0);
  const { ref, rect } = useResizeObserver<HTMLDivElement>(), scroller = useRef<HTMLDivElement>(null), clipboard = usePreviewClipboard();
  const wide = (rect?.width ?? 0) >= 850, split = layout === "split" || layout === "auto" && wide;
  const file = parsed.files[fileIndex], hunk = file?.hunks[hunkIndex], fallback = !parsed.valid, showSource = source || fallback;
  const digits = Math.max(2, String(hunk?.lines.reduce((max, line) => Math.max(max, line.old ?? 0, line.next ?? 0), 0) ?? 0).length);
  const pairs = useMemo(() => splitHunk(hunk?.lines ?? []), [hunk]);
  const pages = Math.max(1, Math.ceil((split ? pairs.length : hunk?.lines.length ?? 0) / 400)), currentPage = Math.min(page, pages - 1);
  useEffect(() => { setPage(0); }, [fileIndex, hunkIndex, split]);
  const stats = parsed.files.reduce((counts, entry) => ({ added: counts.added + entry.added, removed: counts.removed + entry.removed }), { added: 0, removed: 0 });
  useEffect(() => { if (scroller.current) scroller.current.scrollTop = 0; }, [fileIndex, hunkIndex, source, currentPage]);
  const change = (direction: number) => {
    if (!file) return;
    if (hunkIndex + direction >= 0 && hunkIndex + direction < file.hunks.length) setHunkIndex(hunkIndex + direction);
    else if (fileIndex + direction >= 0 && fileIndex + direction < parsed.files.length) { const nextFile = fileIndex + direction; setFileIndex(nextFile); setHunkIndex(direction > 0 ? 0 : Math.max(0, (parsed.files[nextFile]?.hunks.length ?? 1) - 1)); }
  };
  return <ArtifactFrame testId="file-preview-diff">
    <span role="status" className="sr-only">{clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : strings.artifactPreview.copyFailed) : ""}</span>
    <div ref={ref} className="flex min-h-0 min-w-0 flex-1 flex-col">
      <InspectionHeader icon={<GitCompareArrows />} title={t(c.changes)} description={<span className="flex items-center gap-2"><span>{t(c.filesCount, { count: parsed.files.length })}</span><span className="rounded bg-wc-success/10 px-1.5 py-0.5 font-mono text-wc-success">+{stats.added}</span><span className="rounded bg-wc-error-text/10 px-1.5 py-0.5 font-mono text-wc-error-text">−{stats.removed}</span></span>}>
        <PreviewButton label={t(showSource ? c.showChanges : strings.readerPreview.showSource)} disabled={fallback} pressed={showSource} onClick={() => { setSource(!source); }}>{showSource ? <GitCompareArrows /> : <Code2 />}</PreviewButton>
        <InspectionOptions label={t(c.options)}>{() => <>
          <InspectionSetting label={t(c.diffLayout)}><InspectionChoices label={t(c.diffLayout)} value={layout} onChange={setLayout} options={[{ value: "auto", label: t(strings.artifactPreview.autoView) }, { value: "unified", label: t(c.unified) }, { value: "split", label: t(c.split) }]} /></InspectionSetting>
          <label className="flex min-h-12 cursor-pointer items-center gap-3 rounded-xl border border-wc-default px-3 py-2 text-sm"><WrapText className="h-4 w-4 text-wc-text-muted" aria-hidden="true" /><span className="flex-1">{t(strings.messagesFileViewer.wrap)}</span><input type="checkbox" checked={wrap} onChange={(event) => { setWrap(event.target.checked); }} className="peer sr-only" /><span aria-hidden="true" className={cn("wc-inspection-toggle peer-focus-visible:ring-2 peer-focus-visible:ring-wc-accent", wrap && "wc-inspection-toggle-on")} /></label>
        </>}</InspectionOptions>
      </InspectionHeader>
      {!showSource && <div className="flex shrink-0 items-center gap-1 border-b border-wc-default px-3 py-1">
        <select aria-label={t(c.changedFile)} value={fileIndex} onChange={(event) => { setFileIndex(Number(event.target.value)); setHunkIndex(0); }} className={`${previewSelectClass} wc-inspection-picker min-w-0 flex-1 font-medium`}>
          {parsed.files.map((entry, index) => <option key={entry.id} value={index}>{entry.name === "/dev/null" ? entry.oldName : entry.name || t(c.fileNumber, { number: index + 1 })} · +{entry.added} −{entry.removed}</option>)}
          {!parsed.files.length && <option>{model.basename}</option>}
        </select>
        {hunk && <span role="status" className="whitespace-nowrap px-1 text-[11px] tabular-nums text-wc-text-muted" title={t(c.changeNumber, { number: hunkIndex + 1, total: file.hunks.length })}>{hunkIndex + 1}/{file.hunks.length}</span>}
        <PreviewButton label={t(c.previousChange)} disabled={fileIndex === 0 && hunkIndex === 0} onClick={() => { change(-1); }}><ChevronUp /></PreviewButton>
        <PreviewButton label={t(c.nextChange)} disabled={fileIndex >= parsed.files.length - 1 && hunkIndex >= (file?.hunks.length ?? 1) - 1} onClick={() => { change(1); }}><ChevronDown /></PreviewButton>

      </div>}
      {text?.truncated && <PreviewNotice message={t(strings.messagesFileViewer.truncatedNotice)} tone="info" />}
      {fallback && <PreviewNotice message={t(c.diffFallback)} tone="info" />}
      {showSource ? <div className="min-h-0 flex-1"><CodeLinePreview content={content} path={model.resolvedPath} highlightLine={model.line ?? null} truncated={!!text?.truncated} /></div>
        : <div ref={scroller} tabIndex={0} role="region" aria-label={t(c.changes)} dir="ltr" className="min-h-0 min-w-0 flex-1 overflow-auto" data-testid={selectors.inspectionPreview.diffBody} style={{ "--wc-diff-digits": digits } as CSSProperties}>
          {hunk ? <section aria-label={hunk.header}>
            <div className="sticky top-0 z-10 border-b border-wc-default bg-wc-surface-raised px-3 py-2 font-mono text-xs text-wc-accent [overflow-wrap:anywhere]">{hunk.header}</div>
            {split ? <div className={cn("wc-diff-split", !wrap && "wc-diff-nowrap")}>
              <p className="wc-diff-column border-e border-wc-default">{t(c.oldVersion)}</p><p className="wc-diff-column">{t(c.newVersion)}</p>
              {pairs.slice(currentPage * 400, (currentPage + 1) * 400).map((pair, index) => <Fragment key={index}><DiffRow line={pair.left} wrap={wrap} side="old" /><DiffRow line={pair.right} wrap={wrap} side="new" /></Fragment>)}
            </div> : <div className={wrap ? "" : "wc-diff-nowrap"}>{hunk.lines.slice(currentPage * 400, (currentPage + 1) * 400).map((line) => <DiffRow key={line.id} line={line} wrap={wrap} />)}</div>}

          </section> : <pre className="m-0 whitespace-pre-wrap p-4 font-mono text-xs text-wc-text-secondary [overflow-wrap:anywhere]">{file?.metadata.join("\n")}</pre>}
        </div>}
      {!showSource && pages > 1 && <div className="flex shrink-0 items-center justify-end border-t border-wc-default px-3 text-xs text-wc-text-muted"><PreviewButton label={t(strings.artifactPreview.previousPage)} disabled={!currentPage} onClick={() => { setPage(currentPage - 1); }}><ChevronLeft /></PreviewButton><span>{t(strings.artifactPreview.tablePage, { page: currentPage + 1, total: pages })}</span><PreviewButton label={t(strings.artifactPreview.nextPage)} disabled={currentPage >= pages - 1} onClick={() => { setPage(currentPage + 1); }}><ChevronRight /></PreviewButton></div>}
    </div>
    <ArtifactFooter model={model} metadata={t(c.diffStats, { files: parsed.files.length, ...stats })}>
      <button type="button" className={previewTextButtonClass} onClick={() => { void clipboard.copy(content); }}><Copy className="h-4 w-4" aria-hidden="true" />{t(c.copySource)}</button>
    </ArtifactFooter>
  </ArtifactFrame>;
}
function DiffRow({ line, wrap, side }: { line: DiffLine | null; wrap: boolean; side?: "old" | "new" }) {
  const { t } = useTranslation();
  return <div data-diff-kind={line?.kind ?? "empty"} aria-hidden={line ? undefined : true} className={cn("wc-diff-line", side === "old" && "border-e border-wc-default", line?.kind === "add" ? "bg-wc-success/10 text-wc-success" : line?.kind === "remove" ? "bg-wc-error-text/10 text-wc-error-text" : "text-wc-text-secondary")}>
    {(!side || side === "old") && <span className="wc-diff-number">{line?.old !== null && line?.old !== undefined && <span className="sr-only">{t(strings.inspectionPreview.oldVersion)} </span>}{line?.old}</span>}
    {(!side || side === "new") && <span className="wc-diff-number">{line?.next !== null && line?.next !== undefined && <span className="sr-only">{t(strings.inspectionPreview.newVersion)} </span>}{line?.next}</span>}
    <pre className={cn("wc-diff-code", wrap ? "whitespace-pre-wrap [overflow-wrap:anywhere]" : "whitespace-pre")}>{line?.text ?? " "}</pre>
  </div>;
}
