import type { DependencyGraphNode, DependencyGraphEdge } from "../../types";

interface HoverState {
  node: DependencyGraphNode;
  position: { x: number; y: number };
}

interface EdgeHoverState {
  edge: DependencyGraphEdge;
  position: { x: number; y: number };
}

interface NodeTooltipProps {
  hover: HoverState;
}

interface EdgeTooltipProps {
  hover: EdgeHoverState;
}

export function NodeTooltip({ hover }: NodeTooltipProps) {
  const { node, position } = hover;
  return (
    <div
      className="pointer-events-none absolute z-20 min-w-[220px] rounded-lg border border-border/40 bg-background/80 p-4 text-sm shadow-xl shadow-black/20 backdrop-blur"
      style={{ left: position.x, top: position.y }}
    >
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {node.type}
      </p>
      <p className="mt-1 font-semibold text-foreground">{node.label}</p>
      {node.metadata && node.metadata["purpose"] ? (
        <p className="mt-2 text-xs text-muted-foreground/90">
          {(node.metadata["purpose"] as string) ?? ""}
        </p>
      ) : null}
      {node.group ? (
        <p className="mt-3 text-[11px] uppercase tracking-widest text-primary/80">
          {node.group}
        </p>
      ) : null}
    </div>
  );
}

export function EdgeTooltip({ hover }: EdgeTooltipProps) {
  const { edge, position } = hover;
  const metadata = (edge.metadata as Record<string, unknown> | undefined) ?? {};
  const configuration = (metadata.configuration as Record<string, unknown> | undefined) ?? {};

  const formatTimestamp = (value?: unknown) => {
    if (typeof value !== "string") return null;
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return null;
    return date.toLocaleString();
  };

  return (
    <div
      className="pointer-events-none absolute z-20 min-w-[240px] rounded-lg border border-border/40 bg-background/90 p-4 text-xs shadow-2xl shadow-black/40 backdrop-blur"
      style={{ left: position.x, top: position.y }}
    >
      <p className="font-semibold text-foreground">{edge.label || edge.type}</p>
      <p className="mt-1 text-[11px] uppercase tracking-widest text-muted-foreground/80">
        {edge.required ? "Required" : "Optional"} · Weight {edge.weight.toFixed(2)}
      </p>
      {metadata.access_method ? (
        <p className="mt-2 text-muted-foreground">Access: {String(metadata.access_method)}</p>
      ) : null}
      {metadata.purpose ? (
        <p className="mt-1 text-muted-foreground">Purpose: {String(metadata.purpose)}</p>
      ) : null}
      {typeof configuration.found_in_file === "string" ? (
        <p className="mt-1 text-muted-foreground">File: {configuration.found_in_file as string}</p>
      ) : null}
      {typeof configuration.pattern_type === "string" ? (
        <p className="text-muted-foreground">Pattern: {configuration.pattern_type as string}</p>
      ) : null}
      {metadata.discovered_at ? (
        <p className="mt-1 text-muted-foreground">Detected: {formatTimestamp(metadata.discovered_at)}</p>
      ) : null}
      {metadata.last_verified ? (
        <p className="text-muted-foreground">Verified: {formatTimestamp(metadata.last_verified)}</p>
      ) : null}
    </div>
  );
}
