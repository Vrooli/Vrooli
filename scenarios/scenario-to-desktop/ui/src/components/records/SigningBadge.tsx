import { useQuery } from "@tanstack/react-query";
import { RefreshCw, AlertCircle, CheckCircle2 } from "lucide-react";
import type { SigningReadinessResponse } from "../../lib/api";
import { checkSigningReadiness } from "../../lib/api";
import { Badge } from "../ui/badge";

export function SigningBadge({ scenarioName }: { scenarioName: string }) {
  const { data, isFetching } = useQuery<SigningReadinessResponse>({
    queryKey: ["signing-readiness-record", scenarioName],
    queryFn: () => checkSigningReadiness(scenarioName),
    enabled: Boolean(scenarioName),
    staleTime: 30000,
  });

  if (isFetching) {
    return (
      <Badge variant="outline" className="gap-1 border-slate-700 text-slate-200">
        <RefreshCw className="h-3.5 w-3.5 animate-spin" />
        Checking signing…
      </Badge>
    );
  }

  if (!data || !data.ready) {
    return (
      <Badge variant="outline" className="gap-1 border-amber-800 text-amber-200">
        <AlertCircle className="h-3.5 w-3.5" />
        Signing not ready
      </Badge>
    );
  }

  return (
    <Badge variant="outline" className="gap-1 border-green-800 text-green-200">
      <CheckCircle2 className="h-3.5 w-3.5" />
      Signing ready
    </Badge>
  );
}
