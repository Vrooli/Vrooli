import { FormEvent, useEffect, useState } from "react";
import { createBinding, listChannels, type ChannelListing } from "../api/channels";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";

export function ChannelsPage() {
  const { t } = useTranslation();
  const [channels, setChannels] = useState<ChannelListing[] | null>(null);
  const [error, setError] = useState<string>();
  const [selected, setSelected] = useState<string>();
  const [agentId, setAgentId] = useState("");
  const [address, setAddress] = useState("");
  const [threadKey, setThreadKey] = useState("");
  const [saving, setSaving] = useState(false);
  const [attached, setAttached] = useState<string>();

  useEffect(() => {
    const controller = new AbortController();
    listChannels(controller.signal).then(setChannels).catch((e: unknown) => {
      if (!controller.signal.aborted) setError(e instanceof Error ? e.message : t(strings.errors.unknown));
    });
    return () => controller.abort();
  }, [t]);

  const attach = async (event: FormEvent) => {
    event.preventDefault();
    if (!selected || !agentId.trim() || !address.trim()) return;
    setSaving(true);
    setError(undefined);
    try {
      await createBinding({ agentId, channelId: selected, address, threadKey });
      setAttached(selected);
      setSelected(undefined);
      setAgentId("");
      setAddress("");
      setThreadKey("");
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t(strings.errors.unknown));
    } finally {
      setSaving(false);
    }
  };

  return <section aria-labelledby="channels-heading" className="flex flex-col gap-4">
    <h2 id="channels-heading" className="text-2xl font-semibold">{t(strings.console.channels.title)}</h2>
    <p className="text-app-muted-foreground">{t(strings.console.channels.description)}</p>
    {error && <div role="alert" className="rounded border border-red-300 p-4">{error}</div>}
    {attached && <div role="status" className="rounded border border-green-300 p-4">{t(strings.console.channels.attached, { channel: attached })}</div>}
    {!channels && !error && <p aria-busy="true">{t(strings.health.loading)}</p>}
    <ExperienceSurface surfaceId="catalog-region" state={error ? "error" : !channels ? "loading" : channels.length === 0 ? "empty" : "ready"} className="grid gap-3 md:grid-cols-2">
    {channels && <>{channels.map((channel) => <article key={channel.descriptor.id} className="rounded-lg border p-4">
      <div className="flex items-center justify-between"><h3 className="font-semibold">{channel.descriptor.displayName}</h3><span>{channel.availability}</span></div>
      <p className="text-sm text-app-muted-foreground">{t(strings.console.channels.setupFriction, { friction: channel.descriptor.setup.friction })}</p>
      {channel.reason && <p className="text-sm">{channel.reason}</p>}
      {channel.availability === "available" && channel.implemented ? <button type="button" className="mt-3 rounded bg-primary px-3 py-2 text-primary-foreground" onClick={() => setSelected(channel.descriptor.id)}>{t(strings.console.channels.attachAgent)}</button> : <button type="button" disabled className="mt-3 rounded border px-3 py-2">{t(strings.console.channels.connectRequirement)}</button>}
    </article>)}</>}
    </ExperienceSurface>
    {selected && <form onSubmit={(event) => void attach(event)} aria-label={t(strings.console.channels.attachFormLabel)} className="rounded-lg border p-6">
      <h3 className="font-semibold">{selected}</h3>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.console.channels.bindingDescription)}</p>
      <label className="mt-4 flex flex-col gap-1 text-sm" htmlFor="binding-agent">{t(strings.console.channels.agentId)}<input id="binding-agent" value={agentId} onChange={(event) => setAgentId(event.target.value)} className="rounded border p-2" required /></label>
      <label className="mt-3 flex flex-col gap-1 text-sm" htmlFor="binding-address">{t(strings.console.channels.address)}<input id="binding-address" value={address} onChange={(event) => setAddress(event.target.value)} className="rounded border p-2" placeholder={t(strings.console.channels.addressPlaceholder)} required /></label>
      <label className="mt-3 flex flex-col gap-1 text-sm" htmlFor="binding-thread">{t(strings.console.channels.threadKey)}<input id="binding-thread" value={threadKey} onChange={(event) => setThreadKey(event.target.value)} className="rounded border p-2" /></label>
      <div className="mt-4 flex gap-2"><button type="submit" disabled={saving} className="rounded bg-primary px-3 py-2 text-primary-foreground">{saving ? t(strings.console.channels.attaching) : t(strings.console.channels.confirmAttachment)}</button><button type="button" className="rounded border px-3 py-2" onClick={() => setSelected(undefined)}>{t(strings.console.channels.cancel)}</button></div>
    </form>}
  </section>;
}
