import { useQuery } from "@tanstack/react-query";
import { fetchMeasureDefinitions } from "../api/statsClient";

export function useMeasureDefinitions() {
  return useQuery({
    queryKey: ["stats", "measure-definitions"],
    queryFn: fetchMeasureDefinitions,
    staleTime: Infinity,
    gcTime: Infinity,
  });
}
