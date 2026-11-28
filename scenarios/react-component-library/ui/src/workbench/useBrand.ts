import { useQuery } from "@tanstack/react-query";

import { API_BASE } from "../api/client";

export interface Brand {
  /** True only when Brand Manager has a record assigned to this scenario. */
  assigned: boolean;
  name: string;
  /** Absolute URL or data URI. */
  mark: string | null;
  /** CSS colour, used for accents inside the shell. */
  accent: string | null;
  wordmark: string | null;
}

const FALLBACK: Brand = {
  assigned: false,
  name: "React Component Library",
  mark: null,
  accent: null,
  wordmark: null,
};

/**
 * The shell's identity comes from Brand Manager, never from a literal in this
 * repository. When no brand is assigned the shell falls back to a neutral
 * monogram and says so — it does not invent a logo.
 */
export function useBrand() {
  return useQuery({
    queryKey: ["brand", "react-component-library"],
    queryFn: async (): Promise<Brand> => {
      const response = await fetch(`${API_BASE.replace(/\/$/, "")}/api/brand`, {
        headers: { accept: "application/json" },
      });
      if (!response.ok) return FALLBACK;
      const body = (await response.json()) as Partial<Brand> | null;
      const name = body?.name?.trim();
      return {
        assigned: Boolean(body?.assigned && name),
        name: name || FALLBACK.name,
        mark: body?.mark ?? null,
        accent: body?.accent ?? null,
        wordmark: body?.wordmark ?? null,
      };
    },
    staleTime: 5 * 60_000,
    retry: false,
  });
}
