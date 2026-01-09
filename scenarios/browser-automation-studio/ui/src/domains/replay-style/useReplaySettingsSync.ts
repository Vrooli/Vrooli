import { useEffect, useMemo, useRef } from 'react';
import type { ReplayStyleConfig, ReplayStyleOverrides } from './model';
import { normalizeReplayStyle } from './model';
import { useReplayStyle } from './useReplayStyle';

export interface ReplaySettingsSyncParams {
  executionId?: string;
  styleOverrides: ReplayStyleOverrides;
  extraConfig: Record<string, unknown>;
  onStyleHydrated: (style: ReplayStyleConfig) => void;
  onExtraHydrated?: (extra: Record<string, unknown>) => void;
}

export interface ReplaySettingsSyncController {
  style: ReplayStyleConfig;
  isServerReady: boolean;
  serverExtraConfig: Record<string, unknown> | null;
}

export const useReplaySettingsSync = ({
  executionId,
  styleOverrides,
  extraConfig,
  onStyleHydrated,
  onExtraHydrated,
}: ReplaySettingsSyncParams): ReplaySettingsSyncController => {
  const controller = useReplayStyle({ executionId, extraConfig });
  const lastHydratedFromControllerRef = useRef<string | null>(null);
  const lastAppliedFromStoreRef = useRef<string | null>(null);
  const lastExtraHydratedRef = useRef<string | null>(null);

  const normalizedFromStore = useMemo(
    () => normalizeReplayStyle(styleOverrides, controller.style),
    [styleOverrides, controller.style],
  );

  useEffect(() => {
    const serialized = JSON.stringify(controller.style);
    if (lastHydratedFromControllerRef.current === serialized) {
      return;
    }
    lastHydratedFromControllerRef.current = serialized;
    onStyleHydrated(controller.style);
  }, [controller.style, onStyleHydrated]);

  useEffect(() => {
    const serialized = JSON.stringify(normalizedFromStore);
    if (lastHydratedFromControllerRef.current === serialized) {
      return;
    }
    if (lastAppliedFromStoreRef.current === serialized) {
      return;
    }
    lastAppliedFromStoreRef.current = serialized;
    controller.setStyle(normalizedFromStore);
  }, [controller, normalizedFromStore]);

  useEffect(() => {
    if (!onExtraHydrated || !controller.isServerReady || !controller.serverExtraConfig) {
      return;
    }
    const serialized = JSON.stringify(controller.serverExtraConfig);
    if (lastExtraHydratedRef.current === serialized) {
      return;
    }
    lastExtraHydratedRef.current = serialized;
    onExtraHydrated(controller.serverExtraConfig);
  }, [controller.isServerReady, controller.serverExtraConfig, onExtraHydrated]);

  return {
    style: controller.style,
    isServerReady: controller.isServerReady,
    serverExtraConfig: controller.serverExtraConfig,
  };
};
