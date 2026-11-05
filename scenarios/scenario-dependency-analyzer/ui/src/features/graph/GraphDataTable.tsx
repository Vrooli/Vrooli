import { Network } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import type { DependencyGraph, DependencyGraphNode, EdgeStatusFilter } from "../../types";
import { buildDegreeMap, matchesDriftFilter } from "./graphUtils";

interface GraphDataTableProps {
  graph: DependencyGraph | null;
  filter: string;
  driftFilter: EdgeStatusFilter;
  selectedNodeId?: string | null;
  onSelectNode: (node: DependencyGraphNode | null) => void;
}

function matchesNodeFilter(node: DependencyGraphNode, filterText: string) {
  if (!filterText) return true;
  return node.label.toLowerCase().includes(filterText) || node.id.toLowerCase().includes(filterText);
}

export function GraphDataTable({
  graph,
  filter,
  driftFilter,
  selectedNodeId,
  onSelectNode
}: GraphDataTableProps) {
  const filterText = filter.trim().toLowerCase();
  const degreeMap = buildDegreeMap(graph?.edges ?? []);
  const nodes = (graph?.nodes ?? [])
    .filter((node) => matchesNodeFilter(node, filterText))
    .sort((a, b) => (degreeMap.get(b.id) ?? 0) - (degreeMap.get(a.id) ?? 0));
  const edges = (graph?.edges ?? [])
    .filter((edge) => matchesDriftFilter(edge, driftFilter))
    .filter((edge) => {
      if (!filterText) return true;
      return edge.source.toLowerCase().includes(filterText) || edge.target.toLowerCase().includes(filterText) || edge.label.toLowerCase().includes(filterText);
    })
    .slice(0, 12);

  return (
    <Card className="border border-border/40 bg-background/40">
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Network className="h-4 w-4 text-primary" aria-hidden="true" />
          Graph data
        </CardTitle>
        <p className="text-xs text-muted-foreground">
          A keyboard-friendly view of the visible graph. Select a row to inspect the same node shown on the canvas.
        </p>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[520px] text-left text-xs">
            <caption className="sr-only">Dependency graph nodes</caption>
            <thead className="border-b border-border/50 text-muted-foreground">
              <tr>
                <th className="py-2 pr-3 font-medium">Node</th>
                <th className="px-3 py-2 font-medium">Type</th>
                <th className="px-3 py-2 font-medium">Group</th>
                <th className="px-3 py-2 text-right font-medium">Connections</th>
                <th className="py-2 pl-3 text-right font-medium">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/30">
              {nodes.length > 0 ? (
                nodes.map((node) => (
                  <tr key={node.id} className={selectedNodeId === node.id ? "bg-primary/10" : undefined}>
                    <td className="max-w-[180px] py-2 pr-3 font-medium text-foreground">
                      <span className="block truncate">{node.label || node.id}</span>
                      <span className="block truncate text-[11px] font-normal text-muted-foreground">{node.id}</span>
                    </td>
                    <td className="px-3 py-2 text-muted-foreground">{node.type}</td>
                    <td className="px-3 py-2 text-muted-foreground">{node.group || "Unassigned"}</td>
                    <td className="px-3 py-2 text-right text-muted-foreground">{degreeMap.get(node.id) ?? 0}</td>
                    <td className="py-2 pl-3 text-right">
                      <Button
                        size="sm"
                        variant={selectedNodeId === node.id ? "secondary" : "outline"}
                        className="h-8 text-xs"
                        onClick={() => onSelectNode(selectedNodeId === node.id ? null : node)}
                      >
                        {selectedNodeId === node.id ? "Clear" : "Select"}
                      </Button>
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td className="py-4 text-muted-foreground" colSpan={5}>
                    No nodes match the current filters.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full min-w-[520px] text-left text-xs">
            <caption className="sr-only">Dependency graph edges</caption>
            <thead className="border-b border-border/50 text-muted-foreground">
              <tr>
                <th className="py-2 pr-3 font-medium">Source</th>
                <th className="px-3 py-2 font-medium">Target</th>
                <th className="px-3 py-2 font-medium">Relationship</th>
                <th className="px-3 py-2 text-right font-medium">Weight</th>
                <th className="py-2 pl-3 font-medium">Required</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/30">
              {edges.length > 0 ? (
                edges.map((edge) => (
                  <tr key={`${edge.source}-${edge.target}-${edge.label}`}>
                    <td className="py-2 pr-3 text-foreground">{edge.source}</td>
                    <td className="px-3 py-2 text-foreground">{edge.target}</td>
                    <td className="px-3 py-2 text-muted-foreground">{edge.label || edge.type}</td>
                    <td className="px-3 py-2 text-right text-muted-foreground">{edge.weight}</td>
                    <td className="py-2 pl-3 text-muted-foreground">{edge.required ? "Yes" : "No"}</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td className="py-4 text-muted-foreground" colSpan={5}>
                    No edges match the current filters.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}
