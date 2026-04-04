import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, RefreshCw, CheckCircle2, XCircle, Shield } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { listProfiles, listApprovals, decideApproval } from "../lib/api";
import type { DeploymentApproval, DeploymentProfile } from "../lib/api";

type StatusFilter = "all" | "pending" | "approved" | "rejected" | "stale";

function statusVariant(status: string): "success" | "warning" | "destructive" | "secondary" {
  switch (status) {
    case "approved": return "success";
    case "pending": return "warning";
    case "rejected": return "destructive";
    default: return "secondary";
  }
}

export function Approvals() {
  const queryClient = useQueryClient();
  const [selectedProfile, setSelectedProfile] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [detailApproval, setDetailApproval] = useState<DeploymentApproval | null>(null);
  const [decideNotes, setDecideNotes] = useState("");
  const [decideReviewer, setDecideReviewer] = useState("");

  const { data: profiles } = useQuery({
    queryKey: ["profiles"],
    queryFn: listProfiles,
  });

  const { data: approvals, isLoading, error, refetch } = useQuery({
    queryKey: ["approvals", selectedProfile],
    queryFn: () => listApprovals(selectedProfile),
    enabled: selectedProfile.length > 0,
  });

  const decideMutation = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: "approved" | "rejected" }) =>
      decideApproval(id, { decision, reviewer: decideReviewer, notes: decideNotes }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["approvals", selectedProfile] });
      setDetailApproval(null);
      setDecideNotes("");
    },
  });

  const filtered = approvals?.filter(
    (a) => statusFilter === "all" || a.status === statusFilter
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Approvals</h1>
          <p className="text-slate-400 mt-1">
            Manage deployment approval gates across profiles
          </p>
        </div>
        {selectedProfile && (
          <Button variant="outline" className="gap-2" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
        )}
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-48">
              <label className="text-sm text-slate-400 block mb-1">Profile</label>
              <select
                data-testid="profile-select"
                value={selectedProfile}
                onChange={(e) => setSelectedProfile(e.target.value)}
                className="w-full rounded-md border border-white/10 bg-white/5 px-3 py-2 text-sm text-slate-50 focus:outline-none focus:ring-1 focus:ring-cyan-500"
              >
                <option value="">Select a profile...</option>
                {profiles?.map((p: DeploymentProfile) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-sm text-slate-400 block mb-1">Status</label>
              <div className="flex gap-1">
                {(["all", "pending", "approved", "rejected", "stale"] as StatusFilter[]).map((s) => (
                  <Button
                    key={s}
                    variant={statusFilter === s ? "default" : "outline"}
                    size="sm"
                    onClick={() => setStatusFilter(s)}
                    className="capitalize"
                  >
                    {s}
                  </Button>
                ))}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Content */}
      {!selectedProfile && (
        <div className="flex flex-col items-center justify-center py-12 text-slate-400">
          <Shield className="h-12 w-12 mb-4 opacity-50" />
          <p className="text-lg font-medium">Select a profile to view approvals</p>
        </div>
      )}

      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />
        </div>
      )}

      {error && (
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-200">
          <p className="text-sm">Failed to load approvals: {(error as Error).message}</p>
        </div>
      )}

      {filtered && filtered.length === 0 && (
        <div className="flex flex-col items-center justify-center py-12 text-slate-400">
          <CheckCircle2 className="h-12 w-12 mb-4 opacity-50" />
          <p className="text-lg font-medium">No approvals found</p>
          <p className="text-sm mt-1">
            {statusFilter !== "all" ? "Try a different status filter." : "Create approvals via the CLI to get started."}
          </p>
        </div>
      )}

      {filtered && filtered.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Approval List</CardTitle>
            <CardDescription>
              {filtered.length} approval{filtered.length !== 1 ? "s" : ""}
              {statusFilter !== "all" ? ` (${statusFilter})` : ""}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {filtered.map((a) => (
                <div
                  key={a.id}
                  className="flex items-center justify-between rounded-lg border border-white/10 bg-white/5 px-4 py-3 hover:bg-white/10 cursor-pointer transition-colors"
                  onClick={() => setDetailApproval(a)}
                >
                  <div className="flex items-center gap-4">
                    <div>
                      <span className="text-sm font-medium capitalize">{a.platform}</span>
                      <span className="text-xs text-slate-500 ml-2">
                        {a.git_commit_hash.substring(0, 12)}
                      </span>
                    </div>
                    <Badge variant={statusVariant(a.status)}>{a.status}</Badge>
                  </div>
                  <div className="flex items-center gap-2">
                    {a.status === "pending" && (
                      <>
                        <Button
                          size="sm"
                          variant="outline"
                          className="text-green-400 border-green-500/30 hover:bg-green-500/10"
                          onClick={(e) => {
                            e.stopPropagation();
                            if (!decideReviewer) {
                              setDetailApproval(a);
                              return;
                            }
                            decideMutation.mutate({ id: a.id, decision: "approved" });
                          }}
                        >
                          <CheckCircle2 className="h-3 w-3 mr-1" />
                          Approve
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          className="text-red-400 border-red-500/30 hover:bg-red-500/10"
                          onClick={(e) => {
                            e.stopPropagation();
                            if (!decideReviewer) {
                              setDetailApproval(a);
                              return;
                            }
                            decideMutation.mutate({ id: a.id, decision: "rejected" });
                          }}
                        >
                          <XCircle className="h-3 w-3 mr-1" />
                          Reject
                        </Button>
                      </>
                    )}
                    <span className="text-xs text-slate-500">
                      {new Date(a.updated_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Detail Panel */}
      {detailApproval && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span>Approval Detail</span>
              <Button variant="outline" size="sm" onClick={() => setDetailApproval(null)}>
                Close
              </Button>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-slate-400">ID</span>
                <p className="font-mono">{detailApproval.id}</p>
              </div>
              <div>
                <span className="text-slate-400">Platform</span>
                <p className="capitalize">{detailApproval.platform}</p>
              </div>
              <div>
                <span className="text-slate-400">Commit</span>
                <p className="font-mono">{detailApproval.git_commit_hash}</p>
              </div>
              <div>
                <span className="text-slate-400">Status</span>
                <p><Badge variant={statusVariant(detailApproval.status)}>{detailApproval.status}</Badge></p>
              </div>
              {detailApproval.approved_by && (
                <div>
                  <span className="text-slate-400">Reviewer</span>
                  <p>{detailApproval.approved_by}</p>
                </div>
              )}
              {detailApproval.notes && (
                <div>
                  <span className="text-slate-400">Notes</span>
                  <p>{detailApproval.notes}</p>
                </div>
              )}
              {detailApproval.validation_id && (
                <div>
                  <span className="text-slate-400">Validation</span>
                  <p className="font-mono text-cyan-400">{detailApproval.validation_id}</p>
                </div>
              )}
              <div>
                <span className="text-slate-400">Created</span>
                <p>{new Date(detailApproval.created_at).toLocaleString()}</p>
              </div>
            </div>

            {detailApproval.status === "pending" && (
              <div className="space-y-3 pt-4 border-t border-white/10">
                <h4 className="text-sm font-medium">Make Decision</h4>
                <div className="flex gap-4">
                  <div className="flex-1">
                    <label className="text-xs text-slate-400 block mb-1">Reviewer *</label>
                    <input
                      type="text"
                      value={decideReviewer}
                      onChange={(e) => setDecideReviewer(e.target.value)}
                      placeholder="Your name"
                      className="w-full rounded-md border border-white/10 bg-white/5 px-3 py-2 text-sm text-slate-50 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                    />
                  </div>
                  <div className="flex-1">
                    <label className="text-xs text-slate-400 block mb-1">Notes</label>
                    <input
                      type="text"
                      value={decideNotes}
                      onChange={(e) => setDecideNotes(e.target.value)}
                      placeholder="Optional notes"
                      className="w-full rounded-md border border-white/10 bg-white/5 px-3 py-2 text-sm text-slate-50 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                    />
                  </div>
                </div>
                <div className="flex gap-2">
                  <Button
                    onClick={() => decideMutation.mutate({ id: detailApproval.id, decision: "approved" })}
                    disabled={!decideReviewer || decideMutation.isPending}
                    className="gap-2 bg-green-600 hover:bg-green-700"
                  >
                    <CheckCircle2 className="h-4 w-4" />
                    Approve
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={() => decideMutation.mutate({ id: detailApproval.id, decision: "rejected" })}
                    disabled={!decideReviewer || decideMutation.isPending}
                    className="gap-2"
                  >
                    <XCircle className="h-4 w-4" />
                    Reject
                  </Button>
                </div>
                {decideMutation.isError && (
                  <p className="text-sm text-red-300">
                    {(decideMutation.error as Error).message}
                  </p>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
