import { useEffect, useState } from 'react';
import { KeyRound, Loader2, RefreshCw, ShieldCheck, Trash2, XCircle } from 'lucide-react';
import type { RemoteAPIKey, RemoteProfile, RemoteStripeSettings } from '../../../shared/api';
import {
  createRemoteAPIKeyAdmin,
  deleteRemoteAPIKeyAdmin,
  getRemoteStripeSettingsAdmin,
  listRemoteAPIKeysAdmin,
  setRemoteAPIKeyActiveAdmin,
  testRemoteAPIKeyAdmin,
  updateRemoteStripeSettingsAdmin,
} from '../../../shared/api';
import { PROVIDER_OPTIONS } from '../../../shared/api/credits';
import { Button } from '../../../shared/ui/button';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../../shared/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../../shared/ui/select';
import { Switch } from '../../../shared/ui/switch';
import { inputClassName } from '../components/formFieldClasses';
import { SetupTask, type SetupTaskStatus } from '@vrooli/react-component-library/SetupTask/0';

type SettingsTab = 'credentials' | 'payments';

interface RemoteProfileOwnerSettingsDialogProps {
  profile: RemoteProfile | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const emptyStripeForm = { publishableKey: '', secretKey: '', webhookSecret: '', dashboardUrl: '' };

export function RemoteProfileOwnerSettingsDialog({ profile, open, onOpenChange }: RemoteProfileOwnerSettingsDialogProps) {
  const [tab, setTab] = useState<SettingsTab>('credentials');
  const [keys, setKeys] = useState<RemoteAPIKey[]>([]);
  const [stripe, setStripe] = useState<RemoteStripeSettings | null>(null);
  const [stripeForm, setStripeForm] = useState(emptyStripeForm);
  const [provider, setProvider] = useState<string>('');
  const [keyValue, setKeyValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const remoteTarget = profile?.label?.trim() || profile?.tag || 'remote instance';
  const providerKeysStatus: SetupTaskStatus = keys.length > 0 ? 'unknown' : 'needs_attention';
  const stripeReadyToVerify = Boolean(stripe?.publishable_key_set && stripe?.secret_key_set && stripe?.webhook_secret_set);
  const stripeStatus: SetupTaskStatus = stripeReadyToVerify ? 'unknown' : 'needs_attention';

  const load = async () => {
    if (!profile) return;
    setLoading(true);
    setError(null);
    try {
      const [keyResponse, stripeResponse] = await Promise.all([
        listRemoteAPIKeysAdmin(profile.id),
        getRemoteStripeSettingsAdmin(profile.id),
      ]);
      setKeys(keyResponse.keys);
      setStripe(stripeResponse);
      setStripeForm((current) => ({ ...current, dashboardUrl: stripeResponse.dashboard_url ?? '' }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to load remote owner settings');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!open) return;
    setTab('credentials');
    setProvider('');
    setKeyValue('');
    setNotice(null);
    void load();
    // The profile ID and open state identify the remote target for this dialog.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, profile?.id]);

  const run = async (operation: string, action: () => Promise<void>) => {
    setBusy(operation);
    setError(null);
    setNotice(null);
    try {
      await action();
      setNotice('Remote settings updated.');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Remote settings update failed');
    } finally {
      setBusy(null);
    }
  };

  const addKey = async () => {
    if (!profile || !provider || !keyValue.trim()) {
      setError('Choose a provider and enter its API key.');
      return;
    }
    await run('add-key', async () => {
      const key = await createRemoteAPIKeyAdmin(profile.id, provider, keyValue.trim());
      setKeys((current) => [...current.filter((item) => item.provider !== key.provider), key]);
      setProvider('');
      setKeyValue('');
    });
  };

  const saveStripe = async () => {
    if (!profile) return;
    const payload = Object.fromEntries(
      Object.entries({
        publishable_key: stripeForm.publishableKey,
        secret_key: stripeForm.secretKey,
        webhook_secret: stripeForm.webhookSecret,
        dashboard_url: stripeForm.dashboardUrl,
      }).filter(([, value]) => value.trim().length > 0),
    );
    if (Object.keys(payload).length === 0) {
      setError('Enter at least one Stripe setting before saving.');
      return;
    }
    await run('save-stripe', async () => {
      const updated = await updateRemoteStripeSettingsAdmin(profile.id, payload);
      setStripe(updated);
      setStripeForm((current) => ({ ...current, publishableKey: '', secretKey: '', webhookSecret: '', dashboardUrl: updated.dashboard_url ?? '' }));
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2"><ShieldCheck className="h-5 w-5 text-sky-400" />Owner settings</DialogTitle>
          <DialogDescription>
            Securely manage redacted provider credentials and Stripe configuration on {profile?.label?.trim() || profile?.tag || 'the remote instance'}.
            Secret values are sent only for the save operation and are never read back.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-5 md:grid-cols-[170px_1fr]">
          <nav className="flex gap-2 md:flex-col" aria-label="Remote owner settings sections">
            <button type="button" onClick={() => setTab('credentials')} className={`rounded-lg px-3 py-2 text-left text-sm ${tab === 'credentials' ? 'bg-sky-500/15 text-sky-200' : 'text-slate-400 hover:bg-slate-800'}`}>
              <KeyRound className="mr-2 inline h-4 w-4" />Provider keys
            </button>
            <button type="button" onClick={() => setTab('payments')} className={`rounded-lg px-3 py-2 text-left text-sm ${tab === 'payments' ? 'bg-sky-500/15 text-sky-200' : 'text-slate-400 hover:bg-slate-800'}`}>
              Payments
            </button>
          </nav>

          <div className="min-w-0 space-y-4">
            {loading && <div className="flex items-center gap-2 rounded-lg border border-slate-700 bg-slate-950/40 p-4 text-sm text-slate-400"><Loader2 className="h-4 w-4 animate-spin" />Loading remote settings…</div>}
            {error && <div className="flex items-start gap-2 rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 text-sm text-rose-300"><XCircle className="mt-0.5 h-4 w-4 shrink-0" />{error}</div>}
            {notice && <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3 text-sm text-emerald-300">{notice}</div>}

            {!loading && tab === 'credentials' && (
              <SetupTask
                title="Remote provider keys"
                purpose="Manage the provider identities used by this remote owner without exposing secret values."
                target={remoteTarget}
                account="Remote owner"
                status={providerKeysStatus}
                statusLabel={keys.length > 0 ? 'Stored · test each key' : 'No keys configured'}
                guidance="A stored key is not treated as verified. Test it after saving, and keep only the providers this remote owner should be allowed to use."
                testId="remote-provider-keys-task"
              >
              <section className="space-y-4" aria-label="Remote provider keys">
                <div>
                  <p className="text-sm text-slate-400">Only provider, status, and a safe key hint are displayed.</p>
                </div>
                <div className="space-y-2">
                  {keys.length === 0 ? <p className="rounded-lg border border-dashed border-slate-700 p-4 text-sm text-slate-400">No provider keys configured.</p> : keys.map((item) => (
                    <div key={item.id || item.provider} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-slate-700 bg-slate-950/30 p-3">
                      <div><p className="font-medium text-slate-200">{item.provider}</p><p className="font-mono text-xs text-slate-500">{item.key_hint || 'redacted'} · {item.is_active ? 'Active' : 'Inactive'}</p></div>
                      <div className="flex items-center gap-2">
                        <Button variant="outline" size="sm" disabled={busy !== null} onClick={() => { void run(`test-${item.provider}`, async () => { const result = await testRemoteAPIKeyAdmin(profile!.id, item.provider); if (!result.success) throw new Error(result.message); }); }}>Test</Button>
                        <Switch checked={item.is_active} disabled={busy !== null} aria-label={`Enable ${item.provider}`} onCheckedChange={(active) => { void run(`toggle-${item.provider}`, async () => { await setRemoteAPIKeyActiveAdmin(profile!.id, item.provider, active); setKeys((current) => current.map((key) => key.provider === item.provider ? { ...key, is_active: active } : key)); }); }} />
                        <Button variant="ghost" size="sm" disabled={busy !== null} aria-label={`Delete ${item.provider}`} onClick={() => { if (confirm(`Delete the ${item.provider} key on the remote instance?`)) void run(`delete-${item.provider}`, async () => { await deleteRemoteAPIKeyAdmin(profile!.id, item.provider); setKeys((current) => current.filter((key) => key.provider !== item.provider)); }); }}><Trash2 className="h-4 w-4 text-rose-400" /></Button>
                      </div>
                    </div>
                  ))}
                </div>
                <div className="rounded-lg border border-slate-700 bg-slate-950/30 p-4 space-y-3">
                  <h4 className="text-sm font-medium text-slate-200">Add provider key</h4>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <Select value={provider} onValueChange={setProvider}>
                      <SelectTrigger aria-label="Provider"><SelectValue placeholder="Choose provider" /></SelectTrigger>
                      <SelectContent>{PROVIDER_OPTIONS.filter((option) => !keys.some((key) => key.provider === option.value)).map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}</SelectContent>
                    </Select>
                    <input className={inputClassName} type="password" value={keyValue} onChange={(event) => setKeyValue(event.target.value)} placeholder="API key" aria-label="API key" autoComplete="new-password" />
                  </div>
                  <Button size="sm" onClick={() => { void addKey(); }} disabled={busy !== null}>Save provider key</Button>
                </div>
              </section>
              </SetupTask>
            )}

            {!loading && tab === 'payments' && (
              <SetupTask
                title="Remote Stripe settings"
                purpose="Configure payment credentials for the selected remote owner without exposing secret values."
                target={remoteTarget}
                account="Stripe"
                status={stripeStatus}
                statusLabel={stripeReadyToVerify ? 'Stored · payment verification required' : 'Payment credentials needed'}
                guidance="Stored Stripe values remain redacted. Saving configuration does not prove payment operations; verify the remote payment flow before treating it as ready."
                testId="remote-stripe-task"
              >
              <section className="space-y-4" aria-label="Remote Stripe settings">
                <div><p className="text-sm text-slate-400">Existing secrets remain redacted. Leave secret fields blank to keep them unchanged.</p></div>
                <div className="grid gap-3 sm:grid-cols-2 text-sm">
                  <div className="rounded-lg border border-slate-700 p-3"><span className="text-slate-500">Source</span><p className="text-slate-200">{stripe?.source || 'Unknown'}</p></div>
                  <div className="rounded-lg border border-slate-700 p-3"><span className="text-slate-500">Publishable key</span><p className="font-mono text-slate-200">{stripe?.publishable_key_preview || (stripe?.publishable_key_set ? 'Configured' : 'Not configured')}</p></div>
                  <div className="rounded-lg border border-slate-700 p-3"><span className="text-slate-500">Secret key</span><p className="text-slate-200">{stripe?.secret_key_set ? 'Configured' : 'Not configured'}</p></div>
                  <div className="rounded-lg border border-slate-700 p-3"><span className="text-slate-500">Webhook secret</span><p className="text-slate-200">{stripe?.webhook_secret_set ? 'Configured' : 'Not configured'}</p></div>
                </div>
                <div className="space-y-3 rounded-lg border border-slate-700 bg-slate-950/30 p-4">
                  <input className={inputClassName} type="text" value={stripeForm.publishableKey} onChange={(event) => setStripeForm((current) => ({ ...current, publishableKey: event.target.value }))} placeholder="Publishable key (optional)" aria-label="Publishable key" />
                  <input className={inputClassName} type="password" value={stripeForm.secretKey} onChange={(event) => setStripeForm((current) => ({ ...current, secretKey: event.target.value }))} placeholder="Secret key (optional)" aria-label="Secret key" autoComplete="new-password" />
                  <input className={inputClassName} type="password" value={stripeForm.webhookSecret} onChange={(event) => setStripeForm((current) => ({ ...current, webhookSecret: event.target.value }))} placeholder="Webhook secret (optional)" aria-label="Webhook secret" autoComplete="new-password" />
                  <input className={inputClassName} type="url" value={stripeForm.dashboardUrl} onChange={(event) => setStripeForm((current) => ({ ...current, dashboardUrl: event.target.value }))} placeholder="Stripe dashboard URL" aria-label="Stripe dashboard URL" />
                  <Button onClick={() => { void saveStripe(); }} disabled={busy !== null}>Save Stripe settings</Button>
                </div>
              </section>
              </SetupTask>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" size="sm" onClick={() => { void load(); }} disabled={loading || busy !== null}><RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />Refresh</Button>
          <DialogClose asChild><Button variant="ghost">Close</Button></DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
