import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

export type CompositionEvidence = { path: string; line: number; sha256: string };
export type CompositionResult = {
  id: string;
  path: string;
  title: string;
  text: string;
  score: number;
  evidence: CompositionEvidence[];
  attestation: "DERIVED" | "PARTIAL";
};
export type CompositionResponse = {
  results: CompositionResult[];
  degradations: { code: string; message: string }[];
  scope: string;
};

const restBase = resolveApiBase({ appendSuffix: true });

export async function searchComposition(query: string, scenario: string): Promise<CompositionResponse> {
  const response = await fetch(buildApiUrl("/api/v1/cli-composition/search", { baseUrl: restBase }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    body: JSON.stringify({ query, scenario }),
  });
  if (!response.ok) throw new Error(`CLI composition search failed (HTTP ${response.status})`);
  return (await response.json()) as CompositionResponse;
}
