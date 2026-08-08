import type { Page } from 'rebrowser-playwright';
import type { ElectronTargetSpec, SessionSpec } from '../types';

type JsonTarget = {
  id?: unknown;
  type?: unknown;
  url?: unknown;
  title?: unknown;
};

const isLoopback = (hostname: string): boolean =>
  hostname === '127.0.0.1' || hostname === '[::1]';

export function validateElectronTargetSpec(target: ElectronTargetSpec): void {
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
      throw new Error(`electron_target.${field} is required`);
    }
  }
  if (target.cdp_transport !== 'loopback-authenticated' && target.cdp_transport !== 'bridge-authenticated') {
    throw new Error('unsupported Electron CDP transport');
  }
  const endpoint = new URL(target.cdp_endpoint);
  if (endpoint.protocol !== 'http:' || !isLoopback(endpoint.hostname) || !endpoint.port) {
    throw new Error('Electron CDP endpoint must be an explicit loopback HTTP endpoint with a port');
  }
  if (endpoint.username || endpoint.password || endpoint.search || endpoint.hash) {
    throw new Error('Electron CDP endpoint must not contain credentials or opaque query data');
  }
}

/**
 * External desktop contexts cannot retroactively acquire Playwright context
 * capabilities that are selected at browser/context creation time. Refuse
 * those requirements instead of creating a session that silently omits the
 * requested evidence.
 */
export function validateElectronTargetCapabilities(
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
    throw new Error(`Electron target does not support required capabilities: ${unsupported.join(', ')}`);
  }
}

export async function verifyElectronRenderer(target: ElectronTargetSpec): Promise<void> {
  validateElectronTargetSpec(target);
  const response = await fetch(new URL('/json/list', target.cdp_endpoint), {
    signal: AbortSignal.timeout(3000),
  });
  if (!response.ok) {
    throw new Error(`Electron CDP renderer discovery failed with HTTP ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!Array.isArray(payload)) {
    throw new Error('Electron CDP renderer discovery returned a non-list payload');
  }
  const matches = payload.filter((entry): entry is JsonTarget => {
    if (!entry || typeof entry !== 'object') return false;
    const candidate = entry as JsonTarget;
    return candidate.id === target.renderer_id && candidate.type === 'page';
  });
  if (matches.length !== 1) {
    throw new Error(`Electron renderer identity is ${matches.length === 0 ? 'missing' : 'ambiguous'}`);
  }
  const renderer = matches[0];
  if (!renderer) throw new Error('Electron renderer identity is missing');
  if (!isAllowedRendererNavigation(target.renderer_url, renderer.url)) {
    throw new Error('Electron renderer navigated outside the admitted origin');
  }
}

export async function selectElectronPage(pages: Page[], target: ElectronTargetSpec): Promise<Page> {
  const matches = pages.filter((page) => page.url() === target.renderer_url);
  if (matches.length !== 1) {
    if (matches.length === 0 && pages.length === 1) {
      const onlyPage = pages[0];
      if (onlyPage && isAllowedRendererNavigation(target.renderer_url, onlyPage.url())) {
        return onlyPage;
      }
    }
    throw new Error(`Electron renderer page identity is ${matches.length === 0 ? 'missing' : 'ambiguous'}`);
  }
  const page = matches[0];
  if (!page) throw new Error('Electron renderer page identity is missing');
  return page;
}

function isAllowedRendererNavigation(admittedURL: string, currentURL: string): boolean {
  try {
    const admitted = new URL(admittedURL);
    const current = new URL(currentURL);
    return admitted.protocol === current.protocol && admitted.host === current.host;
  } catch {
    return false;
  }
}
