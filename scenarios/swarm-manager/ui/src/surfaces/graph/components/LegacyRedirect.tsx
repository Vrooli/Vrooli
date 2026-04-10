/**
 * LegacyRedirect - Client-side redirects from old tabbed routes to /graph.
 */

import { Navigate, useParams } from "react-router-dom";

/**
 * Redirect /backlog → /graph?lens=topology
 */
export function BacklogRedirect() {
  return <Navigate to="/graph?lens=topology" replace />;
}

/**
 * Redirect /backlog/:kind/:name or /details/backlog/:kind/:name → /graph with detail params
 */
export function BacklogDetailsRedirect() {
  const { kind, name } = useParams<{ kind: string; name: string }>();
  return <Navigate to={`/graph?detail=backlog&kind=${kind}&name=${name}`} replace />;
}

/**
 * Redirect /scenarios → /graph?lens=topology
 */
export function ScenariosRedirect() {
  return <Navigate to="/graph?lens=topology" replace />;
}

/**
 * Redirect /scenarios/:name or /details/scenario/:name → /graph with detail params
 */
export function ScenarioDetailsRedirect() {
  const { name } = useParams<{ name: string }>();
  return <Navigate to={`/graph?detail=scenario&name=${name}`} replace />;
}

/**
 * Redirect /execution → /graph?lens=flow
 */
export function ExecutionRedirect() {
  return <Navigate to="/graph?lens=flow" replace />;
}

/**
 * Redirect /prompts → /graph?lens=topology (prompts accessible via settings drawer)
 */
export function PromptsRedirect() {
  return <Navigate to="/graph?lens=topology" replace />;
}

/**
 * Redirect /settings → /graph?lens=topology (settings accessible via gear icon)
 */
export function SettingsRedirect() {
  return <Navigate to="/graph?lens=topology" replace />;
}
