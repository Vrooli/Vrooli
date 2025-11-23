/**
 * RateLimitsTab Component
 * View and manage rate limit status
 */

import { useEffect, useState } from 'react';
import { Clock, AlertCircle, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useQueueStatus } from '@/hooks/useQueueStatus';
import { useResetRateLimit } from '@/hooks/useRateLimits';

export function RateLimitsTab() {
  const { data: queueStatus } = useQueueStatus();
  const resetRateLimit = useResetRateLimit();
  const [countdown, setCountdown] = useState(0);

  const isRateLimited = queueStatus?.rate_limited || false;
  const retryAfter = queueStatus?.rate_limit_retry_after || 0;

  // Countdown timer
  useEffect(() => {
    if (isRateLimited && retryAfter > 0) {
      setCountdown(retryAfter);
      const interval = setInterval(() => {
        setCountdown((prev) => {
          if (prev <= 1) {
            clearInterval(interval);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);

      return () => clearInterval(interval);
    }
  }, [isRateLimited, retryAfter]);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const handleReset = () => {
    if (confirm('Reset rate limit? This should only be done if you have confirmed with the API provider that the limit has been lifted.')) {
      resetRateLimit.mutate();
    }
  };

  return (
    <div className="space-y-6">
      {/* Status Display */}
      <div className="space-y-4">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <Clock className="h-4 w-4" />
          <span>Monitor and manage API rate limit status</span>
        </div>

        {/* Rate Limit Status Card */}
        <div className={`border rounded-lg p-6 ${
          isRateLimited
            ? 'bg-red-900/20 border-red-400/30'
            : 'bg-green-900/20 border-green-400/30'
        }`}>
          <div className="flex items-start gap-4">
            <div className={`p-3 rounded-full ${
              isRateLimited ? 'bg-red-400/20' : 'bg-green-400/20'
            }`}>
              {isRateLimited ? (
                <AlertCircle className="h-6 w-6 text-red-400" />
              ) : (
                <Clock className="h-6 w-6 text-green-400" />
              )}
            </div>
            <div className="flex-1">
              <h3 className="text-lg font-semibold mb-2">
                {isRateLimited ? 'Rate Limited' : 'No Active Limits'}
              </h3>
              <p className="text-sm text-slate-300 mb-4">
                {isRateLimited
                  ? 'The task processor is currently paused due to API rate limiting. Processing will resume automatically when the limit expires.'
                  : 'The task processor is operating normally with no active rate limits.'}
              </p>

              {isRateLimited && countdown > 0 && (
                <div className="bg-black/30 border border-white/10 rounded-md px-4 py-3 mb-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-slate-300">Time until retry:</span>
                    <span className="text-2xl font-mono font-bold text-red-400">
                      {formatTime(countdown)}
                    </span>
                  </div>
                  <div className="mt-2 h-2 bg-slate-700 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-red-400 transition-all duration-1000"
                      style={{
                        width: `${((retryAfter - countdown) / retryAfter) * 100}%`,
                      }}
                    />
                  </div>
                </div>
              )}

              {isRateLimited && (
                <div className="space-y-2 text-xs text-slate-400">
                  <p>
                    <strong>What this means:</strong> The API provider has temporarily
                    blocked requests due to exceeding usage quotas.
                  </p>
                  <p>
                    <strong>What happens next:</strong> Processing will automatically
                    resume when the countdown reaches zero.
                  </p>
                  <p>
                    <strong>Manual override:</strong> Only use the reset button below
                    if you've confirmed the limit has been lifted by the provider.
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="pt-4 border-t border-white/10">
        <Button
          variant="outline"
          onClick={handleReset}
          disabled={!isRateLimited || resetRateLimit.isPending}
        >
          <RotateCcw className="h-4 w-4 mr-2" />
          {resetRateLimit.isPending ? 'Resetting...' : 'Reset Rate Limit'}
        </Button>
        <p className="text-xs text-slate-500 mt-2">
          Only reset if confirmed with API provider that limit has been lifted
        </p>
      </div>

      {/* Info Section */}
      <div className="bg-blue-900/20 border border-blue-400/30 rounded-md p-4">
        <h4 className="text-sm font-medium text-blue-400 mb-2">Rate Limit Information</h4>
        <ul className="text-xs text-slate-300 space-y-1">
          <li>• Rate limits are imposed by the AI API provider (OpenRouter, Anthropic, etc.)</li>
          <li>• Limits typically reset hourly or daily depending on your plan</li>
          <li>• The processor automatically pauses and resumes to handle limits gracefully</li>
          <li>• Consider upgrading your API plan if you frequently hit limits</li>
        </ul>
      </div>
    </div>
  );
}
