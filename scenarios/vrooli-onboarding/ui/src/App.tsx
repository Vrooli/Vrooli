import { useCallback, useRef, useState } from "react";
import { Wand2, Activity, BookOpen } from "lucide-react";
import { WizardShell } from "./components/wizard/WizardShell";
import { StepWelcome } from "./components/wizard/StepWelcome";
import { StepSelectResources } from "./components/wizard/StepSelectResources";
import { StepReview } from "./components/wizard/StepReview";
import { StepComplete } from "./components/wizard/StepComplete";
import { HealthDashboard } from "./components/dashboard/HealthDashboard";
import { GlossaryPanel } from "./components/glossary/GlossaryPanel";
import { useGlobalKeyboardShortcuts } from "./hooks/useGlobalKeyboardShortcuts";
import { useWizardState } from "./hooks/useWizardState";
import { cn } from "./lib/utils";

type AppView = "wizard" | "dashboard" | "glossary";

const NAV_ITEMS: { id: AppView; label: string; icon: React.ReactNode; testId: string }[] = [
  { id: "wizard", label: "Setup Wizard", icon: <Wand2 className="h-4 w-4" aria-hidden="true" />, testId: "nav-wizard" },
  { id: "dashboard", label: "Health Dashboard", icon: <Activity className="h-4 w-4" aria-hidden="true" />, testId: "nav-dashboard" },
  { id: "glossary", label: "Glossary", icon: <BookOpen className="h-4 w-4" aria-hidden="true" />, testId: "nav-glossary" },
];

const VIEW_IDS = NAV_ITEMS.map((item) => item.id);

export default function App() {
  const [view, setView] = useState<AppView>("wizard");
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);

  const {
    currentStep,
    selectedResources,
    resumeAvailable,
    resumeStep,
    stepContentRef,
    handleResume,
    toggleResource,
    goNext,
    goPrev,
    goToStep,
    startOver,
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
    <div className="min-h-screen bg-slate-950 text-slate-50">
      {/* Skip to content link for screen readers */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-lg focus:bg-emerald-500 focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-white focus:outline-none"
        data-testid="skip-to-content"
      >
        Skip to main content
      </a>

      {/* Navigation */}
      <nav
        data-testid="app-nav"
        aria-label="Main navigation"
        className="sticky top-0 z-50 border-b border-white/10 bg-slate-950/95 backdrop-blur-sm"
      >
        <div className="mx-auto flex max-w-5xl items-center gap-0.5 px-2 py-1.5 sm:gap-1 sm:px-6 sm:py-3" role="tablist" aria-label="Application views">
          {NAV_ITEMS.map((item, idx) => (
            <button
              key={item.id}
              ref={(el) => { tabRefs.current[idx] = el; }}
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
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/50",
                view === item.id
                  ? "bg-white/10 text-white"
                  : "text-slate-300 hover:bg-white/5 hover:text-slate-100"
              )}
            >
              {item.icon}
              <span className="hidden sm:inline">{item.label}</span>
              <span className="text-xs sm:hidden">{item.label.split(" ")[0]}</span>
              <kbd className="hidden lg:inline-flex ml-1 h-4 min-w-4 items-center justify-center rounded bg-white/5 px-1 text-[9px] font-mono text-slate-300/60" aria-hidden="true">
                Alt+{idx + 1}
              </kbd>
              {item.id === "wizard" && selectedResources.size > 0 && (
                <span
                  className="ml-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-emerald-500/20 px-1 text-[10px] font-medium text-emerald-400"
                  aria-label={`${selectedResources.size} resources selected`}
                  data-testid="nav-wizard-badge"
                >
                  {selectedResources.size}
                </span>
              )}
            </button>
          ))}
        </div>
      </nav>

      {/* Screen reader step announcement */}
      <div className="sr-only" aria-live="assertive" aria-atomic="true" data-testid="step-announcement">
        {view === "wizard" && `Step ${currentStep + 1} of ${totalSteps}`}
      </div>

      {/* Content */}
      <main id="main-content">
        <div role="tabpanel" id="tabpanel-wizard" aria-labelledby="tab-wizard" hidden={view !== "wizard"} className={view === "wizard" ? "animate-panel-enter" : ""}>
          {view === "wizard" && (
            <WizardShell
              currentStep={currentStep}
              onNext={goNext}
              onPrev={goPrev}
              onGoToStep={goToStep}
              nextDisabled={currentStep === 1 && selectedResources.size === 0}
              nextLabel={nextLabel}
              showPrev={currentStep > 0 && !isLastStep}
              showNext={!isLastStep}
            >
              <div ref={stepContentRef} key={currentStep} className="animate-step-enter">
              {currentStep === 0 && (
                <>
                  <StepWelcome />
                  {resumeAvailable && (
                    <div
                      data-testid="resume-prompt"
                      className="mt-6 rounded-xl border border-blue-500/30 bg-blue-500/10 p-4 text-center"
                      role="alert"
                    >
                      <p className="text-sm text-slate-300">You have saved progress at step {resumeStep + 1}.</p>
                      <button
                        data-testid="resume-button"
                        onClick={handleResume}
                        className="mt-3 inline-flex items-center rounded-full bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
                      >
                        Resume
                      </button>
                    </div>
                  )}
                </>
              )}
              {currentStep === 1 && (
                <StepSelectResources selected={selectedResources} onToggle={toggleResource} />
              )}
              {currentStep === 2 && <StepReview selected={selectedResources} onRemove={toggleResource} onGoBack={goPrev} />}
              {currentStep === 3 && <StepComplete selected={selectedResources} onStartOver={startOver} />}
              </div>
            </WizardShell>
          )}
        </div>
        <div role="tabpanel" id="tabpanel-dashboard" aria-labelledby="tab-dashboard" className={cn("mx-auto max-w-5xl px-3 py-4 sm:px-6 sm:py-8", view === "dashboard" && "animate-panel-enter")} hidden={view !== "dashboard"}>
          {view === "dashboard" && <HealthDashboard onNavigateToWizard={() => { setView("wizard"); tabRefs.current[0]?.focus(); }} />}
        </div>
        <div role="tabpanel" id="tabpanel-glossary" aria-labelledby="tab-glossary" className={cn("mx-auto max-w-3xl px-3 py-4 sm:px-6 sm:py-8", view === "glossary" && "animate-panel-enter")} hidden={view !== "glossary"}>
          {view === "glossary" && <GlossaryPanel />}
        </div>
      </main>
    </div>
  );
}
