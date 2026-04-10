import type { Brand } from "../lib/api";
import { formatDate } from "../lib/utils";

interface BrandCardProps {
  brand: Brand;
  onClick: () => void;
}

export function BrandCard({ brand, onClick }: BrandCardProps) {
  const colors = brand.colors;
  return (
    <button
      onClick={onClick}
      data-testid={`brand-card-${brand.id}`}
      className="w-full text-left rounded-xl border border-white/10 bg-white/5 p-5 hover:bg-white/10 transition-colors"
    >
      <div className="flex items-start justify-between">
        <div className="min-w-0 flex-1">
          <h3 className="text-lg font-semibold text-slate-50 truncate">{brand.name}</h3>
          {brand.description && (
            <p className="mt-1 text-sm text-slate-400 line-clamp-2">{brand.description}</p>
          )}
        </div>
        {colors && (
          <div className="flex gap-1 ml-3 shrink-0">
            {[colors.primary, colors.secondary, colors.accent].filter(Boolean).map((c, i) => (
              <div
                key={i}
                className="h-6 w-6 rounded-full border border-white/20"
                style={{ backgroundColor: c }}
              />
            ))}
          </div>
        )}
      </div>
      <div className="mt-3 flex items-center gap-3 text-xs text-slate-500">
        <span>v{brand.version}</span>
        <span>Updated {formatDate(brand.updated_at)}</span>
      </div>
    </button>
  );
}
