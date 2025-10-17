import { useCallback, useEffect, useMemo, useRef, useState, type SetStateAction } from 'react';
import { AppSettings, type AgentSettings } from '../data/sampleData';
import { fetchAgentSettings, patchAgentSettings } from '../services/issues';
import {
  agentSettingsSnapshotsEqual,
  buildAgentSettingsSnapshot,
  type AgentSettingsSnapshot,
} from '../utils/issueHelpers';

interface UseAgentSettingsManagerOptions {
  apiBaseUrl: string;
  onSaveError?: (message: string) => void;
}

interface UseAgentSettingsManagerResult {
  agentSettings: AgentSettings;
  updateAgentSettings: (updater: SetStateAction<AgentSettings>) => void;
}

function parseAgentSettings(
  payload: Record<string, unknown>,
  previous: AgentSettings,
): { settings: AgentSettings; snapshot: AgentSettingsSnapshot } {
  const backendData = (payload.agent_backend ?? {}) as Record<string, unknown>;
  const providersData = (payload.providers ?? {}) as Record<string, any>;

  const rawProvider = typeof backendData.provider === 'string' ? backendData.provider.trim() : '';
  const provider: 'codex' | 'claude-code' =
    rawProvider === 'codex' || rawProvider === 'claude-code'
      ? rawProvider
      : previous.backend?.provider ?? 'codex';

  const autoFallback =
    typeof backendData.auto_fallback === 'boolean'
      ? (backendData.auto_fallback as boolean)
      : previous.backend?.autoFallback ?? true;

  const skipPermissions =
    typeof backendData.skip_permissions === 'boolean'
      ? (backendData.skip_permissions as boolean)
      : previous.skipPermissionChecks;

  const providerConfig = (providersData?.[provider] ?? {}) as Record<string, any>;
  const operationsConfig = (providerConfig?.operations ?? {}) as Record<string, any>;
  const investigateConfig = (operationsConfig?.investigate ?? {}) as Record<string, any>;
  const fixConfig = (operationsConfig?.fix ?? {}) as Record<string, any>;

  const resolveNumber = (value: unknown): number | undefined =>
    typeof value === 'number' && Number.isFinite(value) ? (value as number) : undefined;

  const resolveString = (value: unknown): string | undefined =>
    typeof value === 'string' && value.trim().length > 0 ? (value as string).trim() : undefined;

  const selectedMaxTurns =
    resolveNumber(investigateConfig.max_turns) ??
    resolveNumber(fixConfig.max_turns) ??
    previous.maximumTurns;

  const timeoutSeconds =
    resolveNumber(investigateConfig.timeout_seconds) ??
    resolveNumber(fixConfig.timeout_seconds) ??
    previous.taskTimeout * 60;

  const allowedToolsRaw =
    resolveString(investigateConfig.allowed_tools) ??
    resolveString(fixConfig.allowed_tools);

  const parsedAllowedTools =
    allowedToolsRaw
      ?.split(',')
      .map((tool: string) => tool.trim())
      .filter((tool: string) => tool.length > 0) ?? previous.allowedTools;

  const next: AgentSettings = {
    ...previous,
    backend: {
      provider,
      autoFallback,
    },
    maximumTurns: selectedMaxTurns,
    taskTimeout: Math.max(5, Math.round(timeoutSeconds / 60)),
    allowedTools: parsedAllowedTools.length > 0 ? parsedAllowedTools : previous.allowedTools,
    skipPermissionChecks: skipPermissions,
  };

  const allowedToolsKey = next.allowedTools
    .map((tool) => tool.trim())
    .filter((tool) => tool.length > 0)
    .join(',');

  const snapshot = buildAgentSettingsSnapshot(next, allowedToolsKey);

  return { settings: next, snapshot };
}

export function useAgentSettingsManager({
  apiBaseUrl,
  onSaveError,
}: UseAgentSettingsManagerOptions): UseAgentSettingsManagerResult {
  const [agentSettings, setAgentSettings] = useState<AgentSettings>(AppSettings.agent);

  const lastSnapshotRef = useRef<AgentSettingsSnapshot | null>(null);
  const initializedRef = useRef(false);

  const allowedToolsSignature = useMemo(
    () =>
      agentSettings.allowedTools
        .map((tool) => tool.trim())
        .filter((tool) => tool.length > 0)
        .join(','),
    [agentSettings.allowedTools],
  );

  useEffect(() => {
    const loadAgentBackendSettings = async () => {
      try {
        const { data: settingsData } = await fetchAgentSettings(apiBaseUrl);
        if (!settingsData) {
          return;
        }

        setAgentSettings((previous) => {
          const { settings, snapshot } = parseAgentSettings(settingsData as Record<string, unknown>, previous);
          lastSnapshotRef.current = snapshot;
          return settings;
        });
      } catch (error) {
        console.warn('Failed to load agent backend settings', error);
      }
    };

    void loadAgentBackendSettings();
  }, [apiBaseUrl]);

  useEffect(() => {
    if (!agentSettings.backend) {
      return;
    }

    const snapshot = buildAgentSettingsSnapshot(agentSettings, allowedToolsSignature);

    if (!initializedRef.current) {
      initializedRef.current = true;
      if (!lastSnapshotRef.current) {
        lastSnapshotRef.current = snapshot;
      }
      return;
    }

    if (agentSettingsSnapshotsEqual(lastSnapshotRef.current, snapshot)) {
      return;
    }

    const payload: Record<string, unknown> = {
      provider: snapshot.provider,
      auto_fallback: snapshot.autoFallback,
      max_turns: snapshot.maximumTurns,
      timeout_seconds: snapshot.taskTimeoutMinutes * 60,
      skip_permissions: snapshot.skipPermissionChecks,
    };

    if (snapshot.allowedToolsKey.length > 0 || agentSettings.allowedTools.length === 0) {
      payload.allowed_tools = snapshot.allowedToolsKey;
    }

    const saveAgentBackendSettings = async () => {
      try {
        await patchAgentSettings(apiBaseUrl, payload);
        lastSnapshotRef.current = snapshot;
      } catch (error) {
        console.error('Failed to save agent backend settings', error);
        onSaveError?.('Failed to save agent settings');
      }
    };

    void saveAgentBackendSettings();
  }, [
    agentSettings,
    allowedToolsSignature,
    apiBaseUrl,
    onSaveError,
  ]);

  const updateAgentSettings = useCallback((updater: SetStateAction<AgentSettings>) => {
    setAgentSettings((previous) =>
      typeof updater === 'function'
        ? (updater as (value: AgentSettings) => AgentSettings)(previous)
        : updater,
    );
  }, []);

  return {
    agentSettings,
    updateAgentSettings,
  };
}
