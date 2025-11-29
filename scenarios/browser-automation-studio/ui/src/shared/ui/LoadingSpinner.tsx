import { Loader2, Zap } from "lucide-react";

interface LoadingSpinnerProps {
  /** Size of the spinner in pixels */
  size?: number;
  /** Text to display below the spinner */
  message?: string;
  /** Variant style */
  variant?: "default" | "branded" | "minimal";
  /** Additional CSS class */
  className?: string;
}

/**
 * A loading spinner component with multiple variants.
 * Use this for indicating loading states throughout the app.
 *
 * @example
 * <LoadingSpinner message="Loading workflows..." />
 *
 * @example
 * <LoadingSpinner variant="branded" size={48} />
 */
function LoadingSpinner({
  size = 24,
  message,
  variant = "default",
  className = "",
}: LoadingSpinnerProps) {
  if (variant === "minimal") {
    return (
      <div className={`flex items-center justify-center ${className}`}>
        <Loader2 size={size} className="animate-spin text-gray-400" />
      </div>
    );
  }

  if (variant === "branded") {
    return (
      <div className={`flex flex-col items-center justify-center gap-4 ${className}`}>
        <div className="relative">
          {/* Outer glow */}
          <div className="absolute inset-0 bg-flow-accent/20 rounded-full blur-xl animate-pulse" />
          {/* Inner container */}
          <div className="relative p-4 bg-flow-node border border-gray-700 rounded-full">
            <Zap size={size} className="text-flow-accent animate-pulse" />
          </div>
          {/* Spinning ring */}
          <div
            className="absolute inset-[-4px] border-2 border-transparent border-t-flow-accent rounded-full animate-spin"
            style={{ animationDuration: "1.5s" }}
          />
        </div>
        {message && (
          <p className="text-gray-400 text-sm animate-pulse">{message}</p>
        )}
      </div>
    );
  }

  // Default variant
  return (
    <div className={`flex flex-col items-center justify-center gap-3 ${className}`}>
      <div className="relative">
        <Loader2 size={size} className="animate-spin text-flow-accent" />
      </div>
      {message && (
        <p className="text-gray-400 text-sm">{message}</p>
      )}
    </div>
  );
}

/**
 * Full-page loading overlay
 */
export function LoadingOverlay({
  message = "Loading...",
  className = "",
}: {
  message?: string;
  className?: string;
}) {
  return (
    <div
      className={`fixed inset-0 z-50 flex items-center justify-center bg-flow-bg/90 backdrop-blur-sm ${className}`}
    >
      <LoadingSpinner variant="branded" size={32} message={message} />
    </div>
  );
}

/**
 * Inline loading indicator for use within content
 */
export function InlineLoading({
  message,
  className = "",
}: {
  message?: string;
  className?: string;
}) {
  return (
    <div className={`flex items-center gap-2 text-gray-400 text-sm ${className}`}>
      <Loader2 size={14} className="animate-spin" />
      {message && <span>{message}</span>}
    </div>
  );
}

/**
 * Loading state for cards or content areas
 */
export function ContentLoading({
  message = "Loading content...",
  className = "",
}: {
  message?: string;
  className?: string;
}) {
  return (
    <div
      className={`flex flex-col items-center justify-center py-12 px-6 bg-flow-node/50 border border-gray-800 rounded-lg ${className}`}
    >
      <LoadingSpinner variant="default" size={28} message={message} />
    </div>
  );
}

export default LoadingSpinner;
