import { useCallback } from "react";
import { ErrorBoundary } from "./ErrorBoundary";
import { ScenarioViewer, useScenarioViewerRoute } from "./scenarios/ScenarioViewer";

/** Scenario viewer wrapper component */
function ScenarioViewerWrapper() {
  const { scenarioName, path } = useScenarioViewerRoute();

  const handleBack = useCallback(() => {
    window.history.back();
  }, []);

  if (!scenarioName) {
    return null;
  }

  return (
    <ScenarioViewer
      scenarioName={scenarioName}
      path={path ?? undefined}
      onBack={handleBack}
    />
  );
}

/** App router - decides which view to render based on URL */
export function AppRouter({ children }: { children: React.ReactNode }) {
  const { isScenarioViewer } = useScenarioViewerRoute();

  if (isScenarioViewer) {
    return (
      <ErrorBoundary name="ScenarioViewer">
        <ScenarioViewerWrapper />
      </ErrorBoundary>
    );
  }

  return <>{children}</>;
}
