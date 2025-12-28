import { useState, useCallback } from "react";
import { Layout, View } from "./components/layout/Layout";
import { Dashboard } from "./components/Dashboard";
import { WizardContainer } from "./components/wizard";
import { DeploymentsPage } from "./components/deployments/DeploymentsPage";
import { DocsPage } from "./components/docs";

export default function App() {
  const [currentView, setCurrentView] = useState<View>("dashboard");

  const handleStartNew = useCallback(() => {
    // Clear any saved progress before starting new
    try {
      localStorage.removeItem("scenario-to-cloud:deployment");
    } catch {
      // Ignore storage errors
    }
    setCurrentView("wizard");
  }, []);

  const handleResume = useCallback(() => {
    setCurrentView("wizard");
  }, []);

  const handleBackToDashboard = useCallback(() => {
    setCurrentView("dashboard");
  }, []);

  const handleNavigate = useCallback((view: View) => {
    setCurrentView(view);
  }, []);

  const handleViewDeployments = useCallback(() => {
    setCurrentView("deployments");
  }, []);

  return (
    <Layout currentView={currentView} onNavigate={handleNavigate}>
      {currentView === "dashboard" && (
        <Dashboard onStartNew={handleStartNew} onResume={handleResume} />
      )}
      {currentView === "wizard" && (
        <WizardContainer
          onBackToDashboard={handleBackToDashboard}
          onViewDeployments={handleViewDeployments}
        />
      )}
      {currentView === "deployments" && (
        <DeploymentsPage onBack={handleBackToDashboard} />
      )}
      {currentView === "docs" && (
        <DocsPage onBack={handleBackToDashboard} />
      )}
    </Layout>
  );
}
