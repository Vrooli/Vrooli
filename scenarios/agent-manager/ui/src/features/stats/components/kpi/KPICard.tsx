// KPI Card component for displaying key metrics

import { TrendingUp, TrendingDown, Minus } from "lucide-react";
import type { ReactNode } from "react";

interface KPICardProps {
  title: string;
  value: string;
  subtitle?: string;
  trend?: number; // Percent change, positive is good
  trendLabel?: string;
  icon?: ReactNode;
  loading?: boolean;
  error?: string;
  variant?: "default" | "success" | "warning" | "error";
}

export function KPICard({
  title,
  value,
  subtitle,
  trend,
  trendLabel,
  icon,
  loading,
  error,
  variant = "default",
}: KPICardProps) {
  const getTrendIcon = () => {
    if (trend === undefined || trend === 0) {
      return <Minus className="h-3 w-3" />;
    }
    return trend > 0 ? (
      <TrendingUp className="h-3 w-3" />
    ) : (
      <TrendingDown className="h-3 w-3" />
    );
  };

  const getTrendColor = () => {
    if (trend === undefined || trend === 0) return "text-muted-foreground";
    return trend > 0 ? "text-emerald-500" : "text-red-500";
  };

  const getVariantStyles = () => {
    switch (variant) {
      case "success":
        return "border-emerald-500/30 bg-emerald-500/5";
      case "warning":
        return "border-amber-500/30 bg-amber-500/5";
      case "error":
        return "border-red-500/30 bg-red-500/5";
      default:
        return "border-border bg-card/50";
    }
  };

  if (loading) {
    return (
      <div className={`rounded-lg border p-4 ${getVariantStyles()}`}>
        <div className="animate-pulse space-y-3">
          <div className="h-4 w-20 rounded bg-muted/30" />
          <div className="h-8 w-24 rounded bg-muted/30" />
          <div className="h-3 w-16 rounded bg-muted/30" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-4">
        <p className="text-xs text-muted-foreground">{title}</p>
        <p className="mt-1 text-sm text-red-500">Error loading</p>
      </div>
    );
  }

  return (
    <div className={`rounded-lg border p-4 ${getVariantStyles()}`}>
      <div className="flex items-start justify-between">
        <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {title}
        </p>
        {icon && <div className="text-muted-foreground">{icon}</div>}
      </div>
      <p className="mt-2 text-2xl font-semibold tabular-nums">{value}</p>
      <div className="mt-1 flex items-center gap-2">
        {subtitle && (
          <span className="text-xs text-muted-foreground">{subtitle}</span>
        )}
        {trend !== undefined && (
          <span className={`flex items-center gap-1 text-xs ${getTrendColor()}`}>
            {getTrendIcon()}
            {Math.abs(trend).toFixed(1)}%
            {trendLabel && <span className="text-muted-foreground">{trendLabel}</span>}
          </span>
        )}
      </div>
    </div>
  );
}
