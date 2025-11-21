import { Compass, Map } from "lucide-react";
import { Button } from "../components/ui/button";
import { StatCard } from "../components/ui/StatCard";
import { LoadingStatCard, Skeleton } from "../components/ui/LoadingStates";
import type { JourneyId } from "../features/journeys/journeySteps";

interface HeroStats {
  overall_score: number;
  readiness_label: string;
  risk_score: number;
  confidence: number;
  vault_configured: number;
  vault_total: number;
  missing_secrets: number;
}

interface VulnerabilityInsight {
  severity: string;
  message: string;
}

interface JourneyCard {
  id: string;
  badge: string;
  title: string;
  description: string;
  primers: string[];
}

interface JourneyStep {
  title: string;
  description: string;
  content?: React.ReactNode;
}

interface OrientationHubProps {
  heroStats?: HeroStats;
  vulnerabilityInsights: VulnerabilityInsight[];
  journeyCards: JourneyCard[];
  activeJourneyCard?: JourneyCard;
  activeJourney: string | null;
  journeySteps: JourneyStep[];
  journeyStep: number;
  updatedAt?: string;
  isLoading: boolean;
  onJourneySelect: (journeyId: JourneyId) => void;
  onJourneyExit: () => void;
  onJourneyNext: () => void;
  onJourneyBack: () => void;
}

export const OrientationHub = ({
  heroStats,
  vulnerabilityInsights,
  journeyCards,
  activeJourneyCard,
  activeJourney,
  journeySteps,
  journeyStep,
  updatedAt,
  isLoading,
  onJourneySelect,
  onJourneyExit,
  onJourneyNext,
  onJourneyBack
}: OrientationHubProps) => {
  const activeStep = journeySteps[journeyStep];

  return (
    <section className="grid gap-6 lg:grid-cols-[2fr,1fr]">
      <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Orientation Hub</p>
            <h2 className="mt-1 text-2xl font-semibold text-white">Readiness Snapshot</h2>
          </div>
          <div className="rounded-full border border-white/10 px-3 py-1 text-xs text-white/70">
            {isLoading ? (
              <Skeleton className="h-3 w-16" variant="text" />
            ) : (
              <>Updated {updatedAt ? new Date(updatedAt).toLocaleTimeString() : "—"}</>
            )}
          </div>
        </div>
        {isLoading ? (
          <div className="mt-4 grid gap-4 md:grid-cols-3">
            <LoadingStatCard />
            <LoadingStatCard />
            <LoadingStatCard />
          </div>
        ) : (
          <>
            <div className="mt-4 grid gap-4 md:grid-cols-3">
              <StatCard label="Overall" value={`${heroStats?.overall_score ?? 0}%`} description={heroStats?.readiness_label} />
              <StatCard label="Risk" value={`${heroStats?.risk_score ?? 0}`} description="Security score" />
              <StatCard
                label="Confidence"
                value={`${Math.round((heroStats?.confidence ?? 0) * 100)}%`}
                description="Best tier coverage"
              />
            </div>
            <div className="mt-6 flex flex-wrap gap-3">
              {vulnerabilityInsights.map((highlight) => (
                <span
                  key={highlight.severity}
                  className="rounded-full border border-white/10 bg-black/30 px-3 py-1 text-xs uppercase tracking-[0.2em] text-white/70"
                >
                  {highlight.message}
                </span>
              ))}
            </div>
          </>
        )}
      </div>
      <div className="space-y-4">
        <div className="rounded-3xl border border-white/10 bg-white/5 p-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold uppercase tracking-[0.3em] text-white/70">Journeys</h3>
            <Compass className="h-4 w-4 text-white/60" />
          </div>
          <div className="mt-3 grid gap-3">
            {journeyCards.map((card) => (
              <button
                key={card.id}
                onClick={() => onJourneySelect(card.id as JourneyId)}
                className={`w-full rounded-2xl border px-4 py-3 text-left transition hover:border-white/40 ${
                  activeJourney === card.id ? "border-emerald-400 bg-emerald-500/10" : "border-white/10 bg-black/30"
                }`}
              >
                <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-white/60">
                  <span>{card.badge}</span>
                  <span>{card.primers[0]}</span>
                </div>
                <p className="mt-1 text-base font-semibold text-white">{card.title}</p>
                <p className="text-sm text-white/60">{card.description}</p>
              </button>
            ))}
          </div>
        </div>
        <div className="rounded-3xl border border-white/10 bg-white/5 p-4">
          <div className="flex items-center justify-between text-xs uppercase tracking-[0.3em] text-white/60">
            <span>Guided Flow</span>
            <Map className="h-4 w-4 text-white/60" />
          </div>
          {!activeJourney || journeySteps.length === 0 ? (
            <p className="mt-3 text-sm text-white/60">Select a journey to start a guided experience.</p>
          ) : (
            <div className="mt-3 space-y-3">
              <p className="text-xs text-white/50">
                Step {journeyStep + 1} of {journeySteps.length} — {activeJourneyCard?.title}
              </p>
              <h4 className="text-lg font-semibold text-white">{activeStep?.title}</h4>
              <p className="text-sm text-white/70">{activeStep?.description}</p>
              {activeStep?.content}
              <div className="flex items-center justify-between pt-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onJourneyExit}
                  className="text-xs uppercase tracking-[0.2em]"
                >
                  Exit
                </Button>
                <div className="space-x-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onJourneyBack}
                    disabled={journeyStep === 0}
                  >
                    Back
                  </Button>
                  <Button
                    size="sm"
                    onClick={onJourneyNext}
                    disabled={journeyStep >= journeySteps.length - 1}
                  >
                    Next
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  );
};
