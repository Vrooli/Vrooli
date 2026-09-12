import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import { getPolicy, type SafetyPolicy } from "../../api/safety";

/** Shared react-query key so every consumer reuses one fetched policy. */
export const SAFETY_POLICY_QUERY_KEY = ["safety-policy"] as const;

/**
 * Fetch the resolved Responsible-Use policy once and share it across the app
 * (the Settings panel, the Create consent gate, smart-select). The policy is a
 * deploy-time server setting, so a single cached read is enough; consumers read
 * `requireConsent` + `opWeights` to decide whether to ask for consent.
 */
export const useSafetyPolicy = (): UseQueryResult<SafetyPolicy> =>
  useQuery({ queryKey: SAFETY_POLICY_QUERY_KEY, queryFn: getPolicy });
