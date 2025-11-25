import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

export interface Campaign {
  id: string;
  name: string;
  from_agent: string;
  description?: string;
  patterns: string[];
  status: "active" | "completed" | "archived";
  total_files: number;
  visited_files: number;
  coverage_percent: number;
  created_at: string;
  updated_at: string;
}

export interface Visit {
  path: string;
  visited_at: string;
  staleness_score: number;
  visit_count: number;
  last_visited?: string;
  last_modified: string;
  notes: string[];
}

export interface CampaignDetail extends Campaign {
  visits: Visit[];
}

export interface CreateCampaignRequest {
  name: string;
  from_agent: string;
  description?: string;
  patterns: string[];
}

export interface RecordVisitRequest {
  paths: string[];
  notes?: string;
}

export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json();
}

export async function fetchCampaigns(): Promise<{ campaigns: Campaign[] }> {
  const url = buildApiUrl("/campaigns", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch campaigns: ${res.status}`);
  }

  return res.json();
}

export async function fetchCampaign(id: string): Promise<CampaignDetail> {
  const url = buildApiUrl(`/campaigns/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch campaign: ${res.status}`);
  }

  return res.json();
}

export async function createCampaign(data: CreateCampaignRequest): Promise<Campaign> {
  const url = buildApiUrl("/campaigns", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data)
  });

  if (!res.ok) {
    if (res.status === 409) {
      throw new Error("A campaign with this name already exists");
    }
    throw new Error(`Failed to create campaign: ${res.status}`);
  }

  return res.json();
}

export async function deleteCampaign(id: string): Promise<void> {
  const url = buildApiUrl(`/campaigns/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE"
  });

  if (!res.ok) {
    throw new Error(`Failed to delete campaign: ${res.status}`);
  }
}

export async function recordVisit(campaignId: string, data: RecordVisitRequest): Promise<void> {
  const url = buildApiUrl(`/campaigns/${campaignId}/visit`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data)
  });

  if (!res.ok) {
    throw new Error(`Failed to record visit: ${res.status}`);
  }
}

export async function fetchLeastVisited(campaignId: string, limit: number = 10): Promise<{ files: Visit[] }> {
  const url = buildApiUrl(`/campaigns/${campaignId}/least-visited?limit=${limit}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch least visited files: ${res.status}`);
  }

  return res.json();
}

export async function fetchMostStale(campaignId: string, limit: number = 10): Promise<{ files: Visit[] }> {
  const url = buildApiUrl(`/campaigns/${campaignId}/most-stale?limit=${limit}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch most stale files: ${res.status}`);
  }

  return res.json();
}
