import type { ReactNode } from "react";
import { RotateCcw } from "lucide-react";
import { Button } from "./button";

interface QueryStateProps {
  isLoading: boolean;
  error: Error | null;
  refetch?: () => void;
  loadingLabel: string;
  errorLabel: string;
  skeleton: ReactNode;
  children: ReactNode;
}

export function QueryState({ isLoading, error, refetch, loadingLabel, errorLabel, skeleton, children }: QueryStateProps) {
  if (isLoading) {
    return (
      <div className="mt-4 animate-pulse" role="status">
        <span className="sr-only">{loadingLabel}</span>
        {skeleton}
      </div>
    );
  }

  if (error) {
    return (
      <div className="mt-4 rounded-lg border border-red-500/20 bg-red-500/5 p-4" role="alert">
        <p className="text-sm text-red-400">{errorLabel}</p>
        {refetch && (
          <Button variant="outline" size="sm" onClick={refetch} className="mt-3 text-red-400 border-red-400/30 hover:bg-red-500/10">
            <RotateCcw className="h-3 w-3 mr-2" aria-hidden="true" />
            Retry
          </Button>
        )}
      </div>
    );
  }

  return <>{children}</>;
}
