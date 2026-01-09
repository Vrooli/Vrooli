import { useQuery } from "@tanstack/react-query";
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export interface DocSection {
  id: string;
  title: string;
  visibility?: string;
  documents: Array<{
    path: string;
    title: string;
  }>;
}

export interface DocsManifest {
  version: string;
  title: string;
  defaultDocument: string;
  sections: DocSection[];
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
  return res.json();
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
  const data = await res.json();
  return data.content ?? "";
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
