import { useMemo } from "react";
import { useTranslation } from "react-i18next";

import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import { parseDelimited } from "../format";
import { CodeLinePreview } from "./TextRenderers";
import { PreviewNotice } from "./shared";
import type { PreviewRendererProps } from "../types";

const MAX_CSV_ROWS = 1000;

// CsvPreview renders CSV/TSV as a scrollable table with a sticky header,
// falling back to raw text when parsing yields nothing useful.
export function CsvPreview({ model, text }: PreviewRendererProps) {
  const { t } = useTranslation();
  const content = text?.content ?? "";
  const delimiter = model.resolvedPath.toLowerCase().endsWith(".tsv") ? "\t" : ",";

  const parsed = useMemo(() => parseDelimited(content, delimiter), [content, delimiter]);

  if (content.trim() === "") {
    return (
      <div className="flex h-full items-center justify-center text-sm text-wc-text-muted" data-testid="file-preview-empty">
        {t(strings.messagesFileViewer.emptyFile)}
      </div>
    );
  }
  if (parsed.length === 0 || parsed.every((r) => r.length <= 1 && (r[0] ?? "") === "")) {
    return (
      <div className="flex h-full flex-col" data-testid="file-preview-csv-fallback">
        <div className="px-4 pt-3">
          <PreviewNotice message={t(strings.messagesFileViewer.csvParseFallback)} tone="info" />
        </div>
        <div className="min-h-0 flex-1">
          <CodeLinePreview content={content} path={model.resolvedPath} highlightLine={null} />
        </div>
      </div>
    );
  }

  const header = parsed[0] ?? [];
  const bodyRows = parsed.slice(1, 1 + MAX_CSV_ROWS);
  const truncated = parsed.length - 1 > MAX_CSV_ROWS;

  return (
    <div className="flex h-full flex-col" data-testid="file-preview-csv">
      {(truncated || text?.truncated) && (
        <div className="px-4 pt-3">
          <PreviewNotice message={t(strings.messagesFileViewer.truncatedNotice)} tone="info" />
        </div>
      )}
      <div className="min-h-0 flex-1 overflow-auto p-4">
        <table className="w-full border-collapse text-left text-xs">
          <thead className="sticky top-0 z-wc-chrome">
            <tr>
              {header.map((cell, i) => (
                <th
                  key={i}
                  className="border border-wc-default bg-wc-surface-raised px-2 py-1 font-semibold text-wc-text-primary"
                >
                  {cell}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {bodyRows.map((r, ri) => (
              <tr key={ri} className={cn(ri % 2 === 1 && "bg-wc-surface-input/40")}>
                {header.map((_, ci) => (
                  <td key={ci} className="border border-wc-default px-2 py-1 align-top text-wc-text-secondary">
                    {r[ci] ?? ""}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

type DiffLineKind = "add" | "remove" | "hunk" | "meta" | "context";

function classifyDiffLine(line: string): DiffLineKind {
  if (line.startsWith("@@")) return "hunk";
  if (line.startsWith("+++") || line.startsWith("---") || line.startsWith("diff ") || line.startsWith("index ")) {
    return "meta";
  }
  if (line.startsWith("+")) return "add";
  if (line.startsWith("-")) return "remove";
  return "context";
}

const DIFF_LINE_CLASS: Record<DiffLineKind, string> = {
  add: "bg-emerald-500/10 text-emerald-300",
  remove: "bg-red-500/10 text-red-300",
  hunk: "bg-wc-accent/10 text-wc-accent",
  meta: "text-wc-text-muted",
  context: "text-[#c9d1d9]",
};

// DiffPreview renders unified diff/patch text with additions/removals/hunk
// highlighting and line numbers. Preserves content verbatim.
export function DiffPreview({ model, text }: PreviewRendererProps) {
  const { t } = useTranslation();
  const content = text?.content ?? "";
  const lines = useMemo(() => content.split("\n"), [content]);

  if (content.trim() === "") {
    return (
      <div className="flex h-full items-center justify-center text-sm text-wc-text-muted" data-testid="file-preview-empty">
        {t(strings.messagesFileViewer.emptyFile)}
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col bg-[#0d1117]" data-testid="file-preview-diff">
      {text?.truncated && (
        <div className="px-4 pt-3">
          <PreviewNotice message={t(strings.messagesFileViewer.truncatedNotice)} tone="info" />
        </div>
      )}
      <div className="min-h-0 flex-1 overflow-auto font-mono text-xs leading-[1.55]">
        {lines.map((line, i) => {
          const kind = classifyDiffLine(line);
          return (
            <div key={i} className={cn("flex items-start gap-3 px-3 py-px", DIFF_LINE_CLASS[kind])}>
              <span className="w-10 shrink-0 select-none text-end text-wc-text-faint/60 tabular-nums">{i + 1}</span>
              <pre className="m-0 flex-1 whitespace-pre-wrap break-words bg-transparent p-0">{line === "" ? " " : line}</pre>
            </div>
          );
        })}
      </div>
      <div className="shrink-0 border-t border-wc-default/60 bg-wc-surface-base px-4 py-2">
        <span className="font-mono text-[11px] uppercase tracking-wide text-wc-text-muted">{model.basename}</span>
      </div>
    </div>
  );
}
