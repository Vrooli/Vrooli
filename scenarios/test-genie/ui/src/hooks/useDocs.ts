import { useQuery } from "@tanstack/react-query";
import { buildTestGenieApiUrl } from "../lib/api";

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

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isDocSection(value: unknown): value is DocSection {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.title === "string" &&
    (value.visibility === undefined || typeof value.visibility === "string") &&
    Array.isArray(value.documents) &&
    value.documents.every(
      (document) =>
        isRecord(document) &&
        typeof document.path === "string" &&
        typeof document.title === "string"
    )
  );
}

function isDocsManifest(value: unknown): value is DocsManifest {
  return (
    isRecord(value) &&
    typeof value.version === "string" &&
    typeof value.title === "string" &&
    typeof value.defaultDocument === "string" &&
    Array.isArray(value.sections) &&
    value.sections.every(isDocSection)
  );
}

async function fetchDocsManifest(): Promise<DocsManifest> {
  const url = buildTestGenieApiUrl("/docs/manifest");
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    throw new Error(`Failed to fetch docs manifest: ${res.status}`);
  }
  const payload: unknown = await res.json();
  if (!isDocsManifest(payload)) {
    throw new Error("Received invalid docs manifest payload");
  }
  return payload;
}

async function fetchDocContent(path: string): Promise<string> {
  const url = buildTestGenieApiUrl(`/docs/content?path=${encodeURIComponent(path)}`);
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    throw new Error(`Failed to fetch doc content: ${res.status}`);
  }
  const payload: unknown = await res.json();
  if (!isRecord(payload)) {
    return "";
  }
  return typeof payload.content === "string" ? payload.content : "";
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
