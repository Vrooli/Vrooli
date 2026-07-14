import { useEffect, useMemo, useState } from "react";
import { Check, Copy, Columns2, Rows3 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { useTheme } from "../../components/theme/useTheme";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DiffOp, type DiffCell, type DiffRow } from "../../api/versions";

type ViewMode = "split" | "unified";

interface VersionDiffViewerProps {
  rows: DiffRow[];
}

/**
 * VersionDiffViewer consumes the server-aligned diff rows and presents them as
 * a source-oriented diff surface. Shiki is loaded on demand so opening the
 * Info panel does not add its language bundle to the initial editor route.
 */
export function VersionDiffViewer({ rows }: VersionDiffViewerProps) {
  const { t } = useTranslation();
  const [mode, setMode] = useState<ViewMode>("split");
  const [copied, setCopied] = useState(false);
  const source = useMemo(
    () => rows.map((row) => row.right?.text ?? row.left?.text ?? "").join("\n"),
    [rows],
  );

  const copyDiff = async () => {
    try {
      await navigator.clipboard.writeText(source);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1_500);
    } catch {
      // Clipboard access may be disabled by an embedded browser; rendering is unaffected.
    }
  };

  return (
    <section aria-label={t(strings.versions.diff.viewerLabel)} className="mt-2 overflow-hidden rounded border border-app-border bg-app-surface">
      <header className="flex items-center justify-between gap-2 border-b border-app-border bg-app-surface-muted px-2 py-1.5">
        <div className="flex items-center gap-1" role="group" aria-label={t(strings.versions.diff.modeLabel)}>
          <Button
            type="button"
            variant={mode === "split" ? "primary" : "secondary"}
            className="h-7 gap-1 px-2 text-xs"
            onClick={() => setMode("split")}
            aria-pressed={mode === "split"}
          >
            <Columns2 aria-hidden className="h-3.5 w-3.5" /> {t(strings.versions.diff.split)}
          </Button>
          <Button
            type="button"
            variant={mode === "unified" ? "primary" : "secondary"}
            className="h-7 gap-1 px-2 text-xs"
            onClick={() => setMode("unified")}
            aria-pressed={mode === "unified"}
          >
            <Rows3 aria-hidden className="h-3.5 w-3.5" /> {t(strings.versions.diff.unified)}
          </Button>
        </div>
        <Button type="button" variant="secondary" className="h-7 gap-1 px-2 text-xs" onClick={() => void copyDiff()}>
          {copied ? <Check aria-hidden className="h-3.5 w-3.5" /> : <Copy aria-hidden className="h-3.5 w-3.5" />}
          {copied ? t(strings.versions.diff.copied) : t(strings.versions.diff.copy)}
        </Button>
      </header>
      <div className="max-h-[32rem] overflow-auto font-mono text-[0.7rem] leading-5">
        {mode === "split" ? (
          <div role="table" className="min-w-[42rem]">
            {rows.map((row, index) => <SplitHunkRow key={index} row={row} />)}
          </div>
        ) : (
          <div role="table" className="min-w-[28rem]">
            {rows.flatMap((row, index) => unifiedLines(row).map((line, lineIndex) => (
              <UnifiedHunkLine key={`${index}-${lineIndex}`} line={line} />
            )))}
          </div>
        )}
      </div>
    </section>
  );
}

function SplitHunkRow({ row }: { row: DiffRow }) {
  return (
    <div data-testid={selectors.versions.diff.row} role="row" className="grid grid-cols-2 align-top border-b border-app-border/40 last:border-b-0">
      <DiffLine cell={row.left} />
      <DiffLine cell={row.right} />
    </div>
  );
}

function UnifiedHunkLine({ line }: { line: DiffCell }) {
  return <DiffLine cell={line} />;
}

function unifiedLines(row: DiffRow): DiffCell[] {
  if (row.left?.op === DiffOp.EQUAL && row.right?.op === DiffOp.EQUAL) return [row.right];
  return [row.left, row.right].filter((cell): cell is DiffCell => Boolean(cell && cell.op !== DiffOp.EMPTY));
}

function DiffLine({ cell }: { cell: DiffCell | undefined }) {
  if (!cell || cell.op === DiffOp.EMPTY) return <div role="cell" className="min-h-5" />;
  const marker = cell.op === DiffOp.ADD ? "+" : cell.op === DiffOp.REMOVE ? "-" : " ";
  const tone = cell.op === DiffOp.ADD
    ? "bg-app-success/10"
    : cell.op === DiffOp.REMOVE
      ? "bg-app-danger/10"
      : "text-app-muted-foreground";
  return (
    <div role="cell" className={`flex min-w-0 px-2 ${tone}`}>
      <span className="w-8 shrink-0 select-none text-right text-app-muted-foreground">{cell.lineNumber || ""}</span>
      <span className="mx-2 w-3 shrink-0 select-none text-app-muted-foreground">{marker}</span>
      <HighlightedSource source={cell.text} />
    </div>
  );
}

function HighlightedSource({ source }: { source: string }) {
  const { resolved } = useTheme();
  const [html, setHTML] = useState<string>();

  useEffect(() => {
    let active = true;
    void import("shiki")
      .then(({ codeToHtml }) => codeToHtml(source, { lang: "tsx", theme: resolved === "dark" ? "github-dark" : "github-light" }))
      .then((rendered) => {
        if (active) setHTML(rendered);
      })
      .catch(() => {
        if (active) setHTML(undefined);
      });
    return () => { active = false; };
  }, [resolved, source]);

  if (!html) return <span className="min-w-0 whitespace-pre-wrap">{source}</span>;
  // Shiki escapes source text before emitting its token spans.
  return <span className="min-w-0 whitespace-pre-wrap [&_pre]:m-0 [&_pre]:inline [&_pre]:bg-transparent! [&_pre]:p-0! [&_code]:bg-transparent!" dangerouslySetInnerHTML={{ __html: html }} />;
}
