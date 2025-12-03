// Context for sharing check metadata (titles, descriptions) across components
// This allows any component to look up human-friendly names for checkIds
import { createContext, useContext, useMemo, ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchChecks, CheckInfo } from "../lib/api";

interface CheckMetadataContextValue {
  /** Get human-friendly title for a checkId, falls back to checkId if not found */
  getTitle: (checkId: string) => string;
  /** Get full metadata for a checkId */
  getMetadata: (checkId: string) => CheckInfo | undefined;
  /** Whether metadata is still loading */
  isLoading: boolean;
}

const CheckMetadataContext = createContext<CheckMetadataContextValue | null>(null);

export function CheckMetadataProvider({ children }: { children: ReactNode }) {
  const { data: checks, isLoading } = useQuery({
    queryKey: ["checks-metadata"],
    queryFn: fetchChecks,
    staleTime: 60000, // Cache for 60s since metadata rarely changes
  });

  const metadataMap = useMemo(() => {
    const map = new Map<string, CheckInfo>();
    if (checks) {
      for (const check of checks) {
        map.set(check.id, check);
      }
    }
    return map;
  }, [checks]);

  const value = useMemo<CheckMetadataContextValue>(() => ({
    getTitle: (checkId: string) => metadataMap.get(checkId)?.title ?? checkId,
    getMetadata: (checkId: string) => metadataMap.get(checkId),
    isLoading,
  }), [metadataMap, isLoading]);

  return (
    <CheckMetadataContext.Provider value={value}>
      {children}
    </CheckMetadataContext.Provider>
  );
}

export function useCheckMetadata(): CheckMetadataContextValue {
  const context = useContext(CheckMetadataContext);
  if (!context) {
    throw new Error("useCheckMetadata must be used within a CheckMetadataProvider");
  }
  return context;
}
