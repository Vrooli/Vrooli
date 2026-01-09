import { useEffect, useState } from 'react';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import type { AppProxyMetadata, LocalhostUsageReport } from '@/types';

/**
 * Loads and manages proxy metadata and localhost diagnostic reports for an app.
 * Automatically refreshes when the app ID changes.
 */
interface UseProxyMetadataSynchronizationOptions {
  /** Current app ID to load metadata for */
  currentAppId: string | null;
}

export interface UseProxyMetadataSynchronizationReturn {
  /** Proxy metadata including routes and port mappings */
  proxyMetadata: AppProxyMetadata | null;
  /** Localhost usage diagnostic report */
  localhostReport: LocalhostUsageReport | null;
}

export const useProxyMetadataSynchronization = ({
  currentAppId,
}: UseProxyMetadataSynchronizationOptions): UseProxyMetadataSynchronizationReturn => {
  const [proxyMetadata, setProxyMetadata] = useState<AppProxyMetadata | null>(null);
  const [localhostReport, setLocalhostReport] = useState<LocalhostUsageReport | null>(null);

  useEffect(() => {
    const appIdentifier = currentAppId;
    if (!appIdentifier) {
      setProxyMetadata(null);
      setLocalhostReport(null);
      return;
    }

    let cancelled = false;
    setProxyMetadata(prev => (prev && prev.appId === appIdentifier ? prev : null));
    setLocalhostReport(prev => {
      if (!prev) {
        return prev;
      }
      return prev.scenario === appIdentifier ? prev : null;
    });

    const loadDiagnostics = async () => {
      const [metadata, localhostDiagnostics] = await Promise.all([
        appService.getAppProxyMetadata(appIdentifier),
        appService.getAppLocalhostReport(appIdentifier),
      ]);

      if (!cancelled) {
        setProxyMetadata(metadata);
        setLocalhostReport(localhostDiagnostics);
      }
    };

    loadDiagnostics().catch((error) => {
      logger.warn('Failed to load proxy diagnostics', error);
    });

    return () => {
      cancelled = true;
    };
  }, [currentAppId]);

  return {
    proxyMetadata,
    localhostReport,
  };
};
