/* eslint-disable no-restricted-syntax, @typescript-eslint/use-unknown-in-catch-callback-variable */
import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { auditRecords } from "../api/deviceControl";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type RecordRow = { id: string; actor: string; device_id: string; verb: string; outcome: string; created_at: string };

export function EvidencePage() {
  const { t } = useTranslation();
  const [records, setRecords] = useState<RecordRow[]>([]);
  const [error, setError] = useState("");
  useEffect(() => { auditRecords().then(({ records: items }) => setRecords(items)).catch((e: Error) => setError(e.message)); }, []);
  return <section className="flex flex-col gap-4" aria-labelledby="evidence-heading"><h2 id="evidence-heading" className="text-2xl font-semibold">{t(strings.pages.evidence.title)}</h2><p className="text-app-muted-foreground">Every dispatched verb is attributable to an actor, device, and lease.</p><Card><CardHeader><CardTitle>Recent device verbs</CardTitle></CardHeader><CardContent>{error && <p role="alert">{error}</p>}{records.length === 0 && !error && <p className="text-app-muted-foreground">No device verbs have been dispatched.</p>}<div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr><th className="p-2">Verb</th><th className="p-2">Device</th><th className="p-2">Actor</th><th className="p-2">Outcome</th><th className="p-2">Created</th></tr></thead><tbody>{records.map((record) => <tr key={record.id} className="border-t"><td className="p-2">{record.verb}</td><td className="p-2">{record.device_id}</td><td className="p-2">{record.actor}</td><td className="p-2">{record.outcome}</td><td className="p-2">{record.created_at}</td></tr>)}</tbody></table></div></CardContent></Card></section>;
}
