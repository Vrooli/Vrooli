import { RefreshCw, Scan, Share2, Workflow } from "lucide-react";
import { Button } from "./ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue
} from "./ui/select";
import type { EdgeStatusFilter, GraphType, LayoutMode } from "../types";

interface ControlPanelProps {
  graphType: GraphType;
  layout: LayoutMode;
  filter: string;
  driftFilter: EdgeStatusFilter;
  onGraphTypeChange: (value: GraphType) => void;
  onLayoutChange: (value: LayoutMode) => void;
  onFilterChange: (value: string) => void;
  onDriftFilterChange: (value: EdgeStatusFilter) => void;
  onRefresh: () => void;
  onAnalyzeAll: () => Promise<boolean> | void;
  onExport: () => void;
  loading: boolean;
}

const graphTypeLabels: Record<GraphType, string> = {
  combined: "Combined Intelligence Graph",
  resource: "Resource Relationships",
  scenario: "Scenario Interactions"
};

const layoutLabels: Record<LayoutMode, string> = {
  force: "Adaptive Force Layout",
  radial: "Radial Orbit Layout",
  grid: "Structured Grid Layout"
};

const driftLabels: Record<EdgeStatusFilter, string> = {
  all: "Show all dependencies",
  missing: "Only missing/undeclared",
  "declared-only": "Declared but unused"
};

export function ControlPanel({
  graphType,
  layout,
  filter,
  driftFilter,
  onGraphTypeChange,
  onLayoutChange,
  onFilterChange,
  onDriftFilterChange,
  onRefresh,
  onAnalyzeAll,
  onExport,
  loading
}: ControlPanelProps) {
  return (
    <Card className="glass border border-border/40">
      <CardHeader className="space-y-2">
        <CardTitle className="flex items-center gap-2 text-base font-semibold">
          <Workflow className="h-4 w-4 text-primary" /> Scenario Controls
        </CardTitle>
        <CardDescription>
          Shape the visualization, run holistic analysis, and export intelligence artifacts.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="space-y-2">
          <Label htmlFor="graph-type">Graph Focus</Label>
          <Select
            value={graphType}
            onValueChange={(value) => onGraphTypeChange(value as GraphType)}
          >
            <SelectTrigger id="graph-type" aria-label="Select graph focus">
              <SelectValue placeholder="Select a graph" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectLabel>Available Graphs</SelectLabel>
                <SelectItem value="combined">{graphTypeLabels.combined}</SelectItem>
                <SelectItem value="resource">{graphTypeLabels.resource}</SelectItem>
                <SelectItem value="scenario">{graphTypeLabels.scenario}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="layout-mode">Layout Strategy</Label>
          <Select value={layout} onValueChange={(value) => onLayoutChange(value as LayoutMode)}>
            <SelectTrigger id="layout-mode" aria-label="Select layout strategy">
              <SelectValue placeholder="Select a layout" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectLabel>Visual Architectures</SelectLabel>
                <SelectItem value="force">{layoutLabels.force}</SelectItem>
                <SelectItem value="radial">{layoutLabels.radial}</SelectItem>
                <SelectItem value="grid">{layoutLabels.grid}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="search-nodes">Focus Search</Label>
          <Input
            id="search-nodes"
            placeholder="Filter by scenario or resource"
            value={filter}
            onChange={(event) => onFilterChange(event.target.value)}
            autoComplete="off"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="drift-filter">Dependency Drift</Label>
          <Select
            value={driftFilter}
            onValueChange={(value) => onDriftFilterChange(value as EdgeStatusFilter)}
          >
            <SelectTrigger id="drift-filter" aria-label="Highlight drift status">
              <SelectValue placeholder="Select drift filter" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectLabel>Highlight</SelectLabel>
                <SelectItem value="all">{driftLabels.all}</SelectItem>
                <SelectItem value="missing">{driftLabels.missing}</SelectItem>
                <SelectItem value="declared-only">{driftLabels["declared-only"]}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div className="grid grid-cols-1 gap-3 pt-1">
          <Button variant="outline" onClick={onRefresh} disabled={loading} className="gap-2">
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            Refresh Graph
          </Button>
          <Button
            onClick={() => void onAnalyzeAll()}
            disabled={loading}
            className="gap-2"
          >
            <Scan className="h-4 w-4" aria-hidden="true" /> Analyze All Scenarios
          </Button>
          <Button
            variant="secondary"
            className="gap-2"
            onClick={onExport}
            type="button"
          >
            <Share2 className="h-4 w-4" aria-hidden="true" /> Export Graph JSON
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
