import { useCallback, useEffect, useRef, useState } from "react";
import { Wand2, Activity, BookOpen, Search } from "lucide-react";
import { WizardShell } from "./components/wizard/WizardShell";
import { HealthDashboard } from "./components/dashboard/HealthDashboard";
import { GlossaryPanel } from "./components/glossary/GlossaryPanel";
import { useGlobalKeyboardShortcuts } from "./hooks/useGlobalKeyboardShortcuts";
import { useWizardState } from "./hooks/useWizardState";
import { cn } from "./lib/utils";
import { Button } from "@vrooli/react-component-library/Button/2";
import { TargetSwitcher, type TargetSwitcherOption } from "@vrooli/react-component-library/TargetSwitcher/0";
import { AppShell } from "./components/layout/AppShell";
import { stepRegistry } from "./components/wizard/stepRegistry";
import { fetchTargets } from "./api/host";
import type { OnboardingTarget } from "./api/host";
import { i18n } from "./i18n";
import { ConfigurationSearchPanel } from "./components/configuration/ConfigurationSearchPanel";
import type { ConfigurationDescriptor } from "./api/configuration";
import vrooliWordmark from "../../../../assets/readme-display.png";

type AppView = "wizard" | "dashboard" | "glossary" | "configuration";

function initialViewForPath(pathname: string): AppView {
  if (pathname === "/health-dashboard") return "dashboard";
  if (pathname === "/glossary") return "glossary";
  if (pathname === "/configuration") return "configuration";
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
    label: i18n.t("onboarding.app.setupWizard"),
    icon: <Wand2 className="h-4 w-4" aria-hidden="true" />,
    testId: "nav-wizard",
  },
  {
    id: "dashboard",
    label: i18n.t("onboarding.app.healthDashboard"),
    icon: <Activity className="h-4 w-4" aria-hidden="true" />,
    testId: "nav-dashboard",
  },
  {
    id: "glossary",
    label: i18n.t("onboarding.app.glossary"),
    icon: <BookOpen className="h-4 w-4" aria-hidden="true" />,
    testId: "nav-glossary",
  },
  {
    id: "configuration",
    label: i18n.t("onboarding.app.configuration"),
    icon: <Search className="h-4 w-4" aria-hidden="true" />,
    testId: "nav-configuration",
  },
];

const VIEW_IDS = NAV_ITEMS.map((item) => item.id);

function capitalizeLabel(value: string): string {
  const trimmed = value.trim();
  return trimmed ? `${trimmed.charAt(0).toUpperCase()}${trimmed.slice(1)}` : value;
}

function formatTargetEnumLabel(value: string): string {
  const normalized = value.trim().replace(/^(node_status|node_kind)_/i, "").replace(/[_-]+/g, " ").toLowerCase();
  return normalized ? normalized.split(/\s+/).map((word) => `${word.charAt(0).toUpperCase()}${word.slice(1)}`).join(" ") : value;
}

export default function App() {
  const [view, setView] = useState<AppView>(() =>
    initialViewForPath(window.location.pathname),
  );
  const [target, setTarget] = useState(() => new URLSearchParams(window.location.search).get("target") || "local");
  const [targetOptions, setTargetOptions] = useState<OnboardingTarget[]>([{ id: "local", name: "This machine", status: "local" }]);
  useEffect(() => {
    fetchTargets().then((result) => {
      if (Array.isArray(result?.targets) && result.targets.length > 0) setTargetOptions(result.targets);
    }).catch(() => undefined);
  }, []);
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
    setCoreSeed,
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
    planAccepted,
    operatorStateError,
    operatorStateSaveState,
    retryOperatorStateSave,
    profileSession,
    profileSessionError,
    profileSessionSaveState,
    persistProfileSession,
    retryProfileSessionSave,
    acceptRecommendation,
  } = useWizardState(target);

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

  // The shared FormWizard owns step navigation; each step still receives the
  // same durable state and callbacks, so the question schema is rendered once
  // by the registry rather than reimplemented in the shell.
  const renderStep = (step: (typeof steps)[number]) => stepRegistry[step.id]?.({
    step,
    selectedScenarios,
    operatorState,
    toggleScenario,
    setCoreSeed,
    setScenarioAutoRestart,
    setHostOptIn,
    setHostConfig,
    setResourceEnabled,
    target,
    acceptRecommendation,
    onAdjustRecommendation: () => goToStep(1),
    profileSession,
    profileSessionBaseRevision: operatorState?.updatedAt ?? operatorState?.version ?? "",
    onProfileSessionChange: persistProfileSession,
    profileSessionError,
    profileSessionSaveState,
    onRetryProfileSessionSave: () => { void retryProfileSessionSave(); },
  });

  const openConfiguration = (descriptor: ConfigurationDescriptor) => {
    const params = new URLSearchParams(window.location.search);
    params.set("target", target || "local");
    params.set("setting", descriptor.id);
    params.set("draft", "current");
    const query = params.toString();
    window.history.pushState({}, "", `${descriptor.route}${query ? `?${query}` : ""}`);
    setView("wizard");
    tabRefs.current[0]?.focus();
    window.dispatchEvent(new PopStateEvent("popstate"));
  };

  const selectedTarget = targetOptions.find((option) => option.id === target);
  const selectedTargetStatus = formatTargetEnumLabel(selectedTarget?.status ?? (target === "local" ? "local" : "ready"));
  const selectedTargetUnavailable = selectedTarget?.available === false || selectedTarget?.online === false;
  const targetTone = (option: OnboardingTarget): TargetSwitcherOption["statusTone"] => {
    if (option.id === "local") return "local";
    if (option.available === false || option.online === false) return "warning";
    return "success";
  };
  const targetSwitcherOptions: TargetSwitcherOption[] = targetOptions.map((option) => ({
    id: option.id,
    label: option.id === "local" ? i18n.t("onboarding.app.local") : capitalizeLabel(option.name ?? option.id),
    meta: [option.os, option.architecture, option.kind].filter(Boolean).map((value) => capitalizeLabel(value as string)).join(" · ") || undefined,
    description: option.reason,
    status: option.available === false || option.online === false ? i18n.t("onboarding.app.targetUnavailable") : formatTargetEnumLabel(option.status ?? (option.id === "local" ? "local" : "ready")),
    statusTone: targetTone(option),
    badge: option.kind ? formatTargetEnumLabel(option.kind) : undefined,
    disabled: option.id !== "local" && (option.available === false || option.online === false),
  }));
  const selectTarget = (nextTarget: string) => {
    setTarget(nextTarget);
    const params = new URLSearchParams(window.location.search);
    params.set("target", nextTarget);
    window.history.replaceState({}, "", `${window.location.pathname}?${params.toString()}`);
  };

  return (
    <AppShell>
    <div className="min-h-full bg-surface text-foreground" data-plan-accepted={planAccepted ? "true" : "false"}>
      {/* Skip to content link for screen readers */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-lg focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-foreground focus:outline-none"
        data-testid="skip-to-content"
      >
        {i18n.t("onboarding.app.skip")}
      </a>

      {/* Navigation */}
      <div
        role="navigation"
        data-testid="app-nav"
        aria-label={i18n.t("onboarding.app.mainNavigation")}
        className="app-bar"
      >
        <div className="app-bar__inner">
          <div className="app-brand" aria-label={i18n.t("onboarding.app.brand")}>
            <img
              className="app-brand__wordmark"
              src={vrooliWordmark}
              alt={i18n.t("onboarding.app.brand")}
            />
          </div>
          <div
            className="app-tabs"
            role="tablist"
            aria-label={i18n.t("onboarding.app.applicationViews")}
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
              style={{ minBlockSize: "var(--tap-target-min)" }}
              aria-selected={view === item.id}
              aria-controls={`tabpanel-${item.id}`}
              id={`tab-${item.id}`}
              aria-label={item.label}
              title={item.label}
              tabIndex={view === item.id ? 0 : -1}
              className={cn(
                "app-tab min-h-11 inline-flex items-center rounded-lg px-2.5 py-2 text-sm font-medium transition-colors sm:px-3",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus/50",
                view === item.id
                  ? "bg-surface-subtle text-foreground"
                  : "text-muted hover:bg-surface-muted hover:text-foreground",
              )}
            >
              <span className="app-tab__content">
                <span data-control-slot="icon" className="app-tab__icon" aria-hidden="true">{item.icon}</span>
                <span className="app-tab__label">{item.label}</span>
                <kbd
                  className="hidden lg:inline-flex h-4 min-w-4 items-center justify-center rounded bg-surface-muted px-1 text-[9px] font-mono text-muted/60"
                  aria-hidden="true"
                >
                  Alt+{idx + 1}
                </kbd>
                {item.id === "wizard" && selectedScenarios.size > 0 && (
                  <span
                    className="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-primary/20 px-1 text-[10px] font-medium text-primary"
                    aria-label={i18n.t("onboarding.app.selected", { count: selectedScenarios.size })}
                    data-testid="nav-wizard-badge"
                  >
                    {selectedScenarios.size}
                  </span>
                )}
              </span>
            </Button>
          ))}
          </div>
          <TargetSwitcher
            className="target-switcher"
            value={target}
            options={targetSwitcherOptions}
            onValueChange={selectTarget}
            label={i18n.t("onboarding.app.setupTarget")}
            eyebrow={i18n.t("onboarding.app.availableTargets")}
            description={selectedTarget?.reason || i18n.t("onboarding.app.targetDetails")}
            statusLabel={i18n.t("onboarding.app.targetStatus")}
            statusValue={selectedTargetUnavailable ? i18n.t("onboarding.app.targetUnavailable") : selectedTargetStatus}
            statusTone={targetTone(selectedTarget ?? { id: "local" })}
            addTargetLabel={i18n.t("onboarding.app.addTarget")}
            onAddTarget={() => window.open("/apps/vrooli-bridge/proxy/", "_blank", "noopener,noreferrer")}
            testId="setup-target-trigger"
          />
        </div>
      </div>

      {/* Screen reader step announcement */}
      <div
        className="sr-only"
        aria-live="assertive"
        aria-atomic="true"
        data-testid="step-announcement"
      >
        {view === "wizard" && i18n.t("onboarding.app.step", { current: currentStep + 1, total: totalSteps })}
      </div>

      {/* Content */}
      <div id="main-content">
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
                <h1 className="text-2xl font-semibold">{i18n.t("onboarding.app.welcome")}</h1>
                {i18n.t("onboarding.app.loadingSteps")}
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
                <h1 className="text-2xl font-semibold">{i18n.t("onboarding.app.welcome")}</h1>
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
                target={target}
                operatorStateError={operatorStateError}
                operatorStateSaveState={operatorStateSaveState}
                onRetryOperatorStateSave={() => { void retryOperatorStateSave(); }}
                onTargetChange={(nextTarget) => {
                  const normalized = nextTarget.trim() || "local";
                  setTarget(normalized);
                  const params = new URLSearchParams(window.location.search);
                  params.set("target", normalized);
                  window.history.replaceState({}, "", `${window.location.pathname}?${params.toString()}`);
                }}
                targetOptions={targetOptions}
              >
                <div ref={stepContentRef} key={currentStep} className="animate-step-enter">
                  {steps[currentStep] && renderStep(steps[currentStep])}
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
        <div
          role="tabpanel"
          id="tabpanel-configuration"
          aria-labelledby="tab-configuration"
          className={cn(
            "mx-auto max-w-3xl px-3 py-4 sm:px-6 sm:py-8",
            view === "configuration" && "animate-panel-enter",
          )}
          hidden={view !== "configuration"}
        >
          {view === "configuration" && <ConfigurationSearchPanel target={target} onNavigate={openConfiguration} />}
        </div>
      </div>
    </div>
    </AppShell>
  );
}
