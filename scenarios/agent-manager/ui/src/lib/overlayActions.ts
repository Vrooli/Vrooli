import type {
  OverlayActionList,
  OverlayPrimaryAction,
  OverlaySecondaryAction,
} from "@vrooli/react-component-library/OverlayActionBar/1";

/**
 * Keeps conditional overlay action lists typed with their primary action first.
 * The component's usage contract allows at most four actions, so the builder
 * returns one of the tuple lengths the contract declares; extra conditional
 * actions are not silently typed as valid.
 */
export function overlayActions(
  primary: OverlayPrimaryAction,
  ...secondary: OverlaySecondaryAction[]
): OverlayActionList {
  const [first, second, third] = secondary;
  if (first === undefined) return [primary];
  if (second === undefined) return [primary, first];
  if (third === undefined) return [primary, first, second];
  return [primary, first, second, third];
}
