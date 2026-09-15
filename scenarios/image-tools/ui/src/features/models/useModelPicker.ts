import { useCallback, useEffect, useRef, useState } from "react";

import { JobState, jobsClient } from "../../api/jobs";
import { listOperationModels, modelsClient, type CandidateModel, type HostSummary } from "../../api/models";

/** The narrow seam the picker drives; tests inject a fake to run offline. */
export interface ModelPickerClient {
  list: (operation: string) => Promise<{
    candidates: CandidateModel[];
    host?: HostSummary;
    selectedId: string;
    selectedReason: string;
  }>;
  installModel: (id: string) => Promise<{ jobId: string; alreadyInstalled: boolean }>;
  ensureBackend: (tool: string) => Promise<{ jobId: string; alreadyInstalled: boolean; manual: boolean; detail: string }>;
  waitJob: (jobId: string) => Promise<{ ok: boolean; error: string }>;
  setEnabled: (id: string, enabled: boolean) => Promise<void>;
}

/** Production client: the Models + Jobs Connect clients. */
export const liveModelPickerClient: ModelPickerClient = {
  async list(operation) {
    const resp = await listOperationModels(operation);
    return {
      candidates: resp.candidates,
      host: resp.host,
      selectedId: resp.selectedId,
      selectedReason: resp.selectedReason,
    };
  },
  async installModel(id) {
    const resp = await modelsClient.installModel({ id });
    return { jobId: resp.jobId, alreadyInstalled: resp.alreadyInstalled };
  },
  async ensureBackend(tool) {
    const resp = await modelsClient.ensureBackend({ tool });
    return {
      jobId: resp.jobId,
      alreadyInstalled: resp.alreadyInstalled,
      manual: resp.manual,
      detail: resp.detail,
    };
  },
  async waitJob(jobId) {
    const resp = await jobsClient.waitJob({ id: jobId });
    const job = resp.job;
    return { ok: job?.state === JobState.SUCCEEDED, error: job?.error ?? "" };
  },
  async setEnabled(id, enabled) {
    await modelsClient.setModelEnabled({ id, enabled });
  },
};

export interface UseModelPicker {
  /** The operation this picker menu is for (surfaced in the derived-op caveat). */
  operation: string;
  candidates: CandidateModel[];
  host?: HostSummary;
  selectedId: string;
  selectedReason: string;
  loading: boolean;
  error: string | null;
  /** Model id currently being installed/enabled (for per-row spinners). */
  busyId: string;
  /** Per-row failure message keyed by model id. */
  rowError: Record<string, string>;
  refresh: () => void;
  installModel: (id: string) => void;
  installBackend: (tool: string, modelId: string) => void;
  enable: (id: string) => void;
}

export interface UseModelPickerArgs {
  operation: string;
  /** Whether to actively load (the picker is open). Avoids fetching while closed. */
  active: boolean;
  client?: ModelPickerClient;
}

const errorText = (err: unknown): string => (err instanceof Error ? err.message : String(err));

/**
 * Drives the model picker for one operation: loads the host-aware candidate menu
 * and runs the inline install/enable actions (each a durable job it blocks on,
 * then refetches so the row flips to "ready"). Guarded by a mounted ref so a
 * close/unmount mid-install can't set state on a dead component.
 */
export function useModelPicker({
  operation,
  active,
  client = liveModelPickerClient,
}: UseModelPickerArgs): UseModelPicker {
  const [candidates, setCandidates] = useState<CandidateModel[]>([]);
  const [host, setHost] = useState<HostSummary | undefined>(undefined);
  const [selectedId, setSelectedId] = useState("");
  const [selectedReason, setSelectedReason] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState("");
  const [rowError, setRowError] = useState<Record<string, string>>({});

  const mounted = useRef(true);
  const loadId = useRef(0);
  useEffect(
    () => () => {
      mounted.current = false;
    },
    [],
  );

  const load = useCallback(async () => {
    if (!operation) {
      return;
    }
    const id = ++loadId.current;
    setLoading(true);
    setError(null);
    try {
      const res = await client.list(operation);
      if (!mounted.current || loadId.current !== id) {
        return;
      }
      setCandidates(res.candidates);
      setHost(res.host);
      setSelectedId(res.selectedId);
      setSelectedReason(res.selectedReason);
    } catch (err) {
      if (mounted.current && loadId.current === id) {
        setError(errorText(err));
      }
    } finally {
      if (mounted.current && loadId.current === id) {
        setLoading(false);
      }
    }
  }, [client, operation]);

  useEffect(() => {
    if (active) {
      void load();
    }
  }, [active, load]);

  const setRowFailure = useCallback((id: string, message: string) => {
    setRowError((prev) => ({ ...prev, [id]: message }));
  }, []);

  const runJobThenRefresh = useCallback(
    async (modelId: string, submit: () => Promise<{ jobId: string; alreadyInstalled: boolean }>) => {
      setBusyId(modelId);
      setRowError((prev) => {
        const { [modelId]: _omit, ...rest } = prev;
        void _omit;
        return rest;
      });
      try {
        const res = await submit();
        if (!res.alreadyInstalled && res.jobId) {
          const waited = await client.waitJob(res.jobId);
          if (!waited.ok) {
            if (mounted.current) {
              setRowFailure(modelId, waited.error || "install failed");
            }
            return;
          }
        }
        await load();
      } catch (err) {
        if (mounted.current) {
          setRowFailure(modelId, errorText(err));
        }
      } finally {
        if (mounted.current) {
          setBusyId("");
        }
      }
    },
    [client, load, setRowFailure],
  );

  const installModel = useCallback(
    (id: string) => {
      void runJobThenRefresh(id, () => client.installModel(id));
    },
    [client, runJobThenRefresh],
  );

  const installBackend = useCallback(
    (tool: string, modelId: string) => {
      void runJobThenRefresh(modelId, async () => {
        const res = await client.ensureBackend(tool);
        if (res.manual) {
          throw new Error(res.detail || "manual install required");
        }
        return { jobId: res.jobId, alreadyInstalled: res.alreadyInstalled };
      });
    },
    [client, runJobThenRefresh],
  );

  const enable = useCallback(
    (id: string) => {
      setBusyId(id);
      void (async () => {
        try {
          await client.setEnabled(id, true);
          await load();
        } catch (err) {
          if (mounted.current) {
            setRowFailure(id, errorText(err));
          }
        } finally {
          if (mounted.current) {
            setBusyId("");
          }
        }
      })();
    },
    [client, load, setRowFailure],
  );

  return {
    operation,
    candidates,
    host,
    selectedId,
    selectedReason,
    loading,
    error,
    busyId,
    rowError,
    refresh: () => void load(),
    installModel,
    installBackend,
    enable,
  };
}
