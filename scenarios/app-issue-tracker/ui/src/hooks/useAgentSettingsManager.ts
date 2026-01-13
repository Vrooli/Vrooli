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

export interface ProviderOption {
  value: string;
  label: string;
  description: string;
}

export interface SettingConstraint {
  min: number;
  max: number;
  default: number;
  description: string;
}

export interface SettingsConstraints {
  numeric: Record<string, SettingConstraint>;
  runners: ProviderOption[];
}

interface UseAgentSettingsManagerResult {
  agentSettings: AgentSettings;
  updateAgentSettings: (updater: SetStateAction<AgentSettings>) => void;
  constraints: SettingsConstraints | null;
}

function parseAgentSettings(
  payload: Record<string, unknown>,
  previous: AgentSettings,
): { settings: AgentSettings; snapshot: AgentSettingsSnapshot } {
  const agentManager = (payload.agent_manager ?? payload) as Record<string, unknown>;

  const rawRunner =
    typeof agentManager.runner_type === 'string' ? agentManager.runner_type.trim() : '';
  const runnerType: 'claude-code' | 'codex' | 'opencode' =
    rawRunner === 'claude-code' || rawRunner === 'codex' || rawRunner === 'opencode'
      ? rawRunner
      : previous.runnerType ?? 'claude-code';

  const skipPermissions =
    typeof agentManager.skip_permissions === 'boolean'
      ? (agentManager.skip_permissions as boolean)
      : previous.skipPermissionChecks;

  const resolveNumber = (value: unknown): number | undefined =>
    typeof value === 'number' && Number.isFinite(value) ? (value as number) : undefined;

  const resolveString = (value: unknown): string | undefined =>
    typeof value === 'string' && value.trim().length > 0 ? (value as string).trim() : undefined;

  const selectedMaxTurns =
    resolveNumber(agentManager.max_turns) ??
    previous.maximumTurns;

  const timeoutSeconds =
    resolveNumber(agentManager.timeout_seconds) ??
    previous.taskTimeout * 60;

  const allowedToolsRaw =
    resolveString(agentManager.allowed_tools);

  const parsedAllowedTools =
    allowedToolsRaw
      ?.split(',')
      .map((tool: string) => tool.trim())
      .filter((tool: string) => tool.length > 0) ?? previous.allowedTools;

  const next: AgentSettings = {
    ...previous,
    runnerType,
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
  const [constraints, setConstraints] = useState<SettingsConstraints | null>(null);

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

        const payload = settingsData as Record<string, unknown>;

        // Extract constraints if available
        if (payload.constraints && typeof payload.constraints === 'object') {
          setConstraints(payload.constraints as SettingsConstraints);
        }

        // Extract settings
        const actualSettings = (payload.settings as Record<string, unknown>) || payload;

        setAgentSettings((previous) => {
          const { settings, snapshot } = parseAgentSettings(actualSettings, previous);
          lastSnapshotRef.current = snapshot;
          return settings;
        });
      } catch (error) {
        // Failed to load settings - will use defaults
      }
    };

    void loadAgentBackendSettings();
  }, [apiBaseUrl]);

  useEffect(() => {
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
      runner_type: snapshot.runnerType,
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
    constraints,
  };
}
