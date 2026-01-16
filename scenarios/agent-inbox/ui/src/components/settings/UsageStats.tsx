import { useEffect, useState } from "react";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "../ui/dialog";
import { Button } from "../ui/button";
import { fetchUsageStats, UsageStats as UsageStatsType, ModelUsage } from "../../lib/api";

interface UsageStatsProps {
  isOpen: boolean;
  onClose: () => void;
}

// Format cost in cents to a readable string
function formatCost(cents: number): string {
  if (cents === 0) return "$0.00";
  if (cents < 1) return `$${(cents / 100).toFixed(4)}`;
  return `$${(cents / 100).toFixed(2)}`;
}

// Format token count with thousands separator
function formatTokens(count: number): string {
  return count.toLocaleString();
}

// Extract model name from full model ID (e.g., "anthropic/claude-3.5-sonnet" -> "claude-3.5-sonnet")
function formatModelName(modelId: string): string {
  const parts = modelId.split("/");
  return parts[parts.length - 1] ?? modelId;
}

export function UsageStats({ isOpen, onClose }: UsageStatsProps) {
  const [stats, setStats] = useState<UsageStatsType | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isOpen) return;

    setLoading(true);
    setError(null);

    fetchUsageStats()
      .then(setStats)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [isOpen]);

  const modelUsages = stats?.by_model
    ? Object.values(stats.by_model).sort((a, b) => b.total_cost - a.total_cost)
    : [];
  const dailyUsages = stats?.by_day
    ? Object.values(stats.by_day).sort((a, b) => b.date.localeCompare(a.date))
    : [];

  return (
    <Dialog open={isOpen} onClose={onClose} className="sm:max-w-[600px]">
      <DialogHeader onClose={onClose}>Usage Statistics</DialogHeader>
      <DialogBody className="max-h-[60vh] overflow-y-auto">
        {loading && (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-500" />
          </div>
        )}

        {error && (
          <div className="text-red-500 py-4 text-center">
            Failed to load usage statistics: {error}
          </div>
        )}

        {stats && !loading && (
          <div className="space-y-6">
            {/* Overall Summary */}
            <div className="grid grid-cols-2 gap-4">
              <div className="bg-slate-800 rounded-lg p-4">
                <div className="text-sm text-slate-400">Total Tokens</div>
                <div className="text-2xl font-semibold text-white">
                  {formatTokens(stats.total_tokens)}
                </div>
                <div className="text-xs text-slate-500 mt-1">
                  {formatTokens(stats.total_prompt_tokens)} input / {formatTokens(stats.total_completion_tokens)} output
                </div>
              </div>
              <div className="bg-slate-800 rounded-lg p-4">
                <div className="text-sm text-slate-400">Total Cost</div>
                <div className="text-2xl font-semibold text-green-400">
                  {formatCost(stats.total_cost)}
                </div>
                <div className="text-xs text-slate-500 mt-1">
                  Estimated based on model pricing
                </div>
              </div>
            </div>

            {/* By Model Breakdown */}
            {modelUsages.length > 0 && (
              <div>
                <h3 className="text-sm font-medium text-slate-300 mb-3">Usage by Model</h3>
                <div className="space-y-2">
                  {modelUsages.map((model: ModelUsage) => (
                    <div
                      key={model.model}
                      className="flex items-center justify-between bg-slate-800/50 rounded-lg px-4 py-3"
                    >
                      <div className="flex-1">
                        <div className="text-sm font-medium text-white">
                          {formatModelName(model.model)}
                        </div>
                        <div className="text-xs text-slate-500">
                          {model.request_count} request{model.request_count !== 1 ? "s" : ""} &bull; {formatTokens(model.total_tokens)} tokens
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-sm font-medium text-green-400">
                          {formatCost(model.total_cost)}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Daily Usage (last 7 days) */}
            {dailyUsages.length > 0 && (
              <div>
                <h3 className="text-sm font-medium text-slate-300 mb-3">Recent Activity</h3>
                <div className="space-y-2">
                  {dailyUsages.slice(0, 7).map((day) => (
                    <div
                      key={day.date}
                      className="flex items-center justify-between bg-slate-800/50 rounded-lg px-4 py-2"
                    >
                      <div className="text-sm text-slate-300">{day.date}</div>
                      <div className="flex items-center gap-4 text-xs text-slate-500">
                        <span>{day.request_count} requests</span>
                        <span>{formatTokens(day.total_tokens)} tokens</span>
                        <span className="text-green-400">{formatCost(day.total_cost)}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Empty state */}
            {stats.total_tokens === 0 && (
              <div className="text-center py-8 text-slate-500">
                No usage data yet. Start chatting to see your statistics!
              </div>
            )}
          </div>
        )}
      </DialogBody>
      <DialogFooter>
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
      </DialogFooter>
    </Dialog>
  );
}
