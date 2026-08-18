import { useEffect, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { StatusBadge } from "../components/ui/status-badge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { listCollections, listDocuments, type Collection, type DocumentRecord } from "../api/documentManager";

export function CorpusPage() {
  const { t } = useTranslation();
  const [collections, setCollections] = useState<Collection[]>([]);
  const [documents, setDocuments] = useState<DocumentRecord[]>([]);
  const [error, setError] = useState<string>();
  useEffect(() => {
    void Promise.all([listCollections(), listDocuments()])
      .then(([c, d]) => { setCollections(c.collections); setDocuments(d.documents); })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : t(strings.errors.unknown)));
  }, [t]);
  return <div role="region" data-testid={selectors.pages.corpus} aria-labelledby="corpus-heading" className="flex min-w-0 max-w-full flex-col gap-4 overflow-x-hidden">
    <div><h2 id="corpus-heading" className="text-2xl font-semibold">{t(strings.pages.corpus.title)}</h2><p className="text-app-muted-foreground">{t(strings.pages.corpus.description)}</p></div>
    {error && <StatusBadge tone="danger" data-testid={selectors.corpus.error}>{error}</StatusBadge>}
    <div className="grid min-w-0 max-w-full gap-4 lg:grid-cols-2">
      <Card className="min-w-0"><CardHeader><CardTitle>{t(strings.pages.corpus.collections)}</CardTitle><CardDescription>{t(strings.pages.corpus.collectionDescription)}</CardDescription></CardHeader><CardContent><div role="list" aria-label={t(strings.pages.corpus.collections)} data-testid={selectors.corpus.collections}>{collections.length === 0 ? <p data-testid={selectors.corpus.empty}>{t(strings.pages.corpus.empty)}</p> : <ul className="space-y-2">{collections.map((c) => <li key={c.id} className="flex min-w-0 items-center justify-between gap-3 rounded-control border border-app-border p-3"><span className="min-w-0"><strong className="block truncate">{c.name}</strong><span className="block truncate text-xs text-app-muted-foreground">{c.id}</span></span><StatusBadge tone={c.federated ? "info" : "neutral"}>{t(c.federated ? strings.pages.corpus.federated : strings.pages.corpus.localOnly)}</StatusBadge></li>)}</ul>}</div></CardContent></Card>
      <Card className="min-w-0"><CardHeader><CardTitle>{t(strings.pages.corpus.documents)}</CardTitle><CardDescription>{t(strings.pages.corpus.documentDescription)}</CardDescription></CardHeader><CardContent><div role="list" aria-label={t(strings.pages.corpus.documents)} data-testid={selectors.corpus.documents}>{documents.length === 0 ? <><p>{t(strings.pages.corpus.noDocuments)}</p><span role="status" data-testid={selectors.corpus.locality}>{t(strings.pages.reader.local)}</span></> : <ul className="space-y-2">{documents.map((d) => <li key={d.id} className="min-w-0 rounded-control border border-app-border p-3"><div className="flex min-w-0 items-center justify-between gap-3"><strong className="min-w-0 truncate">{d.source_name || t(strings.pages.corpus.unnamed)}</strong><StatusBadge>{t(strings.pages.corpus.privacy, { value: d.privacy_class })}</StatusBadge></div><div className="flex min-w-0 items-center gap-2 text-xs"><span role="status" data-testid={selectors.corpus.locality}>{t(strings.pages.reader.local)}</span><span className="truncate">{d.detected_mime}</span></div><code className="block max-w-full truncate text-xs text-app-muted-foreground">{d.content_sha256}</code></li>)}</ul>}</div></CardContent></Card>
    </div>
  </div>;
}
