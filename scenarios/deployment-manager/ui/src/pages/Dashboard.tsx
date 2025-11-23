import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Package, Rocket, AlertCircle } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { listProfiles } from "../lib/api";
import { GuidedFlow } from "../components/GuidedFlow";

export function Dashboard() {
  const [guidedOpen, setGuidedOpen] = useState(false);
  const { data: profiles, isLoading, error } = useQuery({
    queryKey: ["profiles"],
    queryFn: listProfiles,
  });

  return (
    <div className="space-y-8">
      {/* Hero */}
      <div className="rounded-2xl border border-white/10 bg-gradient-to-r from-slate-950 via-slate-900 to-slate-950 p-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div className="space-y-2">
            <p className="text-xs uppercase tracking-wide text-cyan-300">Guided deployment</p>
            <h1 className="text-3xl font-bold">Make any scenario deployable</h1>
            <p className="text-slate-300 max-w-2xl">
              Follow the scenario-to-desktop style 3-step flow: pick a target, fix blockers (swaps + secrets),
              then export or hand off to scenario-to-*. Clear CTAs and empty states keep new users oriented.
            </p>
          </div>
          <div className="flex flex-wrap gap-3">
            <Button className="gap-2" onClick={() => setGuidedOpen(true)} data-testid="start-guided-flow">
              <Rocket className="h-4 w-4" />
              Start guided flow
            </Button>
            <Link to="/analyze">
              <Button variant="outline" className="gap-2">
                <Package className="h-4 w-4" />
                Open analyzer
              </Button>
            </Link>
            <Link to="/profiles/new">
              <Button variant="ghost" className="gap-2">
                <Plus className="h-4 w-4" />
                New profile
              </Button>
            </Link>
          </div>
        </div>
        <div className="mt-4 grid gap-3 md:grid-cols-3">
          {[
            { title: "Step 1: Choose scenario + tier", desc: "Auto-run analysis to surface fitness & blockers." },
            { title: "Step 2: Plan swaps & secrets", desc: "Use profiles to capture decisions and lift scores." },
            { title: "Step 3: Export or send to packagers", desc: "Bundle manifests or hand-off to scenario-to-*." },
          ].map((item) => (
            <div key={item.title} className="rounded-lg border border-white/5 bg-white/5 p-3">
              <p className="text-sm font-semibold">{item.title}</p>
              <p className="text-xs text-slate-400">{item.desc}</p>
            </div>
          ))}
        </div>
      </div>

      {/* First-run checklist */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>First-run checklist</CardTitle>
            <CardDescription>Get oriented in under a minute</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {[
              { title: "Run analysis", cta: "Open guided flow", action: () => setGuidedOpen(true) },
              { title: "Create profile", cta: "Create profile", link: "/profiles/new" },
              { title: "Review swaps/secrets", cta: "Open Profiles list", link: "/profiles" },
              { title: "Export or deploy", cta: "Go to deployments", link: "/deployments" },
            ].map((item) => (
              <div key={item.title} className="flex items-center justify-between rounded-lg border border-white/10 bg-white/5 p-3">
                <div className="space-y-1">
                  <p className="text-sm font-semibold">{item.title}</p>
                  <p className="text-xs text-slate-400">Guided CTA keeps new users on the happy path.</p>
                </div>
                {item.link ? (
                  <Link to={item.link}>
                    <Button size="sm" variant="outline">{item.cta}</Button>
                  </Link>
                ) : (
                  <Button size="sm" onClick={item.action}>{item.cta}</Button>
                )}
              </div>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Recently used</CardTitle>
            <CardDescription>Profiles and actions</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <p className="text-sm text-slate-300">
              Jump back into the profiles you were editing or start a fresh deployment path.
            </p>
            <div className="flex flex-wrap gap-2">
              <Link to="/profiles">
                <Button size="sm" variant="secondary" className="gap-2">
                  <Package className="h-4 w-4" />
                  Profiles
                </Button>
              </Link>
              <Link to="/deployments">
                <Button size="sm" variant="outline" className="gap-2">
                  <Rocket className="h-4 w-4" />
                  Deployments
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-slate-400">
              Total Profiles
            </CardTitle>
            <Package className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {isLoading ? "..." : profiles?.length || 0}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-slate-400">
              Active Deployments
            </CardTitle>
            <Rocket className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">0</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-slate-400">
              Failed Deployments
            </CardTitle>
            <AlertCircle className="h-4 w-4 text-slate-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">0</div>
          </CardContent>
        </Card>
      </div>

      {/* Recent Profiles */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Profiles</CardTitle>
          <CardDescription>
            {profiles?.length ? `${profiles.length} deployment profile${profiles.length !== 1 ? "s" : ""}` : "No deployment profiles yet"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {error && (
            <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-200">
              <p className="text-sm">Failed to load profiles: {(error as Error).message}</p>
            </div>
          )}

          {isLoading && (
            <div className="flex items-center justify-center py-8">
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-slate-700 border-t-cyan-400" />
            </div>
          )}

          {!isLoading && !error && profiles?.length === 0 && (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Package className="h-12 w-12 text-slate-600 mb-4" />
              <p className="text-slate-400 mb-4">No deployment profiles yet</p>
              <Link to="/profiles/new">
                <Button variant="outline" className="gap-2">
                  <Plus className="h-4 w-4" />
                  Create Your First Profile
                </Button>
              </Link>
            </div>
          )}

          {!isLoading && !error && profiles && profiles.length > 0 && (
            <div className="space-y-3">
              {profiles.map((profile) => (
                <Link
                  key={profile.id}
                  to={`/profiles/${profile.id}`}
                  className="block rounded-lg border border-white/10 bg-white/5 p-4 transition-colors hover:bg-white/10"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <h3 className="font-medium">{profile.name}</h3>
                      <p className="text-sm text-slate-400 mt-1">
                        Scenario: {profile.scenario}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      {profile.tiers?.map((tier) => (
                        <Badge key={tier} variant="secondary">
                          Tier {tier}
                        </Badge>
                      ))}
                      <Badge variant="outline">v{profile.version}</Badge>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <GuidedFlow open={guidedOpen} onClose={() => setGuidedOpen(false)} />
    </div>
  );
}
