import { useCallback, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2, CheckCircle2, AlertCircle, Copy, Check, Download, RotateCcw } from "lucide-react";
import { generateConfig } from "../../lib/api";
import { formatQueryError } from "../../lib/formatQueryError";
import { useConfirmAction } from "../../hooks/useConfirmAction";
import { Button } from "../ui/button";

interface StepCompleteProps {
  selected: Set<string>;
  onStartOver?: () => void;
}

export function StepComplete({ selected, onStartOver }: StepCompleteProps) {
  const [copied, setCopied] = useState(false);
  const { confirming: confirmingStartOver, requestConfirm, confirm: confirmStartOver, cancel: cancelStartOver } =
    useConfirmAction(onStartOver ?? (() => {}));

  const resourcesList = useMemo(() => Array.from(selected), [selected]);

  const { data: config, isLoading: loading, error: queryError } = useQuery({
    queryKey: ["generate-config", resourcesList],
    queryFn: () => generateConfig(resourcesList),
    enabled: resourcesList.length > 0,
  });

  const error = formatQueryError(queryError, "Failed to generate config");

  const configText = useMemo(() => config ? JSON.stringify(config, null, 2) : "", [config]);

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(configText).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [configText]);

  const handleDownload = useCallback(() => {
    const blob = new Blob([configText], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "vrooli-config.json";
    a.click();
    URL.revokeObjectURL(url);
  }, [configText]);

  return (
    <div className="flex flex-col items-center" data-testid="step-complete">
      <h1 className="text-2xl font-semibold">
        {!loading && !error && config ? "Configuration Ready" : "Generating Config"}
      </h1>

      {loading && (
        <div className="flex items-center gap-2 py-16 text-slate-300" data-testid="config-loading" role="status">
          <Loader2 className="h-6 w-6 animate-spin" aria-hidden="true" />
          Generating configuration...
        </div>
      )}

      {error && (
        <div className="flex items-center gap-2 py-16 text-red-400" data-testid="config-error" role="alert">
          <AlertCircle className="h-5 w-5" aria-hidden="true" />
          {error}
        </div>
      )}

      {!loading && !error && config && (
        <>
          <div className="mt-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-emerald-500/20 text-emerald-400">
            <CheckCircle2 className="h-8 w-8" aria-hidden="true" />
          </div>
          <p className="mt-2 text-slate-300 text-center max-w-md">
            Your Vrooli configuration has been generated with {selected.size} resource{selected.size !== 1 ? "s" : ""}.
            Copy or download the config below.
          </p>

          <div className="mt-6 w-full">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium text-slate-300">Generated Config</span>
              <div className="flex items-center gap-2">
                <Button variant="outline" size="sm" onClick={handleDownload} data-testid="download-config" aria-label="Download JSON config file">
                  <Download className="mr-1 h-3 w-3" aria-hidden="true" />
                  <span className="hidden sm:inline">Download</span>
                </Button>
                <Button variant="outline" size="sm" onClick={handleCopy} data-testid="copy-config" aria-label={copied ? "Copied to clipboard" : "Copy config to clipboard"}>
                  {copied ? (
                    <>
                      <Check className="mr-1 h-3 w-3" aria-hidden="true" />
                      Copied
                    </>
                  ) : (
                    <>
                      <Copy className="mr-1 h-3 w-3" aria-hidden="true" />
                      Copy
                    </>
                  )}
                </Button>
              </div>
            </div>
            <pre
              className="config-line-numbers w-full overflow-auto rounded-xl border border-white/10 bg-black/30 p-4 text-sm text-slate-200 leading-relaxed"
              data-testid="config-output"
              tabIndex={0}
              aria-label="Generated configuration JSON"
            >
              {configText.split("\n").map((line, i) => (
                <span key={i} className="block">{line}</span>
              ))}
            </pre>
          </div>

          {onStartOver && (
            <div className="mt-6 pt-4 border-t border-white/10 w-full text-center">
              <p className="text-xs text-slate-300 mb-3">Want to change your selections?</p>
              {confirmingStartOver ? (
                <div className="flex items-center justify-center gap-2" role="alert">
                  <span className="text-xs text-yellow-400">This will clear all selections.</span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={confirmStartOver}
                    data-testid="start-over-confirm"
                    className="border-yellow-500/50 text-yellow-400 hover:bg-yellow-500/10"
                  >
                    Confirm
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={cancelStartOver}
                    data-testid="start-over-cancel"
                  >
                    Cancel
                  </Button>
                </div>
              ) : (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={requestConfirm}
                  data-testid="start-over"
                >
                  <RotateCcw className="mr-1.5 h-3 w-3" aria-hidden="true" />
                  Start Over
                </Button>
              )}
            </div>
          )}
        </>
      )}

      {!loading && !error && !config && selected.size === 0 && (
        <div className="py-16 text-center">
          <p className="text-slate-200">No resources were selected.</p>
        </div>
      )}
    </div>
  );
}
