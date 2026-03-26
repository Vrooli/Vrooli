import { useQuery } from "@tanstack/react-query";
import { Wand2, Check, X } from "lucide-react";
import { fetchGenerateOptions } from "../lib/api";
import { Section } from "./ui/section";

// [REQ:BM-REQ-UI-GENERATE]

export function GenerateOptions() {
  const { data: options, isLoading } = useQuery({
    queryKey: ["generate-options"],
    queryFn: fetchGenerateOptions,
  });

  if (isLoading) return <div className="text-slate-500 text-sm py-4 text-center">Loading options...</div>;

  if (!options) return null;

  return (
    <Section testId="generate-options-section">
      <h2 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2">
        <Wand2 className="h-4 w-4" /> Generation Options
      </h2>

      <div className="space-y-3">
        {options.providers.map((provider) => (
          <div
            key={provider.id}
            className={`rounded-lg border p-3 transition-colors ${
              provider.available
                ? "border-white/10 bg-white/5 hover:bg-white/10"
                : "border-white/5 bg-white/[0.02] opacity-60"
            }`}
            data-testid={`provider-${provider.id}`}
          >
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm font-medium text-slate-200">{provider.name}</span>
              {provider.available ? (
                <span className="flex items-center gap-1 text-xs text-emerald-400">
                  <Check className="h-3 w-3" /> Available
                </span>
              ) : (
                <span className="flex items-center gap-1 text-xs text-slate-500">
                  <X className="h-3 w-3" /> Not configured
                </span>
              )}
            </div>
            <p className="text-xs text-slate-500 mb-2">{provider.description}</p>
            <div className="flex flex-wrap gap-1">
              {provider.capabilities.map((cap) => (
                <span
                  key={cap}
                  className="rounded-full bg-white/10 px-2 py-0.5 text-[10px] text-slate-400"
                >
                  {cap}
                </span>
              ))}
            </div>
            {provider.requires && !provider.available && (
              <p className="text-[10px] text-slate-600 mt-2">Requires: {provider.requires}</p>
            )}
          </div>
        ))}
      </div>

      <div className="mt-3 text-xs text-slate-600">
        Available elements: {options.elements.join(", ")}
      </div>
    </Section>
  );
}
