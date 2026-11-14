import { fireEvent, screen, within } from "@testing-library/react";
export {
  configureTestProviders,
  createHookWrapper,
  createTestQueryClient,
  renderWithProviders,
} from "./renderWithProviders.js";
export type {
  ProviderRenderOptions,
  ProviderRenderResult,
  TestProvider,
} from "./renderWithProviders.js";
export { expectNoA11yViolations } from "./a11y.js";

// chooseOption lives in this module rather than its own file on purpose:
// consumer copies of this package are hard links to dist/, so a rebuild updates
// existing files everywhere at once while a new file only appears after each
// consumer re-runs setup. A new module imported from here would break every
// consumer that has not.

/** Where to look for the control; defaults to the whole document. */
export interface ChooseOptionOptions {
  container?: HTMLElement;
}

type Matcher = string | RegExp;

function matches(text: string, matcher: Matcher): boolean {
  const normalized = text.replace(/\s+/g, " ").trim();
  return typeof matcher === "string" ? normalized === matcher : matcher.test(normalized);
}

/**
 * Chooses `option` in the select labelled `label`, the way a person does.
 *
 * It works for a visible native `<select>` and for a React Component Library
 * Select, which keeps a hidden native select for form state behind a trigger
 * button named by the same label. For the library Select it opens the trigger
 * and clicks the option in the listbox; for a native select it changes the
 * value. `option` matches an option's visible label or its value, so tests
 * written against `userEvent.selectOptions(…, value)` migrate unchanged.
 */
export async function chooseOption(
  label: Matcher,
  option: Matcher,
  { container }: ChooseOptionOptions = {},
): Promise<void> {
  const scope = container ? within(container) : screen;
  const labelled = scope.queryAllByLabelText(label);
  const native = labelled.find(
    (element): element is HTMLSelectElement =>
      element instanceof HTMLSelectElement && element.getAttribute("aria-hidden") !== "true",
  );
  if (native) {
    const choice = Array.from(native.options).find(
      (candidate) => matches(candidate.textContent ?? "", option) || matches(candidate.value, option),
    );
    if (!choice) throw new Error(`select "${String(label)}" has no option "${String(option)}"`);
    fireEvent.change(native, { target: { value: choice.value } });
    return;
  }

  const trigger = scope.getByRole("button", { name: label });
  const hidden = trigger.parentElement?.querySelector("select");
  // Resolve a value to the label the listbox shows for it.
  const optionLabel =
    hidden &&
    Array.from(hidden.options).find((candidate) => typeof option === "string" && candidate.value === option)
      ?.textContent;
  fireEvent.click(trigger);
  const listboxOption = await screen.findByRole("option", {
    name: optionLabel ? optionLabel.replace(/\s+/g, " ").trim() : option,
  });
  fireEvent.click(listboxOption);
}
