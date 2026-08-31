/**
 * @libraryId react-component-library:NetworkGraph
 * @displayName NetworkGraph
 * @description A canvas-rendered dependency graph with a keyboard-accessible node list.
 * @version 1.0.6
 * @tags ["visualization","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:NetworkGraph */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
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
export const NetworkGraph = withClassName(function NetworkGraph({
  nodes = [],
  edges = [],
}: {
  nodes?: GraphNode[];
  edges?: GraphEdge[];
}) {
  const strings = useStrings();
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
      aria-label={strings("visualization.network-graph.dependency-network", "Dependency network")}
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
        aria-label={strings(
          "visualization.network-graph.keyboard-accessible-dependency-nodes-style-displ",
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
            <span
              style={{
                position: "absolute",
                inlineSize: 1,
                blockSize: 1,
                overflow: "hidden",
                clipPath: "inset(50%)",
                whiteSpace: "nowrap",
              }}
            >
              {strings(
                "visualization.network-graph.select-dependency-node",
                "Select dependency node",
              )}
            </span>
          </button>
        ))}
      </div>
    </section>
  );
});
