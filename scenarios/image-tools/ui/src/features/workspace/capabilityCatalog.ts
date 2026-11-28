import {
  ImagePlus,
  Layers,
  Palette,
  ScanText,
  Sparkles,
  Wand2,
  type LucideIcon,
} from "lucide-react";

import {
  CAPABILITIES,
  capabilityMeta,
  type CapabilityMeta,
  type CapabilityEngine,
} from "./capabilityCatalog.generated";
import { iconByName } from "./capabilityIcons";
import { strings } from "../../consts/strings";

export { CAPABILITIES, capabilityMeta };
export type { CapabilityEngine, CapabilityMeta };

/**
 * Family → fallback icon. Used when a manifest declares no icon, or names one
 * this build does not have.
 */
const FAMILY_ICON: Readonly<Record<string, LucideIcon>> = {
  generate: ImagePlus,
  edit: Wand2,
  enhance: Sparkles,
  restore: Layers,
  extract: ScanText,
  compose: Palette,
};

/** Fallback icon for an operation's family (generic image icon when unknown). */
export const familyIcon = (family?: string): LucideIcon =>
  (family ? FAMILY_ICON[family] : undefined) ?? ImagePlus;

type WorkspaceStrings = typeof strings.workspace;
type OpKeys = WorkspaceStrings["op"][keyof WorkspaceStrings["op"]];
type AiOpKeys = WorkspaceStrings["aiOp"][keyof WorkspaceStrings["aiOp"]];
type CreateOpKeys = WorkspaceStrings["createOp"][keyof WorkspaceStrings["createOp"]];
type AnalyzeOpKeys = WorkspaceStrings["analyzeOp"][keyof WorkspaceStrings["analyzeOp"]];

/**
 * The i18n key unions the generated catalog must resolve into. Manifests carry
 * dotted key strings (data); these unions keep `t()` type-checked. A conformance
 * test asserts every generated key exists in the locale, so the cast below
 * cannot hide a typo.
 */
export type OperationLabelKey =
  | OpKeys["label"]
  | AiOpKeys["label"]
  | CreateOpKeys["label"]
  | AnalyzeOpKeys["label"];
export type OperationDescKey =
  | OpKeys["desc"]
  | AiOpKeys["desc"]
  | CreateOpKeys["desc"]
  | AnalyzeOpKeys["desc"];

/**
 * Presentation for one operation, resolved from the generated manifest catalog.
 * `labelKey`/`descKey` are i18n keys and are absent for operations that have no
 * bespoke copy yet — callers fall back to the operation name or the manifest
 * summary. `Icon` is always present.
 */
export interface CapabilityPresentation {
  labelKey?: OperationLabelKey;
  descKey?: OperationDescKey;
  Icon: LucideIcon;
  /** Workspace surface this operation appears on, or undefined when it is reached elsewhere. */
  panel?: WorkspacePanel;
}

/** The workspace surfaces the capability manifests declare. */
export type WorkspacePanel = "create" | "enhance" | "analyze";

export const capabilityPresentation = (id: string): CapabilityPresentation | undefined => {
  const meta = capabilityMeta(id);
  if (!meta) {
    return undefined;
  }
  return {
    labelKey: meta.labelKey as OperationLabelKey | undefined,
    descKey: meta.descKey as OperationDescKey | undefined,
    Icon: iconByName(meta.icon) ?? familyIcon(meta.family),
    panel: meta.panel as WorkspacePanel | undefined,
  };
};

/**
 * Whether an operation belongs on a workspace panel. Panel membership is declared
 * by the manifest (`panel`), separately from the operation's family: the family
 * says what it does, the panel says where a person finds it. An operation with no
 * declared panel is reached through a dedicated editor, another service's list,
 * or the CLI.
 */
export const isPanelOperation = (id: string, panel: WorkspacePanel): boolean =>
  capabilityPresentation(id)?.panel === panel;

/** Icon for an operation from the manifest catalog, or undefined if unknown. */
export const capabilityIcon = (id: string): LucideIcon | undefined =>
  capabilityPresentation(id)?.Icon;

/**
 * Canonical one-line description from the manifest. Used as the description
 * fallback for operations that do not yet have translated `desc` copy.
 */
export const capabilitySummary = (id: string): string | undefined => capabilityMeta(id)?.summary;
