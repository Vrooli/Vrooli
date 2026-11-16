import type {
  OverlayActionList,
  OverlayPrimaryAction,
  OverlaySecondaryAction,
} from "@vrooli/react-component-library/OverlayActionBar/1";

/** Keeps conditional overlay action lists typed with their primary action first. */
export function overlayActions(
  primary: OverlayPrimaryAction,
  ...secondary: OverlaySecondaryAction[]
): OverlayActionList {
  return [primary, ...secondary];
}
