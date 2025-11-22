import { useQuery } from "@tanstack/react-query";
import { Activity, Play, Pause, StopCircle } from "lucide-react";
import { fetchCampaigns } from "../lib/api";
import { PageHeader } from "../components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Spinner } from "../components/ui/spinner";
import { Alert, AlertDescription } from "../components/ui/alert";
import { cn } from "../lib/utils";

export default function CampaignsView() {
  const { data: campaigns = [], isLoading, error } = useQuery({
    queryKey: ["campaigns"],
    queryFn: () => fetchCampaigns(),
  });

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
        description="Manage automated code tidiness campaigns across scenarios"
        actions={
          <Button variant="primary" size="sm">
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
          <CardContent className="text-center py-12">
            <Activity className="h-12 w-12 text-slate-600 mx-auto mb-4" />
            <p className="text-slate-400 mb-4">No campaigns yet</p>
            <Button variant="primary">Create Your First Campaign</Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {campaigns.map((campaign) => (
            <Card key={campaign.id}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <CardTitle>{campaign.scenario}</CardTitle>
                      {getStatusBadge(campaign.status)}
                    </div>
                    <CardDescription>
                      Session {campaign.current_session} of {campaign.max_sessions} ·
                      {campaign.files_visited}/{campaign.files_total} files visited
                    </CardDescription>
                  </div>
                  <div className="flex gap-2">
                    {campaign.status === "active" && (
                      <Button variant="outline" size="sm">
                        <Pause className="h-4 w-4 mr-2" />
                        Pause
                      </Button>
                    )}
                    {campaign.status === "paused" && (
                      <Button variant="outline" size="sm">
                        <Play className="h-4 w-4 mr-2" />
                        Resume
                      </Button>
                    )}
                    {(campaign.status === "active" || campaign.status === "paused") && (
                      <Button variant="destructive" size="sm">
                        <StopCircle className="h-4 w-4 mr-2" />
                        Stop
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
                      <span className="text-slate-400">Progress</span>
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
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
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
                    <div>
                      <span className="text-slate-500">Files per Session</span>
                      <p className="text-slate-200">{campaign.config.max_files_per_session}</p>
                    </div>
                    <div>
                      <span className="text-slate-500">Priority Rules</span>
                      <p className="text-slate-200">
                        {campaign.config.priority_rules?.length || 0} rules
                      </p>
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
