import { useQuery } from "@tanstack/react-query";
import { fetchValidationTargets, type ValidationTarget } from "../lib/api";

export function useTargets() {
  const query = useQuery<ValidationTarget[]>({
    queryKey: ["validation-targets"],
    queryFn: fetchValidationTargets,
    staleTime: 60_000
  });
  return { targets: query.data ?? [], ...query };
}
