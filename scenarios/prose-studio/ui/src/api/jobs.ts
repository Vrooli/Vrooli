import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE, decodeApiError } from "./client";

export type Opening = {
  id: string;
  title: string;
  company: string;
  location?: string;
  url?: string;
  category?: string;
  status?: string;
  source?: string;
  description?: string;
};

export type Fact = { id: string; label: string; content: string; tags?: string[] };
export type ReviewFinding = string | { severity?: string; claim?: string; reason?: string; evidence?: string };
export type ResumeVariant = { id: string; title: string; opening_id?: string; content: string; fact_ids: string[]; review_findings?: ReviewFinding[] };
export type CoverLetter = { id: string; title: string; opening_id?: string; content: string; fact_ids: string[] };
export type SearchResult = { source: string; query: string; url?: string; status: string; openings?: Opening[]; error?: string };

async function request<T>(path: string, method: "GET" | "POST" | "PATCH", body?: unknown): Promise<T> {
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    method,
    headers: body === undefined ? undefined : { "content-type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as T;
}

export const jobsApi = {
  openings: () => request<{ openings: Opening[] }>("/api/v1/jobs/openings", "GET"),
  addOpening: (opening: Omit<Opening, "id">) => request<{ opening: Opening }>("/api/v1/jobs/openings", "POST", opening),
  updateOpening: (id: string, update: Partial<Opening>) => request<{ opening: Opening }>(`/api/v1/jobs/openings/${encodeURIComponent(id)}`, "PATCH", update),
  facts: () => request<{ facts: Fact[] }>("/api/v1/jobs/facts", "GET"),
  addFact: (fact: Omit<Fact, "id">) => request<{ fact: Fact }>("/api/v1/jobs/facts", "POST", fact),
  resumes: () => request<{ resumes: ResumeVariant[] }>("/api/v1/jobs/resumes", "GET"),
  addResume: (resume: Omit<ResumeVariant, "id" | "review_findings">) => request<{ resume: ResumeVariant }>("/api/v1/jobs/resumes", "POST", resume),
  reviewResume: (id: string) => request<{ resume: ResumeVariant }>(`/api/v1/jobs/resumes/${encodeURIComponent(id)}/review`, "POST", {}),
  letters: () => request<{ cover_letters: CoverLetter[] }>("/api/v1/jobs/cover-letters", "GET"),
  addLetter: (letter: Omit<CoverLetter, "id">) => request<{ cover_letter: CoverLetter }>("/api/v1/jobs/cover-letters", "POST", letter),
  sources: () => request<{ sources: Array<{ name: string; query: string; url?: string; available: boolean }> }>("/api/v1/jobs/sources", "GET"),
  discover: (query: string) => request<{ results: SearchResult[]; partial: boolean }>("/api/v1/jobs/discover", "POST", { query }),
};
