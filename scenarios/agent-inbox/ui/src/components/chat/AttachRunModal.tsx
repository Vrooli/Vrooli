import { useState, useEffect, useCallback } from "react";
import { Link2, Loader2, AlertCircle, RefreshCw, ArrowLeft } from "lucide-react";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "../ui/dialog";
import { Button } from "../ui/button";
import type { AgentEvent, AgentRunSummary } from "../../lib/api";
import { listAgentRuns, getRunEvents } from "../../lib/api";
import { AgentEventList } from "./agent/AgentEventList";
import { RunCard, STATUS_OPTIONS } from "./RunCard";

interface AttachRunModalProps {
  isOpen: boolean;
  onClose: () => void;
  onAttach: (run: AgentRunSummary) => void;
  isLoading?: boolean;
}

export function AttachRunModal({
  isOpen,
  onClose,
  onAttach,
  isLoading = false,
}: AttachRunModalProps) {
  const [runs, setRuns] = useState<AgentRunSummary[]>([]);
  const [selectedRun, setSelectedRun] = useState<AgentRunSummary | null>(null);
  const [statusFilter, setStatusFilter] = useState("");
  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Preview state
  const [previewEvents, setPreviewEvents] = useState<AgentEvent[]>([]);
  const [isFetchingPreview, setIsFetchingPreview] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);

  const fetchRuns = useCallback(async (status: string) => {
    setIsFetching(true);
    setError(null);
    try {
      const result = await listAgentRuns({
        status: status || undefined,
        limit: 50,
      });
      setRuns(result.runs);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to fetch runs");
      setRuns([]);
    } finally {
      setIsFetching(false);
    }
  }, []);

  const fetchPreview = useCallback(async (runId: string) => {
    setIsFetchingPreview(true);
    setPreviewError(null);
    try {
      const result = await getRunEvents(runId);
      setPreviewEvents(result.events);
    } catch (e) {
      setPreviewError(e instanceof Error ? e.message : "Failed to load preview");
      setPreviewEvents([]);
    } finally {
      setIsFetchingPreview(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      setSelectedRun(null);
      setPreviewEvents([]);
      setPreviewError(null);
      fetchRuns(statusFilter);
    }
  }, [isOpen, statusFilter, fetchRuns]);

  useEffect(() => {
    if (selectedRun) {
      fetchPreview(selectedRun.run_id);
    }
  }, [selectedRun, fetchPreview]);

  const handleBack = () => {
    setSelectedRun(null);
    setPreviewEvents([]);
    setPreviewError(null);
  };

  return (
    <Dialog open={isOpen} onClose={onClose} className="max-w-lg max-h-[80vh]">
      <DialogHeader onClose={onClose}>
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-blue-500/10">
            <Link2 className="h-5 w-5 text-blue-400" />
          </div>
          <div>
            <span className="text-lg font-semibold text-white">Attach to Run</span>
            <p className="text-sm text-slate-400 font-normal">
              {selectedRun ? "Preview run before attaching" : "Select an existing agent-manager run"}
            </p>
          </div>
        </div>
      </DialogHeader>

      <DialogBody className="space-y-3 min-h-0">
        {selectedRun ? (
          <>
            <div className="space-y-2">
              <button
                type="button"
                onClick={handleBack}
                className="flex items-center gap-1.5 text-sm text-slate-400 hover:text-white transition-colors"
              >
                <ArrowLeft className="h-3.5 w-3.5" />
                Back to list
              </button>
              <RunCard run={selectedRun} isSelected />
            </div>

            <div className="border-t border-white/10 pt-3">
              <h4 className="text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                Conversation Preview
              </h4>
              <div className="max-h-[40vh] overflow-y-auto rounded-lg border border-white/5 bg-slate-950/50">
                {isFetchingPreview && (
                  <div className="flex items-center justify-center py-8">
                    <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
                  </div>
                )}
                {previewError && (
                  <div className="flex items-center gap-2 p-3 text-red-400 text-sm">
                    <AlertCircle className="h-4 w-4 shrink-0" />
                    <span>{previewError}</span>
                  </div>
                )}
                {!isFetchingPreview && !previewError && previewEvents.length === 0 && (
                  <div className="text-center py-8 text-slate-500 text-sm">No events yet</div>
                )}
                {!isFetchingPreview && previewEvents.length > 0 && (
                  <div className="p-2">
                    <AgentEventList events={previewEvents} autoScroll={false} viewMode="compact" />
                  </div>
                )}
              </div>
            </div>
          </>
        ) : (
          <>
            <div className="flex items-center gap-2 shrink-0">
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="px-3 py-1.5 rounded-lg text-sm bg-slate-800 border border-white/10 text-white focus:outline-none focus:border-indigo-500"
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
              <button
                type="button"
                onClick={() => fetchRuns(statusFilter)}
                disabled={isFetching}
                className="p-1.5 rounded-md text-slate-400 hover:text-white hover:bg-white/10 transition-colors disabled:opacity-50"
                title="Refresh"
              >
                <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
              </button>
            </div>

            <div className="space-y-2 min-h-0 overflow-y-auto max-h-[50vh]">
              {isFetching && runs.length === 0 && (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
                </div>
              )}
              {error && (
                <div className="flex items-center gap-2 p-3 rounded-lg bg-red-500/10 text-red-400 text-sm">
                  <AlertCircle className="h-4 w-4 shrink-0" />
                  <span>{error}</span>
                </div>
              )}
              {!isFetching && !error && runs.length === 0 && (
                <div className="text-center py-8 text-slate-500 text-sm">No runs found</div>
              )}
              {runs.map((run) => (
                <RunCard key={run.run_id} run={run} onClick={() => setSelectedRun(run)} />
              ))}
            </div>
          </>
        )}
      </DialogBody>

      <DialogFooter>
        <Button variant="ghost" onClick={onClose} disabled={isLoading}>Cancel</Button>
        <Button onClick={() => selectedRun && onAttach(selectedRun)} disabled={isLoading || !selectedRun}>
          {isLoading ? "Attaching..." : "Attach"}
        </Button>
      </DialogFooter>
    </Dialog>
  );
}

export default AttachRunModal;
