import { ArrowRight } from "lucide-react";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import type { SuiteExecutionResult, SuiteRequest } from "../../lib/api";

interface ContinueSectionProps {
  lastFailedExecution?: SuiteExecutionResult;
  actionableRequest?: SuiteRequest | null;
  recentScenario?: string;
  hasHistory: boolean;
  onResumeDebugging: () => void;
  onRunQueuedTests: () => void;
  onContinueScenario: () => void;
  onGetStarted: () => void;
}

export function ContinueSection({
  lastFailedExecution,
  actionableRequest,
  recentScenario,
  hasHistory,
  onResumeDebugging,
  onRunQueuedTests,
  onContinueScenario,
  onGetStarted
}: ContinueSectionProps) {
  // Determine what to show based on state
  const getContent = () => {
    if (lastFailedExecution) {
      return {
        title: "Resume debugging",
        subtitle: `${lastFailedExecution.scenarioName} failed ${lastFailedExecution.phaseSummary.failed} phase(s)`,
        action: "View failed run",
        onClick: onResumeDebugging
      };
    }

    if (actionableRequest) {
      return {
        title: "Queued tests waiting",
        subtitle: `${actionableRequest.scenarioName} (${actionableRequest.priority} priority) is ready to run`,
        action: "Run queued tests",
        onClick: onRunQueuedTests
      };
    }

    if (recentScenario) {
      return {
        title: "Continue where you left off",
        subtitle: `Pick up with ${recentScenario} or explore other scenarios`,
        action: `Continue with ${recentScenario}`,
        onClick: onContinueScenario
      };
    }

    return {
      title: "Get started",
      subtitle: "Run your first tests to start tracking coverage",
      action: "Run tests",
      onClick: onGetStarted
    };
  };

  const content = getContent();

  return (
    <section
      className="rounded-2xl border border-white/10 bg-gradient-to-r from-cyan-500/10 via-transparent to-purple-500/10 p-6"
      data-testid={selectors.dashboard.continueSection}
    >
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Quick action</p>
          <h2 className="mt-2 text-2xl font-semibold">{content.title}</h2>
          <p className="mt-2 text-sm text-slate-300">{content.subtitle}</p>
        </div>
        <Button onClick={content.onClick}>
          {content.action}
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </section>
  );
}
