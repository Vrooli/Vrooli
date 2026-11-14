import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

const restBase = resolveApiBase({ appendSuffix: true });
const previewPath = "/proposal-preview/service-offer";

export type ServiceOfferPreview = {
  offer: {
    id: string;
    name: string;
    summary: string;
    audience: string;
    proposed_price: { amount_minor: number; currency: string };
    delivery: { target_minutes: number; format: string };
    verification_promise: string;
    refund_rule: string;
    legal: { reviewed_at: string; status: string; notes: string[]; evidence: string[] };
    operator_approval: string;
  };
  surfaces: {
    checkout: { mode: string; payment_mode: string; capture_when: string };
    mcp: { name: string; description: string; payment_path: string };
    mpp: { path: string; mode: string; capture_when: string };
    plugin: { disposition: string; landing_path: string };
    page: { free_sample_path: string; publication_state: string };
  };
  sample: string;
};

export async function fetchServiceOfferPreview(): Promise<ServiceOfferPreview> {
  const response = await fetch(buildApiUrl(previewPath, { baseUrl: restBase }), { cache: "no-store" });
  if (!response.ok) throw new Error(`Offer preview unavailable (${response.status})`);
  return response.json() as Promise<ServiceOfferPreview>;
}

export function serviceOfferSampleUrl(): string {
  return buildApiUrl(`${previewPath}/sample`, { baseUrl: restBase });
}
