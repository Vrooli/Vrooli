import { useMemo, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  AlertCircle,
  Clock,
  Eye,
  RefreshCw,
  Search,
  Trash2,
  XCircle,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";
import { InvestigationReport } from "../components/InvestigationReport";
import type { ApplyFixesRequest } from "../types";
import {
  useInvestigations,
  useInvestigation,
  useDeleteInvestigation,
  useStopInvestigation,
  useApplyFixes,
} from "../hooks/useInvestigations";

import { MasterDetailLayout, ListPanel, DetailPanel } from "../components/patterns/MasterDetail";
import { SearchToolbar, type FilterConfig, type SortOption } from "../components/patterns/SearchToolbar";
import { ListItem, ListItemTitle, ListItemSubtitle } from "../components/patterns/ListItem";

interface InvestigationsPageProps {
  onViewRun?: (runId: string) => void;
}

const STATUS_FILTER_OPTIONS = [
  { value: "pending", label: "Pending" },
  { value: "running", label: "Running" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
  { value: "cancelled", label: "Cancelled" },
];

const SORT_OPTIONS: SortOption[] = [
  { value: "newest", label: "Newest First" },
  { value: "oldest", label: "Oldest First" },
];

export function InvestigationsPage({ onViewRun }: InvestigationsPageProps) {
  const { investigationId } = useParams<{ investigationId?: string }>();
  const navigate = useNavigate();
  const { data: investigations, loading, error, refetch } = useInvestigations();
  const { data: selectedInvestigation, refetch: refetchSelected } = useInvestigation(investigationId ?? null);
  const { deleteInvestigation, loading: deleteLoading } = useDeleteInvestigation();
  const { stop: stopInvestigation, loading: stopLoading } = useStopInvestigation();
  const { applyFixes, loading: applyLoading } = useApplyFixes();

  // Filter/sort/search state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<string>("newest");

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
        return <Search className="h-5 w-5 text-success flex-shrink-0" />;
      case "failed":
        return <XCircle className="h-5 w-5 text-destructive flex-shrink-0" />;
      case "running":
      case "pending":
        return <Search className="h-5 w-5 text-primary animate-pulse flex-shrink-0" />;
      case "cancelled":
        return <XCircle className="h-5 w-5 text-muted-foreground flex-shrink-0" />;
      default:
        return <Clock className="h-5 w-5 text-muted-foreground flex-shrink-0" />;
    }
  };

  const filteredAndSortedInvestigations = useMemo(() => {
    if (!investigations) return [];

    let result = [...investigations];

    if (statusFilter !== "all") {
      result = result.filter((inv) => inv.status === statusFilter);
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter((inv) =>
        inv.id.toLowerCase().includes(query) ||
        inv.run_ids.some((id) => id.toLowerCase().includes(query))
      );
    }

    result.sort((a, b) => {
      const aTime = a.created_at ? new Date(a.created_at).getTime() : 0;
      const bTime = b.created_at ? new Date(b.created_at).getTime() : 0;
      return sortBy === "newest" ? bTime - aTime : aTime - bTime;
    });

    return result;
  }, [investigations, statusFilter, searchQuery, sortBy]);

  const filters: FilterConfig[] = [
    {
      id: "status",
      label: "Filter by status",
      value: statusFilter,
      options: STATUS_FILTER_OPTIONS,
      onChange: setStatusFilter,
      allLabel: "All Status",
    },
  ];

  const listPanel = (
    <ListPanel
      title="All Investigations"
      count={filteredAndSortedInvestigations.length}
      loading={loading}
      headerActions={
        <Button variant="outline" size="sm" onClick={refetch}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      }
      toolbar={
        <SearchToolbar
          searchValue={searchQuery}
          onSearchChange={setSearchQuery}
          searchPlaceholder="Search investigations..."
          filters={filters}
          sortOptions={SORT_OPTIONS}
          currentSort={sortBy}
          onSortChange={setSortBy}
        />
      }
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Search className="h-12 w-12 mb-3 opacity-50" />
          <p className="font-medium">
            {!investigations || investigations.length === 0 ? "No Investigations Yet" : "No Matching Investigations"}
          </p>
          <p className="text-sm text-center mt-1">
            {!investigations || investigations.length === 0
              ? 'Select runs and click "Investigate" to start'
              : "Try adjusting your filters"}
          </p>
        </div>
      }
    >
      {filteredAndSortedInvestigations.map((inv) => (
        <ListItem
          key={inv.id}
          selected={investigationId === inv.id}
          onClick={() => handleSelect(inv.id)}
          icon={getStatusIcon(inv.status)}
          actions={
            <div className="flex items-center gap-1">
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
                  aria-label="Stop investigation"
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
                  aria-label="Delete investigation"
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              )}
            </div>
          }
        >
          <ListItemTitle>
            {inv.run_ids.length} Run{inv.run_ids.length !== 1 ? "s" : ""} Analyzed
          </ListItemTitle>
          <ListItemSubtitle>{formatDate(inv.created_at)}</ListItemSubtitle>
        </ListItem>
      ))}
    </ListPanel>
  );

  const detailPanel = (
    <DetailPanel
      title="Investigation Details"
      hasSelection={!!selectedInvestigation}
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Eye className="h-12 w-12 mb-3 opacity-50" />
          <p className="text-sm">Select an investigation to view details</p>
        </div>
      }
    >
      {selectedInvestigation && (
        selectedInvestigation.status === "running" || selectedInvestigation.status === "pending" ? (
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
        )
      )}
    </DetailPanel>
  );

  // Build header content with error banner
  const headerContent = error ? (
    <Card className="border-destructive/50 bg-destructive/10">
      <CardContent className="flex items-center gap-3 py-4">
        <AlertCircle className="h-4 w-4 text-destructive" />
        <p className="text-sm text-destructive">{error}</p>
      </CardContent>
    </Card>
  ) : null;

  return (
    <MasterDetailLayout
      pageTitle="Investigations"
      pageSubtitle="View analysis reports and apply recommendations"
      storageKey="investigations"
      headerContent={headerContent}
      listPanel={listPanel}
      detailPanel={detailPanel}
      selectedId={investigationId ?? null}
      onDeselect={() => navigate("/investigations")}
      detailTitle={
        selectedInvestigation
          ? `${selectedInvestigation.run_ids.length} Run${selectedInvestigation.run_ids.length !== 1 ? "s" : ""} Analyzed`
          : "Investigation Details"
      }
    />
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
