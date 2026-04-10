import { useQuery } from "@tanstack/react-query";
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export interface DocSection {
  id: string;
  title: string;
  icon?: string;
  description?: string;
  visibility?: string;
  documents: Array<{
    path: string;
    title: string;
    description?: string;
  }>;
}

export interface DocsManifest {
  version: string;
  title: string;
  description?: string;
  defaultDocument: string;
  sections: DocSection[];
}

function isDocsManifest(value: unknown): value is DocsManifest {
  if (!value || typeof value !== "object") return false;
  const record = value as Record<string, unknown>;
  return (
    typeof record.version === "string" &&
    typeof record.title === "string" &&
    typeof record.defaultDocument === "string" &&
    Array.isArray(record.sections)
  );
}

async function fetchDocsManifest(): Promise<DocsManifest> {
  const url = buildApiUrl("/docs/manifest", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    throw new Error(`Failed to fetch docs manifest: ${res.status}`);
  }
  const data: unknown = await res.json();
  if (!isDocsManifest(data)) {
    throw new Error("Invalid docs manifest response");
  }
  return data;
}

async function fetchDocContent(path: string): Promise<string> {
  const url = buildApiUrl(`/docs/content?path=${encodeURIComponent(path)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    throw new Error(`Failed to fetch doc content: ${res.status}`);
  }
  const data: unknown = await res.json();
  if (!data || typeof data !== "object") {
    return "";
  }
  const content = (data as Record<string, unknown>).content;
  return typeof content === "string" ? content : "";
}

export function useDocsManifest() {
  return useQuery<DocsManifest>({
    queryKey: ["docs-manifest"],
    queryFn: fetchDocsManifest,
    staleTime: 5 * 60 * 1000 // 5 minutes
  });
}

export function useDocContent(path: string | null) {
  return useQuery<string>({
    queryKey: ["doc-content", path],
    queryFn: () => (path ? fetchDocContent(path) : Promise.resolve("")),
    enabled: Boolean(path),
    staleTime: 2 * 60 * 1000 // 2 minutes
  });
}
