import { useState, useCallback } from "react";
import { Layout } from "./components/layout/Layout";
import { Dashboard } from "./components/Dashboard";
import { WizardContainer } from "./components/wizard";

type View = "dashboard" | "wizard";

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

  return (
    <Layout>
      {currentView === "dashboard" && (
        <Dashboard onStartNew={handleStartNew} onResume={handleResume} />
      )}
      {currentView === "wizard" && (
        <WizardContainer onBackToDashboard={handleBackToDashboard} />
      )}
    </Layout>
  );
}
