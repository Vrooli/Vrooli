// Read hooks for the Usage Intelligence Measures procedures. Every surface
// (Usage page, dashboard strip, run usage panel) reads the same queries so
// the numbers agree wherever they are shown.

import { createClient } from "@connectrpc/connect";
import { createScenarioConnectTransport } from "@vrooli/api-base";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { CohortSelectorSchema, type CohortSelector } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { TimeWindowSchema, TimeWindowToken, type TimeWindow } from "@vrooli/proto-types/measures/v1/measures_pb";
import { FrictionService } from "@vrooli/proto-types/agent-manager/v1/domain/friction_pb";
import { REQUEST_DEADLINE_MS, resolveAgentManagerApiBase } from "../../lib/api";
import { measuresClient } from "./measuresClient";

export const USAGE_GROUP_BY = ["session", "run", "profile", "workload", "model", "runner", "modality", "tool_kind"] as const;
export type UsageGroupBy = (typeof USAGE_GROUP_BY)[number];

/** The provider window a request is narrowed to. */
export interface UsageWindowScope {
  provider: string;
  pool: string;
  window: string;
}

const REFRESH_MS = 60_000;

/** Attribution can take seconds while the API compacts; past the shared deadline it is an error, not a spinner. */
export const CALL_OPTIONS = { timeoutMs: REQUEST_DEADLINE_MS };

const frictionClient = createClient(FrictionService, createScenarioConnectTransport({ baseUrl: resolveAgentManagerApiBase() }));

const usageQueryKeys = {
  forecast: ["usage", "forecast"] as const,
  burners: (groupBy: UsageGroupBy, runId: string) => ["usage", "burners", groupBy, runId] as const,
  recommendations: (runId: string, scope?: UsageWindowScope) => ["usage", "recommendations", runId, scope?.provider ?? "", scope?.pool ?? "", scope?.window ?? ""] as const,
  coverage: ["usage", "coverage"] as const,
  modes: ["usage", "modes"] as const,
  outcomes: (window: EvidenceWindow, by: string) => ["usage", "outcomes", window, by] as const,
  parity: (window: EvidenceWindow) => ["usage", "parity", window] as const,
  definitions: ["usage", "definitions"] as const,
  compare: (modeId: string, from: string, to: string) => ["usage", "compare", modeId, from, to] as const,
  experiments: ["usage", "experiments"] as const,
  experimentPlan: (runner: string, role: string, split: number) => ["usage", "experiments", "plan", runner, role, split] as const,
  friction: (status: string) => ["usage", "friction", status] as const,
};

export function useUsageForecast() {
  return useQuery({
    queryKey: usageQueryKeys.forecast,
    queryFn: () => measuresClient.usageForecast({}, CALL_OPTIONS),
    refetchInterval: REFRESH_MS,
    retry: false,
  });
}

export function useUsageBurners(groupBy: UsageGroupBy = "session", runId = "") {
  return useQuery({
    queryKey: usageQueryKeys.burners(groupBy, runId),
    queryFn: () => measuresClient.usageBurners({ groupBy, runId }, CALL_OPTIONS),
    // Switching group-by keeps the previous grouping on screen until the new one answers.
    placeholderData: keepPreviousData,
    refetchInterval: REFRESH_MS,
    retry: false,
  });
}

/** Grouped recommendations for a run, or for one provider window when `scope` is given. */
export function useUsageRecommendations(runId = "", scope?: UsageWindowScope) {
  return useQuery({
    queryKey: usageQueryKeys.recommendations(runId, scope),
    queryFn: () => measuresClient.usageRecommendations({ runId, ...scope }, CALL_OPTIONS),
    placeholderData: keepPreviousData,
    refetchInterval: REFRESH_MS,
    retry: false,
  });
}

export function useUsageCoverage() {
  return useQuery({
    queryKey: usageQueryKeys.coverage,
    queryFn: () => measuresClient.usageCoverage({}, CALL_OPTIONS),
    refetchInterval: 5 * REFRESH_MS,
    retry: false,
  });
}

/** Every material friction finding shown on one page; the server's default is a short CLI list. */
export const FRICTION_PAGE_LIMIT = 500;

/** Material friction findings; an empty status returns every status. */
export function useUsageFriction(status = "") {
  return useQuery({
    queryKey: usageQueryKeys.friction(status),
    queryFn: () => measuresClient.usageFriction({ status, limit: FRICTION_PAGE_LIMIT }, CALL_OPTIONS),
    refetchInterval: REFRESH_MS,
    retry: false,
  });
}

/** A finding as listed: its fingerprint and the start of the window it was listed in, where the API looks it up. */
export interface FrictionTarget {
  fingerprint: string;
  since: string;
}

/** Report files the problem to its owner and resolves with the bug id; dismiss resolves with the new status. */
export function useFrictionActions() {
  const queryClient = useQueryClient();
  const refresh = () => queryClient.invalidateQueries({ queryKey: ["usage", "friction"] });
  const report = useMutation({ mutationFn: (target: FrictionTarget) => frictionClient.reportFriction(target, CALL_OPTIONS), onSuccess: refresh });
  const dismiss = useMutation({ mutationFn: (target: FrictionTarget) => frictionClient.dismissFriction(target, CALL_OPTIONS), onSuccess: refresh });
  return { report, dismiss };
}

/** The windows the evidence view can read; the API's own tokens. */
export const EVIDENCE_WINDOWS = ["last_7d", "last_30d"] as const;
export type EvidenceWindow = (typeof EVIDENCE_WINDOWS)[number];
const WINDOW_TOKENS = { last_7d: TimeWindowToken.LAST_7D, last_30d: TimeWindowToken.LAST_30D } as const;

function tokenWindow(window: EvidenceWindow): TimeWindow {
  return create(TimeWindowSchema, { window: { case: "token", value: WINDOW_TOKENS[window] } });
}

/** What accepted outcomes cost, by model, role or mode, with ranges: the same call as `usage outcomes`. */
export function useUsageOutcomes(window: EvidenceWindow, by: "model" | "role" | "mode" | "arm") {
  return useQuery({
    queryKey: usageQueryKeys.outcomes(window, by),
    queryFn: () => measuresClient.usageOutcomes({ window: tokenWindow(window), by }, CALL_OPTIONS),
    placeholderData: keepPreviousData,
    refetchInterval: 5 * REFRESH_MS,
    retry: false,
  });
}

/** Coverage by start path and harness: the same call as `usage parity`. */
export function useUsageParity(window: EvidenceWindow) {
  return useQuery({
    queryKey: usageQueryKeys.parity(window),
    queryFn: () => measuresClient.usageParity({ window: tokenWindow(window) }, CALL_OPTIONS),
    placeholderData: keepPreviousData,
    refetchInterval: 5 * REFRESH_MS,
    retry: false,
  });
}

/** The measure id the operator-signals worker will deliver; the view reads it only once it is defined. */
export const OPERATOR_SIGNALS_MEASURE = "usage.operator_signals";

/**
 * Whether `usage.operator_signals` is a defined measure. A failed read counts as
 * absent: the view then says "not available yet" rather than guessing.
 */
export function useOperatorSignalsDefined() {
  return useQuery({
    queryKey: usageQueryKeys.definitions,
    queryFn: async () => {
      try {
        const { definitions } = await measuresClient.allMeasureDefinitions({}, CALL_OPTIONS);
        return definitions.some((definition) => definition.id === OPERATOR_SIGNALS_MEASURE);
      } catch {
        return false;
      }
    },
    staleTime: 10 * REFRESH_MS,
    retry: false,
  });
}

/** One mode's before and after: the equal-length span before it started against the span it ran. */
export function useModeBeforeAfter(modeId: string, startedAt: string, endedAt: string, enabled: boolean) {
  const from = Date.parse(startedAt);
  const to = Date.parse(endedAt);
  return useQuery({
    queryKey: usageQueryKeys.compare(modeId, startedAt, endedAt),
    enabled: enabled && Number.isFinite(from) && Number.isFinite(to) && to > from,
    queryFn: () => {
      const span = (a: number, b: number): CohortSelector => create(CohortSelectorSchema, { window: create(TimeWindowSchema, { window: { case: "custom", value: { from: timestampFromMs(a), to: timestampFromMs(b) } } }) });
      return measuresClient.usageCompare({ left: span(from - (to - from), from), right: span(from, to) }, CALL_OPTIONS);
    },
    retry: false,
  });
}

/** The active usage modes with burn since start and the reach table. */
export function useUsageModes() {
  return useQuery({
    queryKey: usageQueryKeys.modes,
    queryFn: () => measuresClient.listUsageModes({}, CALL_OPTIONS),
    refetchInterval: REFRESH_MS,
    retry: false,
  });
}

/** What a start or an end needs: the mode request, or the ids to end. Both refresh the modes and the forecast. */
export function useUsageModeActions() {
  const queryClient = useQueryClient();
  const refresh = () => Promise.all([queryClient.invalidateQueries({ queryKey: usageQueryKeys.modes }), queryClient.invalidateQueries({ queryKey: usageQueryKeys.forecast })]);
  const start = useMutation({
    mutationFn: (request: { preset: string; runner: string; role: string; profile: string; until: string }) => measuresClient.startUsageMode(request, CALL_OPTIONS),
    onSuccess: refresh,
  });
  const end = useMutation({ mutationFn: (ids: string[]) => measuresClient.endUsageMode({ ids }, CALL_OPTIONS), onSuccess: refresh });
  // Extending keeps the mode's id, start and settings, so its burn since start stays continuous.
  const extend = useMutation({ mutationFn: (request: { id: string; to: string }) => measuresClient.extendUsageMode(request, CALL_OPTIONS), onSuccess: refresh });
  return { start, end, extend };
}

/**
 * What switching running work would do, per run, with the estimated re-read
 * cost. A preview changes nothing; it is read when the switch panel opens.
 */
export function useRunSwitchPreview(modeId: string) {
  return useQuery({
    queryKey: ["usage", "modes", "switch-preview", modeId] as const,
    queryFn: () => measuresClient.rebaseUsageModeRuns({ modeId, runIds: [], dryRun: true }, CALL_OPTIONS),
    staleTime: 0,
    gcTime: 0,
    retry: false,
  });
}

/** The explicit switch: managed runs change at their next resume; every run gets a result. */
export function useRunSwitch() {
  return useMutation({
    mutationFn: (request: { modeId: string; runIds: string[] }) => measuresClient.rebaseUsageModeRuns({ ...request, dryRun: false }, CALL_OPTIONS),
  });
}

/** Every Learn comparison with its arms, verdict and recommendation: the same call as `usage experiment status`. */
export function useUsageExperiments() {
  return useQuery({
    queryKey: usageQueryKeys.experiments,
    queryFn: () => measuresClient.usageExperiment({}, CALL_OPTIONS),
    refetchInterval: 5 * REFRESH_MS,
    retry: false,
  });
}

/** The expected time to a verdict for one role, read before anything starts. */
export function useExperimentPlan(runner: string, role: string, split: number) {
  return useQuery({
    queryKey: usageQueryKeys.experimentPlan(runner, role, split),
    queryFn: () => measuresClient.planUsageExperiment({ runner, role, splitPercent: split }, CALL_OPTIONS),
    staleTime: 5 * REFRESH_MS,
    retry: false,
  });
}

/** Start a comparison and ask for a role-default approval. Both are operator writes the API enforces. */
export function useExperimentActions() {
  const queryClient = useQueryClient();
  const refresh = () => Promise.all([queryClient.invalidateQueries({ queryKey: usageQueryKeys.experiments }), queryClient.invalidateQueries({ queryKey: usageQueryKeys.modes })]);
  const start = useMutation({
    mutationFn: (request: { runner: string; role: string; split: number; sessions: boolean }) =>
      measuresClient.startUsageExperiment({ runner: request.runner, role: request.role, splitPercent: request.split, sessions: request.sessions, until: "off" }, CALL_OPTIONS),
    onSuccess: refresh,
  });
  const propose = useMutation({ mutationFn: (modeId: string) => measuresClient.proposeRoleDefault({ modeId }, CALL_OPTIONS), onSuccess: refresh });
  return { start, propose };
}
