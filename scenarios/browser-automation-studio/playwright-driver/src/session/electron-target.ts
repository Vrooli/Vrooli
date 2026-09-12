import type { Page } from 'rebrowser-playwright';
import type { AppTargetKind, AppTargetSpec, SessionSpec } from '../types';
import path from 'node:path';

type JsonTarget = {
  id?: unknown;
  type?: unknown;
  url?: unknown;
  title?: unknown;
};

const isLoopback = (hostname: string): boolean => hostname === '127.0.0.1' || hostname === '[::1]';

const targetKind = (target: AppTargetSpec): AppTargetKind => target.target_kind ?? 'electron';

export function validateAppTargetSpec(target: AppTargetSpec): void {
  for (const [field, value] of [
    ['target_id', target.target_id],
    ['cdp_endpoint', target.cdp_endpoint],
    ['renderer_id', target.renderer_id],
    ['renderer_url', target.renderer_url],
    ['scenario_name', target.scenario_name],
    ['artifact_digest', target.artifact_digest],
    ['context_id', target.context_id],
  ] as const) {
    if (typeof value !== 'string' || value.trim() === '') {
      throw new Error(`app_target.${field} is required`);
    }
  }
  if (targetKind(target) !== 'electron' && targetKind(target) !== 'android-webview') {
    throw new Error(`unsupported app target kind: ${String(target.target_kind)}`);
  }
  if (
    target.cdp_transport !== 'loopback-authenticated' &&
    target.cdp_transport !== 'bridge-authenticated'
  ) {
    throw new Error('unsupported app-target CDP transport');
  }
  const endpoint = new URL(target.cdp_endpoint);
  if (endpoint.protocol !== 'http:' || !isLoopback(endpoint.hostname) || !endpoint.port) {
    throw new Error(
      'app-target CDP endpoint must be an explicit loopback HTTP endpoint with a port'
    );
  }
  if (endpoint.username || endpoint.password || endpoint.search || endpoint.hash) {
    throw new Error('app-target CDP endpoint must not contain credentials or opaque query data');
  }
}

/**
 * External desktop contexts cannot retroactively acquire Playwright context
 * capabilities that are selected at browser/context creation time. Refuse
 * those requirements instead of creating a session that silently omits the
 * requested evidence.
 */
export function validateAppTargetCapabilities(
  capabilities: SessionSpec['required_capabilities']
): void {
  const unsupported = [
    ['har', capabilities?.har],
    ['video', capabilities?.video],
    ['tracing', capabilities?.tracing],
    ['performance_trace', capabilities?.performance_trace],
    ['accessibility', capabilities?.accessibility],
  ]
    .filter(([, requested]) => requested === true)
    .map(([name]) => name);
  if (unsupported.length > 0) {
    throw new Error(`app target does not support required capabilities: ${unsupported.join(', ')}`);
  }
}

export async function verifyAppTargetRenderer(target: AppTargetSpec): Promise<void> {
  validateAppTargetSpec(target);
  const response = await fetch(new URL('/json/list', target.cdp_endpoint), {
    signal: AbortSignal.timeout(3000),
  });
  if (!response.ok) {
    throw new Error(`app-target CDP renderer discovery failed with HTTP ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!Array.isArray(payload)) {
    throw new Error('app-target CDP renderer discovery returned a non-list payload');
  }
  const matches = payload.filter((entry): entry is JsonTarget => {
    if (!entry || typeof entry !== 'object') return false;
    const candidate = entry as JsonTarget;
    return candidate.id === target.renderer_id && candidate.type === 'page';
  });
  if (matches.length !== 1) {
    throw new Error(
      `app-target renderer identity is ${matches.length === 0 ? 'missing' : 'ambiguous'}`
    );
  }
  const renderer = matches[0];
  if (!renderer) throw new Error('app-target renderer identity is missing');
  if (typeof renderer.url !== 'string' || !isAllowedRendererNavigation(target, renderer.url)) {
    throw new Error('app-target renderer navigated outside the admitted origin');
  }
}

export async function selectAppTargetPage(pages: Page[], target: AppTargetSpec): Promise<Page> {
  await Promise.resolve();
  const matches = pages.filter((page) => page.url() === target.renderer_url);
  // Android WebView CDP can expose more than one Playwright Page wrapper
  // for the same device-owned renderer. The renderer ID was already
  // authenticated by verifyAppTargetRenderer against /json/list, so exact
  // URL matches are safe to collapse for this target kind. Desktop targets
  // retain strict page-identity ambiguity rejection below.
  if (targetKind(target) === 'android-webview' && matches.length > 0) {
    return matches[0] as Page;
  }
  if (matches.length !== 1) {
    if (matches.length === 0 && pages.length === 1) {
      const onlyPage = pages[0];
      if (onlyPage && isAllowedRendererNavigation(target, onlyPage.url())) {
        return onlyPage;
      }
    }
    throw new Error(
      `app-target renderer page identity is ${matches.length === 0 ? 'missing' : 'ambiguous'}`
    );
  }
  const page = matches[0];
  if (!page) throw new Error('app-target renderer page identity is missing');
  return page;
}

function isAllowedRendererNavigation(target: AppTargetSpec, currentURL: string): boolean {
  try {
    const admitted = new URL(target.renderer_url);
    const current = new URL(currentURL);
    if (admitted.protocol !== current.protocol || admitted.host !== current.host) return false;
    if (admitted.protocol === 'file:') {
      const directory = path.posix.dirname(admitted.pathname);
      return current.pathname === admitted.pathname || current.pathname.startsWith(`${directory}/`);
    }
    return true;
  } catch {
    return false;
  }
}
