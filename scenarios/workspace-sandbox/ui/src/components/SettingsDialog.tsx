import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  Settings,
  RefreshCw,
  Check,
  X,
  ChevronDown,
  ChevronRight,
  Info,
  Cpu,
  HardDrive,
  Shield,
  Loader2,
  Copy,
  CheckCircle2,
  AlertTriangle,
  Play,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { useDriverOptions, useSelectDriver } from "../lib/hooks";
import { queryKeys } from "../lib/hooks";
import type { DriverOption, DriverRequirement, SelectDriverResponse } from "../lib/api";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function RequirementItem({ requirement }: { requirement: DriverRequirement }) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement("textarea");
      textarea.value = text;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="flex items-start gap-2 py-1.5">
      {requirement.met ? (
        <Check className="h-4 w-4 text-emerald-400 mt-0.5 flex-shrink-0" />
      ) : (
        <X className="h-4 w-4 text-red-400 mt-0.5 flex-shrink-0" />
      )}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span
            className={`text-sm ${
              requirement.met ? "text-slate-300" : "text-slate-400"
            }`}
          >
            {requirement.name}
          </span>
          {requirement.current && (
            <span className="text-xs text-slate-500 font-mono">
              ({requirement.current})
            </span>
          )}
        </div>
        {!requirement.met && requirement.howToFix && (
          <div className="mt-1.5 flex items-start gap-2 group">
            <code className="flex-1 text-xs bg-slate-800 px-2 py-1 rounded text-amber-300 font-mono break-all">
              {requirement.howToFix}
            </code>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0"
              onClick={() => copyToClipboard(requirement.howToFix!)}
              title="Copy to clipboard"
            >
              {copied ? (
                <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />
              ) : (
                <Copy className="h-3.5 w-3.5 text-slate-400" />
              )}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

function DriverOptionCard({
  option,
  isCurrentDriver,
  onSelect,
  isSelecting,
}: {
  option: DriverOption;
  isCurrentDriver: boolean;
  onSelect: (driverId: string) => void;
  isSelecting: boolean;
}) {
  const [expanded, setExpanded] = useState(!option.available || isCurrentDriver);

  const metCount = option.requirements.filter((r) => r.met).length;
  const totalCount = option.requirements.length;
  const allMet = metCount === totalCount;

  return (
    <div
      className={`rounded-lg border ${
        isCurrentDriver
          ? "border-emerald-700 bg-emerald-950/30"
          : option.available
          ? "border-slate-700 bg-slate-800/50"
          : "border-slate-800 bg-slate-900/50"
      }`}
    >
      {/* Header */}
      <button
        type="button"
        className="w-full px-4 py-3 flex items-center gap-3 text-left"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-slate-500 flex-shrink-0" />
        ) : (
          <ChevronRight className="h-4 w-4 text-slate-500 flex-shrink-0" />
        )}

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-medium text-slate-200">{option.name}</span>
            {isCurrentDriver && (
              <Badge variant="success" className="text-[10px]">
                Current
              </Badge>
            )}
            {option.recommended && !isCurrentDriver && (
              <Badge variant="default" className="text-[10px] bg-blue-900/50 text-blue-300 border-blue-700">
                Recommended
              </Badge>
            )}
            {option.directAccess && (
              <Badge variant="default" className="text-[10px] bg-slate-700">
                Direct Access
              </Badge>
            )}
          </div>
          <p className="text-xs text-slate-500 mt-0.5 line-clamp-2">
            {option.description}
          </p>
        </div>

        {/* Status indicator */}
        <div className="flex-shrink-0">
          {totalCount === 0 ? (
            <Badge variant="success" className="text-[10px]">
              Always Available
            </Badge>
          ) : allMet ? (
            <Badge variant="success" className="text-[10px]">
              {metCount}/{totalCount} Ready
            </Badge>
          ) : (
            <Badge variant="warning" className="text-[10px]">
              {metCount}/{totalCount} Met
            </Badge>
          )}
        </div>
      </button>

      {/* Requirements List and Select Button */}
      {expanded && (
        <div className="px-4 pb-3 pt-0 border-t border-slate-700/50">
          {option.requirements.length > 0 && (
            <div className="mt-3 space-y-1">
              {option.requirements.map((req, i) => (
                <RequirementItem key={i} requirement={req} />
              ))}
            </div>
          )}

          {/* Select Button */}
          {option.available && !isCurrentDriver && (
            <div className="mt-3 pt-3 border-t border-slate-700/50">
              <Button
                size="sm"
                variant="outline"
                onClick={(e) => {
                  e.stopPropagation();
                  onSelect(option.id);
                }}
                disabled={isSelecting}
                className="w-full"
              >
                {isSelecting ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                    Selecting...
                  </>
                ) : (
                  <>
                    <Play className="h-3.5 w-3.5 mr-1.5" />
                    Use this driver
                  </>
                )}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  const queryClient = useQueryClient();
  const driverOptionsQuery = useDriverOptions();
  const selectDriverMutation = useSelectDriver();
  const data = driverOptionsQuery.data;
  const [selectResult, setSelectResult] = useState<SelectDriverResponse | null>(null);

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.driverOptions });
    setSelectResult(null);
  };

  const handleSelectDriver = (driverId: string) => {
    setSelectResult(null);
    selectDriverMutation.mutate(driverId, {
      onSuccess: (result) => {
        setSelectResult(result);
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col">
        <DialogClose onClose={() => onOpenChange(false)} />

        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Settings className="h-5 w-5 text-slate-400" />
            Settings
          </DialogTitle>
          <DialogDescription>
            Configure sandbox driver options and view system requirements.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto mt-4 space-y-6 pr-2">
          {/* System Info */}
          {data && (
            <section>
              <h3 className="text-sm font-medium text-slate-300 mb-3 flex items-center gap-2">
                <Cpu className="h-4 w-4 text-slate-500" />
                System Information
              </h3>
              <div className="grid grid-cols-2 gap-3">
                <div className="rounded-md bg-slate-800/50 px-3 py-2">
                  <div className="text-[10px] uppercase tracking-wider text-slate-500">
                    Operating System
                  </div>
                  <div className="text-sm text-slate-200 font-medium capitalize">
                    {data.os}
                  </div>
                </div>
                {data.kernel && (
                  <div className="rounded-md bg-slate-800/50 px-3 py-2">
                    <div className="text-[10px] uppercase tracking-wider text-slate-500">
                      Kernel Version
                    </div>
                    <div className="text-sm text-slate-200 font-medium font-mono">
                      {data.kernel}
                    </div>
                  </div>
                )}
                <div className="rounded-md bg-slate-800/50 px-3 py-2">
                  <div className="text-[10px] uppercase tracking-wider text-slate-500">
                    User Namespace
                  </div>
                  <div className="text-sm text-slate-200 font-medium flex items-center gap-1.5">
                    {data.inUserNamespace ? (
                      <>
                        <Shield className="h-3.5 w-3.5 text-emerald-400" />
                        <span>Enabled</span>
                      </>
                    ) : (
                      <>
                        <Shield className="h-3.5 w-3.5 text-slate-500" />
                        <span>Not Active</span>
                      </>
                    )}
                  </div>
                </div>
                <div className="rounded-md bg-slate-800/50 px-3 py-2">
                  <div className="text-[10px] uppercase tracking-wider text-slate-500">
                    Current Driver
                  </div>
                  <div className="text-sm text-emerald-400 font-medium font-mono">
                    {data.currentDriver}
                  </div>
                </div>
              </div>
            </section>
          )}

          {/* Driver Options */}
          <section>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-slate-300 flex items-center gap-2">
                <HardDrive className="h-4 w-4 text-slate-500" />
                Driver Options
              </h3>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleRefresh}
                disabled={driverOptionsQuery.isLoading || driverOptionsQuery.isFetching}
                className="h-7 px-2"
              >
                <RefreshCw
                  className={`h-3.5 w-3.5 mr-1.5 ${
                    driverOptionsQuery.isFetching ? "animate-spin" : ""
                  }`}
                />
                Refresh
              </Button>
            </div>

            {/* Success/Info Message */}
            {selectResult && (
              <div
                className={`rounded-lg border p-3 mb-4 ${
                  selectResult.requiresRestart
                    ? "border-amber-700 bg-amber-950/30 text-amber-200"
                    : "border-emerald-700 bg-emerald-950/30 text-emerald-200"
                }`}
              >
                <div className="flex items-start gap-2">
                  {selectResult.requiresRestart ? (
                    <AlertTriangle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                  ) : (
                    <CheckCircle2 className="h-4 w-4 mt-0.5 flex-shrink-0" />
                  )}
                  <div className="text-sm">
                    <p>{selectResult.message}</p>
                    {selectResult.requiresRestart && (
                      <p className="text-xs mt-1 opacity-80">
                        Restart the API server for the change to take effect.
                      </p>
                    )}
                  </div>
                </div>
              </div>
            )}

            {/* Error Message */}
            {selectDriverMutation.error && (
              <div className="rounded-lg border border-red-800 bg-red-950/50 p-3 mb-4 text-red-300 text-sm">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4" />
                  {(selectDriverMutation.error as Error).message}
                </div>
              </div>
            )}

            {driverOptionsQuery.isLoading ? (
              <div className="flex items-center justify-center py-8 text-slate-500">
                <Loader2 className="h-5 w-5 animate-spin mr-2" />
                Loading driver options...
              </div>
            ) : driverOptionsQuery.error ? (
              <div className="rounded-lg border border-red-800 bg-red-950/50 p-4 text-red-300 text-sm">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4" />
                  Failed to load driver options
                </div>
                <p className="text-xs text-red-400 mt-1">
                  {(driverOptionsQuery.error as Error).message}
                </p>
              </div>
            ) : data?.options ? (
              <div className="space-y-3">
                {data.options.map((option) => (
                  <DriverOptionCard
                    key={option.id}
                    option={option}
                    isCurrentDriver={option.id === data.currentDriver}
                    onSelect={handleSelectDriver}
                    isSelecting={selectDriverMutation.isPending}
                  />
                ))}
              </div>
            ) : null}
          </section>

          {/* Info Section */}
          <section className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
            <div className="flex items-start gap-3">
              <Info className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
              <div className="space-y-2 text-sm text-slate-400">
                <p>
                  <strong className="text-slate-300">Direct Access</strong> means
                  the sandbox merged directory is accessible directly on the
                  filesystem, allowing tools like IDEs or file managers to browse
                  files normally.
                </p>
                <p>
                  Without direct access, files are only accessible through the
                  API&apos;s <code className="text-xs bg-slate-800 px-1 rounded">/exec</code> endpoint
                  or file operation APIs.
                </p>
                <p className="text-xs">
                  Selected driver preferences are saved and will be used on the next API restart.
                  If a driver becomes unavailable, the system will automatically fall back to the next best option.
                </p>
              </div>
            </div>
          </section>
        </div>

        {/* Footer with close button */}
        <div className="flex justify-end pt-4 border-t border-slate-800 mt-4">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
