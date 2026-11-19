import { capabilityPresentation, type OperationLabelKey } from "./capabilityCatalog";

/**
 * The friendly i18n label key for an operation, resolved from the generated
 * capability catalog (one source for every registry), or `null` when the
 * operation has no bespoke copy yet. Lets Library and Activity render
 * "Remove background" instead of the raw `background_removal` token without each
 * owning a catalog.
 */
export function operationLabelKey(operation: string): OperationLabelKey | null {
  return capabilityPresentation(operation)?.labelKey ?? null;
}
