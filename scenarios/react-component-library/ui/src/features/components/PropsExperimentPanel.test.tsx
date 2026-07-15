import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeComponentExample } from "./mocks/factories";
import { PropsExperimentPanel } from "./PropsExperimentPanel";

describe("PropsExperimentPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(cleanup);

  it("shows indexed props and applies only a valid JSON object", async () => {
    const onApply = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<PropsExperimentPanel example={makeComponentExample({ propsJson: '{"title":"Short"}' })} onApply={onApply} onReset={vi.fn()} />);
    const draft = screen.getByTestId<HTMLTextAreaElement>(selectors.components.editor.propsDraft);
    expect(draft.value).toContain('"title": "Short"');
    fireEvent.change(draft, { target: { value: '{"title":"A much longer temporary value"}' } });
    await user.click(screen.getByTestId(selectors.components.editor.propsApply));
    expect(onApply).toHaveBeenCalledWith({ title: "A much longer temporary value" });
  });

  it("keeps invalid JSON in the host and explains the error", async () => {
    const onApply = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<PropsExperimentPanel example={makeComponentExample()} onApply={onApply} onReset={vi.fn()} />);
    const draft = screen.getByTestId<HTMLTextAreaElement>(selectors.components.editor.propsDraft);
    fireEvent.change(draft, { target: { value: "not json" } });
    await user.click(screen.getByTestId(selectors.components.editor.propsApply));
    expect(onApply).not.toHaveBeenCalled();
    expect(screen.getByTestId(selectors.components.editor.propsError)).toHaveTextContent("Fix the JSON before applying the experiment.");
  });

  it("resets through the provided session-only callback", async () => {
    const onReset = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<PropsExperimentPanel example={makeComponentExample()} onApply={vi.fn()} onReset={onReset} />);
    await user.click(screen.getByTestId(selectors.components.editor.propsReset));
    expect(onReset).toHaveBeenCalledTimes(1);
  });
});
