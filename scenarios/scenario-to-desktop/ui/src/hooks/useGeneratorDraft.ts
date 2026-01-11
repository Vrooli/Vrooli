/**
 * Hook for managing generator form draft persistence.
 * Handles loading, saving, and clearing draft state from localStorage.
 */

import { useEffect, useRef, useState, useCallback } from "react";
import {
  loadGeneratorDraft,
  saveGeneratorDraft,
  clearGeneratorDraft
} from "../lib/draftStorage";
import type { BundlePreflightResponse, ProbeResponse } from "../lib/api";
import type { DeploymentMode, ServerType } from "../domain/deployment";
import type { OutputLocation, PlatformSelection } from "../domain/generator";

const DRAFT_SAVE_DEBOUNCE_MS = 600;

export type GeneratorDraftState = {
  selectedTemplate: string;
  appDisplayName: string;
  appDescription: string;
  iconPath: string;
  displayNameEdited: boolean;
  descriptionEdited: boolean;
  iconPathEdited: boolean;
  framework: string;
  serverType: ServerType;
  deploymentMode: DeploymentMode;
  platforms: PlatformSelection;
  locationMode: OutputLocation;
  outputPath: string;
  proxyUrl: string;
  bundleManifestPath: string;
  serverPort: number;
  localServerPath: string;
  localApiEndpoint: string;
  autoManageTier1: boolean;
  vrooliBinaryPath: string;
  connectionResult: ProbeResponse | null;
  connectionError: string | null;
  preflightResult: BundlePreflightResponse | null;
  preflightError: string | null;
  preflightOverride: boolean;
  preflightSecrets: Record<string, string>;
  preflightStartServices: boolean;
  preflightAutoRefresh: boolean;
  preflightSessionId: string | null;
  preflightSessionExpiresAt: string | null;
  preflightSessionTTL: number;
  deploymentManagerUrl: string | null;
  signingEnabledForBuild: boolean;
};

export type DraftTimestamps = {
  createdAt: string;
  updatedAt: string;
};

/**
 * Sanitize preflight result by removing log_tails (too verbose for storage).
 */
function sanitizePreflightResult(
  result: BundlePreflightResponse | null
): BundlePreflightResponse | null {
  if (!result) return null;
  // Remove log_tails (too verbose for storage)
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { log_tails, ...rest } = result;
  return rest;
}

/**
 * Sanitize draft state before persisting by clearing sensitive data.
 */
function sanitizeDraft(draft: GeneratorDraftState): GeneratorDraftState {
  const sanitizedSecrets = Object.keys(draft.preflightSecrets).reduce<Record<string, string>>((acc, key) => {
    acc[key] = "";
    return acc;
  }, {});

  return {
    ...draft,
    preflightSecrets: sanitizedSecrets,
    preflightResult: sanitizePreflightResult(draft.preflightResult)
  };
}

/**
 * Normalize draft on load by validating session expiry.
 */
function normalizeDraftOnLoad(draft: GeneratorDraftState): GeneratorDraftState {
  const now = Date.now();
  const sessionExpiry = draft.preflightSessionExpiresAt ? Date.parse(draft.preflightSessionExpiresAt) : 0;
  const sessionValid = Boolean(draft.preflightSessionId) && sessionExpiry > now;

  return {
    ...draft,
    preflightSessionId: sessionValid ? draft.preflightSessionId : null,
    preflightSessionExpiresAt: sessionValid ? draft.preflightSessionExpiresAt : null,
    preflightAutoRefresh: true,
    preflightStartServices: true
  };
}

export interface UseGeneratorDraftOptions {
  scenarioName: string;
  draftState: GeneratorDraftState;
  onDraftLoaded: (draft: GeneratorDraftState) => void;
  onDraftCleared: () => void;
}

export interface UseGeneratorDraftResult {
  timestamps: DraftTimestamps | null;
  loadedScenario: string | null;
  clearDraft: () => void;
  skipNextSave: () => void;
}

/**
 * Hook for managing generator draft persistence.
 */
export function useGeneratorDraft({
  scenarioName,
  draftState,
  onDraftLoaded,
  onDraftCleared
}: UseGeneratorDraftOptions): UseGeneratorDraftResult {
  const [timestamps, setTimestamps] = useState<DraftTimestamps | null>(null);
  const [loadedScenario, setLoadedScenario] = useState<string | null>(null);
  const draftSaveTimeoutRef = useRef<number | null>(null);
  const skipNextDraftSaveRef = useRef(false);

  // Load draft when scenario changes
  useEffect(() => {
    if (!scenarioName) {
      setTimestamps(null);
      setLoadedScenario(null);
      return;
    }

    const stored = loadGeneratorDraft<GeneratorDraftState>(scenarioName);
    if (!stored) {
      onDraftCleared();
      setTimestamps(null);
      setLoadedScenario(null);
      return;
    }

    const normalized = normalizeDraftOnLoad(stored.data);
    skipNextDraftSaveRef.current = true;
    onDraftLoaded(normalized);
    setTimestamps({ createdAt: stored.createdAt, updatedAt: stored.updatedAt });
    setLoadedScenario(scenarioName);
    // Note: onDraftLoaded and onDraftCleared are intentionally excluded from deps.
    // They are callback props that should be memoized by the parent component.
    // Including them would cause unwanted re-runs when parent re-renders.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scenarioName]);

  // Auto-save draft when state changes
  useEffect(() => {
    if (!scenarioName) return;

    if (skipNextDraftSaveRef.current) {
      skipNextDraftSaveRef.current = false;
      return;
    }

    if (draftSaveTimeoutRef.current) {
      window.clearTimeout(draftSaveTimeoutRef.current);
    }

    draftSaveTimeoutRef.current = window.setTimeout(() => {
      const sanitized = sanitizeDraft(draftState);
      const record = saveGeneratorDraft(scenarioName, sanitized);
      setTimestamps({ createdAt: record.createdAt, updatedAt: record.updatedAt });
    }, DRAFT_SAVE_DEBOUNCE_MS);

    return () => {
      if (draftSaveTimeoutRef.current) {
        window.clearTimeout(draftSaveTimeoutRef.current);
      }
    };
  }, [scenarioName, draftState]);

  const clearDraft = useCallback(() => {
    if (!scenarioName) return;
    clearGeneratorDraft(scenarioName);
    setTimestamps(null);
    setLoadedScenario(null);
    onDraftCleared();
  }, [scenarioName, onDraftCleared]);

  const skipNextSave = useCallback(() => {
    skipNextDraftSaveRef.current = true;
  }, []);

  return {
    timestamps,
    loadedScenario,
    clearDraft,
    skipNextSave
  };
}
