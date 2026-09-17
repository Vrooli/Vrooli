import type { SignInContext, SignInFlow } from '../../../shared/api';

const ALLOWED_LOOPBACK_HOSTS = new Set(['127.0.0.1', 'localhost', '[::1]', '::1']);

/**
 * App callbacks must stay on this machine: loopback HTTP for native apps or the
 * vrooli: scheme for the desktop shell. Anything else is refused before a
 * credential is sent.
 */
export function isAllowedCallbackUrl(urlString: string): boolean {
  try {
    const url = new URL(urlString);
    if (url.protocol === 'vrooli:') return true;
    if (url.protocol === 'http:' || url.protocol === 'https:') return ALLOWED_LOOPBACK_HOSTS.has(url.hostname);
    return false;
  } catch {
    return false;
  }
}

/** Reads app parameters from the sign-in page URL. */
export function signInContextFromSearch(params: URLSearchParams): SignInContext | undefined {
  const redirect = params.get('redirect_uri');
  const desktop = params.get('desktop_link') === 'true';
  if (!redirect && !desktop) return undefined;
  const scopes = params.get('scopes')?.split(',').map((scope) => scope.trim()).filter(Boolean);
  const context: SignInContext = {
    ...(redirect ? { redirect_uri: redirect } : {}),
    app: params.get('app') || undefined,
    state: params.get('state') || undefined,
    code_challenge: params.get('code_challenge') || undefined,
    code_challenge_method: params.get('code_challenge_method') || undefined,
    ...(desktop ? { desktop_link: true } : {}),
    installation_id: params.get('installation_id') || undefined,
    resource: params.get('resource') || undefined,
    audience: params.get('audience') || undefined,
    ...(scopes && scopes.length > 0 ? { scopes } : {}),
    business_account_id: params.get('business_account_id') || undefined,
  };
  return JSON.parse(JSON.stringify(context)) as SignInContext;
}

export function flowForContext(context?: SignInContext): SignInFlow {
  if (context?.desktop_link) return 'desktop_link';
  if (context?.redirect_uri) return 'native_app';
  return 'browser';
}

/** Why an app context cannot be completed, or null when it can. */
export function contextProblem(context: SignInContext): string | null {
  if (!context.redirect_uri || !isAllowedCallbackUrl(context.redirect_uri)) {
    return 'This app asked to return to an address that is not on this computer.';
  }
  if (context.code_challenge_method !== 'S256' || !context.code_challenge) {
    return 'This app sent an incomplete security check. Start sign-in again from the app.';
  }
  if (context.desktop_link) {
    if (!context.installation_id || !context.resource || context.audience !== `scenario:${context.resource}` || !context.scopes?.length) {
      return 'This desktop connection request is incomplete. Start again from the desktop app.';
    }
  }
  return null;
}

/** Only same-site paths are honored as a post-sign-in destination. A visitor
 * with no destination lands on their account hub, not the marketing homepage. */
export function safeNextPath(raw: string | null): string {
  if (!raw || !raw.startsWith('/') || raw.startsWith('//') || raw.startsWith('/\\')) return '/account';
  return raw;
}

/** Human names for requested capabilities, e.g. "demo:read" → "Read demo". */
export function describeScope(scope: string): string {
  const [resource, action] = scope.split(':');
  if (!resource || !action) return scope;
  const verb = { read: 'Read', write: 'Change', admin: 'Manage', use: 'Use' }[action] ?? action;
  return `${verb} ${resource.replace(/[-_]/g, ' ')}`;
}
