import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Rocket, Loader2, Focus, Package, AlertCircle, Shield, CheckCircle2, XCircle } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { getProfile, deployProfile, checkReleaseGate, getRequiredPlatforms, setRequiredPlatforms } from "../lib/api";
import type { ReleaseGateStatus } from "../lib/api";

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

const ALL_PLATFORMS = ["windows", "macos", "linux"];

function statusVariant(status: string): "success" | "warning" | "destructive" | "secondary" {
  switch (status) {
    case "approved": return "success";
    case "pending": return "warning";
    case "rejected": return "destructive";
    default: return "secondary";
  }
}

export function ProfileDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [gateCommit, setGateCommit] = useState("");
  const [platformEditing, setPlatformEditing] = useState(false);
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>([]);

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

  // Release gate query — uses gateCommit if set, otherwise disabled until user enters one
  const { data: gateStatus, isLoading: gateLoading, error: gateError } = useQuery({
    queryKey: ["release-gate", id, gateCommit],
    queryFn: () => checkReleaseGate(id!, gateCommit),
    enabled: !!id && gateCommit.length > 0,
  });

  // Required platforms
  const { data: requiredPlatforms } = useQuery({
    queryKey: ["required-platforms", id],
    queryFn: () => getRequiredPlatforms(id!),
    enabled: !!id,
  });

  const savePlatformsMutation = useMutation({
    mutationFn: () => setRequiredPlatforms(id!, selectedPlatforms),
    onSuccess: () => {
      setPlatformEditing(false);
      queryClient.invalidateQueries({ queryKey: ["required-platforms", id] });
      queryClient.invalidateQueries({ queryKey: ["release-gate", id] });
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

  // Initialize selectedPlatforms from server data on first load
  if (requiredPlatforms && selectedPlatforms.length === 0 && requiredPlatforms.platforms.length > 0 && !platformEditing) {
    setSelectedPlatforms(requiredPlatforms.platforms);
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
          <Link to="/deployments">
            <Button variant="secondary" className="gap-2">
              <Package className="h-4 w-4" />
              Open deployments
            </Button>
          </Link>
          <Button
            onClick={() => deployMutation.mutate(profile.id)}
            disabled={deployMutation.isPending}
            className="gap-2"
            title="Deploy hand-off is stubbed here; use the packager CLI after exporting a bundle."
          >
            {deployMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Rocket className="h-4 w-4" />
            )}
            Deploy (stub)
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

      <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 p-4 text-sm text-amber-100 flex gap-3 items-start">
        <AlertCircle className="h-4 w-4 mt-0.5" />
        <div className="space-y-1">
          <p className="font-semibold">How to finish deployment</p>
          <p className="text-amber-50/90">
            Export a bundle for your target tier and run the matching scenario-to-* packager (desktop/mobile/saas). The Deploy button here is a stub and does not call a packager yet.
          </p>
          <p className="text-amber-50/80">
            After running the packager, come back to Deployments to track status and upload telemetry.
          </p>
        </div>
      </div>

      {/* Release Gate Status */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Release Gate
          </CardTitle>
          <CardDescription>
            Check approval status for a specific commit
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <input
              type="text"
              placeholder="Enter git commit hash..."
              value={gateCommit}
              onChange={(e) => setGateCommit(e.target.value)}
              className="flex-1 rounded-md border border-white/10 bg-white/5 px-3 py-2 text-sm text-slate-50 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            />
          </div>

          {gateLoading && (
            <div className="flex items-center gap-2 text-slate-400">
              <Loader2 className="h-4 w-4 animate-spin" />
              Checking gate status...
            </div>
          )}

          {gateError && (
            <div className="text-sm text-red-300">
              Failed to check gate: {(gateError as Error).message}
            </div>
          )}

          {gateStatus && <GateStatusDisplay gate={gateStatus} />}

          {!gateCommit && (
            <p className="text-sm text-slate-500">
              Enter a commit hash to check release gate status.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Required Platforms */}
      <Card>
        <CardHeader>
          <CardTitle>Required Platforms</CardTitle>
          <CardDescription>
            Platforms that must be approved before deployment can proceed
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-4">
            {ALL_PLATFORMS.map((plat) => (
              <label key={plat} className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={selectedPlatforms.includes(plat)}
                  onChange={(e) => {
                    setPlatformEditing(true);
                    setSelectedPlatforms(
                      e.target.checked
                        ? [...selectedPlatforms, plat]
                        : selectedPlatforms.filter((p) => p !== plat)
                    );
                  }}
                  className="rounded border-white/20 bg-white/5 text-cyan-500 focus:ring-cyan-500"
                />
                <span className="text-sm capitalize">{plat}</span>
              </label>
            ))}
          </div>
          {platformEditing && (
            <Button
              size="sm"
              onClick={() => savePlatformsMutation.mutate()}
              disabled={savePlatformsMutation.isPending}
            >
              {savePlatformsMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : null}
              Save Platforms
            </Button>
          )}
          {savePlatformsMutation.isSuccess && !platformEditing && (
            <p className="text-sm text-green-400">Platforms saved.</p>
          )}
        </CardContent>
      </Card>

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

function GateStatusDisplay({ gate }: { gate: ReleaseGateStatus }) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        {gate.ready ? (
          <>
            <CheckCircle2 className="h-5 w-5 text-green-400" />
            <Badge variant="success">Ready</Badge>
          </>
        ) : (
          <>
            <XCircle className="h-5 w-5 text-red-400" />
            <Badge variant="destructive">Blocked</Badge>
          </>
        )}
        <span className="text-sm text-slate-400 ml-2">
          Commit: {gate.git_commit_hash.substring(0, 12)}...
        </span>
      </div>

      {gate.platforms.length > 0 && (
        <div className="space-y-2">
          {gate.platforms.map((p) => (
            <div
              key={p.platform}
              className="flex items-center justify-between rounded-lg border border-white/10 bg-white/5 px-3 py-2"
            >
              <span className="text-sm font-medium capitalize">{p.platform}</span>
              <div className="flex items-center gap-2">
                {p.required && (
                  <span className="text-xs text-slate-500">required</span>
                )}
                <Badge variant={statusVariant(p.status)}>{p.status}</Badge>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
