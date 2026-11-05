import { strings } from "../../consts/strings";
import { AI_CATALOG } from "./aiCatalog";
import { CREATE_CATALOG } from "./createCatalog";
import { OP_CATALOG } from "./opCatalog";

type AnyLabelKey =
  | (typeof strings.workspace.op)[keyof typeof strings.workspace.op]["label"]
  | (typeof strings.workspace.aiOp)[keyof typeof strings.workspace.aiOp]["label"]
  | (typeof strings.workspace.createOp)[keyof typeof strings.workspace.createOp]["label"];

/**
 * The friendly i18n label key for an operation across all three op catalogs
 * (deterministic / enhancement / generation), or `null` if the op isn't named.
 * Lets Library and Activity render "Remove background" instead of the raw
 * `background_removal` token without each owning a catalog.
 */
export function operationLabelKey(operation: string): AnyLabelKey | null {
  return (
    OP_CATALOG[operation]?.labelKey ??
    AI_CATALOG[operation]?.labelKey ??
    CREATE_CATALOG[operation]?.labelKey ??
    null
  );
}
