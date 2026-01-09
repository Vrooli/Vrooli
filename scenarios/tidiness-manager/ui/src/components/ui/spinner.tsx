import { cn } from "../../lib/utils";

export interface SpinnerProps extends React.HTMLAttributes<HTMLDivElement> {
  size?: "sm" | "md" | "lg";
}

export function Spinner({ className, size = "md", ...props }: SpinnerProps) {
  const sizeClasses = {
    sm: "h-4 w-4 border-2",
    md: "h-8 w-8 border-3",
    lg: "h-12 w-12 border-4",
  };

  return (
    <div
      className={cn(
        "inline-block animate-spin rounded-full border-solid border-slate-600 border-t-slate-100",
        sizeClasses[size],
        className
      )}
      role="status"
      aria-label="Loading"
      data-testid="spinner"
      {...props}
    >
      <span className="sr-only">Loading...</span>
    </div>
  );
}
