import { useEffect, useState, type ReactNode } from "react";
import {
  ArrowDown,
  ArrowUp,
  Box,
  ChevronDown,
  ChevronRight,
  Component,
  Copy,
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
  Trash2,
  Type,
  UserRound,
} from "lucide-react";

import { useTranslation } from "../../i18n";
import type { DesignDocument, DesignFrame } from "../designs";
import { layerLabel, type TemplateNodeType } from "../templates/spec";
import type { Coverage } from "./rcl";
import { rclPrimitive } from "./rcl";
import {
  ADDABLE_NODES,
  CONTAINER_TYPES,
  FRAME_SIZE,
  type AddableNode,
  type LayerRow,
} from "./model";
import { cx } from "../cx";
import { Chip } from "../primitives";

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
  avatar: UserRound,
  chip: Tag,
  meter: Gauge,
  divider: Minus,
  spacer: MoveVertical,
  slot: Component,
  swatch: Shapes,
};

export function StudioLayers({
  design,
  frame,
  rows,
  selectedId,
  coverage,
  onToggleCollapse,
  onSelect,
  onMove,
  onDuplicate,
  onRemove,
  onAdd,
  onRename,
  footer,
}: {
  design: DesignDocument;
  frame: DesignFrame;
  rows: LayerRow[];
  selectedId: string | null;
  coverage: Coverage;
  onToggleCollapse: (id: string) => void;
  onSelect: (id: string) => void;
  onMove: (direction: -1 | 1) => void;
  onDuplicate: () => void;
  onRemove: () => void;
  onAdd: (entry: AddableNode, parentId: string) => void;
  onRename: (name: string) => void;
  footer?: ReactNode;
}) {
  const { t } = useTranslation();
  const [nameDraft, setNameDraft] = useState(design.name);
  const selected = rows.find((row) => row.node.id === selectedId);
  const parent = selected && CONTAINER_TYPES.has(selected.node.type) ? selected.node : rows[0]?.node;
  const size = FRAME_SIZE[frame.device];

  useEffect(() => {
    setNameDraft(design.name);
  }, [design.id, design.name]);

  const commitName = () => {
    const next = nameDraft.trim();
    if (!next || next === design.name) {
      setNameDraft(design.name);
      return;
    }
    onRename(next);
  };

  return (
    <>
      <div className="studio-pane-head">
        <input
          className="studio-name-input"
          value={nameDraft}
          aria-label={t("workbench.studio.designName", { defaultValue: "Design name" })}
          onChange={(event) => setNameDraft(event.target.value)}
          onBlur={commitName}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.currentTarget.blur();
            }
            if (event.key === "Escape") setNameDraft(design.name);
          }}
        />
        <span className="studio-frame-chip">
          {frame.device} · {size.width} × {size.height}
        </span>
      </div>

      <div className="studio-tree" role="tree" aria-label={t("workbench.studio.layers", { defaultValue: "Layers" })}>
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
                      ? t("workbench.studio.expand", { defaultValue: "Expand {{name}}", name: layerLabel(row.node) })
                      : t("workbench.studio.collapse", { defaultValue: "Collapse {{name}}", name: layerLabel(row.node) })
                  }
                  onClick={() => onToggleCollapse(row.node.id)}
                >
                  {row.collapsed ? <ChevronRight aria-hidden size={11} /> : <ChevronDown aria-hidden size={11} />}
                </button>
              ) : (
                <span className="studio-tree-caret" aria-hidden />
              )}
              <button
                type="button"
                className="studio-tree-select"
                onClick={() => onSelect(row.node.id)}
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

      <div className="studio-pane-foot">
        <div className="studio-add-row">
          <select
            className="wb-select studio-add-select"
            aria-label={t("workbench.studio.addLayer", { defaultValue: "Add a layer" })}
            value=""
            onChange={(event) => {
              const entry = ADDABLE_NODES.find((item) => item.id === event.target.value);
              if (entry && parent) onAdd(entry, parent.id);
              event.target.value = "";
            }}
          >
            <option value="" disabled>
              {t("workbench.studio.addLayer", { defaultValue: "Add a layer" })}
            </option>
            {ADDABLE_NODES.map((entry) => (
              <option key={entry.id} value={entry.id}>
                {t(entry.labelKey, { defaultValue: entry.label })}
              </option>
            ))}
          </select>
          <button
            type="button"
            className="studio-iconbtn"
            aria-label={t("workbench.studio.moveUp", { defaultValue: "Move up" })}
            disabled={!selected || selected.node.id === rows[0]?.node.id}
            onClick={() => onMove(-1)}
          >
            <ArrowUp aria-hidden size={13} />
          </button>
          <button
            type="button"
            className="studio-iconbtn"
            aria-label={t("workbench.studio.moveDown", { defaultValue: "Move down" })}
            disabled={!selected || selected.node.id === rows[0]?.node.id}
            onClick={() => onMove(1)}
          >
            <ArrowDown aria-hidden size={13} />
          </button>
          <button
            type="button"
            className="studio-iconbtn"
            aria-label={t("workbench.studio.duplicateLayer", { defaultValue: "Duplicate layer" })}
            disabled={!selected || selected.node.id === rows[0]?.node.id}
            onClick={onDuplicate}
          >
            <Copy aria-hidden size={13} />
          </button>
          <button
            type="button"
            className="studio-iconbtn is-danger"
            aria-label={t("workbench.studio.remove", { defaultValue: "Remove" })}
            disabled={!selected || selected.node.id === rows[0]?.node.id}
            onClick={onRemove}
          >
            <Trash2 aria-hidden size={13} />
          </button>
        </div>

        <div className="studio-legend">
          <Chip tone="green">{t("workbench.studio.legendRcl", { defaultValue: "RCL-backed" })}</Chip>
          <Chip>{t("workbench.studio.legendCustom", { defaultValue: "Custom" })}</Chip>
          <Chip>
            {t("workbench.studio.layerCount", {
              defaultValue: "{{count}} layers",
              count: coverage.total,
            })}
          </Chip>
          <span className="studio-coverage">
            {t("workbench.studio.coverage", {
              defaultValue: "{{percent}}% resolve to RCL",
              percent: Math.round(coverage.ratio * 100),
            })}
          </span>
        </div>

        {footer}
      </div>
    </>
  );
}
