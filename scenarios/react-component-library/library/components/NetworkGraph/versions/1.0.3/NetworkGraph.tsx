/**
 * @libraryId react-component-library:NetworkGraph
 * @displayName NetworkGraph
 * @description A canvas-rendered dependency graph with a keyboard-accessible node list.
 * @version 1.0.3
 * @tags ["visualization","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:NetworkGraph */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { useEffect, useRef } from "react";
export interface GraphNode {
  id: string;
  label: string;
  health?: string;
}
export interface GraphEdge {
  from: string;
  to: string;
}
export function NetworkGraph({
  nodes = [],
  edges = [],
}: {
  nodes?: GraphNode[];
  edges?: GraphEdge[];
}) {
  const canvas = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const element = canvas.current;
    if (!element) return;
    const context = element.getContext("2d");
    if (!context) return;
    context.clearRect(0, 0, element.width, element.height);
    const positions = new Map(
      nodes.map((node, index) => [
        node.id,
        { x: 24 + (index % 12) * 72, y: 24 + Math.floor(index / 12) * 44 },
      ]),
    );
    context.strokeStyle = "var(--color-border)";
    for (const edge of edges) {
      const from = positions.get(edge.from);
      const to = positions.get(edge.to);
      if (from && to) {
        context.beginPath();
        context.moveTo(from.x, from.y);
        context.lineTo(to.x, to.y);
        context.stroke();
      }
    }
    context.fillStyle = "currentColor";
    for (const point of positions.values()) {
      context.beginPath();
      context.arc(point.x, point.y, 5, 0, Math.PI * 2);
      context.fill();
    }
  }, [nodes, edges]);
  return (
    <section
      aria-label={translate("visualization.network-graph.aria-label.1", "Dependency network")}
      style={{ display: "grid", gap: "var(--space-xs)" }}
    >
      <div role="img" aria-label={`${nodes.length} dependency nodes`}>
        <canvas
          ref={canvas}
          width={900}
          height={160}
          aria-hidden="true"
          style={{
            width: "100%",
            border: "1px solid var(--color-border)",
            borderRadius: "var(--radius-control)",
          }}
        />
      </div>
      <div
        aria-label={translate(
          "visualization.network-graph.aria-label.2",
          "Keyboard accessible dependency nodes",
        )}
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(10rem, 1fr))",
          gap: "var(--space-2xs)",
        }}
      >
        {nodes.map((node) => (
          <button
            data-testid="visualization.network-graph"
            type="button"
            key={node.id}
            data-node-id={node.id}
            aria-label={`${node.label}, ${node.health ?? "unknown"}`}
            style={{ textAlign: "start" }}
          >
            <span className="sr-only">
              {translate("visualization.network-graph.text.1", "Select dependency node")}
            </span>
          </button>
        ))}
      </div>
    </section>
  );
}
