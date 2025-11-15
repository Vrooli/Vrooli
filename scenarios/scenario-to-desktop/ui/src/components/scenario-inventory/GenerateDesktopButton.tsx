import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { Loader2, Zap, CheckCircle, XCircle } from "lucide-react";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface GenerateDesktopButtonProps {
  scenarioName: string;
}

export function GenerateDesktopButton({ scenarioName }: GenerateDesktopButtonProps) {
  const queryClient = useQueryClient();
  const [buildId, setBuildId] = useState<string | null>(null);

  const generateMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(buildUrl('/desktop/generate/quick'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scenario_name: scenarioName,
          template_type: 'universal' // Universal template works for any scenario
        })
      });
      if (!res.ok) {
        const contentType = res.headers.get('content-type');
        let errorMessage = 'Failed to generate desktop app';

        if (contentType?.includes('application/json')) {
          const errorData = await res.json();
          errorMessage = errorData.message || errorData.error || errorMessage;
        } else {
          const textError = await res.text();
          errorMessage = textError || errorMessage;
        }

        // Add status code context
        if (res.status === 404) {
          throw new Error(`Scenario '${scenarioName}' not found. Check that the scenario exists in the scenarios directory.`);
        } else if (res.status === 400) {
          throw new Error(`Invalid request: ${errorMessage}`);
        } else if (res.status === 500) {
          throw new Error(`Server error: ${errorMessage}. Check API logs for details.`);
        } else {
          throw new Error(`${errorMessage} (HTTP ${res.status})`);
        }
      }
      return res.json();
    },
    onSuccess: (data) => {
      setBuildId(data.build_id);
      // Refresh scenarios list to show the new desktop version
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
      }, 3000);
    }
  });

  // Poll build status if we have a buildId
  const { data: buildStatus } = useQuery({
    queryKey: ['build-status', buildId],
    queryFn: async () => {
      if (!buildId) return null;
      const res = await fetch(buildUrl(`/desktop/status/${buildId}`));
      if (!res.ok) throw new Error('Failed to fetch build status');
      return res.json();
    },
    enabled: !!buildId,
    refetchInterval: (data) => {
      // Stop polling if build is complete or failed
      if (data?.status === 'ready' || data?.status === 'failed') {
        return false;
      }
      return 2000; // Poll every 2 seconds while building
    }
  });

  const isBuilding = generateMutation.isPending || buildStatus?.status === 'building';
  const isComplete = buildStatus?.status === 'ready';
  const isFailed = buildStatus?.status === 'failed' || generateMutation.isError;

  if (isComplete) {
    return (
      <div className="ml-4 flex flex-col items-end gap-2">
        <Badge variant="success" className="gap-1">
          <CheckCircle className="h-3 w-3" />
          Generated!
        </Badge>
        <p className="text-xs text-slate-400">
          {buildStatus.output_path}
        </p>
      </div>
    );
  }

  if (isFailed) {
    const errorMessage = buildStatus?.error_log?.join('\n\n') || generateMutation.error?.message || 'Unknown error';

    // Parse error to provide actionable suggestions
    const getSuggestion = (error: string): string | null => {
      if (error.includes('not found') || error.includes('404')) {
        return `💡 Ensure the scenario '${scenarioName}' exists in /scenarios/ directory`;
      }
      if (error.includes('ui/dist') || error.includes('UI not built')) {
        return `💡 Build the scenario UI first: cd scenarios/${scenarioName}/ui && npm run build`;
      }
      if (error.includes('permission') || error.includes('EACCES')) {
        return '💡 Check file permissions in the scenarios directory';
      }
      if (error.includes('ENOSPC') || error.includes('no space')) {
        return '💡 Free up disk space and try again';
      }
      if (error.includes('port') || error.includes('EADDRINUSE')) {
        return '💡 Another process is using the required port. Stop it or change ports.';
      }
      return null;
    };

    const suggestion = getSuggestion(errorMessage);

    const copyError = () => {
      navigator.clipboard.writeText(errorMessage);
    };

    return (
      <div className="ml-4 flex flex-col items-end gap-2 max-w-md">
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Failed
        </Badge>
        <div className="w-full rounded border border-red-900 bg-red-950/20 p-2">
          <p className="text-xs text-red-300 font-mono whitespace-pre-wrap max-h-32 overflow-y-auto">
            {errorMessage}
          </p>
          {suggestion && (
            <p className="mt-2 text-xs text-yellow-300 border-t border-red-800 pt-2">
              {suggestion}
            </p>
          )}
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={copyError}
            className="gap-1"
          >
            <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            Copy Error
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setBuildId(null);
              generateMutation.reset();
            }}
          >
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (isBuilding) {
    return (
      <div className="ml-4 flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-blue-400" />
        <span className="text-sm text-slate-400">Generating...</span>
      </div>
    );
  }

  return (
    <Button
      variant="outline"
      size="sm"
      className="ml-4 gap-2"
      onClick={() => generateMutation.mutate()}
      disabled={isBuilding}
    >
      <Zap className="h-4 w-4" />
      Generate Desktop
    </Button>
  );
}
