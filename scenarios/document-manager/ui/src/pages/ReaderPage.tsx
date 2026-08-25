import { useEffect, useRef, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1.1.0";
import { Input } from "../components/ui/input";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { queryCorpus, type QueryResult } from "../api/documentManager";

export function ReaderPage() {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<QueryResult[]>([]);
  const [partial, setPartial] = useState(false);
  const [active, setActive] = useState(0);
  const unitRefs = useRef<Array<HTMLButtonElement | null>>([]);
  useEffect(() => {
    if (!query.trim()) { setResults([]); setPartial(false); return; }
    const timer = window.setTimeout(() => {
      void queryCorpus(query).then((r) => { setResults(r.results); setPartial(r.partial); }).catch(() => setResults([]));
    }, 180);
    return () => window.clearTimeout(timer);
  }, [query]);
  const move = (delta: number) => { if (!results.length) return; const next = (active + delta + results.length) % results.length; setActive(next); unitRefs.current[next]?.focus(); };
  return <section data-testid={selectors.pages.reader} aria-labelledby="reader-heading" className="flex min-h-0 min-w-0 max-w-full flex-col gap-4 overflow-x-hidden">
    <div><h2 id="reader-heading" className="text-2xl font-semibold">{t(strings.pages.reader.title)}</h2><p className="text-app-muted-foreground">{t(strings.pages.reader.description)}</p></div>
    <div className="flex min-w-0 items-center gap-2"><label htmlFor="reader-query" className="sr-only">{t(strings.pages.reader.searchLabel)}</label><Input id="reader-query" value={query} onChange={(e) => setQuery(e.target.value)} placeholder={t(strings.pages.reader.placeholder)} data-testid={selectors.reader.query} onKeyDown={(e) => { if (e.key === "ArrowDown") { e.preventDefault(); move(1); } if (e.key === "ArrowUp") { e.preventDefault(); move(-1); } }} /><span role="status" data-testid={selectors.reader.localitySummary} aria-label={t(strings.pages.reader.local)} className="sr-only">{t(strings.pages.reader.local)}</span>{partial && <StatusBadge tone="warning" data-testid={selectors.reader.partial}>{t(strings.pages.reader.partial)}</StatusBadge>}</div>
    <div className="grid min-h-0 gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]" data-testid={selectors.reader.splitView}>
      <Card role="region" aria-label={t(strings.pages.reader.sourceRegion)} data-testid={selectors.reader.source}><CardHeader><CardTitle>{t(strings.pages.reader.sourceRegion)}</CardTitle></CardHeader><CardContent><div role="status" aria-label={t(strings.pages.reader.sourceRegion)} data-testid={selectors.reader.anchorHighlight} className="min-h-64 rounded-control border border-dashed border-app-border bg-app-surface-muted p-4" aria-live="polite">{t(strings.pages.reader.sourceHint)}</div></CardContent></Card>
      <Card role="region" aria-label={t(strings.pages.reader.derivedUnits)} data-testid={selectors.reader.units}><CardHeader><CardTitle>{t(strings.pages.reader.derivedUnits)}</CardTitle></CardHeader><CardContent><div role="listbox" tabIndex={0} aria-label={t(strings.pages.reader.derivedUnitsLabel)} data-testid={selectors.reader.unitsList} onKeyDown={(e) => { if (e.key === "ArrowDown") { e.preventDefault(); move(1); } if (e.key === "ArrowUp") { e.preventDefault(); move(-1); } }} className="space-y-2">{results.length === 0 ? <p className="text-sm text-app-muted-foreground">{t(strings.pages.reader.noMatches)}</p> : results.map((r, index) => <div key={r.unit_id}><button ref={(node) => { unitRefs.current[index] = node; }} type="button" role="option" aria-selected={active === index} aria-controls={`source-${r.unit_id}`} data-testid={selectors.reader.unit({ index })} onClick={() => setActive(index)} onFocus={() => setActive(index)} className={`w-full rounded-control border p-3 text-left ${active === index ? "border-app-primary bg-app-surface-muted" : "border-app-border"}`}><span className="block text-xs text-app-muted-foreground">{t(strings.pages.reader.confidence, { value: Math.max(0, Math.min(1, r.score)).toFixed(2) })} · <span role="status" data-testid={selectors.reader.locality} aria-label={t(strings.pages.reader.local)}>{t(strings.pages.reader.local)}</span></span><span className="block truncate">{r.document_hash}</span><code className="block truncate text-xs">{r.anchor_uri}</code></button><div id={`source-${r.unit_id}`} className="sr-only">{t(strings.pages.reader.sourceFor, { anchor: r.anchor_uri })}</div></div>)}</div></CardContent></Card>
    </div>
  </section>;
}
