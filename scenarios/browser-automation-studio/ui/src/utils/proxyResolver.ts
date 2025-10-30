/**
 * Proxy path resolution utilities for app-monitor integration.
 * Only loaded when proxy path resolution is needed.
 */

function toOptionalTrimmedString(value: unknown): string | null {
  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : null;
  }

  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }

  return null;
}

function normalizeProxyPath(path: string): string {
  if (!path) {
    return '/';
  }

  let normalized = path.trim();
  if (!normalized.startsWith('/')) {
    normalized = `/${normalized}`;
  }

  return normalized.endsWith('/') ? normalized : `${normalized}/`;
}

/**
 * Resolves proxy path for a given port from app-monitor globals.
 * Returns normalized proxy path if found, null otherwise.
 */
export function resolveProxyPathForPort(port?: string): string | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const globalWindow = window as Window & {
    __APP_MONITOR_PROXY_INFO__?: unknown;
    __APP_MONITOR_PROXY_INDEX__?: unknown;
  };

  const normalizedPort = toOptionalTrimmedString(port)?.toLowerCase() ?? '';

  const readPathFromEntry = (entry: unknown): string | null => {
    if (!entry || typeof entry !== 'object') {
      return null;
    }
    const record = entry as Record<string, unknown>;
    const candidate = toOptionalTrimmedString(record.path);
    return candidate ? normalizeProxyPath(candidate) : null;
  };

  const tryAliasMapLookup = (indexCandidate: unknown): string | null => {
    if (!indexCandidate || typeof indexCandidate !== 'object') {
      return null;
    }

    const indexRecord = indexCandidate as Record<string, unknown>;
    const aliasMap = indexRecord.aliasMap as unknown;

    if (aliasMap instanceof Map) {
      if (normalizedPort) {
        const entry = aliasMap.get(normalizedPort);
        const path = readPathFromEntry(entry);
        if (path) {
          return path;
        }
      }

      if (!normalizedPort && indexRecord.primary) {
        const primaryPath = readPathFromEntry(indexRecord.primary);
        if (primaryPath) {
          return primaryPath;
        }
      }
    } else if (aliasMap && typeof aliasMap === 'object') {
      if (normalizedPort && normalizedPort in (aliasMap as Record<string, unknown>)) {
        const entry = (aliasMap as Record<string, unknown>)[normalizedPort];
        const path = readPathFromEntry(entry);
        if (path) {
          return path;
        }
      }

      if (!normalizedPort && indexRecord.primary) {
        const primaryPath = readPathFromEntry(indexRecord.primary);
        if (primaryPath) {
          return primaryPath;
        }
      }
    }

    return null;
  };

  // Try index-based lookup first
  const indexPath = tryAliasMapLookup(globalWindow.__APP_MONITOR_PROXY_INDEX__);
  if (indexPath) {
    return indexPath;
  }

  // Try ports array lookup
  const tryPortsArray = (source: unknown): string | null => {
    if (!source || typeof source !== 'object') {
      return null;
    }

    const record = source as Record<string, unknown>;
    const ports = record.ports;
    if (Array.isArray(ports)) {
      for (const entry of ports) {
        if (!entry || typeof entry !== 'object') {
          continue;
        }

        const entryRecord = entry as Record<string, unknown>;
        const entryPort = toOptionalTrimmedString(entryRecord.port)?.toLowerCase();
        if (normalizedPort && entryPort && entryPort === normalizedPort) {
          const path = readPathFromEntry(entryRecord);
          if (path) {
            return path;
          }
        }

        if (normalizedPort && Array.isArray(entryRecord.aliases)) {
          const aliasMatch = entryRecord.aliases
            .map((alias) => toOptionalTrimmedString(alias)?.toLowerCase())
            .filter(Boolean)
            .includes(normalizedPort);
          if (aliasMatch) {
            const path = readPathFromEntry(entryRecord);
            if (path) {
              return path;
            }
          }
        }
      }
    }

    if (!normalizedPort && record.primary) {
      const primaryPath = readPathFromEntry(record.primary);
      if (primaryPath) {
        return primaryPath;
      }
    }

    return null;
  };

  const infoPath = tryPortsArray(globalWindow.__APP_MONITOR_PROXY_INFO__);
  if (infoPath) {
    return infoPath;
  }

  return null;
}
