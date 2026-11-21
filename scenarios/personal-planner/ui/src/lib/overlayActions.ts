import type {
  OverlayActionList,
  OverlayPrimaryAction,
  OverlaySecondaryAction,
} from "@vrooli/react-component-library/OverlayActionBar/1";

/** Keeps PlannerDialog actions typed and primary-first, including conditional actions. */
export function overlayActions(
  primary: OverlayPrimaryAction,
  ...secondary: OverlaySecondaryAction[]
): OverlayActionList {
  return [primary, ...secondary];
}

/** Submit through the form so native validation and the existing onSubmit stay authoritative. */
export function submitForm(formId: string) {
  return () => {
    const form = document.getElementById(formId);
    if (form instanceof HTMLFormElement) form.requestSubmit();
  };
}
