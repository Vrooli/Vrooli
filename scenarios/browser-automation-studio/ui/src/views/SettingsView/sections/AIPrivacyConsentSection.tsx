import { useCallback, useEffect, useState } from 'react';
import { ShieldCheck } from 'lucide-react';
import { AI_CONSENT_RECEIPT_KEY, AI_CONSENT_SOURCE_KEY, createRemoteAIConsent, listRemoteAIConsent, revokeRemoteAIConsent, type RemoteAIConsentGrant, type RemoteAISource } from '@/api/aiPrivacyConsent';

const sourceName = (source: RemoteAISource) => source === 'own-key' ? 'your configured OpenRouter account' : 'Vrooli AI credits';

export function AIPrivacyConsentSection() {
  const [source, setSource] = useState<RemoteAISource>('own-key');
  const [grants, setGrants] = useState<RemoteAIConsentGrant[]>([]);
  const [activeReceipt, setActiveReceipt] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const refresh = useCallback(async () => {
    try {
      const result = await listRemoteAIConsent();
      setGrants(result.grants ?? []);
      setError('');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Consent status is unavailable. Sign in and try again.');
    }
  }, []);

  useEffect(() => {
    void refresh();
    setActiveReceipt(Boolean(sessionStorage.getItem(AI_CONSENT_RECEIPT_KEY)));
  }, [refresh]);

  const allow = async () => {
    setBusy(true);
    setError('');
    try {
      const result = await createRemoteAIConsent(source);
      sessionStorage.setItem(AI_CONSENT_RECEIPT_KEY, result.receipt);
      sessionStorage.setItem(AI_CONSENT_SOURCE_KEY, source);
      setActiveReceipt(true);
      await refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Consent could not be saved.');
    } finally {
      setBusy(false);
    }
  };

  const revoke = async (grant: RemoteAIConsentGrant) => {
    setBusy(true);
    setError('');
    try {
      await revokeRemoteAIConsent(grant.id);
      if (sessionStorage.getItem(AI_CONSENT_SOURCE_KEY) === grant.source) {
        sessionStorage.removeItem(AI_CONSENT_RECEIPT_KEY);
        sessionStorage.removeItem(AI_CONSENT_SOURCE_KEY);
        setActiveReceipt(false);
      }
      await refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Consent could not be revoked.');
    } finally {
      setBusy(false);
    }
  };

  const activeGrants = grants.filter((grant) => !grant.revoked_at && Date.parse(grant.expires_at) > Date.now());
  return (
    <section data-testid="ai-privacy-consent" aria-labelledby="ai-privacy-consent-title" className="rounded-lg border border-gray-700 bg-gray-800/60 p-4">
      <div className="flex items-start gap-3">
        <ShieldCheck size={21} className="mt-0.5 shrink-0 text-purple-300" aria-hidden />
        <div className="min-w-0 flex-1">
          <h3 id="ai-privacy-consent-title" className="text-sm font-semibold text-surface">Confidential data sharing</h3>
          <p className="mt-1 text-xs leading-5 text-gray-300">
            Choosing a provider or saving a key does not grant permission. If allowed, workflow prompts and relevant browser or session context may be sent to the selected remote source and follow its data policies. Secret material is always blocked from remote providers.
          </p>
          <label htmlFor="ai-privacy-consent-source" className="mt-3 block text-xs font-medium text-surface">Remote source</label>
          <select id="ai-privacy-consent-source" data-testid="ai-privacy-consent-source" value={source} onChange={(event) => setSource(event.target.value as RemoteAISource)} className="mt-1 min-h-10 w-full rounded border border-gray-600 bg-gray-900 px-2 text-sm text-surface">
            <option value="own-key">{sourceName('own-key')}</option>
            <option value="vrooli-credits">{sourceName('vrooli-credits')}</option>
          </select>
          <div className="mt-2 flex flex-wrap items-center gap-3">
            <button type="button" data-testid="ai-privacy-consent-allow" disabled={busy} onClick={() => void allow()} className="min-h-10 rounded bg-purple-600 px-3 text-sm font-semibold text-white disabled:opacity-60">
              {busy ? 'Saving…' : `Allow confidential context for ${sourceName(source)}`}
            </button>
            <span role="status" className="text-xs text-gray-300">{activeReceipt ? 'A consent receipt is active for this browser tab.' : 'No consent receipt is active in this browser tab.'}</span>
          </div>
          {activeGrants.length > 0 && <ul className="mt-3 space-y-2 border-t border-gray-700 pt-3" aria-label="Active consent grants">
            {activeGrants.map((grant) => <li key={grant.id} className="flex flex-wrap items-center justify-between gap-2 text-xs text-gray-300">
              <span>Allowed for {sourceName(grant.source)} until {new Date(grant.expires_at).toLocaleDateString()}.</span>
              <button type="button" data-testid={`ai-privacy-consent-revoke-${grant.id}`} disabled={busy} onClick={() => void revoke(grant)} className="min-h-10 rounded border border-gray-600 px-3 font-medium text-surface disabled:opacity-60">Revoke</button>
            </li>)}
          </ul>}
          {error && <p role="alert" className="mt-2 text-xs text-red-300">{error}</p>}
        </div>
      </div>
    </section>
  );
}
