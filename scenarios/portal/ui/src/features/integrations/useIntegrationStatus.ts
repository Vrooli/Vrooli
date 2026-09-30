import { useCallback, useEffect, useState } from "react";

import {
	BehaviorOverride,
	fetchIntegrationsStatus,
	type StatusResponse,
	updateBehaviorOverride,
} from "../../api/integrations";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

interface IntegrationStatusState {
  status: StatusResponse | null;
  loading: boolean;
  error: string;
  refresh: () => Promise<void>;
  setOverride: (override: BehaviorOverride) => Promise<void>;
}

export function useIntegrationStatus(): IntegrationStatusState {
	const { t } = useTranslation();
	const [status, setStatus] = useState<StatusResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setStatus(await fetchIntegrationsStatus());
	} catch (err) {
		setError(errorMessage(err, t));
    } finally {
      setLoading(false);
    }
	}, [t]);

  const setOverride = useCallback(async (override: BehaviorOverride) => {
    setLoading(true);
    setError("");
    try {
      setStatus(await updateBehaviorOverride(override));
	} catch (err) {
		setError(errorMessage(err, t));
    } finally {
      setLoading(false);
    }
	}, [t]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return { status, loading, error, refresh, setOverride };
}
