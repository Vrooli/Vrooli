import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sourcesClient, type AdapterState, uploadArchive } from "../../api/sources";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { Input } from "../../components/ui/input";

const sourcesKey = ["sources", "adapters"] as const;

function AdapterRow({ adapter }: { adapter: AdapterState }) {
  const client = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const enabled = useMutation({
    mutationFn: () => sourcesClient.setAdapterEnabled({ adapterId: adapter.adapterId, enabled: !adapter.enabled }),
    onSuccess: () => void client.invalidateQueries({ queryKey: sourcesKey }),
  });
  const imported = useMutation({
    mutationFn: async () => uploadArchive(adapter.adapterId, file!),
    onSuccess: () => { setFile(null); void client.invalidateQueries({ queryKey: sourcesKey }); },
  });
  // Wire enum value 0 is unspecified; value 1 represents the semantic tier 0.
  const risk = adapter.riskTier === 1 ? "tier 0 — local export" : `tier ${adapter.riskTier - 1} — explicit enablement required`;
  return <li className="flex flex-col gap-2 rounded border border-app-border p-3">
    <div className="flex flex-wrap items-center justify-between gap-2"><strong>{adapter.adapterId}</strong><span className="text-sm text-app-muted-foreground">{risk}</span></div>
    <p className="text-sm text-app-muted-foreground">{adapter.enabled ? "Enabled" : `Disabled${adapter.disabledReason ? `: ${adapter.disabledReason}` : ""}`}{adapter.lastError ? ` · Last error: ${adapter.lastError}` : ""}</p>
    <div className="flex flex-wrap gap-2"><Button size="sm" variant="secondary" onClick={() => enabled.mutate()} disabled={enabled.isPending}>{adapter.enabled ? "Disable" : "Enable"}</Button><Input aria-label={`Archive for ${adapter.adapterId}`} type="file" accept="text/html,application/json,application/zip,.html,.json,.zip" onChange={(event) => setFile(event.target.files?.[0] ?? null)} /><Button size="sm" onClick={() => imported.mutate()} disabled={!adapter.enabled || !file || imported.isPending}>{imported.isPending ? "Importing…" : "Import export"}</Button></div>
    {imported.data?.result && <p className="text-sm text-app-muted-foreground">Imported {imported.data.result.created}; duplicates {imported.data.result.duplicated}; failed {imported.data.result.failed}.</p>}
    {(enabled.error || imported.error) && <p className="text-sm text-app-danger">Adapter operation failed. No platform request was made by archive import.</p>}
  </li>;
}

export function SourcesCard() {
  const adapters = useQuery({ queryKey: sourcesKey, queryFn: () => sourcesClient.listAdapters({}) });
  return <Card aria-label="Source adapters"><CardHeader><CardTitle>Source adapters</CardTitle></CardHeader><CardContent className="flex flex-col gap-3">
    <p className="text-sm text-app-muted-foreground">Archive imports are local tier-0 reads. Networked adapters remain disabled until you explicitly enable them.</p>
    {adapters.isLoading && <p>Loading adapters…</p>}
    {adapters.error && <p className="text-sm text-app-danger">Could not load adapter state.</p>}
    {adapters.data && <ul className="space-y-2">{adapters.data.adapters.map((adapter) => <AdapterRow key={adapter.adapterId} adapter={adapter} />)}</ul>}
  </CardContent></Card>;
}
