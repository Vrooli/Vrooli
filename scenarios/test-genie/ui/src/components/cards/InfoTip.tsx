import { useState } from "react";
import { Info } from "lucide-react";
import { cn } from "../../lib/utils";

interface InfoTipProps {
  title: string;
  description: string;
}

export function InfoTip({ title, description }: InfoTipProps) {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <div className="relative inline-flex" onMouseLeave={() => setIsOpen(false)}>
      <button
        type="button"
        className="flex h-7 w-7 items-center justify-center rounded-full border border-white/20 text-slate-300 transition hover:border-cyan-400 hover:text-white"
        onMouseEnter={() => setIsOpen(true)}
        onFocus={() => setIsOpen(true)}
        onBlur={() => setIsOpen(false)}
        onClick={() => setIsOpen((prev) => !prev)}
        aria-expanded={isOpen}
        aria-label={`More about ${title}`}
      >
        <Info className="h-4 w-4" />
      </button>
      <div
        className={cn(
          "absolute right-0 top-9 z-20 w-64 rounded-2xl border border-white/10 bg-slate-900/95 p-4 text-left text-xs text-slate-200 shadow-2xl transition",
          isOpen ? "pointer-events-auto opacity-100" : "pointer-events-none opacity-0"
        )}
        role="status"
      >
        <p className="font-semibold text-white">{title}</p>
        <p className="mt-1 leading-relaxed">{description}</p>
      </div>
    </div>
  );
}
