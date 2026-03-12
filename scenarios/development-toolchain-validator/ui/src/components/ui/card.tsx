import { cn } from "../../lib/utils";

// ─────────────────────────────────────────────────────────────────────────────
// Card Primitive
// [REQ:P0-001] Reference Scenario Registry - Reusable card component
// ─────────────────────────────────────────────────────────────────────────────
//
// A flexible card primitive for content containers. Supports multiple variants
// for different contexts (default surface, interactive clickable, muted).
// ─────────────────────────────────────────────────────────────────────────────

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  /** Visual variant */
  variant?: "default" | "interactive" | "muted";
}

export function Card({ className, variant = "default", ...props }: CardProps) {
  const variantClasses = {
    default: "border-white/10 bg-white/5",
    interactive: "border-white/10 bg-white/5 transition-colors hover:border-indigo-500/30 hover:bg-white/8 cursor-pointer",
    muted: "border-white/5 bg-white/3"
  };

  return (
    <div
      className={cn(
        "rounded-xl border",
        variantClasses[variant],
        className
      )}
      {...props}
    />
  );
}

export type CardHeaderProps = React.HTMLAttributes<HTMLDivElement>;

export function CardHeader({ className, ...props }: CardHeaderProps) {
  return (
    <div
      className={cn("p-6 pb-4", className)}
      {...props}
    />
  );
}

export type CardContentProps = React.HTMLAttributes<HTMLDivElement>;

export function CardContent({ className, ...props }: CardContentProps) {
  return (
    <div
      className={cn("p-6 pt-0", className)}
      {...props}
    />
  );
}

export type CardFooterProps = React.HTMLAttributes<HTMLDivElement>;

export function CardFooter({ className, ...props }: CardFooterProps) {
  return (
    <div
      className={cn("p-6 pt-0 flex items-center", className)}
      {...props}
    />
  );
}
