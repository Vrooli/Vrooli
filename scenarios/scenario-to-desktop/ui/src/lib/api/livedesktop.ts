import type { JsonObject } from "@bufbuild/protobuf";
import type {
  DesktopControlResponse,
  DesktopSession,
  DesktopSessionRequest,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/evidence_pb";
import { evidenceConnectClient } from "./connect";
import { buildUrl } from "./client";

// ============================================================================
// Types
// ============================================================================

export type { DesktopSession };
export type DesktopSessionConfig = Pick<
  DesktopSessionRequest,
  "scenarioName" | "artifactPath" | "platform" | "width" | "height" | "target"
>;

export type ConnectionStatus =
  | "disconnected"
  | "connecting"
  | "connected"
  | "error";

// ============================================================================
// API Functions
// ============================================================================

export async function startDesktopSession(
  config: DesktopSessionConfig,
): Promise<DesktopSession> {
  return evidenceConnectClient.startDesktopSession(config);
}

export async function stopDesktopSession(id: string): Promise<void> {
  await evidenceConnectClient.stopDesktopSession({ sessionId: id });
}

export async function heartbeatSession(id: string): Promise<void> {
  await evidenceConnectClient.heartbeatDesktopSession({ sessionId: id });
}

export async function getDesktopSession(id: string): Promise<DesktopSession> {
  return evidenceConnectClient.getDesktopSession({ sessionId: id });
}

export async function listDesktopSessions(): Promise<DesktopSession[]> {
  const response = await evidenceConnectClient.listDesktopSessions({});
  return response.sessions;
}

export async function launchAppOnDesktop(
  id: string,
  appPath?: string,
): Promise<void> {
  await evidenceConnectClient.launchDesktopArtifact({
    sessionId: id,
    artifactPath: appPath,
  });
}

export async function findArtifact(id: string): Promise<string> {
  const session = await evidenceConnectClient.getDesktopSession({
    sessionId: id,
  });
  return (
    await evidenceConnectClient.findDesktopArtifact({
      scenarioName: session.scenarioName,
    })
  ).artifactPath;
}

// ============================================================================
// Control Actions
// ============================================================================

export interface ControlRequest {
  action: string;
  params?: Record<string, unknown>;
}

export type ControlResult = DesktopControlResponse;

export function controlResultString(
  result: DesktopControlResponse,
  field: string,
): string | undefined {
  const fields = result.result?.fields;
  if (!fields || Array.isArray(fields) || typeof fields !== "object") {
    return undefined;
  }
  const value = fields[field];
  return typeof value === "string" ? value : undefined;
}

export async function executeDesktopControl(
  id: string,
  req: ControlRequest,
): Promise<ControlResult> {
  const response = await evidenceConnectClient.controlDesktop({
    sessionId: id,
    action: req.action,
    params: req.params as JsonObject | undefined,
  });
  return response;
}

/**
 * Build the WebSocket URL for the VNC proxy endpoint.
 * Uses the same origin as the API, switching protocol to ws/wss.
 */
export function buildVncWsUrl(sessionId: string): string {
  const apiUrl = buildUrl(
    `/livedesktop/sessions/${encodeURIComponent(sessionId)}/ws`,
  );
  return apiUrl.replace(/^http/, "ws");
}
