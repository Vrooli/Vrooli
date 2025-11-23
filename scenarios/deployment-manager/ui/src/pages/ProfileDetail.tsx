import { useParams, useNavigate, Link } from "react-router-dom";
import { useQuery, useMutation } from "@tanstack/react-query";
import { ArrowLeft, Rocket, Loader2, Focus } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { getProfile, deployProfile } from "../lib/api";

const TIER_NAMES: Record<number, string> = {
  1: "Local/Dev",
  2: "Desktop",
  3: "Mobile",
  4: "SaaS/Cloud",
  5: "Enterprise",
};

const TIER_KEYS: Record<number, string> = {
  1: "local",
  2: "desktop",
  3: "mobile",
  4: "saas",
  5: "enterprise",
};

export function ProfileDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { data: profile, isLoading, error } = useQuery({
    queryKey: ["profile", id],
    queryFn: () => getProfile(id!),
    enabled: !!id,
  });

  const deployMutation = useMutation({
    mutationFn: deployProfile,
    onSuccess: (data) => {
      navigate(`/deployments/${data.deployment_id}`);
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-3xl font-bold">Profile Not Found</h1>
        </div>
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-200">
          <p className="text-sm">Failed to load profile: {(error as Error).message}</p>
        </div>
      </div>
    );
  }

  if (!profile) {
    return null;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold">{profile.name}</h1>
            <p className="text-slate-400 mt-1">
              Version {profile.version}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Link to={`/analyze?scenario=${profile.scenario}&tier=${TIER_KEYS[profile.tiers?.[0] ?? 2]}`}>
            <Button variant="outline" className="gap-2">
              <Focus className="h-4 w-4" />
              Analyze (focus tier)
            </Button>
          </Link>
          <Button
            onClick={() => deployMutation.mutate(profile.id)}
            disabled={deployMutation.isPending}
            className="gap-2"
          >
            {deployMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Rocket className="h-4 w-4" />
            )}
            Deploy
          </Button>
        </div>
      </div>

      {deployMutation.isError && (
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-200">
          <p className="text-sm">
            Failed to deploy: {(deployMutation.error as Error).message}
          </p>
        </div>
      )}

      {/* Basic Info */}
      <Card>
        <CardHeader>
          <CardTitle>Profile Configuration</CardTitle>
          <CardDescription>
            Basic deployment profile information
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <div className="text-sm text-slate-400">Scenario</div>
            <div className="text-lg font-medium mt-1">{profile.scenario}</div>
          </div>

          <div>
            <div className="text-sm text-slate-400">Target Tiers</div>
            <div className="flex flex-wrap gap-2 mt-2">
              {profile.tiers.map((tier) => (
                <Badge key={tier} variant="secondary">
                  Tier {tier}: {TIER_NAMES[tier]}
                </Badge>
              ))}
            </div>
          </div>

          {profile.created_at && (
            <div>
              <div className="text-sm text-slate-400">Created</div>
              <div className="text-lg font-medium mt-1">
                {new Date(profile.created_at).toLocaleString()}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Swaps */}
      {profile.swaps && Object.keys(profile.swaps).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Dependency Swaps</CardTitle>
            <CardDescription>
              Modified dependencies for this deployment
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {Object.entries(profile.swaps).map(([from, to]) => (
                <div
                  key={from}
                  className="flex items-center gap-3 rounded-lg border border-white/10 bg-white/5 p-3"
                >
                  <Badge variant="destructive">{from}</Badge>
                  <span className="text-slate-400">→</span>
                  <Badge variant="success">{to}</Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Secrets */}
      {profile.secrets && Object.keys(profile.secrets).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Secret Configuration</CardTitle>
            <CardDescription>
              Environment variables and secrets for this deployment
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="text-sm text-slate-400">
              {Object.keys(profile.secrets).length} secret(s) configured
            </div>
          </CardContent>
        </Card>
      )}

      {/* Settings */}
      {profile.settings && Object.keys(profile.settings).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Additional Settings</CardTitle>
            <CardDescription>
              Platform-specific configuration options
            </CardDescription>
          </CardHeader>
          <CardContent>
            <pre className="rounded-lg bg-black/40 p-4 text-xs overflow-auto">
              {JSON.stringify(profile.settings, null, 2)}
            </pre>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
