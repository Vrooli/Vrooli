import { Loader2 } from "lucide-react";
import * as config from "../../config";
import { cn } from "../../lib";
import { Card } from "./card";

interface SkeletonBlockProps {
  className?: string;
}

interface PageLoadingStateProps {
  label: string;
  testId?: string;
  variant?: "list" | "detail" | "settings";
}

interface InlineLoadingIndicatorProps {
  label?: string;
  className?: string;
  testId?: string;
}

const shouldUseSkeletonLoading = (): boolean => {
  try {
    const uiBehaviorConfig = Reflect.get(
      config as Record<string, unknown>,
      "uiBehaviorConfig"
    ) as { useSkeletonLoading?: boolean } | undefined;
    return uiBehaviorConfig?.useSkeletonLoading ?? true;
  } catch {
    return true;
  }
};

function SkeletonBlock({ className }: SkeletonBlockProps) {
  return (
    <div
      className={cn("animate-pulse rounded-md bg-slate-700/60", className)}
      aria-hidden="true"
    />
  );
}

function ListSkeleton() {
  return (
    <div className="space-y-3">
      <div className="space-y-2">
        <SkeletonBlock className="h-4 w-40" />
        <SkeletonBlock className="h-3 w-56" />
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: 6 }).map((_, index) => (
          <div key={`list-skeleton-${index}`} className="rounded-xl border border-white/10 bg-slate-800/35 p-4">
            <SkeletonBlock className="h-5 w-3/5" />
            <SkeletonBlock className="mt-3 h-4 w-full" />
            <SkeletonBlock className="mt-2 h-4 w-4/5" />
            <div className="mt-4 flex gap-2">
              <SkeletonBlock className="h-5 w-16 rounded-full" />
              <SkeletonBlock className="h-5 w-20 rounded-full" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="space-y-3">
      <SkeletonBlock className="h-5 w-56" />
      <SkeletonBlock className="h-4 w-80" />
      <div className="grid gap-3 lg:grid-cols-3">
        <div className="space-y-3 lg:col-span-2">
          <SkeletonBlock className="h-52 w-full" />
          <SkeletonBlock className="h-44 w-full" />
        </div>
        <SkeletonBlock className="h-64 w-full" />
      </div>
    </div>
  );
}

function SettingsSkeleton() {
  return (
    <div className="space-y-3">
      <SkeletonBlock className="h-5 w-36" />
      <SkeletonBlock className="h-4 w-72" />
      <div className="space-y-3">
        <SkeletonBlock className="h-14 w-full" />
        <SkeletonBlock className="h-14 w-full" />
        <SkeletonBlock className="h-14 w-full" />
      </div>
    </div>
  );
}

function getSkeletonForVariant(variant: PageLoadingStateProps["variant"]) {
  if (variant === "detail") return <DetailSkeleton />;
  if (variant === "settings") return <SettingsSkeleton />;
  return <ListSkeleton />;
}

export function PageLoadingState({ label, testId, variant = "list" }: PageLoadingStateProps) {
  if (!shouldUseSkeletonLoading()) {
    return (
      <Card padding="lg" centered data-testid={testId} role="status" aria-live="polite" aria-busy="true">
        <div className="inline-flex items-center gap-2 text-slate-300">
          <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
          <p className="text-slate-300">{label}</p>
        </div>
      </Card>
    );
  }

  return (
    <Card padding="lg" data-testid={testId} role="status" aria-live="polite" aria-busy="true">
      <p className="mb-4 text-sm font-medium text-slate-300">{label}</p>
      {getSkeletonForVariant(variant)}
    </Card>
  );
}

export function InlineLoadingIndicator({
  label = "Syncing latest data...",
  className,
  testId,
}: InlineLoadingIndicatorProps) {
  return (
    <div
      className={cn(
        "inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-500/10 px-3 py-1.5 text-xs text-cyan-200",
        className
      )}
      data-testid={testId}
      role="status"
      aria-live="polite"
      aria-busy="true"
    >
      <Loader2 className="h-3.5 w-3.5 animate-spin" />
      <span>{label}</span>
    </div>
  );
}

export function PanelLoadingState({ label, testId }: { label: string; testId?: string }) {
  if (!shouldUseSkeletonLoading()) {
    return (
      <div
        className="rounded-lg border border-white/10 bg-slate-800/40 p-4 text-sm text-slate-300"
        data-testid={testId}
        role="status"
        aria-live="polite"
        aria-busy="true"
      >
        <div className="inline-flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
          <span>{label}</span>
        </div>
      </div>
    );
  }

  return (
    <div
      className="space-y-3 rounded-lg border border-white/10 bg-slate-800/40 p-4"
      data-testid={testId}
      role="status"
      aria-live="polite"
      aria-busy="true"
    >
      <p className="text-sm text-slate-300">{label}</p>
      <SkeletonBlock className="h-8 w-full" />
      <SkeletonBlock className="h-8 w-11/12" />
      <SkeletonBlock className="h-8 w-10/12" />
    </div>
  );
}
