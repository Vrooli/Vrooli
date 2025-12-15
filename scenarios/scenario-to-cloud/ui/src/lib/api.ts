import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true });

export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json() as Promise<{ status: string; service: string; timestamp: string }>;
}

export type ValidationIssue = {
  path: string;
  message: string;
  hint?: string;
  severity: "error" | "warn";
};

export type ManifestValidateResponse = {
  valid: boolean;
  issues?: ValidationIssue[];
  timestamp: string;
};

export type PlanStep = { id: string; title: string; description: string };
export type PlanResponse = { plan: PlanStep[]; timestamp: string };

export type BundleArtifact = { path: string; sha256: string; size_bytes: number };
export type BundleBuildResponse = { artifact: BundleArtifact; timestamp: string };

export async function validateManifest(manifest: unknown) {
  const url = buildApiUrl("/manifest/validate", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Manifest validation failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<ManifestValidateResponse>;
}

export async function buildPlan(manifest: unknown) {
  const url = buildApiUrl("/plan", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Plan generation failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<PlanResponse>;
}

export async function buildBundle(manifest: unknown) {
  const url = buildApiUrl("/bundle/build", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Bundle build failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<BundleBuildResponse>;
}
