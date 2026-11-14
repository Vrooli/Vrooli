import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE } from "./client";

export type RampTarget = { id?: string; label?: string; platform?: string; device_kind?: string; available?: boolean; reason?: string; next_action?: string; transport?: { kind?: string; id?: string; available?: boolean; reason?: string } };
export async function getRampJson<T>(path: string): Promise<T> { const response = await fetch(buildApiUrl(path, { baseUrl: `${API_BASE}/api/v1` }), { cache: "no-store" }); if (!response.ok) throw new Error(`Android ramp request failed (${response.status})`); return (await response.json()) as T; }
export function targetsFromPayload(payload: unknown): RampTarget[] { if (Array.isArray(payload)) return payload as RampTarget[]; if (payload && typeof payload === "object" && Array.isArray((payload as { targets?: unknown }).targets)) return (payload as { targets: RampTarget[] }).targets; return []; }
