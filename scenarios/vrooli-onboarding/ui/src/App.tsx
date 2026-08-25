import { useCallback, useRef, useState } from "react";
import { Wand2, Activity, BookOpen } from "lucide-react";
import { WizardShell } from "./components/wizard/WizardShell";
import { HealthDashboard } from "./components/dashboard/HealthDashboard";
import { GlossaryPanel } from "./components/glossary/GlossaryPanel";
import { useGlobalKeyboardShortcuts } from "./hooks/useGlobalKeyboardShortcuts";
import { useWizardState } from "./hooks/useWizardState";
import { cn } from "./lib/utils";
import { Button } from "./components/ui/button";
import { stepRegistry } from "./components/wizard/stepRegistry";

type AppView = "wizard" | "dashboard" | "glossary";

function initialViewForPath(pathname: string): AppView {
  if (pathname === "/health-dashboard") return "dashboard";
  if (pathname === "/glossary") return "glossary";
  return "wizard";
}

const NAV_ITEMS: {
  id: AppView;
  label: string;
  icon: React.ReactNode;
  testId: string;
}[] = [
  {
    id: "wizard",
    label: "Setup Wizard",
    icon: <Wand2 className="h-4 w-4" aria-hidden="true" />,
    testId: "nav-wizard",
  },
  {
    id: "dashboard",
    label: "Health Dashboard",
    icon: <Activity className="h-4 w-4" aria-hidden="true" />,
    testId: "nav-dashboard",
  },
  {
    id: "glossary",
    label: "Glossary",
    icon: <BookOpen className="h-4 w-4" aria-hidden="true" />,
    testId: "nav-glossary",
  },
];

const VIEW_IDS = NAV_ITEMS.map((item) => item.id);

export default function App() {
  const [view, setView] = useState<AppView>(() =>
    initialViewForPath(window.location.pathname),
  );
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);

  const {
    currentStep,
    steps,
    stepsLoading,
    stepsError,
    selectedScenarios,
    operatorState,
    stepContentRef,
    toggleScenario,
    setScenarioAutoRestart,
    setHostOptIn,
    setHostConfig,
    setResourceEnabled,
    goNext,
    goPrev,
    goToStep,
    nextLabel,
    isLastStep,
    totalSteps,
  } = useWizardState();

  // WAI-ARIA tablist keyboard navigation: Left/Right arrows, Home/End
  const handleTabKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLButtonElement>) => {
      const currentIndex = VIEW_IDS.indexOf(view);
      let nextIndex = -1;

      if (e.key === "ArrowRight" || e.key === "ArrowDown") {
        nextIndex = (currentIndex + 1) % VIEW_IDS.length;
      } else if (e.key === "ArrowLeft" || e.key === "ArrowUp") {
        nextIndex = (currentIndex - 1 + VIEW_IDS.length) % VIEW_IDS.length;
      } else if (e.key === "Home") {
        nextIndex = 0;
      } else if (e.key === "End") {
        nextIndex = VIEW_IDS.length - 1;
      }

      if (nextIndex >= 0) {
        e.preventDefault();
        const nextView = VIEW_IDS[nextIndex];
        if (nextView) setView(nextView);
        tabRefs.current[nextIndex]?.focus();
      }
    },
    [view],
  );

  // Global keyboard shortcuts: Alt+1/2/3 to switch views
  useGlobalKeyboardShortcuts(VIEW_IDS, (index) => {
    const viewId = VIEW_IDS[index];
    if (viewId) {
      setView(viewId);
      tabRefs.current[index]?.focus();
    }
  });

  return (
    <div className="min-h-full bg-surface text-foreground">
      {/* Skip to content link for screen readers */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-lg focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-foreground focus:outline-none"
        data-testid="skip-to-content"
      >
        Skip to main content
      </a>

      {/* Navigation */}
      <div
        role="navigation"
        data-testid="app-nav"
        aria-label="Main navigation"
        className="sticky top-0 z-50 border-b border-muted bg-surface/95 backdrop-blur-sm"
      >
        <div
          className="mx-auto flex max-w-5xl items-center gap-0.5 px-2 py-1.5 sm:gap-1 sm:px-6 sm:py-3"
          role="tablist"
          aria-label="Application views"
        >
          {NAV_ITEMS.map((item, idx) => (
            <Button
              variant="ghost"
              key={item.id}
              ref={(el) => {
                tabRefs.current[idx] = el;
              }}
              role="tab"
              data-testid={item.testId}
              onClick={() => setView(item.id)}
              onKeyDown={handleTabKeyDown}
              aria-selected={view === item.id}
              aria-controls={`tabpanel-${item.id}`}
              id={`tab-${item.id}`}
              tabIndex={view === item.id ? 0 : -1}
              className={cn(
                "inline-flex items-center gap-1.5 rounded-lg px-2.5 py-2 text-sm font-medium transition-colors sm:gap-2 sm:px-3",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus/50",
                view === item.id
                  ? "bg-surface-subtle text-foreground"
                  : "text-muted hover:bg-surface-muted hover:text-foreground",
              )}
            >
              {item.icon}
              <span className="hidden sm:inline">{item.label}</span>
              <span className="text-xs sm:hidden">
                {item.label.split(" ")[0]}
              </span>
              <kbd
                className="hidden lg:inline-flex ml-1 h-4 min-w-4 items-center justify-center rounded bg-surface-muted px-1 text-[9px] font-mono text-muted/60"
                aria-hidden="true"
              >
                Alt+{idx + 1}
              </kbd>
              {item.id === "wizard" && selectedScenarios.size > 0 && (
                <span
                  className="ml-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-primary/20 px-1 text-[10px] font-medium text-primary"
                  aria-label={`${selectedScenarios.size} scenarios selected`}
                  data-testid="nav-wizard-badge"
                >
                  {selectedScenarios.size}
                </span>
              )}
            </Button>
          ))}
        </div>
      </div>

      {/* Screen reader step announcement */}
      <div
        className="sr-only"
        aria-live="assertive"
        aria-atomic="true"
        data-testid="step-announcement"
      >
        {view === "wizard" && `Step ${currentStep + 1} of ${totalSteps}`}
      </div>

      {/* Content */}
      <main id="main-content">
        <div
          role="tabpanel"
          id="tabpanel-wizard"
          aria-labelledby="tab-wizard"
          hidden={view !== "wizard"}
          className={view === "wizard" ? "animate-panel-enter" : ""}
        >
          {view === "wizard" && stepsLoading && (
            <div data-testid="wizard-shell" className="contents">
              <div
                className="mx-auto max-w-3xl px-3 py-8"
                data-testid="wizard-loading"
                role="status"
              >
                <h1 className="text-2xl font-semibold">Welcome to Vrooli</h1>
                Loading onboarding steps…
              </div>
            </div>
          )}
          {view === "wizard" && stepsError && !stepsLoading && (
            <div data-testid="wizard-shell" className="contents">
              <div
                className="mx-auto max-w-3xl px-3 py-8"
                data-testid="wizard-error"
                role="alert"
              >
                <h1 className="text-2xl font-semibold">Welcome to Vrooli</h1>
                {stepsError}
              </div>
            </div>
          )}
          {view === "wizard" &&
            !stepsLoading &&
            !stepsError &&
            steps.length > 0 && (
              <WizardShell
                currentStep={currentStep}
                steps={steps}
                onNext={goNext}
                onPrev={goPrev}
                onGoToStep={goToStep}
                nextDisabled={
                  steps[currentStep]?.id === "scenarios" &&
                  selectedScenarios.size === 0
                }
                nextLabel={nextLabel}
                showPrev={currentStep > 0}
                showNext={!isLastStep}
              >
                <div
                  ref={stepContentRef}
                  key={currentStep}
                  className="animate-step-enter"
                >
                  {steps[currentStep] &&
                    stepRegistry[steps[currentStep].id]?.({
                      step: steps[currentStep],
                      selectedScenarios,
                      operatorState,
                      toggleScenario,
                      setScenarioAutoRestart,
                      setHostOptIn,
                      setHostConfig,
                      setResourceEnabled,
                    })}
                </div>
              </WizardShell>
            )}
        </div>
        <div
          role="tabpanel"
          id="tabpanel-dashboard"
          aria-labelledby="tab-dashboard"
          className={cn(
            "mx-auto max-w-5xl px-3 py-4 sm:px-6 sm:py-8",
            view === "dashboard" && "animate-panel-enter",
          )}
          hidden={view !== "dashboard"}
        >
          {view === "dashboard" && (
            <HealthDashboard
              onNavigateToWizard={() => {
                setView("wizard");
                tabRefs.current[0]?.focus();
              }}
            />
          )}
        </div>
        <div
          role="tabpanel"
          id="tabpanel-glossary"
          aria-labelledby="tab-glossary"
          className={cn(
            "mx-auto max-w-3xl px-3 py-4 sm:px-6 sm:py-8",
            view === "glossary" && "animate-panel-enter",
          )}
          hidden={view !== "glossary"}
        >
          {view === "glossary" && <GlossaryPanel />}
        </div>
      </main>
    </div>
  );
}
