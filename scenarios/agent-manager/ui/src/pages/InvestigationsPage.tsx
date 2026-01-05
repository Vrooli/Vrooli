import { useCallback, useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  AlertCircle,
  ChevronRight,
  Clock,
  Eye,
  RefreshCw,
  Search,
  Trash2,
  XCircle,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ScrollArea } from "../components/ui/scroll-area";
import { InvestigationReport } from "../components/InvestigationReport";
import { cn } from "../lib/utils";
import type { ApplyFixesRequest, Investigation } from "../types";
import {
  useInvestigations,
  useInvestigation,
  useDeleteInvestigation,
  useStopInvestigation,
  useApplyFixes,
} from "../hooks/useInvestigations";

interface InvestigationsPageProps {
  onViewRun?: (runId: string) => void;
}

export function InvestigationsPage({ onViewRun }: InvestigationsPageProps) {
  const { investigationId } = useParams<{ investigationId?: string }>();
  const navigate = useNavigate();
  const { data: investigations, loading, error, refetch } = useInvestigations();
  const { data: selectedInvestigation, refetch: refetchSelected } = useInvestigation(investigationId ?? null);
  const { deleteInvestigation, loading: deleteLoading } = useDeleteInvestigation();
  const { stop: stopInvestigation, loading: stopLoading } = useStopInvestigation();
  const { applyFixes, loading: applyLoading } = useApplyFixes();

  const handleSelect = (id: string) => {
    navigate(`/investigations/${id}`);
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this investigation? This cannot be undone.")) return;
    await deleteInvestigation(id);
    if (investigationId === id) {
      navigate("/investigations");
    }
    refetch();
  };

  const handleStop = async (id: string) => {
    if (!confirm("Stop this investigation?")) return;
    await stopInvestigation(id);
    refetch();
    refetchSelected();
  };

  const handleApplyFixes = async (request: ApplyFixesRequest) => {
    if (!selectedInvestigation) return;
    await applyFixes(selectedInvestigation.id, request);
    refetch();
    refetchSelected();
  };

  const handleViewRun = (runId: string) => {
    if (onViewRun) {
      onViewRun(runId);
    } else {
      navigate(`/runs/${runId}`);
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return "N/A";
    return new Date(dateStr).toLocaleString();
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "completed":
        return <Search className="h-5 w-5 text-success" />;
      case "failed":
        return <XCircle className="h-5 w-5 text-destructive" />;
      case "running":
      case "pending":
        return <Search className="h-5 w-5 text-primary animate-pulse" />;
      case "cancelled":
        return <XCircle className="h-5 w-5 text-muted-foreground" />;
      default:
        return <Clock className="h-5 w-5 text-muted-foreground" />;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Investigations</h2>
          <p className="text-sm text-muted-foreground">
            View analysis reports and apply recommendations
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={refetch} className="gap-2">
          <RefreshCw className="h-4 w-4" />
          Refresh
        </Button>
      </div>

      {error && (
        <Card className="border-destructive/50 bg-destructive/10">
          <CardContent className="flex items-center gap-3 py-4">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <p className="text-sm text-destructive">{error}</p>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Investigations List */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Search className="h-5 w-5" />
              All Investigations ({investigations?.length ?? 0})
            </CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="py-8 text-center text-muted-foreground">
                Loading investigations...
              </div>
            ) : !investigations || investigations.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <Search className="h-12 w-12 mb-4 opacity-50" />
                <p className="text-sm">No investigations yet</p>
                <p className="text-xs">Select runs and click "Investigate" to start</p>
              </div>
            ) : (
              <ScrollArea className="h-[500px]">
                <div className="space-y-2">
                  {investigations.map((inv) => (
                    <div
                      key={inv.id}
                      className={cn(
                        "flex items-center justify-between rounded-lg border p-3 cursor-pointer transition-colors",
                        investigationId === inv.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:bg-muted/50"
                      )}
                      onClick={() => handleSelect(inv.id)}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          handleSelect(inv.id);
                        }
                      }}
                    >
                      <div className="flex items-center gap-3">
                        {getStatusIcon(inv.status)}
                        <div>
                          <p className="font-medium text-sm">
                            {inv.run_ids.length} Run{inv.run_ids.length !== 1 ? "s" : ""} Analyzed
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {formatDate(inv.created_at)}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge variant={getStatusVariant(inv.status)}>
                          {inv.status}
                        </Badge>
                        {inv.status === "running" || inv.status === "pending" ? (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleStop(inv.id);
                            }}
                            disabled={stopLoading}
                          >
                            <XCircle className="h-3 w-3" />
                          </Button>
                        ) : (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6 text-destructive hover:text-destructive"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDelete(inv.id);
                            }}
                            disabled={deleteLoading}
                          >
                            <Trash2 className="h-3 w-3" />
                          </Button>
                        )}
                        <ChevronRight className="h-4 w-4 text-muted-foreground" />
                      </div>
                    </div>
                  ))}
                </div>
              </ScrollArea>
            )}
          </CardContent>
        </Card>

        {/* Investigation Details */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Eye className="h-5 w-5" />
              Investigation Details
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!selectedInvestigation ? (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <Eye className="h-12 w-12 mb-4 opacity-50" />
                <p className="text-sm">Select an investigation to view details</p>
              </div>
            ) : selectedInvestigation.status === "running" || selectedInvestigation.status === "pending" ? (
              <div className="flex flex-col items-center justify-center py-12">
                <Search className="h-12 w-12 mb-4 text-primary animate-pulse" />
                <p className="text-sm font-medium">Investigation in progress</p>
                <p className="text-xs text-muted-foreground mt-1">
                  {selectedInvestigation.progress}% complete
                </p>
                <div className="w-48 bg-muted rounded-full h-2 mt-4">
                  <div
                    className="bg-primary h-2 rounded-full transition-all"
                    style={{ width: `${selectedInvestigation.progress}%` }}
                  />
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-4"
                  onClick={() => handleStop(selectedInvestigation.id)}
                  disabled={stopLoading}
                >
                  Stop Investigation
                </Button>
              </div>
            ) : selectedInvestigation.status === "failed" ? (
              <div className="flex flex-col items-center justify-center py-12">
                <XCircle className="h-12 w-12 mb-4 text-destructive" />
                <p className="text-sm font-medium">Investigation failed</p>
                {selectedInvestigation.error_message && (
                  <p className="text-xs text-muted-foreground mt-2 text-center max-w-md">
                    {selectedInvestigation.error_message}
                  </p>
                )}
              </div>
            ) : (
              <InvestigationReport
                investigation={selectedInvestigation}
                onApplyFixes={handleApplyFixes}
                onViewRun={handleViewRun}
                applyingFixes={applyLoading}
              />
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function getStatusVariant(status: string): "default" | "secondary" | "success" | "destructive" | "outline" {
  switch (status) {
    case "completed":
      return "success";
    case "failed":
      return "destructive";
    case "running":
    case "pending":
      return "secondary";
    case "cancelled":
      return "outline";
    default:
      return "default";
  }
}
