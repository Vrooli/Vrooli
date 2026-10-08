import { CONNECT_API_BASE } from './common';
import { getBrowserBinding } from '../../surfaces/user-auth/lib/browserBinding';

async function rpc<T>(method: string, body: Record<string, unknown>, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`${CONNECT_API_BASE}/landing_page_business_suite.v1.PasskeyService/${method}`, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', 'X-Lpbs-Browser-Binding': getBrowserBinding() }, body: JSON.stringify(body), signal });
  if (!response.ok) throw new Error(`Passkey request failed (${response.status})`);
  return response.json() as Promise<T>;
}
export interface PasskeySummary { id: string; nickname: string; createdAt?: string; lastUsedAt?: string; backupState?: string }
export const beginPasskeyRegistration = () => rpc<{ optionsJson: string; ceremonyId: string }>('BeginRegistration', {});
export const finishPasskeyRegistration = (credentialJson: string, ceremonyId: string, nickname = '') => rpc<{ id: string; nickname: string }>('FinishRegistration', { credentialJson, ceremonyId, nickname });
export const beginPasskeyAuthentication = (context = '', signal?: AbortSignal) => rpc<{ optionsJson: string; ceremonyId: string }>('BeginAuthentication', { context }, signal);
export const finishPasskeyAuthentication = (credentialJson: string, ceremonyId: string, context = '') => rpc<{ authenticated: boolean; redirectUrl?: string; desktopContinuation?: boolean }>('FinishAuthentication', { credentialJson, ceremonyId, context });
export const listPasskeys = () => rpc<{ passkeys: PasskeySummary[] }>('ListPasskeys', {});
export const renamePasskey = (id: string, nickname: string) => rpc<{ passkey: PasskeySummary }>('RenamePasskey', { id, nickname });
export const revokePasskey = (id: string) => rpc<{ revoked: boolean }>('RevokePasskey', { id });
