/**
 * external-links
 *
 * Canonical UI seam for building cross-scenario URLs. Every place in the UI
 * that needs to deep-link into agent-manager, prompt-manager, etc. consumes
 * one of these hooks — never builds the URL inline. Bases come from
 * `useEmbeddedServiceUrl(<service>)`, which calls the backend
 * `/embedded/{service}/external-url` seam, so client code never guesses
 * origins (per `feedback_scenario_url_resolution`).
 *
 * Each hook returns `null` while the embedded-service URL is loading or the
 * service is unavailable — call sites should gate the link on `!== null` and
 * either hide the affordance or render a disabled state.
 */

import { useMemo } from "react";
import { useEmbeddedServiceUrl } from "../hooks/useEmbeddedServiceUrl";

function trimTrailingSlash(value: string): string {
  return value.endsWith("/") ? value.slice(0, -1) : value;
}

function joinUrl(base: string | null | undefined, path: string): string | null {
  if (!base) return null;
  return `${trimTrailingSlash(base)}${path}`;
}

/**
 * Pure URL builder for an agent-manager run, used by call sites that already
 * have the agent-manager base URL in scope (typically passed down as a prop
 * from a parent that called `useEmbeddedServiceUrl`). New code should prefer
 * the `useAgentRunUrl` hook below; this helper keeps the URL shape in one
 * place for legacy prop-threaded sites.
 *
 * Returns null when either base or runId is missing.
 */
export function buildAgentRunUrl(
  agentManagerUiUrl: string | null | undefined,
  runId: string | null | undefined,
): string | null {
  if (!runId) return null;
  return joinUrl(agentManagerUiUrl, `/runs/${encodeURIComponent(runId)}`);
}

/**
 * Pure URL builder for an agent-manager profile (deep-link via ?profileKey=).
 * Returns null when either base or profileKey is missing.
 */
export function buildAgentProfileUrl(
  agentManagerUiUrl: string | null | undefined,
  profileKey: string | null | undefined,
): string | null {
  if (!profileKey) return null;
  return joinUrl(agentManagerUiUrl, `/profiles?profileKey=${encodeURIComponent(profileKey)}`);
}

/**
 * Pure URL builder for a prompt-manager skill detail page.
 * Returns null when either base or skillId is missing.
 */
export function buildSkillUrl(
  promptManagerUiUrl: string | null | undefined,
  skillId: string | null | undefined,
): string | null {
  if (!skillId) return null;
  return joinUrl(promptManagerUiUrl, `/skills/${encodeURIComponent(skillId)}`);
}

/**
 * Resolve the agent-manager UI URL for a single run.
 * Returns null when runId is missing or the agent-manager service URL has not resolved.
 */
export function useAgentRunUrl(runId: string | null | undefined): string | null {
  const { url } = useEmbeddedServiceUrl("agent-manager");
  return useMemo(() => buildAgentRunUrl(url, runId), [url, runId]);
}

/**
 * Resolve the agent-manager UI URL for a profile, deep-linked via ?profileKey=.
 * Returns null when profileKey is missing or the agent-manager service URL has not resolved.
 */
export function useAgentProfileUrl(profileKey: string | null | undefined): string | null {
  const { url } = useEmbeddedServiceUrl("agent-manager");
  return useMemo(() => buildAgentProfileUrl(url, profileKey), [url, profileKey]);
}

/**
 * Resolve the prompt-manager UI URL for a single skill.
 * Returns null when skillId is missing or the prompt-manager service URL has not resolved.
 */
export function useSkillUrl(skillId: string | null | undefined): string | null {
  const { url } = useEmbeddedServiceUrl("prompt-manager");
  return useMemo(() => buildSkillUrl(url, skillId), [url, skillId]);
}
