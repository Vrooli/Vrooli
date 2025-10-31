import { ExternalLink, Info, Link2 } from "lucide-react";
import { Badge } from "./ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import type { DependencyGraphEdge, DependencyGraphNode } from "../types";

interface ConnectionInsight {
  node: DependencyGraphNode;
  edge: DependencyGraphEdge;
}

interface SelectedNodePanelProps {
  node: DependencyGraphNode;
  connections: ConnectionInsight[];
}

export function SelectedNodePanel({ node, connections }: SelectedNodePanelProps) {
  return (
    <Card className="glass border border-border/40">
      <CardHeader>
        <CardTitle className="flex flex-col gap-2 text-base font-semibold">
          <span className="flex items-center gap-2">
            <Info className="h-4 w-4 text-primary" />
            {node.label}
          </span>
          <Badge className="self-start uppercase" variant="secondary">
            {node.type}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {node.metadata?.purpose ? (
          <div className="rounded-lg border border-border/40 bg-background/60 p-3 text-sm text-muted-foreground">
            {String(node.metadata.purpose)}
          </div>
        ) : (
          <p className="rounded-lg border border-border/40 bg-background/40 p-3 text-sm text-muted-foreground">
            This node does not yet have a documented purpose. Consider enriching its metadata within the scenario repository.
          </p>
        )}

        <section className="space-y-2">
          <div className="flex items-center justify-between text-xs font-medium uppercase tracking-wide text-muted-foreground">
            <span>Connections</span>
            <span className="text-muted-foreground/70">{connections.length}</span>
          </div>
          {connections.length === 0 ? (
            <p className="rounded-lg border border-border/40 bg-background/40 p-3 text-xs text-muted-foreground">
              No direct dependencies detected for this node in the current graph selection.
            </p>
          ) : (
            <ScrollArea className="max-h-[220px] rounded-lg border border-border/40 bg-background/60">
              <ul className="divide-y divide-border/20 text-sm">
                {connections.map(({ node: peer, edge }) => (
                  <li key={`${edge.source}-${edge.target}-${edge.label}`} className="flex items-start gap-3 p-3">
                    <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                      <Link2 className="h-4 w-4" aria-hidden="true" />
                    </div>
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <p className="font-medium text-foreground">{peer.label}</p>
                        <Badge variant="outline" className="uppercase">
                          {peer.type}
                        </Badge>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {edge.label || "Dependency relationship"}
                      </p>
                      <p className="text-[11px] uppercase tracking-widest text-muted-foreground/80">
                        {edge.required ? "Required" : "Optional"} · Weight {edge.weight.toFixed(2)}
                      </p>
                    </div>
                  </li>
                ))}
              </ul>
            </ScrollArea>
          )}
        </section>

        {node.metadata && Object.keys(node.metadata).length > 0 ? (
          <section className="space-y-2 text-xs">
            <p className="font-medium uppercase tracking-wide text-muted-foreground">
              Metadata
            </p>
            <div className="space-y-1 rounded-lg border border-border/40 bg-background/60 p-3">
              {Object.entries(node.metadata).map(([key, value]) => (
                <div key={key} className="flex items-center justify-between text-muted-foreground">
                  <span className="uppercase tracking-wide">{key}</span>
                  <span className="text-foreground/80">{String(value)}</span>
                </div>
              ))}
            </div>
          </section>
        ) : null}

        <a
          className="group inline-flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-muted-foreground hover:text-primary"
          href={`https://www.google.com/search?q=${encodeURIComponent(node.label + " Vrooli")}`}
          target="_blank"
          rel="noreferrer"
        >
          Deep dive this capability <ExternalLink className="h-3 w-3 transition-transform group-hover:translate-x-0.5" />
        </a>
      </CardContent>
    </Card>
  );
}
