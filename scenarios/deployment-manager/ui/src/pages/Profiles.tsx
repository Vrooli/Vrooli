import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Package, Loader2, HelpCircle } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { listProfiles } from "../lib/api";

const TIER_NAMES: Record<number, string> = {
  1: "Local/Dev",
  2: "Desktop",
  3: "Mobile",
  4: "SaaS/Cloud",
  5: "Enterprise",
};

export function Profiles() {
  const [showHelp, setShowHelp] = useState(false);
  const { data: profiles, isLoading, error } = useQuery({
    queryKey: ["profiles"],
    queryFn: listProfiles,
  });

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-3xl font-bold">Deployment Profiles</h1>
          <p className="text-slate-400 mt-1">
            Manage your deployment configurations
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setShowHelp((v) => !v)} className="gap-2">
            <HelpCircle className="h-4 w-4" />
            {showHelp ? "Hide help" : "How to use"}
          </Button>
          <Link to="/profiles/new">
            <Button className="gap-2">
              <Plus className="h-4 w-4" />
              New Profile
            </Button>
          </Link>
        </div>
      </div>

      {showHelp && (
        <Card>
          <CardContent className="pt-4 space-y-2 text-sm text-slate-300">
            <p>Create a profile to capture swaps/secrets for a scenario + tier. Profiles feed deployments.</p>
            <p>Tip: use the guided flow from the dashboard to prefill scenario/tier and jump straight here.</p>
          </CardContent>
        </Card>
      )}

      {/* Error */}
      {error && (
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-200">
          <p className="text-sm">Failed to load profiles: {(error as Error).message}</p>
        </div>
      )}

      {/* Loading */}
      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
        </div>
      )}

      {/* Empty State */}
      {!isLoading && !error && profiles?.length === 0 && (
        <Card>
          <CardContent className="pt-6">
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
          </CardContent>
        </Card>
      )}

      {/* Profiles Grid */}
      {!isLoading && !error && profiles && profiles.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {profiles.map((profile) => (
            <Link key={profile.id} to={`/profiles/${profile.id}`}>
              <Card className="h-full transition-colors hover:bg-white/10 cursor-pointer">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <CardTitle className="text-lg">{profile.name}</CardTitle>
                    <Badge variant="outline">v{profile.version}</Badge>
                  </div>
                  <CardDescription>{profile.scenario}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div>
                      <div className="text-xs text-slate-400 mb-1">Target Tiers</div>
                      <div className="flex flex-wrap gap-1">
                        {profile.tiers.map((tier) => (
                          <Badge key={tier} variant="secondary" className="text-xs">
                            {TIER_NAMES[tier]}
                          </Badge>
                        ))}
                      </div>
                    </div>
                    {profile.created_at && (
                      <div className="text-xs text-slate-500">
                        Created {new Date(profile.created_at).toLocaleDateString()}
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
