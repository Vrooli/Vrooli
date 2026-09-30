import { FormEvent, useEffect, useMemo, useState } from 'react';
import type { HTMLAttributes, ReactNode } from 'react';
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Clock3,
  Info,
  KeyRound,
  Mail,
  RefreshCw,
  Search,
  Send,
  Server,
  ShieldCheck,
  XCircle,
  type LucideIcon,
} from 'lucide-react';
import { getEmailReadiness, getSignInDeliveryAdminReport, saveProviderCredential, sendDeliveryProbe, type EmailReadinessReport, type ProviderCredentialField, type SignInDeliveryAdminReport } from '../../../shared/api/emailReadiness';
import { Button } from '../../../shared/ui/button';
import { PageHeader } from '../components/PageHeader';

type HealthTone = 'good' | 'warning' | 'critical' | 'neutral';

const toneClasses: Record<HealthTone, { icon: string; text: string; surface: string; border: string }> = {
  good: { icon: 'text-emerald-300', text: 'text-emerald-200', surface: 'bg-emerald-500/10', border: 'border-emerald-400/20' },
  warning: { icon: 'text-amber-300', text: 'text-amber-200', surface: 'bg-amber-500/10', border: 'border-amber-400/20' },
  critical: { icon: 'text-rose-300', text: 'text-rose-200', surface: 'bg-rose-500/10', border: 'border-rose-400/20' },
  neutral: { icon: 'text-slate-300', text: 'text-slate-200', surface: 'bg-white/5', border: 'border-white/10' },
};

function readinessTone(status?: string): HealthTone {
  if (status === 'pass' || status === 'ready') return 'good';
  if (status === 'fail' || status === 'blocked' || status === 'needs_setup') return 'critical';
  if (status === 'warn' || status === 'pending') return 'warning';
  return 'neutral';
}

function StatusIcon({ tone, className = 'h-4 w-4' }: { tone: HealthTone; className?: string }) {
  const Icon = tone === 'good' ? CheckCircle2 : tone === 'critical' ? XCircle : tone === 'warning' ? AlertTriangle : Info;
  return <Icon className={`${className} ${toneClasses[tone].icon}`} aria-hidden="true" />;
}

function formatDate(value?: string) {
  if (!value) return 'Not available';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? 'Not available' : date.toLocaleString();
}

function formatDuration(seconds?: number | null) {
  if (seconds === undefined || seconds === null) return '—';
  if (seconds < 60) return `${String(Math.round(seconds))}s`;
  const minutes = Math.floor(seconds / 60);
  return `${String(minutes)}m ${String(Math.round(seconds % 60))}s`;
}

function Panel({ children, className = '', ...props }: HTMLAttributes<HTMLElement>) {
  return <section className={`rounded-2xl border border-white/10 bg-white/[0.045] shadow-lg shadow-black/5 ${className}`} {...props}>{children}</section>;
}

function PanelHeading({ icon: Icon, title, description, action }: { icon: LucideIcon; title: string; description?: string; action?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4 border-b border-white/10 px-5 py-4 sm:px-6">
      <div className="flex items-start gap-3">
        <div className="mt-0.5 rounded-lg bg-cyan-400/10 p-2 text-cyan-300"><Icon className="h-4 w-4" aria-hidden="true" /></div>
        <div><h2 className="font-semibold text-white">{title}</h2>{description && <p className="mt-1 max-w-2xl text-sm leading-5 text-slate-400">{description}</p>}</div>
      </div>
      {action}
    </div>
  );
}

export function EmailDelivery() {
  const [readiness, setReadiness] = useState<EmailReadinessReport | null>(null);
  const [delivery, setDelivery] = useState<SignInDeliveryAdminReport | null>(null);
  const [address, setAddress] = useState('');
  const [probeMessage, setProbeMessage] = useState('');
  const [historySearch, setHistorySearch] = useState('');
  const [credentialField, setCredentialField] = useState<ProviderCredentialField>('mailgun-api-key');
  const [credentialValue, setCredentialValue] = useState('');
  const [credentialMessage, setCredentialMessage] = useState('');
  const [credentialError, setCredentialError] = useState('');
  const [savingCredential, setSavingCredential] = useState(false);
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

  const providerChecks = useMemo(() => readiness?.Providers ?? [], [readiness]);
  const failedChecks = providerChecks.filter((provider) => provider.Status === 'fail').length + (readiness?.DMARC.Status === 'fail' ? 1 : 0);
  const warningChecks = providerChecks.filter((provider) => provider.Status === 'warn').length + (readiness?.DMARC.Status === 'warn' ? 1 : 0);
  const hasReadyProvider = Boolean(delivery?.chosen_provider) || Boolean(delivery?.providers?.some((provider) => provider.state === 'ready'));
  const overallTone: HealthTone = failedChecks > 0 || (Boolean(delivery?.providers?.length) && !hasReadyProvider) ? 'critical' : warningChecks > 0 ? 'warning' : readiness && delivery ? 'good' : 'neutral';
  const overallLabel = overallTone === 'good' ? 'Healthy' : overallTone === 'warning' ? 'Needs attention' : overallTone === 'critical' ? 'Action required' : 'Checking status';
  const delivery24h = delivery?.delivery_24h;
  const queuedCount = Object.entries(delivery?.outbox ?? {}).reduce((total, [state, count]) => state === 'pending' || state === 'queued' ? total + count : total, 0);

  const submitProbe = async (event: FormEvent) => {
    event.preventDefault();
    setProbeMessage('');
    try {
      const result = await sendDeliveryProbe(address);
      setProbeMessage(`Test queued successfully. Request ID: ${result.request_id}`);
    } catch (error) {
      setProbeMessage(error instanceof Error ? error.message : 'Unable to queue the delivery test.');
    }
  };

  const submitCredential = async (event: FormEvent) => {
    event.preventDefault();
    setCredentialMessage('');
    setCredentialError('');
    setSavingCredential(true);
    try {
      const result = await saveProviderCredential(credentialField, credentialValue);
      setCredentialValue('');
      setCredentialMessage(`${result.field} was stored securely. Provider verification has been refreshed.`);
      await refresh();
    } catch (error) {
      setCredentialError(error instanceof Error ? error.message : 'Unable to store the provider credential.');
    } finally {
      setSavingCredential(false);
    }
  };

  const credentialLabel = credentialField === 'mailgun-api-key' ? 'Mailgun API key' : credentialField === 'smtp-password' ? 'SMTP password' : credentialField === 'sendgrid-api-key' ? 'SendGrid API key' : 'SendGrid webhook public key';

  return (
    <main className="mx-auto max-w-6xl px-4 py-8 text-slate-100 sm:px-6 lg:py-10" aria-labelledby="email-delivery-title">
      <PageHeader title="Email delivery" description="Monitor provider authorization, queue health, and sign-in email delivery from one place." icon={Mail} iconBgClass="bg-cyan-400/10" iconColorClass="text-cyan-300" actions={<Button variant="outline" size="sm" onClick={() => { void refresh(); }} disabled={loading}><RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />{loading ? 'Refreshing' : 'Refresh'}</Button>} />

      <section className={`mb-6 rounded-2xl border ${toneClasses[overallTone].border} ${toneClasses[overallTone].surface} p-5 sm:p-6`} aria-label="Email delivery overview">
        <div className="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex items-start gap-4"><div className="rounded-xl bg-slate-950/30 p-3"><StatusIcon tone={overallTone} className="h-7 w-7" /></div><div><div className="flex flex-wrap items-center gap-3"><h2 className="text-xl font-semibold text-white">{overallLabel}</h2><span className={`rounded-full border px-2.5 py-1 text-xs font-semibold ${toneClasses[overallTone].border} ${toneClasses[overallTone].text}`}>{readiness?.Domain ?? 'Email system'}</span></div><p className="mt-2 max-w-2xl text-sm leading-6 text-slate-300">{overallTone === 'good' ? 'At least one provider is authorized and the delivery system is reporting normally.' : overallTone === 'critical' ? 'Email may not be able to send. Review the blocked checks and provider remedy below before testing.' : overallTone === 'warning' ? 'The system is operating, but one or more checks need attention.' : 'Loading provider and delivery telemetry.'}</p></div></div>
          <div className="grid grid-cols-2 gap-x-8 gap-y-3 border-t border-white/10 pt-4 text-sm lg:border-l lg:border-t-0 lg:pl-8 lg:pt-0"><div><p className="text-slate-400">Selected provider</p><p className="mt-1 font-medium text-white">{delivery?.chosen_provider || 'None'}</p></div><div><p className="text-slate-400">Last checked</p><p className="mt-1 font-medium text-white">{formatDate(readiness?.CheckedAt)}</p></div><div><p className="text-slate-400">Waiting now</p><p className="mt-1 font-medium text-white">{queuedCount}</p></div><div><p className="text-slate-400">Window</p><p className="mt-1 font-medium text-white">Last {delivery?.window_hours ?? 24} hours</p></div></div>
        </div>
      </section>

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label="Delivery summary">
        {[{ label: 'Accepted', value: delivery24h?.sent ?? '—', detail: 'provider handoffs', icon: Send, tone: 'good' as HealthTone }, { label: 'Delivered', value: delivery24h?.delivered ?? '—', detail: 'recipient-server accepts', icon: CheckCircle2, tone: 'good' as HealthTone }, { label: 'Failed', value: delivery24h?.failed ?? '—', detail: delivery24h?.last_error || 'no recent failure recorded', icon: XCircle, tone: delivery24h?.failed ? 'critical' as HealthTone : 'neutral' as HealthTone }, { label: 'Median acceptance', value: formatDuration(delivery?.queue_health?.median_acceptance_seconds_24h), detail: 'over the active window', icon: Activity, tone: 'neutral' as HealthTone }].map(({ label, value, detail, icon: Icon, tone }) => <div key={label} className="rounded-xl border border-white/10 bg-slate-950/25 p-4"><div className="flex items-center justify-between"><span className="text-sm text-slate-400">{label}</span><Icon className={`h-4 w-4 ${toneClasses[tone].icon}`} aria-hidden="true" /></div><p className="mt-3 text-2xl font-semibold text-white">{value}</p><p className="mt-1 truncate text-xs text-slate-500" title={detail}>{detail}</p></div>)}
      </div>

      <Panel className="mb-6" aria-label="Provider credential setup">
        <PanelHeading icon={KeyRound} title="Provider credentials" description="Store a provider credential without putting it in branding settings, deployment files, URLs, or logs." />
        <div className="grid gap-6 p-5 sm:p-6 lg:grid-cols-[1.15fr_0.85fr]">
          <div>
            <div className="rounded-xl border border-cyan-400/15 bg-cyan-400/5 p-4 text-sm leading-6 text-slate-300">
              <p className="font-medium text-cyan-100">Keep two protected copies</p>
              <p className="mt-2">Local onboarding and this production portal use separate credential authorities. For transactional sign-in email, use the Mailgun API key; the legacy SMTP password is only for legacy SMTP and feedback notifications. Keep a recovery copy in your local authority for development and VPS recovery, then store the production copy here. Saving locally does not update production, and saving here does not update your local machine.</p>
              <p className="mt-2 text-slate-400">The provider dashboard or password manager remains the ultimate recovery source. If the VPS is lost, redeploy from the local copy or provision a fresh provider credential; the secret is never shown back on this page.</p>
            </div>
            <form className="mt-5 space-y-4" onSubmit={(event) => { void submitCredential(event); }}>
              <div>
                <label className="text-sm font-medium text-white" htmlFor="provider-credential-field">Credential to store</label>
                <select id="provider-credential-field" className="mt-2 w-full rounded-lg border border-white/15 bg-slate-950 px-3 py-2.5 text-sm text-white outline-none focus:border-cyan-400/60 focus:ring-2 focus:ring-cyan-400/20" value={credentialField} onChange={(event) => { setCredentialField(event.target.value as ProviderCredentialField); }}>
                  <option value="mailgun-api-key">Mailgun API key</option>
                  <option value="smtp-password">Legacy SMTP password</option>
                  <option value="sendgrid-api-key">SendGrid API key</option>
                  <option value="sendgrid-webhook-public-key">SendGrid webhook public key</option>
                </select>
              </div>
              <div>
                <label className="text-sm font-medium text-white" htmlFor="provider-credential-value">{credentialLabel}</label>
                <input id="provider-credential-value" className="mt-2 w-full rounded-lg border border-white/15 bg-slate-950 px-3 py-2.5 text-sm text-white outline-none placeholder:text-slate-600 focus:border-cyan-400/60 focus:ring-2 focus:ring-cyan-400/20" type="password" autoComplete="new-password" required value={credentialValue} onChange={(event) => { setCredentialValue(event.target.value); }} placeholder="Paste the value once; it will not be displayed again" />
              </div>
              <div className="flex flex-wrap items-center gap-3">
                <Button type="submit" disabled={savingCredential || !credentialValue.trim()}><KeyRound className="mr-2 h-4 w-4" />{savingCredential ? 'Storing securely…' : 'Store and verify'}</Button>
                {credentialMessage && <span className="text-sm text-emerald-200" role="status">{credentialMessage}</span>}
                {credentialError && <span className="text-sm text-rose-200" role="alert">{credentialError}</span>}
              </div>
            </form>
          </div>
          <div className="rounded-xl border border-white/10 bg-slate-950/25 p-4 text-sm leading-6 text-slate-400">
            <p className="font-medium text-white">Recommended recovery workflow</p>
            <ol className="mt-3 space-y-3">
              <li><span className="mr-2 text-cyan-300">1.</span>Keep the provider credential in your password manager or provider account.</li>
              <li><span className="mr-2 text-cyan-300">2.</span>Provision the same value through local <code className="rounded bg-slate-900 px-1.5 py-0.5 text-xs text-slate-300">vrooli-onboarding</code> when developing or preparing recovery.</li>
              <li><span className="mr-2 text-cyan-300">3.</span>Use this form for the production authority, then verify and send a real probe.</li>
              <li><span className="mr-2 text-cyan-300">4.</span>Rotate at the provider first, update both authorities, verify, and only then revoke the old credential.</li>
            </ol>
          </div>
        </div>
      </Panel>

      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <Panel aria-label="Provider authorization"><PanelHeading icon={ShieldCheck} title="Authorization checks" description="A provider can send only when its credential and sending-domain requirements are both usable." /><div className="space-y-3 p-5 sm:p-6">{providerChecks.map((provider) => { const tone = readinessTone(provider.Status); return <div key={provider.Provider} className="rounded-xl border border-white/10 bg-slate-950/25 p-4"><div className="flex items-start justify-between gap-4"><div className="flex items-center gap-2"><StatusIcon tone={tone} /><span className="font-medium text-white">{provider.Provider}</span></div><span className={`text-xs font-semibold uppercase tracking-wide ${toneClasses[tone].text}`}>{provider.Status}</span></div><p className="mt-2 pl-6 text-sm leading-5 text-slate-400">{provider.Detail}</p>{provider.Record && <p className="mt-3 rounded-lg bg-slate-900/70 px-3 py-2 font-mono text-xs leading-5 text-slate-400">{provider.Record}</p>}{provider.Cost !== undefined && <p className="mt-2 pl-6 text-xs text-slate-500">SPF lookup cost: {provider.Cost}</p>}</div>; })}{readiness && <div className="rounded-xl border border-white/10 bg-slate-950/25 p-4"><div className="flex items-center justify-between gap-4"><div className="flex items-center gap-2"><StatusIcon tone={readinessTone(readiness.DMARC.Status)} /><span className="font-medium text-white">DMARC</span></div><span className={`text-xs font-semibold uppercase tracking-wide ${toneClasses[readinessTone(readiness.DMARC.Status)].text}`}>{readiness.DMARC.Status}</span></div><p className="mt-2 pl-6 text-sm leading-5 text-slate-400">{readiness.DMARC.Detail}</p></div>}{!readiness && <p className="text-sm text-slate-400">Authorization checks are loading.</p>}</div></Panel>

        <Panel aria-label="Provider routing"><PanelHeading icon={Server} title="Provider routing" description="The router chooses the first eligible provider in the current cost and capacity ladder." /><div className="p-5 sm:p-6">{delivery?.providers?.map((provider) => { const tone = readinessTone(provider.state); return <article key={provider.id} className="mb-3 rounded-xl border border-white/10 bg-slate-950/25 p-4 last:mb-0"><div className="flex items-center justify-between gap-3"><span className="font-medium text-white">{provider.id}</span><span className={`flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide ${toneClasses[tone].text}`}><StatusIcon tone={tone} className="h-3.5 w-3.5" />{provider.state.replace('_', ' ')}</span></div><div className="mt-3 grid gap-2 text-xs text-slate-400 sm:grid-cols-2"><p><span className="text-slate-500">Credential</span><br />{provider.credential.detail}</p><p><span className="text-slate-500">DNS</span><br />{provider.dns.detail}</p></div>{provider.remedy && <p className="mt-3 flex gap-2 rounded-lg bg-amber-400/10 px-3 py-2 text-sm leading-5 text-amber-200"><AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />{provider.remedy}</p>}</article>; })}{!delivery?.providers?.length && <p className="text-sm text-slate-400">Provider registry data is not available.</p>}<div className="mt-5 border-t border-white/10 pt-4"><div className="flex items-center justify-between gap-3"><h3 className="text-sm font-medium text-white">Routing ladder</h3><span className="text-xs text-slate-500">{delivery?.chosen_provider ? `Selected: ${delivery.chosen_provider}` : 'No provider selected'}</span></div><div className="mt-3 space-y-2">{(delivery?.routing ?? []).map((route) => <div key={route.provider_id} className="flex items-center justify-between rounded-lg bg-slate-950/60 px-3 py-2 text-sm"><span className="text-slate-300">{`${route.provider_id} · rank ${String(route.cost_rank)}`}</span><span className={route.eligible ? 'text-emerald-300' : 'text-amber-300'}>{route.eligible ? 'Eligible' : route.skip_reason ?? 'Skipped'}</span></div>)}</div></div></div></Panel>
      </div>

      <div className="mt-6 grid gap-6 xl:grid-cols-[0.9fr_1.1fr]"><Panel aria-label="Queue health"><PanelHeading icon={Clock3} title="Queue health" description="A queued message has been accepted by the application, but not yet by a provider." /><div className="grid gap-4 p-5 sm:grid-cols-2 sm:p-6 xl:grid-cols-1"><div className="rounded-xl bg-slate-950/30 p-4"><p className="text-2xl font-semibold text-white">{`Oldest waiting message: ${formatDuration(delivery?.queue_health?.oldest_wait_seconds)}`}</p><p className="mt-1 text-xs text-slate-500">Long waits usually indicate provider eligibility or capacity issues.</p></div><div className="rounded-xl bg-slate-950/30 p-4"><p className="text-sm text-slate-400">Outbox by state</p><div className="mt-3 space-y-2">{Object.entries(delivery?.outbox ?? {}).map(([state, count]) => <div key={state} className="flex items-center justify-between text-sm"><span className="capitalize text-slate-300">{state.replace('_', ' ')}</span><span className="font-medium text-white">{String(count)}</span></div>)}{!delivery?.outbox && <p className="text-sm text-slate-500">No outbox rows reported.</p>}</div></div><div className="rounded-xl bg-slate-950/30 p-4"><p className="text-sm text-slate-400">Waiting by priority</p><div className="mt-3 space-y-2">{(delivery?.queue_health?.by_priority ?? []).map((item) => <div key={item.priority} className="flex items-center justify-between text-sm"><span className="text-slate-300">{`Priority ${String(item.priority)}`}</span><span className="text-slate-400">{`${String(item.depth)} waiting`}</span></div>)}</div></div><div className="rounded-xl bg-slate-950/30 p-4"><p className="text-sm text-slate-400">Quota windows</p><div className="mt-3 space-y-2">{(delivery?.quotas ?? []).map((quota) => <p key={`${quota.provider_id}-${quota.window}`} className="text-sm text-slate-300">{`${quota.provider_id} · ${quota.window}: ${String(quota.used)}/${String(quota.ceiling)}`}</p>)}{!delivery?.quotas?.length && <p className="text-sm text-slate-500">No active quota windows.</p>}</div></div></div></Panel>

        <Panel aria-label="Delivery probe"><PanelHeading icon={Send} title="Test an external delivery" description="Send a real test message through the same durable queue used by sign-in email." /><div className="p-5 sm:p-6"><div className="mb-5 rounded-xl border border-cyan-400/15 bg-cyan-400/5 p-4"><p className="flex items-center gap-2 text-sm font-medium text-cyan-100"><Info className="h-4 w-4" />Recommended test procedure</p><ol className="mt-3 space-y-2 text-sm leading-5 text-slate-300"><li><span className="mr-2 text-cyan-300">1.</span>Use an inbox you can check immediately.</li><li><span className="mr-2 text-cyan-300">2.</span>Queue the test below and note the request ID.</li><li><span className="mr-2 text-cyan-300">3.</span>Confirm the message arrives, then check its status in history.</li></ol></div><form className="flex flex-col gap-3 sm:flex-row" onSubmit={(event) => { void submitProbe(event); }}><label className="sr-only" htmlFor="delivery-probe-address">Test recipient address</label><input id="delivery-probe-address" className="min-w-0 flex-1 rounded-lg border border-white/15 bg-slate-950 px-3 py-2.5 text-sm text-white outline-none placeholder:text-slate-600 focus:border-cyan-400/60 focus:ring-2 focus:ring-cyan-400/20" type="email" required value={address} onChange={(event) => { setAddress(event.target.value); }} placeholder="you@example.net" /><Button type="submit" disabled={loading}><Send className="mr-2 h-4 w-4" />Queue test</Button></form>{probeMessage && <p className="mt-3 rounded-lg bg-emerald-400/10 px-3 py-2 text-sm leading-5 text-emerald-200" role="status">{probeMessage}</p>}<p className="mt-4 text-xs leading-5 text-slate-500">This queues a message; it does not guarantee recipient delivery. Provider acceptance and recipient-server delivery are tracked separately.</p></div></Panel>
      </div>

      <Panel className="mt-6" aria-label="Message history"><PanelHeading icon={Activity} title="Message history" description="Inspect recent requests, provider attempts, and the reason for any failure." action={<form className="flex w-full gap-2 sm:w-auto" onSubmit={(event) => { event.preventDefault(); void refresh(historySearch); return; }}><label className="sr-only" htmlFor="message-history-search">Search message history by recipient</label><div className="relative min-w-0 flex-1 sm:w-64"><Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-500" /><input id="message-history-search" className="w-full rounded-lg border border-white/15 bg-slate-950 py-2 pl-9 pr-3 text-sm text-white outline-none placeholder:text-slate-600 focus:border-cyan-400/60" value={historySearch} onChange={(event) => { setHistorySearch(event.target.value); }} placeholder="Search recipient" /></div><Button type="submit" variant="outline" size="sm">Search</Button></form>} /><div className="p-5 sm:p-6">{(delivery?.history ?? []).map((item) => <article key={item.id} className="mb-3 rounded-xl border border-white/10 bg-slate-950/25 p-4 last:mb-0"><div className="flex flex-wrap items-start justify-between gap-3"><div><p className="font-medium capitalize text-white">{item.purpose.replace('_', ' ')} <span className="font-normal text-slate-500">to</span> {item.recipient}</p><p className="mt-1 text-xs text-slate-500">Requested {formatDate(item.requested_at)}{item.provider_id ? ` · ${item.provider_id}` : ''}</p></div><span className={`rounded-full px-2.5 py-1 text-xs font-semibold capitalize ${toneClasses[readinessTone(item.status)].surface} ${toneClasses[readinessTone(item.status)].text}`}>{item.status.replace('_', ' ')}</span></div>{item.last_error && <p className="mt-3 rounded-lg bg-rose-400/10 px-3 py-2 text-sm leading-5 text-rose-200">{item.last_error}</p>}{item.attempts?.length ? <details className="mt-3"><summary className="cursor-pointer text-xs text-slate-400 hover:text-slate-200">View {item.attempts.length} provider attempt{item.attempts.length === 1 ? '' : 's'}</summary><div className="mt-2 space-y-1 border-l border-white/10 pl-3 text-xs text-slate-500">{item.attempts.map((attempt) => <p key={`${item.id}-${String(attempt.attempt)}`}>Attempt {String(attempt.attempt)}: {attempt.provider_id} · {attempt.outcome}{attempt.diagnostic ? ` · ${attempt.diagnostic}` : ''}</p>)}</div></details> : null}</article>)}{!delivery?.history?.length && <div className="py-8 text-center"><Mail className="mx-auto h-8 w-8 text-slate-600" /><p className="mt-3 text-sm text-slate-400">No messages match the current search.</p></div>}</div></Panel>

      <p className="mt-5 flex items-center justify-center gap-2 text-center text-xs text-slate-500"><Info className="h-3.5 w-3.5" />Provider acceptance means the provider took responsibility; it is not the same as inbox delivery.</p>
    </main>
  );
}
