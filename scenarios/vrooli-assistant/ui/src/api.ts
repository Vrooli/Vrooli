export interface CaptureIssuePayload {
  description: string;
  scenarioName: string;
  url: string;
  screenshotPath?: string;
  tags: string[];
  contextData: Record<string, unknown>;
  agentType: AgentType;
  spawnAgent: boolean;
}

export type AgentType = 'none' | 'claude-code' | 'agent-s2' | 'agent-s3';

export interface CaptureIssueResponse {
  issue_id: string;
  status: string;
  message: string;
}

export interface HistoryIssueSummary {
  id: string;
  timestamp: string;
  scenario_name: string;
  description: string;
  status: string;
  agent_session_id?: string;
}

export interface IssueDetail extends HistoryIssueSummary {
  screenshot_path?: string;
  url?: string;
  resolution_notes?: string;
  tags?: string[];
  context_data?: Record<string, unknown>;
}

export interface AssistantStatus {
  status: string;
  issues_captured: number;
  agents_spawned: number;
  uptime: string;
}

function resolveProxyAwareBaseUrl(): string {
  if (typeof window === 'undefined') {
    return '';
  }

  const { origin, pathname } = window.location;
  const segments = pathname.split('/').filter(Boolean);
  const proxyIndex = segments.indexOf('proxy');

  if (proxyIndex !== -1) {
    const scopedPath = segments.slice(0, proxyIndex + 1).join('/');
    return `${origin}/${scopedPath}`;
  }

  return origin;
}

const API_BASE_URL = (typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.VITE_API_BASE_URL)
  ? (import.meta.env.VITE_API_BASE_URL as string)
  : resolveProxyAwareBaseUrl();

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Request failed with status ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export async function captureIssue(payload: CaptureIssuePayload): Promise<CaptureIssueResponse> {
  const body = {
    description: payload.description,
    scenario_name: payload.scenarioName,
    url: payload.url,
    screenshot_path: payload.screenshotPath ?? '',
    tags: payload.tags,
    context_data: {
      submitted_via: 'web-ui',
      agent_type: payload.agentType,
      ...payload.contextData,
    },
  };

  return handleResponse<CaptureIssueResponse>(
    await fetch(`${API_BASE_URL}/api/v1/assistant/capture`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  );
}

export async function spawnAgent(issueId: string, agentType: Exclude<AgentType, 'none'>, description: string): Promise<void> {
  const body = {
    issue_id: issueId,
    agent_type: agentType,
    description,
    context: {
      requested_via: 'web-ui',
    },
    screenshot: '',
  };

  await handleResponse(await fetch(`${API_BASE_URL}/api/v1/assistant/spawn-agent`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }));
}

export async function fetchHistory(): Promise<HistoryIssueSummary[]> {
  const response = await handleResponse<{ issues: HistoryIssueSummary[] }>(
    await fetch(`${API_BASE_URL}/api/v1/assistant/history`),
  );
  return response.issues;
}

export async function fetchIssueDetail(issueId: string): Promise<IssueDetail> {
  return handleResponse<IssueDetail>(
    await fetch(`${API_BASE_URL}/api/v1/assistant/issues/${issueId}`),
  );
}

export async function fetchAssistantStatus(): Promise<AssistantStatus> {
  return handleResponse<AssistantStatus>(
    await fetch(`${API_BASE_URL}/api/v1/assistant/status`),
  );
}

export async function updateIssueStatus(issueId: string, status: string): Promise<void> {
  await handleResponse(
    await fetch(`${API_BASE_URL}/api/v1/assistant/issues/${issueId}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status }),
    }),
  );
}
