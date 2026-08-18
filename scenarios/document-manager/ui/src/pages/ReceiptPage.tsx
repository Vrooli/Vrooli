import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { StatusBadge } from "../components/ui/status-badge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { listDocuments, type DocumentRecord } from "../api/documentManager";

export function ReceiptPage() {
  const { t } = useTranslation();
  const [documents, setDocuments] = useState<DocumentRecord[]>([]);
  useEffect(() => { void listDocuments().then((r) => setDocuments(r.documents)).catch(() => setDocuments([])); }, []);
  return <div role="region" data-testid={selectors.pages.receipt} aria-labelledby="receipt-heading" className="flex min-w-0 max-w-full flex-col gap-4 overflow-x-hidden"><div><h2 id="receipt-heading" className="text-2xl font-semibold">{t(strings.pages.receipt.title)}</h2><p className="text-app-muted-foreground">{t(strings.pages.receipt.description)}</p></div><Card className="min-w-0"><CardHeader><CardTitle>{t(strings.pages.receipt.timeline)}</CardTitle></CardHeader><CardContent><div role="status" aria-label={t(strings.pages.receipt.local)} data-testid={selectors.receipt.residencySummary} className="mb-3"><StatusBadge tone="success">{t(strings.pages.receipt.local)}</StatusBadge></div><ol aria-label={t(strings.pages.receipt.timeline)} data-testid={selectors.receipt.timeline} className="space-y-3">{documents.length === 0 ? <li>{t(strings.pages.receipt.empty)}</li> : documents.map((d) => <li key={d.id} className="grid min-w-0 gap-2 rounded-control border border-app-border p-3 sm:grid-cols-[minmax(0,1fr)_auto]"><div className="min-w-0"><strong className="block truncate">{d.source_name || t(strings.pages.receipt.unnamed)}</strong><div className="truncate text-xs text-app-muted-foreground">{d.detected_mime} · {t(strings.pages.receipt.privacy, { value: d.privacy_class })}</div></div><div className="flex items-center gap-2"><StatusBadge tone="success">{t(strings.pages.receipt.local)}</StatusBadge><StatusBadge>{t(strings.pages.receipt.parsed)}</StatusBadge></div></li>)}</ol></CardContent></Card></div>;
}
