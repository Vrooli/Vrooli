import { cn } from "../lib/utils";

interface ColorSwatchProps {
  color?: string;
  label: string;
  className?: string;
}

export function ColorSwatch({ color, label, className }: ColorSwatchProps) {
  if (!color) return null;
  return (
    <div className={cn("flex items-center gap-2", className)} data-testid={`color-swatch-${label.toLowerCase()}`}>
      <div
        className="h-8 w-8 rounded-md border border-white/20 shrink-0"
        style={{ backgroundColor: color }}
      />
      <div className="text-sm">
        <p className="text-slate-300">{label}</p>
        <p className="font-mono text-slate-400">{color}</p>
      </div>
    </div>
  );
}
