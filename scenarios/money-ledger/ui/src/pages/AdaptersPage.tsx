import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { FormSection } from "../components/FormSection";
import { DirtyStateGuard } from "../components/DirtyStateGuard";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Select } from "../components/ui/select";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { fetchAdapters, importFile, registerAdapter, runAdapter } from "../api/ledger";
import { AdapterKind } from "@vrooli/proto-types/money-ledger/v1/ingest/ingest_pb";
import { formatDate } from "../i18n/format";

const timestampLabel = (timestamp?: { seconds: bigint | number; nanos?: number }) => timestamp ? formatDate(new Date(Number(timestamp.seconds) * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000))) : "—";

export function AdaptersPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const adapters = useQuery({ queryKey: ["adapters"], queryFn: fetchAdapters, retry: false });
  const unavailable = Boolean(adapters.data?.adapters.some((adapter) => !adapter.enabled || adapter.availabilityReason));
  const surface = useSurfaceState({
    query: { isLoading: adapters.isLoading, isFetching: adapters.isFetching, isError: adapters.isError, error: adapters.error },
    availability: { partial: unavailable },
    empty: Boolean(adapters.data && adapters.data.adapters.length === 0),
  });
  const [form, setForm] = useState({ id: "", name: "", kind: String(AdapterKind.FILE) });
  const [fileAdapterId, setFileAdapterId] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState(false);
  const complete = async () => { await queryClient.invalidateQueries({ queryKey: ["adapters"] }); setError(false); setMessage(t(strings.pages.adapters.savedNotice)); };
  const registerMutation = useMutation({ mutationFn: () => registerAdapter({ id: form.id.trim(), name: form.name.trim(), kind: Number(form.kind) }), onSuccess: async () => { await complete(); setForm({ id: "", name: "", kind: String(AdapterKind.FILE) }); }, onError: () => setError(true) });
  const runMutation = useMutation({ mutationFn: (adapterId: string) => runAdapter(adapterId), onSuccess: complete, onError: () => setError(true) });
  const importMutation = useMutation({ mutationFn: async (input: { adapterId: string; file: File }) => importFile(input.adapterId, new Uint8Array(await input.file.arrayBuffer())), onSuccess: complete, onError: () => setError(true) });
  const submitRegistration = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setMessage("");
    if (!form.id.trim() || !form.name.trim()) { setError(true); return; }
    registerMutation.mutate();
  };
  const submitImport = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const input = event.currentTarget.elements.namedItem("adapter-file") as HTMLInputElement;
    const file = input.files?.[0];
    setMessage("");
    if (!fileAdapterId || !file) { setError(true); return; }
    importMutation.mutate({ adapterId: fileAdapterId, file });
  };

  return (
    <ExperienceSurface surfaceId="adapters" state={surface.state} statusMessage={surface.reason} data-testid={selectors.pages.adapters} aria-labelledby="adapters-heading" className="flex flex-col gap-4">
      <h2 id="adapters-heading" className="text-2xl font-semibold">{t(strings.pages.adapters.title)}</h2>
      <Card>
        <CardHeader><CardTitle>{t(strings.pages.adapters.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.adapters.description)}</p>
          <ul data-testid={selectors.pages.adapterList} aria-label={t(strings.pages.adapters.cardTitle)} className="mt-4 grid gap-3">
            <li data-testid={selectors.pages.manualAdapterEntry} className="rounded-md border p-3">{t(strings.pages.adapters.manualAdapter)}</li>
            {adapters.data?.adapters.map((adapter) => <li key={adapter.id} className="rounded-md border p-3">
              <div className="flex flex-wrap items-center justify-between gap-3"><span className="font-medium">{adapter.name} · {adapter.id}</span><span>{!adapter.enabled || adapter.availabilityReason ? t(strings.pages.adapters.unavailable) : !adapter.lastSuccessAt ? t(strings.pages.adapters.neverRun) : t(strings.pages.adapters.available)}</span></div>
              <p className="text-sm text-app-muted-foreground">{t(strings.pages.adapters.lastSuccess)}: {timestampLabel(adapter.lastSuccessAt)}</p>
              {adapter.availabilityReason && <p role="alert" className="text-sm text-app-danger">{t(strings.pages.adapters.availabilityReason)}: {adapter.availabilityReason}</p>}
              <Button type="button" className="mt-2" disabled={runMutation.isPending || !adapter.enabled} onClick={() => runMutation.mutate(adapter.id)}>{t(strings.pages.adapters.runAction)}</Button>
            </li>)}
          </ul>
          <p data-testid={selectors.pages.adapterAvailability} role="status" className="mt-3">{unavailable ? t(strings.pages.adapters.unavailable) : t(strings.pages.adapters.available)}</p>
          <p data-testid={selectors.pages.failureReason} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.adapters.failureReason)}</p>
          <p data-testid={selectors.pages.lastSuccessAge} role="status" className="text-sm text-app-muted-foreground">{t(strings.pages.adapters.lastSuccessAge)}</p>
          <p data-testid={selectors.pages.missingImpact} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.adapters.missingImpact)}</p>
          <p data-testid={selectors.pages.credentialGap} role="alert" className="text-sm text-app-muted-foreground">{t(strings.pages.adapters.credentialGap)}</p>
          <p data-testid={selectors.pages.adaptersEmptyGuidance} role="note" className={surface.state === "empty" ? "mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground" : "sr-only"}>{t(strings.pages.adapters.emptyGuidance)}</p>
        </CardContent>
      </Card>
      <div className="grid gap-4 lg:grid-cols-2">
        <DirtyStateGuard isDirty={Boolean(form.id || form.name)} protectUnload title={t(strings.pages.adapters.registerTitle)} description={t(strings.pages.adapters.description)}>
          <FormSection title={t(strings.pages.adapters.registerTitle)}>
            <form className="grid gap-3" onSubmit={submitRegistration}>
              <label className="grid gap-1" htmlFor="adapter-id"><span>{t(strings.pages.adapters.adapterIdLabel)}</span><Input id="adapter-id" value={form.id} onChange={(event) => setForm({ ...form, id: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="adapter-name"><span>{t(strings.pages.adapters.adapterNameLabel)}</span><Input id="adapter-name" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="adapter-kind"><span>{t(strings.pages.adapters.adapterKindLabel)}</span><Select id="adapter-kind" value={form.kind} onChange={(event) => setForm({ ...form, kind: event.target.value })} options={[{ value: String(AdapterKind.FILE), label: "FILE" }, { value: String(AdapterKind.AGGREGATOR), label: "AGGREGATOR" }]} /></label>
              <Button type="submit" disabled={registerMutation.isPending}>{t(strings.pages.adapters.registerAction)}</Button>
            </form>
          </FormSection>
        </DirtyStateGuard>
        <DirtyStateGuard isDirty={Boolean(fileAdapterId)} protectUnload title={t(strings.pages.adapters.importTitle)} description={t(strings.pages.adapters.description)}>
          <FormSection title={t(strings.pages.adapters.importTitle)}>
            <form className="grid gap-3" onSubmit={submitImport}>
              <label className="grid gap-1" htmlFor="import-adapter"><span>{t(strings.pages.adapters.adapterIdLabel)}</span><Select id="import-adapter" value={fileAdapterId} onChange={(event) => setFileAdapterId(event.target.value)} options={(adapters.data?.adapters ?? []).filter((adapter) => adapter.kind === AdapterKind.FILE).map((adapter) => ({ value: adapter.id, label: adapter.name }))} placeholder={t(strings.pages.adapters.adapterIdLabel)} /></label>
              <label className="grid gap-1" htmlFor="adapter-file"><span>{t(strings.pages.adapters.fileLabel)}</span><Input id="adapter-file" name="adapter-file" type="file" accept=".csv,text/csv" /></label>
              <Button type="submit" disabled={importMutation.isPending}>{t(strings.pages.adapters.importTitle)}</Button>
            </form>
          </FormSection>
        </DirtyStateGuard>
      </div>
      {error && <p role="alert" className="text-sm text-app-danger">{t(strings.pages.adapters.requestError)}</p>}
      {message && <p role="status" className="text-sm text-app-success">{message}</p>}
    </ExperienceSurface>
  );
}
