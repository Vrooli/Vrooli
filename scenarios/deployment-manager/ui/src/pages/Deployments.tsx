import { useState } from "react";
import { Link } from "react-router-dom";
import { Package, HelpCircle, Rocket, Plus } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { GuidedFlow } from "../components/GuidedFlow";

export function Deployments() {
  const [showHelp, setShowHelp] = useState(false);
  const [guidedOpen, setGuidedOpen] = useState(false);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between gap-3 flex-wrap">
        <h1 className="text-3xl font-bold">Deployments</h1>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setShowHelp((v) => !v)} className="gap-2">
            <HelpCircle className="h-4 w-4" />
            {showHelp ? "Hide help" : "How this works"}
          </Button>
          <Button size="sm" className="gap-2" onClick={() => setGuidedOpen(true)}>
            <Rocket className="h-4 w-4" />
            Start guided flow
          </Button>
        </div>
      </div>
      <p className="text-slate-400 mt-1">
        Monitor active and past deployments. Start with a profile, then export/hand-off to scenario-to-*.
      </p>

      {showHelp && (
        <Card>
          <CardContent className="pt-4 space-y-2 text-sm text-slate-300">
            <p>Deployments come from profiles. If you see nothing here, create a profile and run a deployment.</p>
            <p>Use the guided flow to pick a scenario + tier, plan swaps/secrets, then export or trigger a deploy.</p>
          </CardContent>
        </Card>
      )}

      {/* Empty State */}
      <Card>
        <CardHeader>
          <CardTitle>Active Deployments</CardTitle>
          <CardDescription>
            No active deployments
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <Package className="h-12 w-12 text-slate-600 mb-4" />
            <p className="text-slate-400 mb-4">No deployments yet</p>
            <p className="text-sm text-slate-500">
              Create a deployment profile and deploy it to see active deployments here
            </p>
            <div className="mt-4 flex flex-wrap items-center justify-center gap-2">
              <Button size="sm" onClick={() => setGuidedOpen(true)} className="gap-2">
                <Rocket className="h-4 w-4" />
                Start guided flow
              </Button>
              <Link to="/profiles/new">
                <Button size="sm" variant="outline" className="gap-2">
                  <Plus className="h-4 w-4" />
                  Create profile
                </Button>
              </Link>
              <Link to="/profiles">
                <Button size="sm" variant="ghost" className="gap-2">
                  View profiles
                </Button>
              </Link>
            </div>
          </div>
        </CardContent>
      </Card>

      <GuidedFlow open={guidedOpen} onClose={() => setGuidedOpen(false)} />
    </div>
  );
}
