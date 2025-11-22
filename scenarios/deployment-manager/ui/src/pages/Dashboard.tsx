import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Package, Rocket, AlertCircle } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { listProfiles } from "../lib/api";

export function Dashboard() {
  const { data: profiles, isLoading, error } = useQuery({
    queryKey: ["profiles"],
    queryFn: listProfiles,
  });

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="text-slate-400 mt-1">
            Manage deployment profiles and monitor your deployments
          </p>
        </div>
        <Link to="/profiles/new">
          <Button className="gap-2">
            <Plus className="h-4 w-4" />
            New Profile
          </Button>
        </Link>
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
            {profiles?.length ? `${profiles.length} deployment profile${profiles.length !== 1 ? 's' : ''}` : 'No deployment profiles yet'}
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
    </div>
  );
}
