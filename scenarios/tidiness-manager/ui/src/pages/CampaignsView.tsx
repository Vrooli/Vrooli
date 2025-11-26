import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Activity, Play, Pause, StopCircle, Info, Terminal, Copy } from "lucide-react";
import { fetchCampaigns } from "../lib/api";
import { useToast } from "../components/ui/toast";
import { PageHeader } from "../components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Spinner } from "../components/ui/spinner";
import { Alert, AlertDescription } from "../components/ui/alert";
import { cn } from "../lib/utils";

export default function CampaignsView() {
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const { data: campaigns = [], isLoading, error } = useQuery({
    queryKey: ["campaigns"],
    queryFn: () => fetchCampaigns(),
    refetchInterval: 10000, // Poll every 10 seconds when page is visible
  });

  const copyCLICommand = async (command: string) => {
    try {
      await navigator.clipboard.writeText(command);
      showToast("CLI command copied to clipboard", "success");
    } catch (err) {
      showToast("Failed to copy to clipboard", "error");
    }
  };

  const handleCampaignAction = (scenario: string, action: "pause" | "resume" | "stop") => {
    // For now, show CLI command since API integration is pending
    const command = `tidiness-manager campaigns ${action} ${scenario}`;
    showToast(
      `To ${action} this campaign, run:\n\n${command}\n\nDirect UI control requires API wiring (POST /api/v1/campaigns/{scenario}/${action})`,
      "info",
      8000
    );
    copyCLICommand(command);
  };

  const getStatusBadge = (status: string) => {
    const variants: Record<string, "active" | "success" | "warning" | "error" | "default"> = {
      active: "active",
      completed: "success",
      paused: "warning",
      error: "error",
      created: "default",
      terminated: "default",
    };
    return <Badge variant={variants[status] || "default"}>{status}</Badge>;
  };

  if (error) {
    return (
      <div>
        <PageHeader title="Campaigns" />
        <Alert variant="destructive">
          <AlertDescription>Failed to load campaigns: {(error as Error).message}</AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title="Tidiness Campaigns"
        description={
          <div className="space-y-1">
            <p>Manage automated code tidiness campaigns across scenarios</p>
            <p className="text-xs text-slate-500 font-mono">
              CLI: tidiness-manager campaigns list | tidiness-manager campaign create &lt;scenario&gt;
            </p>
          </div>
        }
        actions={
          <Button variant="primary" size="sm" title="Create a new automated campaign to systematically analyze files in a scenario">
            <Play className="h-4 w-4 mr-2" />
            New Campaign
          </Button>
        }
      />

      {isLoading ? (
        <div className="flex items-center justify-center p-12">
          <Spinner size="lg" />
        </div>
      ) : campaigns.length === 0 ? (
        <Card>
          <CardContent className="text-center py-12 space-y-6">
            <Activity className="h-12 w-12 text-slate-600 mx-auto" />
            <div className="space-y-2">
              <p className="text-lg font-semibold text-slate-200">No Active Campaigns</p>
              <p className="text-sm text-slate-400 max-w-xl mx-auto leading-relaxed">
                Campaigns systematically analyze files across scenarios using visited-tracker to ensure comprehensive coverage without redundant work. Ideal for ongoing code health monitoring and systematic refactoring efforts.
              </p>
            </div>

            {/* Agent Workflow */}
            <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-5 max-w-2xl mx-auto text-left">
              <div className="flex items-start gap-3 mb-4">
                <Terminal className="h-5 w-5 text-blue-400 mt-0.5 flex-shrink-0" />
                <div className="flex-1">
                  <p className="text-sm font-semibold text-blue-200 mb-2">Recommended: CLI Campaign Management</p>
                  <p className="text-xs text-slate-400 mb-3">
                    Agents can create, monitor, and control campaigns programmatically for automated tidiness workflows.
                  </p>
                </div>
              </div>

              <div className="space-y-3">
                <div>
                  <p className="text-xs text-slate-400 mb-1">1. Create campaign with custom parameters:</p>
                  <code className="text-xs bg-slate-950 px-3 py-2 rounded block text-slate-300 font-mono">
                    tidiness-manager campaigns start &lt;scenario-name&gt; --max-sessions 10 --files-per-session 5
                  </code>
                </div>
                <div>
                  <p className="text-xs text-slate-400 mb-1">2. Monitor progress:</p>
                  <code className="text-xs bg-slate-950 px-3 py-2 rounded block text-slate-300 font-mono">
                    tidiness-manager campaigns list
                  </code>
                </div>
                <div>
                  <p className="text-xs text-slate-400 mb-1">3. Control execution:</p>
                  <code className="text-xs bg-slate-950 px-3 py-2 rounded block text-slate-300 font-mono">
                    tidiness-manager campaigns pause &lt;scenario-name&gt;  # or resume, stop
                  </code>
                </div>
              </div>

              <div className="mt-4 pt-4 border-t border-blue-500/20">
                <p className="text-xs text-slate-500">
                  <span className="font-medium">Use Cases:</span> Systematic refactoring, batch issue discovery, post-merge health checks, pre-release quality gates
                </p>
              </div>
            </div>

            {/* Human Alternative */}
            <div className="space-y-3">
              <p className="text-sm text-slate-400">For Humans:</p>
              <Button variant="primary" disabled title="UI campaign creation coming soon. Use CLI for now.">
                Create Your First Campaign
              </Button>
              <p className="text-xs text-slate-500">Campaign creation UI coming soon. Please use CLI for now.</p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {campaigns.map((campaign) => (
            <Card key={campaign.id}>
              <CardHeader>
                <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-2 flex-wrap">
                      <CardTitle className="text-base sm:text-lg">{campaign.scenario}</CardTitle>
                      {getStatusBadge(campaign.status)}
                    </div>
                    <CardDescription className="flex flex-col sm:flex-row sm:items-center gap-1 text-xs sm:text-sm">
                      <span title="Each session analyzes a batch of files. Campaign completes when all files are visited or max sessions reached.">
                        Session {campaign.current_session} of {campaign.max_sessions}
                      </span>
                      <span className="hidden sm:inline"> ·</span>
                      <span title="Files visited this campaign. Campaign uses visited-tracker to prevent redundant analysis.">
                        {campaign.files_visited}/{campaign.files_total} files visited
                      </span>
                    </CardDescription>
                  </div>
                  <div className="flex gap-2 flex-wrap">
                    {campaign.status === "active" && (
                      <Button
                        variant="outline"
                        size="sm"
                        title="Temporarily pause this campaign. Can be resumed later."
                        className="flex-1 sm:flex-none"
                        onClick={() => handleCampaignAction(campaign.scenario, "pause")}
                      >
                        <Pause className="h-4 w-4 sm:mr-2" />
                        <span className="sm:hidden ml-2">Pause</span>
                        <span className="hidden sm:inline">Pause</span>
                      </Button>
                    )}
                    {campaign.status === "paused" && (
                      <Button
                        variant="outline"
                        size="sm"
                        title="Resume this campaign from where it left off"
                        className="flex-1 sm:flex-none"
                        onClick={() => handleCampaignAction(campaign.scenario, "resume")}
                      >
                        <Play className="h-4 w-4 sm:mr-2" />
                        <span className="sm:hidden ml-2">Resume</span>
                        <span className="hidden sm:inline">Resume</span>
                      </Button>
                    )}
                    {(campaign.status === "active" || campaign.status === "paused") && (
                      <Button
                        variant="destructive"
                        size="sm"
                        title="Permanently stop this campaign. Progress will be saved but campaign cannot be resumed."
                        className="flex-1 sm:flex-none"
                        onClick={() => handleCampaignAction(campaign.scenario, "stop")}
                      >
                        <StopCircle className="h-4 w-4 sm:mr-2" />
                        <span className="sm:hidden ml-2">Stop</span>
                        <span className="hidden sm:inline">Stop</span>
                      </Button>
                    )}
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {/* Progress Bar */}
                  <div>
                    <div className="flex items-center justify-between text-sm mb-2">
                      <div className="flex items-center gap-1">
                        <span className="text-slate-400">Progress</span>
                        <Info
                          className="h-3 w-3 text-slate-500 cursor-help"
                          title="File visit progress. Campaign completes when all files are analyzed or max sessions reached."
                        />
                      </div>
                      <span className="text-slate-300">
                        {Math.round((campaign.files_visited / campaign.files_total) * 100)}%
                      </span>
                    </div>
                    <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
                      <div
                        className={cn(
                          "h-full transition-all",
                          campaign.status === "completed" ? "bg-green-500" :
                          campaign.status === "active" ? "bg-blue-500" :
                          campaign.status === "error" ? "bg-red-500" : "bg-yellow-500"
                        )}
                        style={{ width: `${(campaign.files_visited / campaign.files_total) * 100}%` }}
                      />
                    </div>
                  </div>

                  {/* Details */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 text-xs sm:text-sm">
                    <div>
                      <span className="text-slate-500">Created</span>
                      <p className="text-slate-200">
                        {new Date(campaign.created_at).toLocaleDateString()}
                      </p>
                    </div>
                    <div>
                      <span className="text-slate-500">Updated</span>
                      <p className="text-slate-200">
                        {new Date(campaign.updated_at).toLocaleDateString()}
                      </p>
                    </div>
                    <div className="flex items-start gap-1">
                      <div className="flex-1">
                        <span className="text-slate-500">Files per Session</span>
                        <p className="text-slate-200">{campaign.config.max_files_per_session}</p>
                      </div>
                      <Info
                        className="h-3 w-3 text-slate-500 cursor-help mt-0.5"
                        title="Maximum number of files analyzed in each session. Smaller batches reduce cost but increase session count."
                      />
                    </div>
                    <div className="flex items-start gap-1">
                      <div className="flex-1">
                        <span className="text-slate-500">Priority Rules</span>
                        <p className="text-slate-200">
                          {campaign.config.priority_rules?.length || 0} rules
                        </p>
                      </div>
                      <Info
                        className="h-3 w-3 text-slate-500 cursor-help mt-0.5"
                        title="Custom rules for prioritizing which files to analyze first (e.g., prefer main.go over test fixtures)."
                      />
                    </div>
                  </div>

                  {campaign.error_reason && (
                    <Alert variant="destructive">
                      <AlertDescription>{campaign.error_reason}</AlertDescription>
                    </Alert>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
