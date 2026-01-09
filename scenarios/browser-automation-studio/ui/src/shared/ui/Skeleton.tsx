import { CSSProperties } from "react";

interface SkeletonProps {
  width?: string | number;
  height?: string | number;
  className?: string;
  variant?: "text" | "circular" | "rectangular" | "rounded";
  animation?: "pulse" | "wave" | "none";
}

/**
 * A skeleton loading placeholder that indicates content is loading.
 * Provides visual feedback during data fetching operations.
 */
function Skeleton({
  width,
  height,
  className = "",
  variant = "text",
  animation = "pulse",
}: SkeletonProps) {
  const baseClasses = "bg-gray-700";

  const variantClasses = {
    text: "rounded",
    circular: "rounded-full",
    rectangular: "rounded-none",
    rounded: "rounded-lg",
  };

  const animationClasses = {
    pulse: "animate-pulse",
    wave: "skeleton-wave",
    none: "",
  };

  const style: CSSProperties = {
    width: width || "100%",
    height: height || (variant === "text" ? "1em" : "100%"),
  };

  return (
    <div
      className={`${baseClasses} ${variantClasses[variant]} ${animationClasses[animation]} ${className}`}
      style={style}
      aria-hidden="true"
    />
  );
}

/**
 * A card skeleton showing a typical card layout during loading.
 */
export function CardSkeleton({ className = "" }: { className?: string }) {
  return (
    <div className={`bg-flow-node border border-gray-700 rounded-lg p-6 ${className}`}>
      <div className="flex items-start gap-3 mb-4">
        <Skeleton variant="circular" width={32} height={32} />
        <div className="flex-1">
          <Skeleton width="60%" height={20} className="mb-2" />
          <Skeleton width="40%" height={14} />
        </div>
      </div>
      <Skeleton width="100%" height={14} className="mb-2" />
      <Skeleton width="80%" height={14} className="mb-4" />
      <div className="flex gap-4">
        <Skeleton width={60} height={32} variant="rounded" />
        <Skeleton width={60} height={32} variant="rounded" />
      </div>
    </div>
  );
}

/**
 * A list skeleton showing multiple items during loading.
 */
export function ListSkeleton({
  count = 3,
  className = "",
}: {
  count?: number;
  className?: string;
}) {
  return (
    <div className={`space-y-3 ${className}`}>
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 p-3 bg-flow-node rounded-lg">
          <Skeleton variant="circular" width={40} height={40} />
          <div className="flex-1">
            <Skeleton width="70%" height={16} className="mb-2" />
            <Skeleton width="50%" height={12} />
          </div>
        </div>
      ))}
    </div>
  );
}

/**
 * A table row skeleton for tabular data loading.
 */
export function TableRowSkeleton({
  columns = 4,
  className = "",
}: {
  columns?: number;
  className?: string;
}) {
  return (
    <div className={`flex items-center gap-4 py-3 ${className}`}>
      {Array.from({ length: columns }).map((_, i) => (
        <Skeleton
          key={i}
          width={i === 0 ? "30%" : "20%"}
          height={16}
        />
      ))}
    </div>
  );
}

export default Skeleton;
