import { Skeleton } from "./Skeleton";

export function RouteSkeleton({ label }: { label?: string }) {
  return (
    <div
      data-testid="route-skeleton"
      role="status"
      aria-live="polite"
      aria-label={label}
      className="flex flex-col gap-4 p-6"
    >
      <Skeleton className="h-7 w-1/3" />
      <Skeleton className="h-4 w-2/3" />
      <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
      <Skeleton className="h-64" />
    </div>
  );
}
