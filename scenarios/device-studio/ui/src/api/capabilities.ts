import { getFile } from "./client";

export type CapabilityAvailability = {
  id: string;
  name: string;
  status: "available" | "unavailable" | "unknown";
  message?: string;
  checkedAt?: string;
  actionLabel?: string;
  operatorCommand?: string;
};

export async function fetchOptionalCapabilities(): Promise<CapabilityAvailability[]> {
  const response = await getFile("/capabilities/describe");
  if (!response.ok) throw new Error("capability availability unavailable");
  const payload = await response.json() as { states?: CapabilityAvailability[] };
  if (!Array.isArray(payload.states)) throw new Error("capability availability response invalid");
  return payload.states.filter((state) => state.id === "ai-gateway" || state.id === "workspace-sandbox");
}
