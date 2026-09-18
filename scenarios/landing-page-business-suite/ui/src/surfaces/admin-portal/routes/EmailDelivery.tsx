import { FormEvent, useEffect, useState } from 'react';
import { getEmailReadiness, getSignInDeliveryAdminReport, sendDeliveryProbe, type EmailReadinessReport, type SignInDeliveryAdminReport } from '../../../shared/api/emailReadiness';

export function EmailDelivery() {
  const [readiness, setReadiness] = useState<EmailReadinessReport | null>(null);
  const [delivery, setDelivery] = useState<SignInDeliveryAdminReport | null>(null);
  const [address, setAddress] = useState('');
  const [probeMessage, setProbeMessage] = useState('');
  const [historySearch, setHistorySearch] = useState('');
  const [loading, setLoading] = useState(true);

  const refresh = async (recipient = '') => {
    setLoading(true);
    try {
      const [nextReadiness, nextDelivery] = await Promise.all([getEmailReadiness(), getSignInDeliveryAdminReport(recipient)]);
      setReadiness(nextReadiness);
      setDelivery(nextDelivery);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { void refresh(); }, []);

  const submitProbe = async (event: FormEvent) => {
    event.preventDefault();
    setProbeMessage('');
    try {
      const result = await sendDeliveryProbe(address);
      setProbeMessage(`Probe queued: ${result.request_id}`);
    } catch (error) {
      setProbeMessage(error instanceof Error ? error.message : 'Unable to queue probe');
    }
  };

  return (
    <main className="mx-auto max-w-5xl px-6 py-10 text-slate-100" aria-labelledby="email-delivery-title">
      <div className="mb-8 flex items-start justify-between gap-4">
        <div><h1 id="email-delivery-title" className="text-3xl font-semibold">Email delivery</h1><p className="mt-2 text-slate-400">DNS authorization, provider state, and the durable outbox.</p></div>
        <button type="button" className="rounded-lg border border-white/15 px-4 py-2 text-sm" onClick={() => void refresh()} disabled={loading}>{loading ? 'Refreshing…' : 'Refresh'}</button>
      </div>
      <section className="mb-6 rounded-2xl border border-white/10 bg-white/5 p-5" aria-label="Provider authorization">
        <h2 className="text-lg font-semibold">Provider authorization</h2>
        <p className="mt-1 text-sm text-slate-400">A provider is eligible only when its credential and DNS requirements are usable.</p>
        <div className="mt-4 grid gap-3 md:grid-cols-2">
          {readiness?.Providers.map((provider) => <div key={provider.Provider} className="rounded-xl border border-white/10 p-4"><div className="flex justify-between"><span>{provider.Provider}</span><span className={provider.Status === 'pass' ? 'text-emerald-300' : provider.Status === 'warn' ? 'text-amber-300' : 'text-rose-300'}>{provider.Status}</span></div><p className="mt-2 text-sm text-slate-400">{provider.Detail}</p>{provider.Cost !== undefined && <p className="mt-1 text-xs text-slate-500">SPF lookup cost: {provider.Cost}</p>}</div>)}
          {readiness && <div className="rounded-xl border border-white/10 p-4"><div className="flex justify-between"><span>DMARC</span><span className={readiness.DMARC.Status === 'pass' ? 'text-emerald-300' : 'text-rose-300'}>{readiness.DMARC.Status}</span></div><p className="mt-2 text-sm text-slate-400">{readiness.DMARC.Detail}</p></div>}
        </div>
      </section>
      <section className="mb-6 rounded-2xl border border-white/10 bg-white/5 p-5" aria-label="Outbox state">
        <h2 className="text-lg font-semibold">Durable outbox</h2>
        <div className="mt-4 flex flex-wrap gap-4 text-sm">{Object.entries(delivery?.outbox ?? {}).map(([state, count]) => <span key={state} className="rounded-full bg-slate-800 px-3 py-1">{state}: {count}</span>)}{!delivery?.outbox && <span className="text-slate-400">No outbox rows reported.</span>}</div>
      </section>
      <section className="mb-6 rounded-2xl border border-white/10 bg-white/5 p-5" aria-label="Provider routing">
        <h2 className="text-lg font-semibold">Provider status and routing</h2>
        <div className="mt-4 grid gap-3 md:grid-cols-2">
          {(delivery?.providers ?? []).map((provider) => <article key={provider.id} className="rounded-xl border border-white/10 p-4"><div className="flex items-center justify-between"><span className="font-medium">{provider.id}</span><span className={provider.state === 'ready' ? 'text-emerald-300' : 'text-amber-300'}>{provider.state}</span></div><p className="mt-2 text-sm text-slate-400">Credential: {provider.credential.detail}</p><p className="mt-1 text-sm text-slate-400">DNS: {provider.dns.detail}</p>{provider.remedy && <p className="mt-2 text-sm text-cyan-200">Remedy: {provider.remedy}</p>}</article>)}
          {!delivery?.providers?.length && <p className="text-sm text-slate-400">Provider registry data is not available.</p>}
        </div>
        <h3 className="mt-6 font-medium">Current ladder{delivery?.chosen_provider ? ` · selected ${delivery.chosen_provider}` : ''}</h3>
        <div className="mt-2 space-y-2">{(delivery?.routing ?? []).map((route) => <div key={route.provider_id} className="flex justify-between rounded-lg bg-slate-950/60 px-3 py-2 text-sm"><span>{route.provider_id} · rank {route.cost_rank}</span><span className={route.eligible ? 'text-emerald-300' : 'text-amber-300'}>{route.eligible ? 'eligible' : route.skip_reason ?? 'skipped'}</span></div>)}</div>
      </section>
      <section className="mb-6 rounded-2xl border border-white/10 bg-white/5 p-5" aria-label="Allowance and queue health">
        <h2 className="text-lg font-semibold">Allowance and queue health</h2>
        <div className="mt-4 grid gap-4 md:grid-cols-2"><div><h3 className="font-medium">Quota windows</h3><div className="mt-2 space-y-2">{(delivery?.quotas ?? []).map((quota) => <div key={`${quota.provider_id}-${quota.window}`} className="text-sm text-slate-300">{quota.provider_id} · {quota.window}: {quota.used}/{quota.ceiling}</div>)}{!delivery?.quotas?.length && <p className="text-sm text-slate-400">No active quota windows.</p>}</div></div><div><h3 className="font-medium">Queue</h3><p className="mt-2 text-sm text-slate-300">Oldest waiting message: {delivery?.queue_health?.oldest_wait_seconds ?? 0}s</p><p className="mt-1 text-sm text-slate-300">Median acceptance: {delivery?.queue_health?.median_acceptance_seconds_24h ?? '—'}s</p>{(delivery?.queue_health?.by_priority ?? []).map((item) => <p key={item.priority} className="mt-1 text-xs text-slate-400">Priority {item.priority}: {item.depth} waiting</p>)}</div></div>
      </section>
      <section className="mb-6 rounded-2xl border border-white/10 bg-white/5 p-5" aria-label="Message history">
        <div className="flex flex-wrap items-center justify-between gap-3"><h2 className="text-lg font-semibold">Message history</h2><form className="flex gap-2" onSubmit={(event) => { event.preventDefault(); void refresh(historySearch); }}><input className="rounded-lg border border-white/15 bg-slate-950 px-3 py-2 text-sm" value={historySearch} onChange={(event) => setHistorySearch(event.target.value)} placeholder="Recipient address" aria-label="Search message history" /><button type="submit" className="rounded-lg border border-white/15 px-3 py-2 text-sm">Search</button></form></div>
        <div className="mt-4 space-y-3">{(delivery?.history ?? []).map((item) => <article key={item.id} className="rounded-xl border border-white/10 p-4 text-sm"><div className="flex flex-wrap justify-between gap-2"><span>{item.purpose} · {item.recipient}</span><span className="text-slate-400">{item.status}</span></div><p className="mt-1 text-slate-400">Provider: {item.provider_id || 'not selected'} · requested {new Date(item.requested_at).toLocaleString()}</p>{item.last_error && <p className="mt-1 text-amber-200">{item.last_error}</p>}<div className="mt-2 space-y-1 text-xs text-slate-500">{(item.attempts ?? []).map((attempt) => <p key={`${item.id}-${attempt.attempt}`}>Attempt {attempt.attempt}: {attempt.provider_id} · {attempt.outcome}{attempt.diagnostic ? ` · ${attempt.diagnostic}` : ''}</p>)}</div></article>)}{!delivery?.history?.length && <p className="text-sm text-slate-400">No messages match the current search.</p>}</div>
      </section>
      <section className="rounded-2xl border border-white/10 bg-white/5 p-5" aria-label="Delivery probe">
        <h2 className="text-lg font-semibold">External delivery probe</h2>
        <p className="mt-1 text-sm text-slate-400">Use an external inbox. This tests the same queue used by sign-in.</p>
        <form className="mt-4 flex flex-col gap-3 sm:flex-row" onSubmit={submitProbe}><input className="min-w-0 flex-1 rounded-lg border border-white/15 bg-slate-950 px-3 py-2" type="email" required value={address} onChange={(event) => setAddress(event.target.value)} placeholder="you@example.net" /><button className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium" type="submit">Queue probe</button></form>
        {probeMessage && <p className="mt-3 text-sm text-slate-300" role="status">{probeMessage}</p>}
      </section>
    </main>
  );
}
