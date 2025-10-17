import { formatFileSize } from './files';
import { buildIssueSnapshot } from './issues';
import type { AgentSettings, Issue, IssueAttachment } from '../data/sampleData';
import type {
  CreateIssueInitialFields,
  CreateIssuePrefill,
  LockedAttachmentDraft,
} from '../types/issueCreation';

export interface AgentSettingsSnapshot {
  provider: string;
  autoFallback: boolean;
  maximumTurns: number;
  taskTimeoutMinutes: number;
  allowedToolsKey: string;
  skipPermissionChecks: boolean;
}

export function buildAgentSettingsSnapshot(
  settings: AgentSettings,
  allowedToolsKey: string,
): AgentSettingsSnapshot {
  return {
    provider: settings.backend?.provider ?? 'codex',
    autoFallback: settings.backend?.autoFallback ?? true,
    maximumTurns: settings.maximumTurns,
    taskTimeoutMinutes: settings.taskTimeout,
    allowedToolsKey,
    skipPermissionChecks: settings.skipPermissionChecks,
  };
}

export function agentSettingsSnapshotsEqual(
  previous: AgentSettingsSnapshot | null,
  next: AgentSettingsSnapshot,
): boolean {
  if (!previous) {
    return false;
  }
  return (
    previous.provider === next.provider &&
    previous.autoFallback === next.autoFallback &&
    previous.maximumTurns === next.maximumTurns &&
    previous.taskTimeoutMinutes === next.taskTimeoutMinutes &&
    previous.allowedToolsKey === next.allowedToolsKey &&
    previous.skipPermissionChecks === next.skipPermissionChecks
  );
}

function createLocalId(prefix: string): string {
  try {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
      return `${prefix}-${crypto.randomUUID()}`;
    }
  } catch {
    // Fall through to Math.random fallback
  }
  return `${prefix}-${Math.random().toString(16).slice(2, 10)}`;
}

function encodeBytesToBase64(bytes: Uint8Array): string {
  if (typeof window !== 'undefined' && typeof window.btoa === 'function') {
    let binary = '';
    const chunkSize = 0x8000;
    for (let offset = 0; offset < bytes.length; offset += chunkSize) {
      const chunk = bytes.subarray(offset, Math.min(offset + chunkSize, bytes.length));
      binary += String.fromCharCode(...chunk);
    }
    return window.btoa(binary);
  }

  const bufferCtor = (globalThis as Record<string, unknown>).Buffer as
    | { from: (input: Uint8Array) => { toString: (encoding: string) => string } }
    | undefined;
  if (bufferCtor?.from) {
    return bufferCtor.from(bytes).toString('base64');
  }

  let fallbackBinary = '';
  for (let index = 0; index < bytes.length; index += 1) {
    fallbackBinary += String.fromCharCode(bytes[index]);
  }

  const btoaFn = typeof btoa === 'function' ? btoa : undefined;
  if (btoaFn) {
    return btoaFn(fallbackBinary);
  }

  throw new Error('Base64 encoding is not supported in this environment.');
}

function encodeStringToBase64(value: string): { base64: string; byteLength: number } {
  const encoder = new TextEncoder();
  const bytes = encoder.encode(value);
  return {
    base64: encodeBytesToBase64(bytes),
    byteLength: bytes.length,
  };
}

async function cloneIssueAttachment(attachment: IssueAttachment): Promise<LockedAttachmentDraft> {
  const response = await fetch(attachment.url);
  if (!response.ok) {
    throw new Error(`request failed with status ${response.status}`);
  }

  const arrayBuffer = await response.arrayBuffer();
  const bytes = new Uint8Array(arrayBuffer);
  const base64 = encodeBytesToBase64(bytes);
  const fallbackName = attachment.path.split(/[\\/]+/).pop() ?? 'attachment';
  const name = attachment.name || fallbackName;
  const responseContentType = response.headers.get('Content-Type');
  const contentType = (attachment.type ?? responseContentType ?? 'application/octet-stream').split(';')[0];

  return {
    id: createLocalId('attachment'),
    payload: {
      name,
      content: base64,
      encoding: 'base64',
      contentType,
      category: attachment.category ?? 'attachment',
    },
    name,
    description: 'Attachment from previous issue',
    sizeLabel: formatFileSize(typeof attachment.size === 'number' ? attachment.size : bytes.length),
    category: attachment.category ?? undefined,
  };
}

export async function prepareFollowUpPrefill(issue: Issue): Promise<CreateIssuePrefill> {
  const lockedAttachments: LockedAttachmentDraft[] = [];

  if (issue.attachments.length > 0) {
    try {
      const cloned = await Promise.all(issue.attachments.map(cloneIssueAttachment));
      lockedAttachments.push(...cloned);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      throw new Error(`Failed to copy attachments from ${issue.id}: ${message}`);
    }
  }

  if (issue.investigation?.report) {
    const heading = `# Investigation Report for ${issue.id}\n\n`;
    const { base64, byteLength } = encodeStringToBase64(`${heading}${issue.investigation.report}`);
    const name = `${issue.id}-investigation-report.md`;
    lockedAttachments.push({
      id: createLocalId('report'),
      payload: {
        name,
        content: base64,
        encoding: 'base64',
        contentType: 'text/markdown',
        category: 'investigation',
      },
      name,
      description: 'Investigation findings copied from the previous issue',
      sizeLabel: formatFileSize(byteLength),
      category: 'investigation',
    });
  }

  const snapshotJson = `${JSON.stringify(buildIssueSnapshot(issue), null, 2)}\n`;
  const snapshot = encodeStringToBase64(snapshotJson);
  const snapshotName = `${issue.id}-snapshot.json`;
  lockedAttachments.push({
    id: createLocalId('snapshot'),
    payload: {
      name: snapshotName,
      content: snapshot.base64,
      encoding: 'base64',
      contentType: 'application/json',
      category: 'snapshot',
    },
    name: snapshotName,
    description: 'Structured JSON snapshot of the previous issue',
    sizeLabel: formatFileSize(snapshot.byteLength),
    category: 'snapshot',
  });

  const uniqueTags = Array.from(new Set(['follow-up', ...issue.tags])).filter(Boolean);
  const initial: CreateIssueInitialFields = {
    title: `Follow up: ${issue.title}`,
    description: `Follow-up request for ${issue.id} (${issue.title}).\n\nDescribe the outstanding issues, regressions, or new instructions here.\n`,
    priority: issue.priority ?? 'Medium',
    status: 'open',
    appId: issue.app || 'unknown',
    tags: uniqueTags,
  };

  return {
    key: `follow-up-${issue.id}-${Date.now()}`,
    initial,
    lockedAttachments,
    followUpOf: { id: issue.id, title: issue.title },
  };
}
