import type { PricingSource } from "../../types";

export function pricingSourceColor(source: PricingSource): string {
  switch (source) {
    case "manual_override":
      return "bg-purple-500/20 text-purple-300 border-purple-500/30";
    case "provider_api":
      return "bg-green-500/20 text-green-300 border-green-500/30";
    case "historical_average":
      return "bg-blue-500/20 text-blue-300 border-blue-500/30";
    default:
      return "bg-gray-500/20 text-gray-300 border-gray-500/30";
  }
}

export function pricingSourceLabel(source: PricingSource): string {
  switch (source) {
    case "manual_override":
      return "Manual";
    case "provider_api":
      return "Provider";
    case "historical_average":
      return "Historical";
    default:
      return "Unknown";
  }
}

export function pricingSourceBadgeClass(
  source: PricingSource,
  options?: { isActive?: boolean; clickable?: boolean; className?: string }
): string {
  const activeClasses = options?.isActive ? "ring-2 ring-offset-1 ring-offset-background" : "";
  const clickableClasses = options?.clickable ? "cursor-pointer hover:opacity-80 transition-opacity" : "";
  return `text-xs ${pricingSourceColor(source)} ${activeClasses} ${clickableClasses} ${options?.className ?? ""}`;
}

export function formatPricingDisplay(price: number | undefined): string {
  if (price === undefined || price === 0) return "-";
  if (price < 0.01) return `$${price.toFixed(4)}`;
  return `$${price.toFixed(2)}`;
}

export function formatPricingTimestamp(ts: string | undefined): string {
  if (!ts) return "-";
  try {
    return new Date(ts).toLocaleString();
  } catch {
    return ts;
  }
}
