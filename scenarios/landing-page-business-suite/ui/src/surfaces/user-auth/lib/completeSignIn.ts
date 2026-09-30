import { issueDesktopLink, listBusinessAccounts, withReauthentication, type BusinessAccount, type Reauthenticate, type SignInContext } from '../../../shared/api';

/** Sends the browser to a URL; injectable so tests never navigate. */
export type Redirect = (url: string) => void;

export function redirectBrowser(url: string): void {
  window.location.assign(url);
}

export type DesktopLinkOutcome =
  | { kind: 'redirected' }
  | { kind: 'choose-account'; accounts: BusinessAccount[] };

/**
 * Finishes a desktop connection for the signed-in browser: picks the business
 * account (asking only when there is a real choice) and returns a one-use code
 * to the desktop app's loopback listener.
 */
export async function continueDesktopLink(context: SignInContext, redirect: Redirect): Promise<DesktopLinkOutcome> {
  const accounts = await listBusinessAccounts();
  const requested = context.business_account_id;
  if (requested && !accounts.some((account) => account.id === requested)) {
    throw new Error('The business account this app asked for is not available to you.');
  }
  if (!requested && accounts.length > 1) {
    return { kind: 'choose-account', accounts };
  }
  const accountId = requested || accounts[0]?.id;
  if (!accountId) {
    throw new Error('Your account has no business account to connect yet.');
  }
  await finishDesktopLink(context, accountId, redirect);
  return { kind: 'redirected' };
}

export async function finishDesktopLink(context: SignInContext, accountId: string, redirect: Redirect, reauthenticate?: Reauthenticate): Promise<void> {
  const { installation_id, resource, audience, scopes, code_challenge, redirect_uri } = context;
  if (!installation_id || !resource || !audience || !scopes?.length || !code_challenge || !redirect_uri) {
    throw new Error('This desktop connection request is incomplete. Start again from the desktop app.');
  }
	const issued = await withReauthentication(() => issueDesktopLink({
    business_account_id: accountId,
    installation_id,
    resource,
    audience,
    scopes,
    code_challenge,
    code_challenge_method: 'S256',
    redirect_uri,
	}), reauthenticate);
  const callback = new URL(redirect_uri);
  callback.searchParams.set('code', issued.code);
  if (context.state) callback.searchParams.set('state', context.state);
  redirect(callback.toString());
}
