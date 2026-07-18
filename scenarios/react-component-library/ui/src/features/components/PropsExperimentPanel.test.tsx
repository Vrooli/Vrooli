import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { PropsExperimentPanel } from "./PropsExperimentPanel";

const storyContract = {
  id: "contract", componentId: "component", libraryId: "rcl:component", version: "1.0.0", schemaVersion: 1,
  kind: "component", title: "", argsJson: '{"fields":[{"path":"title","kind":"text","default":"Short"}]}',
  environmentJson: '{"fixtures":[]}', storiesJson: '[{"id":"primary","args":{"title":"Short"}}]', contractJson: "{}", sourcePath: "story.json",
};

describe("PropsExperimentPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(cleanup);

  it("shows indexed props and applies only a valid JSON object", async () => {
    const onApply = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<PropsExperimentPanel storyContract={storyContract} storyId="primary" storyName="Primary" initialArgs={{ title: "Short" }} onApply={onApply} onReset={vi.fn()} />);
    const field = screen.getByLabelText("title");
    fireEvent.change(field, { target: { value: "A much longer temporary value" } });
    await user.click(screen.getByTestId(selectors.components.editor.propsApply));
    expect(onApply).toHaveBeenCalledWith({ title: "A much longer temporary value" });
  });

  it("keeps JSON diagnostics closed until explicitly requested", () => {
    renderWithProviders(<PropsExperimentPanel storyContract={storyContract} onApply={vi.fn()} onReset={vi.fn()} />);
    expect(screen.queryByTestId(selectors.components.editor.propsDraft)).not.toBeInTheDocument();
  });

  it("starts each named story with schema defaults plus that story's overrides", () => {
    const defaultsContract = {
      ...storyContract,
      argsJson: '{"fields":[{"path":"title","kind":"text","default":"Default title"},{"path":"enabled","kind":"boolean","default":true}]}',
      storiesJson: '[{"id":"primary","args":{"title":"Story title"}}]',
    };
    renderWithProviders(<PropsExperimentPanel storyContract={defaultsContract} storyId="primary" initialArgs={{ title: "Story title" }} onApply={vi.fn()} onReset={vi.fn()} />);
    expect(screen.getByLabelText("title")).toHaveValue("Story title");
    expect(screen.getByLabelText("enabled")).toBeChecked();
  });

  it("offers only declared environment fixture options and applies the selected state", async () => {
    const onApply = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<PropsExperimentPanel storyContract={{ ...storyContract, environmentJson: '{"fixtures":[{"key":"voiceInput","adapter":"voice-input","options":["idle","permission-denied"]}]}' }} storyId="primary" initialEnvironment={{ voiceInput: "idle" }} onApply={onApply} onReset={vi.fn()} />);
    await user.selectOptions(screen.getByLabelText("voiceInput"), "permission-denied");
    await user.click(screen.getByTestId(selectors.components.editor.propsApply));
    expect(onApply).toHaveBeenCalledWith({ title: "Short" }, { voiceInput: "permission-denied" });
  });

  it("resets through the provided session-only callback", async () => {
    const onReset = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<PropsExperimentPanel onApply={vi.fn()} onReset={onReset} />);
    await user.click(screen.getByTestId(selectors.components.editor.propsReset));
    expect(onReset).toHaveBeenCalledTimes(1);
  });
});
