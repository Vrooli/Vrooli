import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Rocket,
  Clock,
  ArrowRight,
  Cloud,
  Server,
  CheckCircle2,
  AlertCircle,
  History,
  Trash2,
} from "lucide-react";
import { fetchHealth } from "../lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "./ui/card";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { Alert } from "./ui/alert";
import { EmptyState } from "./ui/empty-state";

const STORAGE_KEY = "scenario-to-cloud:deployment";

type SavedDeployment = {
  manifestJson: string;
  currentStep: number;
  timestamp: number;
};

function loadSavedDeployment(): SavedDeployment | null {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      return JSON.parse(saved);
    }
  } catch {
    // Ignore parse errors
  }
  return null;
}

function clearSavedDeployment(): void {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Ignore storage errors
  }
}

interface DashboardProps {
  onStartNew: () => void;
  onResume: () => void;
}

export function Dashboard({ onStartNew, onResume }: DashboardProps) {
  const [savedDeployment, setSavedDeployment] = useState<SavedDeployment | null>(null);

  useEffect(() => {
    setSavedDeployment(loadSavedDeployment());
  }, []);

  const { data: health, isLoading: healthLoading, error: healthError } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
  });

  const handleClearSaved = () => {
    clearSavedDeployment();
    setSavedDeployment(null);
  };

  // Extract scenario info from saved deployment
  const savedScenarioInfo = savedDeployment
    ? (() => {
        try {
          const manifest = JSON.parse(savedDeployment.manifestJson);
          return {
            scenarioId: manifest?.scenario?.id ?? "Unknown",
            domain: manifest?.edge?.domain ?? "Not set",
            step: savedDeployment.currentStep,
            timestamp: new Date(savedDeployment.timestamp).toLocaleString(),
          };
        } catch {
          return null;
        }
      })()
    : null;

  const stepLabels = ["Manifest", "Validate", "Plan", "Build", "Preflight", "Deploy"];

  return (
    <div className="space-y-8">
      {/* Hero Section */}
      <div className="text-center py-8">
        <div className="mx-auto w-20 h-20 rounded-2xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center mb-6">
          <Cloud className="h-10 w-10 text-white" />
        </div>
        <h1 className="text-3xl font-bold text-white mb-3">
          Deploy Scenarios to the Cloud
        </h1>
        <p className="text-slate-400 max-w-lg mx-auto">
          Package and deploy Vrooli scenarios to VPS infrastructure with automated
          bundle building, preflight checks, and HTTPS configuration.
        </p>
      </div>

      {/* Quick Actions */}
      <div className="grid gap-4 md:grid-cols-2">
        {/* New Deployment Card */}
        <Card className="hover:border-white/20 transition-colors">
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-blue-500/20">
                <Rocket className="h-5 w-5 text-blue-400" />
              </div>
              <div>
                <CardTitle>New Deployment</CardTitle>
                <CardDescription>Start a fresh deployment from scratch</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <Button
              data-testid="dashboard-start-new-button"
              onClick={onStartNew}
              className="w-full sm:w-auto"
            >
              Start New
              <ArrowRight className="h-4 w-4 ml-2" />
            </Button>
          </CardContent>
        </Card>

        {/* Resume Card */}
        {savedDeployment && savedScenarioInfo ? (
          <Card className="border-amber-500/30 hover:border-amber-500/50 transition-colors">
            <CardHeader>
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-amber-500/20">
                  <History className="h-5 w-5 text-amber-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <CardTitle className="flex items-center gap-2">
                    Resume Deployment
                    <Badge variant="warning">In Progress</Badge>
                  </CardTitle>
                  <CardDescription>Continue where you left off</CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-3 mb-4">
                <div className="flex items-center gap-2 text-sm">
                  <Server className="h-4 w-4 text-slate-500" />
                  <span className="text-slate-400">Scenario:</span>
                  <span className="text-slate-200 font-mono text-xs">{savedScenarioInfo.scenarioId}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Clock className="h-4 w-4 text-slate-500" />
                  <span className="text-slate-400">Last saved:</span>
                  <span className="text-slate-200">{savedScenarioInfo.timestamp}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <CheckCircle2 className="h-4 w-4 text-slate-500" />
                  <span className="text-slate-400">Current step:</span>
                  <span className="text-slate-200">{stepLabels[savedScenarioInfo.step]}</span>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Button data-testid="dashboard-resume-button" onClick={onResume}>
                  Resume
                  <ArrowRight className="h-4 w-4 ml-2" />
                </Button>
                <Button
                  data-testid="dashboard-discard-button"
                  variant="outline"
                  onClick={handleClearSaved}
                >
                  <Trash2 className="h-4 w-4 mr-1" />
                  Discard
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <Card className="border-dashed">
            <CardContent className="py-8">
              <EmptyState
                icon={<History className="h-12 w-12" />}
                title="No Saved Progress"
                description="Start a new deployment to begin. Your progress will be saved automatically."
              />
            </CardContent>
          </Card>
        )}
      </div>

      {/* API Status Section */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">System Status</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {healthLoading && (
                <>
                  <div className="w-3 h-3 rounded-full bg-slate-500 animate-pulse" />
                  <span className="text-sm text-slate-400">Checking API status...</span>
                </>
              )}
              {!healthLoading && healthError && (
                <>
                  <div className="w-3 h-3 rounded-full bg-red-500" />
                  <div>
                    <span className="text-sm text-red-400">API Offline</span>
                    <p className="text-xs text-slate-500 mt-0.5">
                      Start the scenario with: vrooli scenario start scenario-to-cloud
                    </p>
                  </div>
                </>
              )}
              {!healthLoading && health && (
                <>
                  <div className="w-3 h-3 rounded-full bg-emerald-500" />
                  <div>
                    <span className="text-sm text-emerald-400">API Online</span>
                    <p className="text-xs text-slate-500 mt-0.5">
                      {health.service} - {new Date(health.timestamp).toLocaleTimeString()}
                    </p>
                  </div>
                </>
              )}
            </div>
            <Badge variant={health ? "success" : healthError ? "error" : "default"}>
              {health ? "Ready" : healthError ? "Offline" : "Checking..."}
            </Badge>
          </div>
        </CardContent>
      </Card>

      {/* API Offline Warning */}
      {healthError && (
        <Alert variant="warning" title="API Not Available">
          <p>
            The scenario-to-cloud API is not running. You can still configure your deployment
            manifest, but validation and bundle building will fail until the API is started.
          </p>
          <div className="mt-2 p-2 rounded bg-slate-800/50 font-mono text-xs">
            vrooli scenario start scenario-to-cloud
          </div>
        </Alert>
      )}

      {/* How It Works */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">How It Works</CardTitle>
          <CardDescription>
            Deploy scenarios to VPS in 6 simple steps
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {[
              { step: 1, title: "Configure Manifest", desc: "Set target server, ports, and options" },
              { step: 2, title: "Validate", desc: "Check manifest for errors" },
              { step: 3, title: "Review Plan", desc: "See what will be deployed" },
              { step: 4, title: "Build Bundle", desc: "Create deployment package" },
              { step: 5, title: "Preflight Checks", desc: "Verify server is ready" },
              { step: 6, title: "Deploy", desc: "Push to server and go live" },
            ].map((item) => (
              <div
                key={item.step}
                className="flex items-start gap-3 p-3 rounded-lg bg-slate-800/30"
              >
                <div className="w-8 h-8 rounded-full bg-slate-700 flex items-center justify-center flex-shrink-0">
                  <span className="text-sm font-medium text-slate-300">{item.step}</span>
                </div>
                <div>
                  <p className="text-sm font-medium text-slate-200">{item.title}</p>
                  <p className="text-xs text-slate-500">{item.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
