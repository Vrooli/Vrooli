import {
  isAppMonitorProxyPreviewTarget,
  isBlockedHostEmbedPreviewTarget,
  normalizeScenarioNavigationInput,
  resolvePreviewUrlCandidate,
} from '@/utils/previewUrl';

const PROXY_BASE_PATTERN = /^\/apps\/([^/]+)\/proxy(?:\/|$)/i;
const URL_PARSE_BASE = 'http://localhost';

export const PREVIEW_NAV_INVALID_MESSAGE = 'Enter an absolute URL or wait for preview to load before using relative paths.';
export const PREVIEW_NAV_BLOCKED_HOST_MESSAGE = 'Preview URL points to App Monitor shell. Use a scenario proxy URL like /apps/<scenario>/proxy/.';

type ProxyBase = {
  origin: string;
  scenarioIdentifier: string;
  basePath: string;
};

const normalizePath = (path: string): string => {
  if (path === '/') {
    return path;
  }
  return path.endsWith('/') ? path.slice(0, -1) : path;
};

const parseProxyBase = (value: string): ProxyBase | null => {
  try {
    const parsed = new URL(value, URL_PARSE_BASE);
    const match = parsed.pathname.match(PROXY_BASE_PATTERN);
    if (!match) {
      return null;
    }

    const rawScenarioIdentifier = decodeURIComponent(match[1] ?? '').trim();
    if (!rawScenarioIdentifier) {
      return null;
    }
    const scenarioIdentifier = rawScenarioIdentifier.toLowerCase();

    return {
      origin: parsed.origin,
      scenarioIdentifier,
      basePath: normalizePath(`/apps/${encodeURIComponent(rawScenarioIdentifier)}/proxy/`),
    };
  } catch {
    return null;
  }
};

const canBridgeNavigateToTarget = (
  targetUrl: string,
  currentReference: string | null,
  childOrigin: string | null,
): boolean => {
  if (!currentReference) {
    return false;
  }

  try {
    const resolvedTarget = new URL(targetUrl, URL_PARSE_BASE);
    if (childOrigin && resolvedTarget.origin !== childOrigin) {
      return false;
    }
  } catch {
    return false;
  }

  const currentBase = parseProxyBase(currentReference);
  const targetBase = parseProxyBase(targetUrl);
  if (!currentBase || !targetBase) {
    return false;
  }

  if (currentBase.origin !== targetBase.origin) {
    return false;
  }

  if (currentBase.scenarioIdentifier !== targetBase.scenarioIdentifier) {
    return false;
  }

  try {
    const resolvedCurrent = new URL(currentReference, URL_PARSE_BASE);
    const resolvedTarget = new URL(targetUrl, URL_PARSE_BASE);
    const currentPath = normalizePath(resolvedCurrent.pathname);
    const targetPath = normalizePath(resolvedTarget.pathname);
    if (!(targetPath === targetBase.basePath || targetPath.startsWith(`${targetBase.basePath}/`))) {
      return false;
    }

    // Bridge GO uses history.pushState in the child; keep it for same-path URL updates.
    // Cross-path changes should force iframe src navigation for stronger compatibility.
    return currentPath === targetPath;
  } catch {
    return false;
  }
};

const normalizeUrl = (value: string): string => value.replace(/\/$/, '');

export const isSameNormalizedUrl = (left: string, right: string): boolean => (
  normalizeUrl(left) === normalizeUrl(right)
);

export type PreviewNavigationPlan =
  | {
    kind: 'empty';
    nextInput: '';
  }
  | {
    kind: 'invalid';
    nextInput: string;
    message: string;
  }
  | {
    kind: 'blocked-host';
    nextInput: string;
    resolvedTarget: string;
    message: string;
  }
  | {
    kind: 'bridge-go';
    nextInput: string;
    resolvedTarget: string;
  }
  | {
    kind: 'local-go';
    nextInput: string;
    resolvedTarget: string;
  };

export interface CreatePreviewNavigationPlanOptions {
  rawValue: string;
  navigationReference: string | null;
  hostOrigin: string | null;
  bridgeSupported: boolean;
  childOrigin: string | null;
}

export const createPreviewNavigationPlan = ({
  rawValue,
  navigationReference,
  hostOrigin,
  bridgeSupported,
  childOrigin,
}: CreatePreviewNavigationPlanOptions): PreviewNavigationPlan => {
  const trimmed = rawValue.trim();
  if (!trimmed) {
    return {
      kind: 'empty',
      nextInput: '',
    };
  }

  const nextInput = normalizeScenarioNavigationInput(trimmed) ?? trimmed;
  const resolvedTarget = resolvePreviewUrlCandidate(nextInput, navigationReference);
  if (!resolvedTarget) {
    return {
      kind: 'invalid',
      nextInput,
      message: PREVIEW_NAV_INVALID_MESSAGE,
    };
  }

  if (isBlockedHostEmbedPreviewTarget(resolvedTarget, hostOrigin)) {
    return {
      kind: 'blocked-host',
      nextInput,
      resolvedTarget,
      message: PREVIEW_NAV_BLOCKED_HOST_MESSAGE,
    };
  }

  if (isAppMonitorProxyPreviewTarget(resolvedTarget)) {
    return {
      kind: 'blocked-host',
      nextInput,
      resolvedTarget,
      message: PREVIEW_NAV_BLOCKED_HOST_MESSAGE,
    };
  }

  if (
    bridgeSupported
    && canBridgeNavigateToTarget(resolvedTarget, navigationReference, childOrigin)
  ) {
    return {
      kind: 'bridge-go',
      nextInput,
      resolvedTarget,
    };
  }

  return {
    kind: 'local-go',
    nextInput,
    resolvedTarget,
  };
};
