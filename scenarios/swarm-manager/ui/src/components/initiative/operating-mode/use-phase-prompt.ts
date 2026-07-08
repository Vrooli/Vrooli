/**
 * usePhasePrompt — lazily resolves the Instructions-tab prompt for a PhaseView.
 *
 * Mirrors the SkillViewerDialog lazy-fetch pattern: the prompt is only fetched
 * when the Instructions tab is active. It dispatches on the request source —
 * the contract template (unfilled slots) via prompt-manager's skill read, or a
 * substituted prompt via the operating-mode render endpoints — and degrades to
 * the resolved variable map when the prompt-manager seam is unavailable.
 */

import { useQuery } from "@tanstack/react-query";
import { defaultQueryOptions } from "../../../lib";
import { promptService } from "../../../services/prompt-service";
import { initiativeModeService } from "../../../services";
import type { PhasePromptRequest } from "./phase-view";

export interface ResolvedPhasePrompt {
  prompt: string;
  /** True for the contract source: prompt still contains {{VARIABLE}} slots. */
  isSlots: boolean;
  degraded: boolean;
  variables: Record<string, string>;
  skillId: string;
  profileKey: string;
}

export interface PhasePromptState extends ResolvedPhasePrompt {
  isLoading: boolean;
  error: Error | null;
  refetch: () => void;
}

function fallbackVariables(request: PhasePromptRequest): Record<string, string> {
  return request.source === "contract" ? {} : request.variables;
}

async function fetchPhasePrompt(request: PhasePromptRequest): Promise<ResolvedPhasePrompt> {
  if (request.source === "contract") {
    const skill = await promptService.getSkill(request.skillId);
    return {
      prompt: skill.current_content ?? "",
      isSlots: true,
      degraded: false,
      variables: {},
      skillId: request.skillId,
      profileKey: request.profileKey,
    };
  }
  const rendered = request.source === "simulation"
    ? await initiativeModeService.renderSimulationPrompt(request.mode, request.preset, request.stepIndex)
    : await initiativeModeService.renderLivePrompt(request.initiative, request.phase, request.round);
  const variables = Object.keys(rendered.variables).length > 0 ? rendered.variables : request.variables;
  return {
    prompt: rendered.prompt,
    isSlots: false,
    degraded: rendered.degraded,
    variables,
    skillId: rendered.skillId || request.skillId,
    profileKey: rendered.profileKey || request.profileKey,
  };
}

function promptQueryKey(request: PhasePromptRequest): unknown[] {
  switch (request.source) {
    case "contract":
      return ["operating-mode", "phase-prompt", "contract", request.skillId];
    case "simulation":
      return ["operating-mode", "phase-prompt", "simulation", request.mode, request.preset, request.stepIndex];
    case "live":
      return ["operating-mode", "phase-prompt", "live", request.initiative, request.phase, request.round ?? 0];
  }
}

function requestEnabled(request: PhasePromptRequest): boolean {
  switch (request.source) {
    case "contract":
      return request.skillId.length > 0;
    case "simulation":
      return request.mode.length > 0 && request.preset.length > 0;
    case "live":
      return request.initiative.length > 0 && request.phase.length > 0;
  }
}

export function usePhasePrompt(request: PhasePromptRequest, enabled = true): PhasePromptState {
  const active = enabled && requestEnabled(request);
  const query = useQuery({
    queryKey: promptQueryKey(request),
    queryFn: () => fetchPhasePrompt(request),
    enabled: active,
    ...defaultQueryOptions,
  });

  const data = query.data;
  return {
    isLoading: active && query.isLoading,
    error: query.error ?? null,
    refetch: () => void query.refetch(),
    prompt: data?.prompt ?? "",
    isSlots: data?.isSlots ?? request.source === "contract",
    degraded: data?.degraded ?? false,
    variables: data?.variables ?? fallbackVariables(request),
    skillId: data?.skillId ?? request.skillId,
    profileKey: data?.profileKey ?? request.profileKey,
  };
}
