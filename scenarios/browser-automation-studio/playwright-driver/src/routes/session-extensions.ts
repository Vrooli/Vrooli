import type { IncomingMessage, ServerResponse } from 'http';
import type { Config } from '../config';
import type { SessionManager } from '../session';
import type { SessionState } from '../types/session';
import { parseJsonBody, sendError, sendJson } from '../middleware';
import {
  installManagedExtension,
  inventoryManagedExtensions,
  loadManagedExtension,
  reloadManagedExtension,
  unloadManagedExtension,
  validateManagedExtension,
  type ManagedExtensionArtifact,
  type ManagedExtensionInventoryArtifact,
} from '../session/managed-extensions';
import { requireSessionLease } from './session-lease';

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string');
}

function parseInstallRequest(body: Record<string, unknown>): { directory: string; artifact: ManagedExtensionArtifact } {
  const directory = typeof body.directory_path === 'string' ? body.directory_path.trim() : '';
  const artifact = body.artifact as Partial<ManagedExtensionArtifact> | undefined;
  if (
    !directory ||
    !artifact ||
    typeof artifact.extension_id !== 'string' ||
    typeof artifact.artifact_ref !== 'string' ||
    !/^[a-f0-9]{64}$/.test(artifact.artifact_sha256 ?? '') ||
    !/^[a-f0-9]{64}$/.test(artifact.manifest_sha256 ?? '') ||
    typeof artifact.version !== 'string' ||
    !isStringArray(artifact.permissions) ||
    !isStringArray(artifact.host_permissions)
  ) {
    throw new Error('managed extension artifact request is invalid');
  }
  if (artifact.permission_probe !== undefined && artifact.permission_probe !== 'chrome.permissions.getAll.v1') {
    throw new Error('managed extension permission probe is unsupported');
  }
  return { directory, artifact: artifact as ManagedExtensionArtifact };
}

function parseInventoryRequest(body: Record<string, unknown>): { sessionID: string; profileID: string; artifacts: ManagedExtensionInventoryArtifact[] } {
  const sessionID = typeof body.session_id === 'string' ? body.session_id.trim() : '';
  const profileID = typeof body.profile_id === 'string' ? body.profile_id.trim() : '';
  if (!sessionID || !profileID || !Array.isArray(body.artifacts) || body.artifacts.length === 0) {
    throw new Error('managed extension inventory request must bind the session, profile, and staged artifacts');
  }
  const artifacts = body.artifacts.map((value) => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('managed extension inventory artifact is invalid');
    const parsed = parseInstallRequest(value as Record<string, unknown>);
    return { directory_path: parsed.directory, artifact: parsed.artifact };
  });
  return { sessionID, profileID, artifacts };
}

async function withExtensionOperation<T>(session: SessionState, operation: () => Promise<T>): Promise<T> {
  if (session.extensionOperationInFlight || session.instructionInFlight || session.phase !== 'ready') {
    throw new Error('BAS session is busy and cannot perform an extension operation');
  }
  session.extensionOperationInFlight = true;
  try {
    return await operation();
  } finally {
    session.extensionOperationInFlight = false;
  }
}

async function handleLifecycle(
  req: IncomingMessage,
  res: ServerResponse,
  sessionId: string,
  sessionManager: SessionManager,
  config: Config,
  operation: 'install' | 'load' | 'reload' | 'unload' | 'validate'
): Promise<void> {
  try {
    const body = await parseJsonBody(req, config);
    const owner = requireSessionLease(req, res, sessionId, sessionManager);
    if (!owner) return;
    const { directory, artifact } = parseInstallRequest(body);
    const { session } = owner;
    const result = await withExtensionOperation(session, async () => {
      switch (operation) {
        case 'install': return installManagedExtension(session, directory, artifact);
        case 'load': return loadManagedExtension(session, directory, artifact);
        case 'reload': return reloadManagedExtension(session, directory, artifact);
        case 'unload': return unloadManagedExtension(session, directory, artifact);
        case 'validate': return validateManagedExtension(session, directory, artifact);
      }
    });
    sendJson(res, 200, { session_id: sessionId, extension: result ?? null });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/extensions/${operation}`);
  }
}

export function handleSessionExtensionInstall(req: IncomingMessage, res: ServerResponse, sessionId: string, sessions: SessionManager, config: Config): Promise<void> {
  return handleLifecycle(req, res, sessionId, sessions, config, 'install');
}

export function handleSessionExtensionLoad(req: IncomingMessage, res: ServerResponse, sessionId: string, sessions: SessionManager, config: Config): Promise<void> {
  return handleLifecycle(req, res, sessionId, sessions, config, 'load');
}

export function handleSessionExtensionReload(req: IncomingMessage, res: ServerResponse, sessionId: string, sessions: SessionManager, config: Config): Promise<void> {
  return handleLifecycle(req, res, sessionId, sessions, config, 'reload');
}

export function handleSessionExtensionUnload(req: IncomingMessage, res: ServerResponse, sessionId: string, sessions: SessionManager, config: Config): Promise<void> {
  return handleLifecycle(req, res, sessionId, sessions, config, 'unload');
}

export function handleSessionExtensionValidate(req: IncomingMessage, res: ServerResponse, sessionId: string, sessions: SessionManager, config: Config): Promise<void> {
  return handleLifecycle(req, res, sessionId, sessions, config, 'validate');
}

export async function handleSessionExtensionInventory(req: IncomingMessage, res: ServerResponse, sessionId: string, sessions: SessionManager, config: Config): Promise<void> {
  try {
    const owner = requireSessionLease(req, res, sessionId, sessions);
    if (!owner) return;
    let extensions;
    if (req.method === 'POST') {
      const body = await parseJsonBody(req, config);
      const inventory = parseInventoryRequest(body);
      if (inventory.sessionID !== sessionId || inventory.sessionID !== owner.session.id) {
        throw new Error('managed extension inventory request session does not match the driver lease');
      }
      extensions = await withExtensionOperation(owner.session, () => inventoryManagedExtensions(owner.session, inventory.artifacts, inventory.sessionID, inventory.profileID));
    } else {
      extensions = await withExtensionOperation(owner.session, () => inventoryManagedExtensions(
        owner.session, [], owner.session.id, owner.session.spec.permission_profile?.profile_id ?? '',
      ));
    }
    sendJson(res, 200, { session_id: sessionId, extensions });
  } catch (error) {
    sendError(res, error as Error, `/session/${sessionId}/extensions/inventory`);
  }
}
