import type { KeyboardEvent } from "react";

/**
 * Arrow-key behaviour for a `role="radiogroup"`: arrows, Home and End move focus
 * and selection together, skip disabled options, and follow reading direction.
 * Attach to the group. It moves focus; `select` receives the new index.
 */
export function handleRadioKeys(event: KeyboardEvent<HTMLElement>, select: (index: number) => void) {
  const radios = [...event.currentTarget.querySelectorAll<HTMLElement>('[role="radio"]')];
  const enabled = radios.map((radio, index) => ({ radio, index })).filter(({ radio }) => !radio.hasAttribute("disabled") && radio.getAttribute("aria-disabled") !== "true");
  if (enabled.length === 0) return;
  const rtl = getComputedStyle(event.currentTarget).direction === "rtl";
  const forward = rtl ? "ArrowLeft" : "ArrowRight";
  const backward = rtl ? "ArrowRight" : "ArrowLeft";
  const position = enabled.findIndex(({ radio }) => radio === document.activeElement);
  let next: number;
  if (event.key === forward || event.key === "ArrowDown") next = position + 1;
  else if (event.key === backward || event.key === "ArrowUp") next = position - 1;
  else if (event.key === "Home") next = 0;
  else if (event.key === "End") next = enabled.length - 1;
  else return;
  event.preventDefault();
  const target = enabled[(next + enabled.length) % enabled.length];
  if (!target) return;
  target.radio.focus();
  select(target.index);
}
