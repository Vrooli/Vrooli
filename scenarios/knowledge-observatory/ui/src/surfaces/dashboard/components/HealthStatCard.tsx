// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { ReactNode } from "react";
import { AlertCircle } from "lucide-react";

export type HealthStatCardProps = {
  title: string;
  value: string;
  subtitle?: string;
  tone?: "good" | "medium" | "poor";
  icon?: ReactNode;
  isLoading?: boolean;
  hasError?: boolean;
  errorMessage?: string;
  testId?: string;
};

const toneClass = (tone?: "good" | "medium" | "poor") => {
  if (tone === "good") return "ko-tone-good";
  if (tone === "poor") return "ko-tone-poor";
  return "ko-tone-medium";
};

export function HealthStatCard({
  title,
  value,
  subtitle,
  tone,
  icon,
  isLoading,
  hasError,
  errorMessage,
  testId,
}: HealthStatCardProps) {
  return (
    <div className={`ko-card ko-stat-card p-4 ${toneClass(tone)}`} data-testid={testId}>
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="ko-meta">{title}</p>
          {isLoading ? (
            <p className="text-xl font-bold mt-1">Syncing…</p>
          ) : hasError ? (
            <p className="text-xl font-bold mt-1">Unavailable</p>
          ) : (
            <p className="text-xl font-bold mt-1">{value}</p>
          )}
        </div>
        {hasError ? <AlertCircle className="h-5 w-5 ko-text-danger" /> : icon}
      </div>
      {hasError && errorMessage ? (
        <p className="ko-text-xs ko-text-danger-muted mt-2">{errorMessage}</p>
      ) : subtitle ? (
        <p className="ko-text-xs ko-subtle mt-2">{subtitle}</p>
      ) : null}
    </div>
  );
}
