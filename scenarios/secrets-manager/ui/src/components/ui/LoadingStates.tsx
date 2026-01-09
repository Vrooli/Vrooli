export const Skeleton = ({ className = "", variant = "default" }: { className?: string; variant?: "default" | "text" | "circular" }) => {
  const baseClasses = "animate-pulse bg-white/10";
  const variantClasses = variant === "circular" ? "rounded-full" : variant === "text" ? "rounded" : "rounded-2xl";
  return <div className={`${baseClasses} ${variantClasses} ${className}`} />;
};

export const LoadingStatCard = () => (
  <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
    <Skeleton className="h-3 w-20" variant="text" />
    <Skeleton className="mt-2 h-9 w-16" variant="text" />
    <Skeleton className="mt-1 h-4 w-32" variant="text" />
  </div>
);

export const LoadingStatusTile = () => (
  <div className="rounded-2xl border border-white/10 bg-white/5 px-4 py-5">
    <div className="flex items-center justify-between">
      <Skeleton className="h-3 w-24" variant="text" />
      <Skeleton className="h-4 w-4" variant="circular" />
    </div>
    <Skeleton className="mt-3 h-8 w-32" variant="text" />
    <Skeleton className="mt-1 h-4 w-24" variant="text" />
  </div>
);

export const LoadingResourceCard = () => (
  <div className="rounded-2xl border border-white/10 bg-black/30 p-4">
    <div className="flex items-center justify-between">
      <div className="flex-1">
        <Skeleton className="h-6 w-32" variant="text" />
        <Skeleton className="mt-1 h-4 w-40" variant="text" />
      </div>
      <Skeleton className="h-8 w-20" variant="text" />
    </div>
    <div className="mt-3 space-y-2">
      <Skeleton className="h-16 w-full" />
      <Skeleton className="h-16 w-full" />
    </div>
  </div>
);

export const LoadingSecretsRow = () => (
  <div className="flex items-center justify-between rounded-xl border border-white/5 bg-white/5 px-4 py-3">
    <div className="flex-1">
      <Skeleton className="h-4 w-32" variant="text" />
      <Skeleton className="mt-1 h-3 w-24" variant="text" />
    </div>
    <div className="flex w-1/2 flex-col items-end gap-2">
      <Skeleton className="h-2 w-full" />
      <Skeleton className="h-5 w-24" variant="text" />
    </div>
  </div>
);
