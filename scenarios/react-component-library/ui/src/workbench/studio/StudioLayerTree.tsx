import { useState } from "react";
import {
  Box,
  ChevronDown,
  ChevronRight,
  Component,
  Gauge,
  Grid3x3,
  LayoutTemplate,
  Minus,
  MousePointerClick,
  MoveVertical,
  Rows3,
  Shapes,
  Square,
  Tag,
  TextCursorInput,
  Type,
  UserRound,
  Sparkles,
} from "lucide-react";
import { useTranslation } from "../../i18n";
import { layerLabel, type TemplateNodeType } from "../templates/spec";
import { rclPrimitive } from "./rcl";
import { CONTAINER_TYPES, type LayerRow } from "./model";
import { cx } from "../cx";
import type { LayerDropPosition } from "./layerMovement";

const NODE_ICON: Partial<Record<TemplateNodeType, typeof Box>> = {
  page: LayoutTemplate,
  stack: Box,
  row: Rows3,
  grid: Grid3x3,
  surface: Square,
  card: Square,
  text: Type,
  button: MousePointerClick,
  input: TextCursorInput,
  stat: Gauge,
  metric: Gauge,
  icon: Sparkles,
  avatar: UserRound,
  chip: Tag,
  meter: Gauge,
  divider: Minus,
  spacer: MoveVertical,
  slot: Component,
  swatch: Shapes,
};
export function StudioLayerTree({ rows, selectedId, onToggleCollapse, onSelect, onReparent }: {
  rows: LayerRow[];
  selectedId: string | null;
  onToggleCollapse: (id: string) => void;
  onSelect: (id: string) => void;
  onReparent: (sourceId: string, targetId: string, position: LayerDropPosition) => void;
}) {
  const { t } = useTranslation();
  const [dragSource, setDragSource] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<{ id: string; position: LayerDropPosition } | null>(null);
  const [keyboardDrag, setKeyboardDrag] = useState(false);
  const finishDrop = (source: string, target: string, position: LayerDropPosition) => {
    if (source !== target) onReparent(source, target, position);
    setDragSource(null);
    setDropTarget(null);
    setKeyboardDrag(false);
  };
  return (
      <div
        className="studio-tree"
        role="tree"
        aria-label={t("workbench.studio.layers", { defaultValue: "Layers" })}
      >
        {rows.map((row) => {
          const icon = NODE_ICON[row.node.type] ?? Box;
          const Icon = icon;
          const primitive = rclPrimitive(row.node.type);
          const current = row.node.id === selectedId;
          return (
            <div
              key={row.node.id}
              className={cx("studio-tree-row", current && "is-selected")}
              style={{ paddingLeft: `${6 + row.depth * 13}px` }}
              data-drag-source={dragSource === row.node.id || undefined}
              data-drop-position={dropTarget?.id === row.node.id ? dropTarget.position : undefined}
              role="treeitem"
              aria-level={row.depth + 1}
              aria-selected={current}
              aria-expanded={row.hasChildren ? !row.collapsed : undefined}
            >
              {row.hasChildren ? (
                <button
                  type="button"
                  className="studio-tree-caret"
                  aria-label={
                    row.collapsed
                      ? t("workbench.studio.expand", {
                          defaultValue: "Expand {{name}}",
                          name: layerLabel(row.node),
                        })
                      : t("workbench.studio.collapse", {
                          defaultValue: "Collapse {{name}}",
                          name: layerLabel(row.node),
                        })
                  }
                  onClick={() => onToggleCollapse(row.node.id)}
                >
                  {row.collapsed ? (
                    <ChevronRight aria-hidden size={11} />
                  ) : (
                    <ChevronDown aria-hidden size={11} />
                  )}
                </button>
              ) : (
                <span className="studio-tree-caret" aria-hidden />
              )}
              <button
                type="button"
                className="studio-tree-select"
                draggable={row.node.id !== rows[0]?.node.id}
                aria-label={
                  keyboardDrag && dragSource === row.node.id
                    ? t("workbench.studio.layerGrabbed", {
                        defaultValue:
                          "Moving {{name}}. Choose a destination with arrow keys, then press Enter.",
                        name: layerLabel(row.node),
                      })
                    : undefined
                }
                onClick={() => onSelect(row.node.id)}
                onDragStart={(event) => {
                  if (row.node.id === rows[0]?.node.id) {
                    event.preventDefault();
                    return;
                  }
                  event.dataTransfer.effectAllowed = "move";
                  event.dataTransfer.setData("text/plain", row.node.id);
                  setDragSource(row.node.id);
                }}
                onDragEnd={() => {
                  setDragSource(null);
                  setDropTarget(null);
                }}
                onDragOver={(event) => {
                  const source = event.dataTransfer.getData("text/plain") || dragSource;
                  if (!source || source === row.node.id || source === rows[0]?.node.id) return;
                  const rect = event.currentTarget.getBoundingClientRect();
                  const relative = (event.clientY - rect.top) / Math.max(rect.height, 1);
                  const position: LayerDropPosition =
                    CONTAINER_TYPES.has(row.node.type) && relative > 0.25 && relative < 0.75
                      ? "inside"
                      : relative < 0.5
                        ? "before"
                        : "after";
                  event.preventDefault();
                  event.dataTransfer.dropEffect = "move";
                  setDropTarget({ id: row.node.id, position });
                }}
                onDragLeave={() =>
                  setDropTarget((value) => (value?.id === row.node.id ? null : value))
                }
                onDrop={(event) => {
                  event.preventDefault();
                  const source = event.dataTransfer.getData("text/plain") || dragSource;
                  const rect = event.currentTarget.getBoundingClientRect();
                  const relative = (event.clientY - rect.top) / Math.max(rect.height, 1);
                  const position: LayerDropPosition =
                    CONTAINER_TYPES.has(row.node.type) && relative > 0.25 && relative < 0.75
                      ? "inside"
                      : relative < 0.5
                        ? "before"
                        : "after";
                  if (source) finishDrop(source, row.node.id, position);
                }}
                onKeyDown={(event) => {
                  if (event.key === " " && !keyboardDrag && row.node.id !== rows[0]?.node.id) {
                    event.preventDefault();
                    setDragSource(row.node.id);
                    setKeyboardDrag(true);
                    setDropTarget({ id: row.node.id, position: "after" });
                    return;
                  }
                  if (!keyboardDrag || !dragSource) return;
                  if (event.key === "Escape") {
                    event.preventDefault();
                    setDragSource(null);
                    setDropTarget(null);
                    setKeyboardDrag(false);
                    return;
                  }
                  if (event.key === "ArrowUp" || event.key === "ArrowDown") {
                    event.preventDefault();
                    const index = rows.findIndex(
                      (entry) => entry.node.id === (dropTarget?.id ?? dragSource),
                    );
                    const next =
                      rows[
                        Math.max(
                          0,
                          Math.min(rows.length - 1, index + (event.key === "ArrowUp" ? -1 : 1)),
                        )
                      ];
                    if (next && next.node.id !== dragSource)
                      setDropTarget({ id: next.node.id, position: "after" });
                    return;
                  }
                  if (
                    event.key === "ArrowRight" &&
                    dropTarget &&
                    CONTAINER_TYPES.has(
                      rows.find((entry) => entry.node.id === dropTarget.id)?.node.type ?? "text",
                    )
                  ) {
                    event.preventDefault();
                    setDropTarget({ ...dropTarget, position: "inside" });
                    return;
                  }
                  if (event.key === "Enter" && dropTarget) {
                    event.preventDefault();
                    finishDrop(dragSource, dropTarget.id, dropTarget.position);
                  }
                }}
                aria-keyshortcuts="Space ArrowUp ArrowDown ArrowRight Enter Escape"
              >
                <Icon aria-hidden size={12} className="studio-tree-icon" />
                <span className="studio-tree-name">{layerLabel(row.node)}</span>
                {row.node.type === "slot" ? (
                  <span className="studio-tree-slot">
                    {typeof row.node.props.role === "string"
                      ? row.node.props.role
                      : t("workbench.studio.slot", { defaultValue: "slot" })}
                  </span>
                ) : null}
                {row.node.type === "slot" ? null : (
                  <span className="studio-tree-type">{row.node.type}</span>
                )}
                {primitive ? (
                  <span className="studio-tree-badge is-rcl" title={primitive}>
                    {t("workbench.studio.badgeRcl", { defaultValue: "RCL" })}
                  </span>
                ) : CONTAINER_TYPES.has(row.node.type) ? null : (
                  <span className="studio-tree-badge is-custom">
                    {t("workbench.studio.custom", { defaultValue: "custom" })}
                  </span>
                )}
              </button>
            </div>
          );
        })}
      </div>

  );
}
