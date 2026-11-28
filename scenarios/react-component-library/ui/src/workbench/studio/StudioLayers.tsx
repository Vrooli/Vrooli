import { useEffect, useState, type ReactNode } from "react";
import { ArrowDown, ArrowUp, Copy, Trash2 } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { DesignDocument, DesignFrame } from "../design-types";
import type { Coverage } from "./rcl";
import {
  ADDABLE_NODES,
  CONTAINER_TYPES,
  FRAME_SIZE,
  type AddableNode,
  type LayerRow,
} from "./model";
import { Chip } from "../primitives";
import { StudioLayerTree } from "./StudioLayerTree";
import type { LayerDropPosition } from "./layerMovement";

export function StudioLayers({
  design,
  frame,
  rows,
  selectedId,
  coverage,
  onToggleCollapse,
  onSelect,
  onMove,
  onReparent,
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
  onReparent: (sourceId: string, targetId: string, position: LayerDropPosition) => void;
  onDuplicate: () => void;
  onRemove: () => void;
  onAdd: (entry: AddableNode, parentId: string) => void;
  onRename: (name: string) => void;
  footer?: ReactNode;
}) {
  const { t } = useTranslation();
  const [nameDraft, setNameDraft] = useState(design.name);
  const selected = rows.find((row) => row.node.id === selectedId);
  const parent =
    selected && CONTAINER_TYPES.has(selected.node.type) ? selected.node : rows[0]?.node;
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

      <StudioLayerTree
        rows={rows}
        selectedId={selectedId}
        onToggleCollapse={onToggleCollapse}
        onSelect={onSelect}
        onReparent={onReparent}
      />

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
          <Chip tone="green">
            {t("workbench.studio.legendRcl", { defaultValue: "RCL-backed" })}
          </Chip>
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
