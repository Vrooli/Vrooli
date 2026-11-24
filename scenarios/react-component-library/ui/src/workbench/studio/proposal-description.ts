import type { TFunction } from "i18next";

import type { DesignDocument } from "../design-types";
import { findNode, type TemplateNode } from "../templates/spec";
import { layerLabel } from "../templates/spec";
import type { DesignPatchOperation } from "./proposal";

function findAnywhere(doc: DesignDocument, id: string | undefined): TemplateNode | undefined {
  if (!id) return undefined;
  for (const frame of doc.frames) {
    const found = findNode(frame.spec, id);
    if (found) return found;
  }
  return undefined;
}

function clip(value: string, limit = 64): string {
  return value.length > limit ? `${value.slice(0, limit - 1)}…` : value;
}

export function describeProposalOperation(
  design: DesignDocument,
  operation: DesignPatchOperation,
  t: TFunction,
): { title: string; detail?: string } {
  const target = findAnywhere(design, operation.nodeId);
  const parent = findAnywhere(design, operation.parentId);
  // A person reads layer names, not node ids. The id stays the identity the
  // patch uses; only the description is humanised.
  const nameOf = (node: TemplateNode | undefined, id: string | undefined) =>
    node ? layerLabel(node) : (id ?? "");
  switch (operation.kind) {
    case "setText": {
      const before = typeof target?.props.text === "string" ? target.props.text : "";
      return {
        title: t("workbench.studio.aiOpText", {
          defaultValue: "Rewrite copy in {{layer}}",
          layer: nameOf(target, operation.nodeId),
        }),
        detail: `${clip(before) || "—"} → ${clip(operation.text ?? "")}`,
      };
    }
    case "setProp": {
      const before = target?.props[operation.key ?? ""];
      return {
        title: t("workbench.studio.aiOpProp", {
          defaultValue: "Set {{key}} on {{layer}}",
          key: operation.key ?? "",
          layer: nameOf(target, operation.nodeId),
        }),
        detail: `${String(before ?? "—")} → ${String(operation.value ?? "—")}`,
      };
    }
    case "addLayer":
      return {
        title: t("workbench.studio.aiOpAdd", {
          defaultValue: "Add {{type}} inside {{layer}}",
          type: operation.node?.type ?? "",
          layer: nameOf(parent, operation.parentId),
        }),
      };
    case "removeLayer":
      return {
        title: t("workbench.studio.aiOpRemove", {
          defaultValue: "Remove {{layer}}",
          layer: nameOf(target, operation.nodeId),
        }),
        detail: target?.type,
      };
    case "moveLayer":
      return {
        title: t("workbench.studio.aiOpMove", {
          defaultValue: "Move {{layer}} into {{parent}}",
          layer: nameOf(target, operation.nodeId),
          parent: nameOf(parent, operation.parentId),
        }),
      };
    case "setTheme":
      return {
        title: t("workbench.studio.aiOpTheme", {
          defaultValue: "Set {{key}}",
          key: operation.key ?? "",
        }),
        detail: String(operation.value ?? ""),
      };
    default:
      return { title: String(operation.kind) };
  }
}
