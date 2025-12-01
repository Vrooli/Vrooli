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
  priorityResource?: string | null;
  prioritySecretKey?: string;
  missingCount?: number;
  onOpenResource?: (resourceName: string, secretKey?: string) => void;
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
  priorityResource,
  prioritySecretKey,
  missingCount,
  onOpenResource,
  onJourneySelect,
  onJourneyExit,
  onJourneyNext,
  onJourneyBack
}: OrientationHubProps) => {
  const activeStep = journeySteps[journeyStep];

  return (
    <section className="grid gap-6 lg:grid-cols-[1fr,2fr]">
      {/* LEFT SIDE: Journey Navigation (Primary - F-pattern) */}
      <div className="space-y-4">
        <div className="rounded-3xl border border-white/10 bg-white/5 p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold uppercase tracking-[0.3em] text-white/70">Journeys</h3>
            <Compass className="h-4 w-4 text-white/60" />
          </div>
          <div className="grid gap-3">
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
          <div className="flex items-center justify-between mb-2">
            <h4 className="text-xs font-semibold uppercase tracking-[0.3em] text-white/60">Quick Stats</h4>
          </div>
          {isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-8 w-full" variant="text" />
              <Skeleton className="h-8 w-full" variant="text" />
            </div>
          ) : (
            <div className="space-y-2 text-sm">
              <div className="flex justify-between items-center py-2 border-b border-white/10">
                <span className="text-white/60">Overall Score</span>
                <span className="font-semibold text-white">{heroStats?.overall_score ?? 0}%</span>
              </div>
              <div className="flex justify-between items-center py-2 border-b border-white/10">
                <span className="text-white/60">Risk Score</span>
                <span className="font-semibold text-white">{heroStats?.risk_score ?? 0}</span>
              </div>
              <div className="flex justify-between items-center py-2">
                <span className="text-white/60">Missing Secrets</span>
                <span className="font-semibold text-white">{heroStats?.missing_secrets ?? 0}</span>
              </div>
            </div>
          )}
        </div>
      </div>
      {/* RIGHT SIDE: Journey Content (Secondary) */}
      <div className="rounded-3xl border border-white/10 bg-white/5 p-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Guided Journey</p>
            <h2 className="mt-1 text-2xl font-semibold text-white">
              {activeJourney ? activeJourneyCard?.title || "Journey" : "Get Started"}
            </h2>
          </div>
          <div className="rounded-full border border-white/10 px-3 py-1 text-xs text-white/70">
            {isLoading ? (
              <Skeleton className="h-3 w-16" variant="text" />
            ) : (
              <>Updated {updatedAt ? new Date(updatedAt).toLocaleTimeString() : "—"}</>
            )}
          </div>
        </div>
        {priorityResource && onOpenResource ? (
          <div className="mt-4 rounded-2xl border border-emerald-400/30 bg-emerald-500/5 p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-[11px] uppercase tracking-[0.2em] text-emerald-200/80">Priority action</p>
                <p className="text-sm text-white">
                  {priorityResource} {missingCount ? `· ${missingCount} missing secret${missingCount !== 1 ? "s" : ""}` : ""}
                </p>
                {prioritySecretKey ? (
                  <p className="text-[11px] text-emerald-100/80">Start with {prioritySecretKey}</p>
                ) : (
                  <p className="text-[11px] text-emerald-100/70">Open the workbench to assign strategies</p>
                )}
              </div>
              <Button size="sm" variant="secondary" onClick={() => onOpenResource(priorityResource, prioritySecretKey)}>
                Open workbench
              </Button>
            </div>
          </div>
        ) : null}
        {!activeJourney || journeySteps.length === 0 ? (
          <div className="mt-6">
            <p className="text-white/70">Select a journey from the left to begin your guided experience.</p>
            <div className="mt-4 rounded-2xl border border-white/10 bg-white/5 p-4">
              <p className="text-sm text-white/60">
                💡 <strong className="text-white">Tip:</strong> Start with the Orientation journey to get familiar with your security posture.
              </p>
            </div>
          </div>
        ) : (
          <div className="mt-6 space-y-4">
            <div className="flex items-center justify-between">
              <p className="text-xs text-white/50">
                Step {journeyStep + 1} of {journeySteps.length}
              </p>
              <Map className="h-4 w-4 text-white/40" />
            </div>
            <div>
              <h3 className="text-xl font-semibold text-white">{activeStep?.title}</h3>
              <p className="mt-1 text-sm text-white/60">{activeStep?.description}</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-black/20 p-4">
              {activeStep?.content}
            </div>
            <div className="flex items-center justify-between pt-4 border-t border-white/10">
              <Button
                variant="ghost"
                size="sm"
                onClick={onJourneyExit}
                className="text-xs uppercase tracking-[0.2em]"
              >
                Exit Journey
              </Button>
              <div className="space-x-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onJourneyBack}
                  disabled={journeyStep === 0}
                >
                  ← Back
                </Button>
                <Button
                  size="sm"
                  onClick={onJourneyNext}
                  disabled={journeyStep >= journeySteps.length - 1}
                >
                  Next →
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </section>
  );
};
