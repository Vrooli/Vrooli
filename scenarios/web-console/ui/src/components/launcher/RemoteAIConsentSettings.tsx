import { useCallback, useEffect, useState } from "react";
import { createAIPrivacyConsent, listAIPrivacyConsent, revokeAIPrivacyConsent, AI_PRIVACY_CONSENT_SESSION_KEY, AI_PRIVACY_CONSENT_SOURCE_SESSION_KEY, type AIPrivacyConsentGrant } from "../../api/ai";

type RemoteSource = AIPrivacyConsentGrant["source"];

const labelFor = (source: RemoteSource) => source === "own-key" ? "your configured OpenRouter account" : "Vrooli AI credits";

export function RemoteAIConsentSettings() {
  const [source, setSource] = useState<RemoteSource>("own-key");
  const [grants, setGrants] = useState<AIPrivacyConsentGrant[]>([]);
  const [hasSessionGrant, setHasSessionGrant] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    try {
      const status = await listAIPrivacyConsent();
      setGrants(status.grants ?? []);
      setError("");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to load consent status.");
    }
  }, []);

  useEffect(() => {
    void refresh();
    setHasSessionGrant(Boolean(window.sessionStorage.getItem(AI_PRIVACY_CONSENT_SESSION_KEY)) && window.sessionStorage.getItem(AI_PRIVACY_CONSENT_SOURCE_SESSION_KEY) === source);
  }, [refresh]);

  const grant = async () => {
    setBusy(true);
    setError("");
    try {
      const created = await createAIPrivacyConsent(source);
      window.sessionStorage.setItem(AI_PRIVACY_CONSENT_SESSION_KEY, created.receipt);
      window.sessionStorage.setItem(AI_PRIVACY_CONSENT_SOURCE_SESSION_KEY, source);
      setHasSessionGrant(true);
      await refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to save consent.");
    } finally {
      setBusy(false);
    }
  };

  const revoke = async (id: string) => {
    setBusy(true);
    setError("");
    try {
      await revokeAIPrivacyConsent(id);
      window.sessionStorage.removeItem(AI_PRIVACY_CONSENT_SESSION_KEY);
      window.sessionStorage.removeItem(AI_PRIVACY_CONSENT_SOURCE_SESSION_KEY);
      setHasSessionGrant(false);
      await refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to revoke consent.");
    } finally {
      setBusy(false);
    }
  };

  const liveGrants = grants.filter((item) => !item.revoked_at && Date.parse(item.expires_at) > Date.now());
  return (
    <section aria-labelledby="remote-ai-consent-title" data-testid="remote-ai-consent" className="mt-3 rounded-lg border border-wc-default bg-wc-surface p-3">
      <h3 id="remote-ai-consent-title" className="text-sm font-semibold text-wc-text-primary">Confidential data sharing</h3>
      <p className="mt-1 text-xs leading-5 text-wc-text-muted">
        Provider selection and saved keys do not grant permission. If you allow a remote source, confidential prompts and relevant session or workspace context may leave this device for that source and follow its data policies. Secret material remains local-only.
      </p>
      <label className="mt-3 block text-xs font-medium text-wc-text-primary" htmlFor="remote-ai-consent-source">Remote source</label>
      <select id="remote-ai-consent-source" data-testid="remote-ai-consent-source" value={source} onChange={(event) => {
        const selected = event.target.value as RemoteSource;
        setSource(selected);
        setHasSessionGrant(Boolean(window.sessionStorage.getItem(AI_PRIVACY_CONSENT_SESSION_KEY)) && window.sessionStorage.getItem(AI_PRIVACY_CONSENT_SOURCE_SESSION_KEY) === selected);
      }} className="mt-1 min-h-10 w-full rounded border border-wc-default bg-wc-surface text-sm text-wc-text-primary">
        <option value="own-key">{labelFor("own-key")}</option>
        <option value="vrooli-credits">{labelFor("vrooli-credits")}</option>
      </select>
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <button type="button" disabled={busy} onClick={() => void grant()} data-testid="remote-ai-consent-grant" className="min-h-10 rounded bg-wc-accent px-3 text-sm font-semibold text-white disabled:opacity-60">
          {busy ? "Saving…" : `Allow confidential data for ${labelFor(source)}`}
        </button>
        <span role="status" className="text-xs text-wc-text-muted">
          {hasSessionGrant ? "A consent receipt is active for this browser tab." : "No consent receipt is active in this browser tab."}
        </span>
      </div>
      {liveGrants.length > 0 && (
        <ul className="mt-3 space-y-2 border-t border-wc-default pt-2" aria-label="Active remote consent grants">
          {liveGrants.map((item) => (
            <li key={item.id} className="flex flex-wrap items-center justify-between gap-2 text-xs text-wc-text-muted">
              <span>Allowed for {labelFor(item.source)} until {new Date(item.expires_at).toLocaleDateString()}.</span>
              <button type="button" disabled={busy} onClick={() => void revoke(item.id)} data-testid={`remote-ai-consent-revoke-${item.id}`} className="min-h-10 rounded border border-wc-default px-3 font-semibold text-wc-text-primary disabled:opacity-60">Revoke</button>
            </li>
          ))}
        </ul>
      )}
      {error && <p role="alert" className="mt-2 text-xs text-wc-error-text">{error}</p>}
    </section>
  );
}
