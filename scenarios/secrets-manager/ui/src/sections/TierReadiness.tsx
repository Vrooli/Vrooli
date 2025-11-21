import { Layers } from "lucide-react";
import { LoadingStatCard } from "../components/ui/LoadingStates";

interface TierReadinessData {
  tier: string;
  label: string;
  ready_percent: number;
  strategized: number;
  total: number;
  blocking_secret_sample: string[];
}

interface TierReadinessProps {
  tierReadiness: TierReadinessData[];
  isLoading: boolean;
}

export const TierReadiness = ({ tierReadiness, isLoading }: TierReadinessProps) => (
  <section className="grid gap-4 md:grid-cols-3">
    {isLoading ? (
      <>
        <LoadingStatCard />
        <LoadingStatCard />
        <LoadingStatCard />
      </>
    ) : (
      tierReadiness.map((tier) => (
        <div key={tier.tier} className="rounded-3xl border border-white/10 bg-white/5 p-4">
          <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-white/60">
            <span>{tier.label}</span>
            <Layers className="h-4 w-4 text-white/60" />
          </div>
          <p className="mt-2 text-3xl font-semibold text-white">{tier.ready_percent}%</p>
          <p className="text-sm text-white/60">
            {tier.strategized}/{tier.total} required secrets covered
          </p>
          {tier.blocking_secret_sample.length ? (
            <p className="mt-2 text-xs text-white/50">Blockers: {tier.blocking_secret_sample.join(", ")}</p>
          ) : null}
        </div>
      ))
    )}
  </section>
);
