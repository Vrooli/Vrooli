import type { BrowserPermissionProfile } from '../types/session';
import { getActionType, getNavigateParams, type HandlerInstruction } from '../proto';

export type PermissionDecision = { allowed: true } | { allowed: false; reason: string };

const safetyPassReceiptKinds = new Set([
  'purchase', 'submit', 'delete', 'credential_use', 'extension_install', 'download',
  'input', 'service_worker_unregister', 'recording_start', 'recording_stop',
  'replay_preview', 'close_page', 'reset', 'audio_playback_restart', 'audio_playback_stop',
]);

/** Second-layer check for the API-admitted profile. The API remains responsible
 * for consent receipts, dynamic revocation, credentials and approval authority. */
export function authorizeInstruction(
  profile: BrowserPermissionProfile | undefined,
  instruction: HandlerInstruction,
  currentURL: string,
  now = Date.now(),
  sessionStartedAt?: number,
  actionsUsed = 0,
): PermissionDecision {
  if (!profile) return { allowed: true };
  const kind = getActionType(instruction).toLowerCase();
  const target = kind === 'navigate' ? getNavigateParams(instruction.action)?.url : undefined;
  return authorizeUnreceiptedOperation(profile, kind, currentURL, target, now, sessionStartedAt, actionsUsed);
}

/** Driver routes do not own an approval verifier. Apply the API operation's
 * approval tier before any route without receipt verification reaches effects. */
export function authorizeUnreceiptedOperation(
  profile: BrowserPermissionProfile | undefined,
  kind: string,
  currentURL: string,
  targetURL?: string,
  now = Date.now(),
  sessionStartedAt?: number,
  actionsUsed = 0,
): PermissionDecision {
  if (profile) {
    const normalizedKind = kind.toLowerCase();
    const mutation = !['assert', 'extract', 'screenshot', 'wait', 'observe', 'done', 'history_read'].includes(normalizedKind);
    if (mutation && profile.approval === 'every_action') {
      return deny('driver route cannot verify the required every-action approval receipt');
    }
    if (mutation && profile.approval === 'safety_pass' && requiresSafetyPassReceipt(normalizedKind)) {
      return deny('driver route cannot verify the required safety-pass approval receipt');
    }
  }
  return authorizeOperation(profile, kind, currentURL, targetURL, now, sessionStartedAt, actionsUsed);
}

function requiresSafetyPassReceipt(kind: string): boolean {
  return safetyPassReceiptKinds.has(kind);
}

export function authorizeOperation(
  profile: BrowserPermissionProfile | undefined,
  kind: string,
  currentURL: string,
  targetURL?: string,
  now = Date.now(),
  sessionStartedAt?: number,
  actionsUsed = 0,
): PermissionDecision {
  if (!profile) return { allowed: true };
  kind = kind.toLowerCase();
  const mutation = !['assert', 'extract', 'screenshot', 'wait', 'observe', 'done', 'history_read'].includes(kind);
  if (profile.effect === 'observe' && mutation) return deny('observe profile cannot perform mutations');
  if (mutation && !profile.control) return deny('browser control is disabled');
  if (['assert', 'extract', 'screenshot', 'observe'].includes(kind) && !profile.view) return deny('browser viewing is disabled');
  if (kind === 'history_read' && (!profile.view || !profile.history_read)) return deny('browser history reading is disabled');
  if (kind === 'recording_stop' && !profile.history_read) return deny('recording stop receipt requires history read');
  if (profile.max_actions && actionsUsed >= profile.max_actions) return deny('action budget exhausted');
  if (profile.max_duration && sessionStartedAt && now - sessionStartedAt >= profile.max_duration / 1_000_000) return deny('session time budget exhausted');
  const target = kind === 'navigate' ? targetURL : currentURL;
  if (!target || !allowsURL(profile, target)) return deny('URL outside allowed scope');
  if ((kind.includes('cdp') || kind === 'raw_protocol') && profile.effect !== 'unrestricted') return deny('raw CDP requires unrestricted effect');
  return { allowed: true };
}

function deny(reason: string): PermissionDecision { return { allowed: false, reason }; }

function allowsURL(profile: BrowserPermissionProfile, raw: string): boolean {
  let candidate: URL;
  try { candidate = new URL(raw); } catch { return false; }
  if (!['http:', 'https:'].includes(candidate.protocol) || candidate.username || candidate.password) return false;
  if ((profile.url_deny ?? []).some(pattern => matches(pattern, candidate))) return false;
  return (profile.url_allow ?? []).some(pattern => matches(pattern, candidate));
}

function matches(pattern: string, candidate: URL): boolean {
  const match = /^(https?):\/\/([^/]+)(\/.*)?$/i.exec(pattern.trim());
  const scheme = match?.[1];
  const hostValue = match?.[2];
  if (!scheme || !hostValue || `${scheme.toLowerCase()}:` !== candidate.protocol) return false;
  const hostPattern = hostValue.toLowerCase();
  const host = candidate.hostname.toLowerCase();
  if (hostPattern.startsWith('*.')) {
    const suffix = hostPattern.slice(1);
    if (!host.endsWith(suffix) || host === suffix.slice(1)) return false;
  } else if (host !== hostPattern) return false;
  const pathPattern = match[3] ?? '/*';
  if (pathPattern === '/*') return true;
  const prefix = pathPattern.endsWith('*') ? pathPattern.slice(0, -1) : pathPattern;
  return candidate.pathname.startsWith(prefix);
}
