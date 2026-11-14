import { useQuery } from "@tanstack/react-query";

import { restUrl } from "./client";

/**
 * The template capability registry's plain-JSON probe (a REST ops exception):
 * which optional scenarios the desk can reach and what each one is used for.
 */
export interface Capability {
  id: string;
  name: string;
  description: string;
  status: string;
  message?: string;
  checkedAt?: string;
  uses?: { id: string; name: string; description: string }[];
}

async function fetchCapabilities(): Promise<Capability[]> {
  const res = await fetch(restUrl("/capabilities/describe"), { cache: "no-store" });
  if (!res.ok) throw new Error(`capabilities: HTTP ${res.status}`);
  const body = (await res.json()) as { states?: Capability[] };
  return body.states ?? [];
}

export function useCapabilities() {
  return useQuery({ queryKey: ["desk", "capabilities"], queryFn: fetchCapabilities, refetchInterval: 60_000 });
}

/** A research-home text file through the read-only file route. */
export function useHomeText(url: string | null) {
  return useQuery({
    queryKey: ["desk", "file", url],
    enabled: url !== null,
    queryFn: async () => {
      const res = await fetch(url ?? "", { cache: "no-store" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return res.text();
    },
  });
}
