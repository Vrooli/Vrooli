import { useCallback } from "react";
import { Layout } from "./components/layout/Layout";
import { Dashboard } from "./components/Dashboard";
import { WizardContainer } from "./components/wizard";
import { DeploymentsPage } from "./components/deployments/DeploymentsPage";
import { DocsPage } from "./components/docs";
import { useHashRouter, View } from "./hooks/useHashRouter";

export default function App() {
  const { view, docPath, navigate, navigateToDoc } = useHashRouter();

  const handleStartNew = useCallback(() => {
    // Clear any saved progress before starting new
    try {
      localStorage.removeItem("scenario-to-cloud:deployment");
    } catch {
      // Ignore storage errors
    }
    navigate("wizard");
  }, [navigate]);

  const handleResume = useCallback(() => {
    navigate("wizard");
  }, [navigate]);

  const handleBackToDashboard = useCallback(() => {
    navigate("dashboard");
  }, [navigate]);

  const handleNavigate = useCallback((newView: View) => {
    navigate(newView);
  }, [navigate]);

  const handleViewDeployments = useCallback(() => {
    navigate("deployments");
  }, [navigate]);

  return (
    <Layout currentView={view} onNavigate={handleNavigate}>
      {view === "dashboard" && (
        <Dashboard onStartNew={handleStartNew} onResume={handleResume} />
      )}
      {view === "wizard" && (
        <WizardContainer
          onBackToDashboard={handleBackToDashboard}
          onViewDeployments={handleViewDeployments}
        />
      )}
      {view === "deployments" && (
        <DeploymentsPage onBack={handleBackToDashboard} />
      )}
      {view === "docs" && (
        <DocsPage
          onBack={handleBackToDashboard}
          initialDocPath={docPath}
          onDocPathChange={navigateToDoc}
        />
      )}
    </Layout>
  );
}
