import { useMemo, useRef, useState, type PointerEventHandler, type WheelEventHandler } from "react";
import { AlertTriangle, Maximize2, Minus, Plus, RefreshCcw } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import { Button } from "../../../shared/ui/button";
import {
  CANVAS_WORLD_HEIGHT,
  CANVAS_WORLD_WIDTH,
  createViewportTransform,
  type CanvasRenderModel,
  type CanvasRenderNode,
  type CanvasViewport,
} from "../graphCanvasModel";

const MIN_SCALE = 0.3;
const MAX_SCALE = 3.2;

const clamp = (value: number, min: number, max: number) => Math.max(min, Math.min(max, value));

const nodeTestID = (nodeID: string) =>
  `${selectors.graph.nodePrefix}-${nodeID.replace(/[^a-zA-Z0-9_-]/g, "-").toLowerCase()}`;

type GraphCanvasProps = {
  model: CanvasRenderModel;
  viewport: CanvasViewport;
  onViewportChange: (next: CanvasViewport) => void;
  selectedNodeID?: string;
  onSelectNode: (nodeID: string) => void;
  onFit: () => void;
  onReset: () => void;
  onExpand?: (node: CanvasRenderNode) => void;
  canExpand: boolean;
  isExpanding: boolean;
};

export function GraphCanvas({
  model,
  viewport,
  onViewportChange,
  selectedNodeID,
  onSelectNode,
  onFit,
  onReset,
  onExpand,
  canExpand,
  isExpanding,
}: GraphCanvasProps) {
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [dragState, setDragState] = useState<{ x: number; y: number } | null>(null);

  const selectedNode = useMemo(
    () => model.nodes.find((node) => node.id === selectedNodeID),
    [model.nodes, selectedNodeID]
  );

  const setScaleAroundPoint = (nextScaleRaw: number, clientX: number, clientY: number) => {
    const svg = svgRef.current;
    if (!svg) return;
    const rect = svg.getBoundingClientRect();
    const localX = clientX - rect.left;
    const localY = clientY - rect.top;
    const nextScale = clamp(nextScaleRaw, MIN_SCALE, MAX_SCALE);

    const worldX = (localX - viewport.offsetX) / viewport.scale;
    const worldY = (localY - viewport.offsetY) / viewport.scale;

    onViewportChange({
      scale: nextScale,
      offsetX: localX - worldX * nextScale,
      offsetY: localY - worldY * nextScale,
    });
  };

  const handleWheel: WheelEventHandler<SVGSVGElement> = (event) => {
    event.preventDefault();
    const zoomAmount = event.deltaY < 0 ? 1.12 : 0.9;
    setScaleAroundPoint(viewport.scale * zoomAmount, event.clientX, event.clientY);
  };

  const handlePointerDown: PointerEventHandler<SVGRectElement> = (event) => {
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragState({ x: event.clientX, y: event.clientY });
  };

  const handlePointerMove: PointerEventHandler<SVGRectElement> = (event) => {
    if (!dragState) return;
    const dx = event.clientX - dragState.x;
    const dy = event.clientY - dragState.y;
    setDragState({ x: event.clientX, y: event.clientY });
    onViewportChange({
      ...viewport,
      offsetX: viewport.offsetX + dx,
      offsetY: viewport.offsetY + dy,
    });
  };

  const handlePointerUp: PointerEventHandler<SVGRectElement> = () => {
    setDragState(null);
  };

  return (
    <div className="ko-card p-3" data-testid={selectors.graph.canvasPanel}>
      <div className="flex flex-wrap items-center gap-2 pb-3">
        <Button
          type="button"
          variant="outline"
          size="compact"
          onClick={onFit}
          data-testid={selectors.graph.fit}
        >
          <Maximize2 className="h-3.5 w-3.5" />
          <span className="ml-1">Fit</span>
        </Button>
        <Button
          type="button"
          variant="outline"
          size="compact"
          onClick={() => {
            const svg = svgRef.current;
            if (!svg) return;
            const rect = svg.getBoundingClientRect();
            setScaleAroundPoint(viewport.scale * 1.1, rect.left + rect.width / 2, rect.top + rect.height / 2);
          }}
          data-testid={selectors.graph.zoomIn}
        >
          <Plus className="h-3.5 w-3.5" />
          <span className="ml-1">Zoom In</span>
        </Button>
        <Button
          type="button"
          variant="outline"
          size="compact"
          onClick={() => {
            const svg = svgRef.current;
            if (!svg) return;
            const rect = svg.getBoundingClientRect();
            setScaleAroundPoint(viewport.scale * 0.9, rect.left + rect.width / 2, rect.top + rect.height / 2);
          }}
          data-testid={selectors.graph.zoomOut}
        >
          <Minus className="h-3.5 w-3.5" />
          <span className="ml-1">Zoom Out</span>
        </Button>
        <Button
          type="button"
          variant="secondary"
          size="compact"
          onClick={onReset}
          data-testid={selectors.graph.resetViewport}
        >
          <RefreshCcw className="h-3.5 w-3.5" />
          <span className="ml-1">Reset View</span>
        </Button>

        {selectedNode && canExpand && onExpand ? (
          <Button
            type="button"
            variant="secondary"
            size="compact"
            onClick={() => onExpand(selectedNode)}
            disabled={isExpanding}
            data-testid={selectors.graph.expand}
          >
            <span>Expand From Selected</span>
          </Button>
        ) : null}

        <span className="ml-auto ko-meta" data-testid={selectors.graph.zoomLabel}>
          Zoom {(viewport.scale * 100).toFixed(0)}%
        </span>
      </div>

      {model.truncatedCount > 0 ? (
        <div className="ko-alert ko-alert-danger mb-3" data-testid={selectors.graph.truncatedWarning}>
          <AlertTriangle className="h-4 w-4 ko-text-warning mt-0.5" />
          <div>
            <p className="ko-alert-title ko-text-warning">Large graph optimized for rendering</p>
            <p className="ko-text-xs ko-text-warning-muted mt-1">
              {model.truncatedCount} low-priority nodes were hidden. Lower your threshold or expand from specific nodes.
            </p>
          </div>
        </div>
      ) : null}

      <svg
        ref={svgRef}
        className="ko-graph-canvas"
        viewBox={`0 0 ${CANVAS_WORLD_WIDTH} ${CANVAS_WORLD_HEIGHT}`}
        role="img"
        aria-label="Knowledge graph canvas"
        onWheel={handleWheel}
        data-testid={selectors.graph.canvas}
      >
        <defs>
          <linearGradient id="ko-graph-edge" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stopColor="rgb(var(--ko-accent) / 0.85)" />
            <stop offset="100%" stopColor="rgb(var(--ko-accent-strong) / 0.5)" />
          </linearGradient>
          <radialGradient id="ko-graph-node" cx="50%" cy="45%" r="65%">
            <stop offset="0%" stopColor="rgb(var(--ko-accent-strong) / 0.95)" />
            <stop offset="100%" stopColor="rgb(var(--ko-accent) / 0.62)" />
          </radialGradient>
        </defs>

        <rect
          x={0}
          y={0}
          width={CANVAS_WORLD_WIDTH}
          height={CANVAS_WORLD_HEIGHT}
          fill="transparent"
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerUp}
        />

        <g transform={createViewportTransform(viewport)} data-testid={selectors.graph.canvasViewport}>
          {model.edges.map((edge) => (
            <line
              key={edge.id}
              x1={edge.sourceX}
              y1={edge.sourceY}
              x2={edge.targetX}
              y2={edge.targetY}
              className={[
                "ko-graph-edge",
                edge.isHighlighted ? "ko-graph-edge-highlighted" : "",
                edge.isFaded ? "ko-graph-edge-faded" : "",
              ].join(" ")}
              strokeWidth={Math.max(1.2, 1 + edge.weight * 2.2)}
            />
          ))}

          {model.nodes.map((node) => (
            <g
              key={node.id}
              transform={`translate(${node.x} ${node.y})`}
              className="ko-graph-node-group"
              data-testid={nodeTestID(node.id)}
              onClick={(event) => {
                event.stopPropagation();
                onSelectNode(node.id);
              }}
            >
              <circle
                r={node.radius}
                className={[
                  "ko-graph-node",
                  node.isCenter ? "ko-graph-node-center" : "",
                  node.isSelected ? "ko-graph-node-selected" : "",
                  node.isNeighbor ? "ko-graph-node-neighbor" : "",
                  node.isFaded ? "ko-graph-node-faded" : "",
                ].join(" ")}
              />
              <text className="ko-graph-node-label" y={node.radius + 16} textAnchor="middle">
                {node.label.length > 28 ? `${node.label.slice(0, 28)}…` : node.label}
              </text>
            </g>
          ))}
        </g>
      </svg>

      {selectedNode ? (
        <div className="ko-graph-details mt-3" data-testid={selectors.graph.details}>
          <div className="flex items-center justify-between gap-3">
            <p className="ko-text-sm ko-text-strong truncate">Selected: {selectedNode.label}</p>
            <span className="ko-pill ko-pill-muted">ID {selectedNode.id}</span>
          </div>
          <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
            <span className="ko-pill ko-pill-muted">Score {selectedNode.score?.toFixed(3) ?? "N/A"}</span>
            <span className="ko-pill ko-pill-muted">Center {selectedNode.isCenter ? "Yes" : "No"}</span>
          </div>
        </div>
      ) : null}
    </div>
  );
}
