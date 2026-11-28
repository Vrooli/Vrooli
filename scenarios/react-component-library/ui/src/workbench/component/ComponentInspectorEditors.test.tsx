import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import en from "../../i18n/locales/en.json";
import { renderWithProviders } from "../../test-utils";
import type { WorkbenchStory } from "../data";
import { PropsEditor } from "./ComponentInspectorEditors";

const story: WorkbenchStory = {
  key: "button@1.0.0:default",
  id: "default",
  name: "Default",
  role: "anatomy",
  args: { disabled: false, label: "Continue" },
  fieldsJson: JSON.stringify({
    fields: [
      { path: "disabled", kind: "boolean" },
      { path: "label", kind: "text" },
      { path: "appearance.size", kind: "enum", options: ["small", "large"] },
    ],
  }),
  version: "1.0.0",
  sourcePath: "components/Button/versions/1.0.0/story.json",
  contractId: "button@1.0.0",
};

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("PropsEditor", () => {
  it("creates controls from the declared schema and applies valid overrides", () => {
    const onApply = vi.fn();
    const onReset = vi.fn();
    renderWithProviders(
      <PropsEditor
        story={{ ...story, args: { ...story.args, appearance: { size: "small" } } }}
        override={null}
        onApply={onApply}
        onReset={onReset}
        status="ready"
      />,
    );

    expect(screen.getByLabelText(/^disabled$/)).toHaveValue("false");
    expect(screen.getByLabelText(/^label$/)).toHaveValue("Continue");
    expect(screen.getByLabelText(/^appearance\.size$/)).toHaveValue('"small"');
    expect(screen.queryByLabelText(en.workbench.component.propsJson)).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText(/^disabled$/), { target: { value: "true" } });
    fireEvent.change(screen.getByLabelText(/^appearance\.size$/), { target: { value: '"large"' } });
    fireEvent.click(screen.getByRole("button", { name: "Apply props" }));

    expect(onApply).toHaveBeenCalledWith({
      disabled: true,
      label: "Continue",
      appearance: { size: "large" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Reset" }));
    expect(onReset).toHaveBeenCalledTimes(1);
  });

  it("rejects malformed override JSON without applying a new specimen", () => {
    const onApply = vi.fn();
    renderWithProviders(
      <PropsEditor
        story={story}
        override={null}
        onApply={onApply}
        onReset={vi.fn()}
        status="ready"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Edit JSON" }));
    fireEvent.change(screen.getByLabelText(en.workbench.component.propsJson), {
      target: { value: '{"disabled":' },
    });
    fireEvent.click(screen.getByRole("button", { name: "Apply props" }));

    expect(screen.getByText(en.workbench.component.propsParse)).toBeInTheDocument();
    expect(onApply).not.toHaveBeenCalled();
  });

  it("rejects valid JSON that is not an object", () => {
    const onApply = vi.fn();
    renderWithProviders(
      <PropsEditor
        story={story}
        override={null}
        onApply={onApply}
        onReset={vi.fn()}
        status="ready"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Edit JSON" }));
    fireEvent.change(screen.getByLabelText(en.workbench.component.propsJson), {
      target: { value: "[]" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Apply props" }));

    expect(screen.getByText(en.workbench.component.propsObject)).toBeInTheDocument();
    expect(onApply).not.toHaveBeenCalled();
  });

  it("uses inferred controls when fields are absent and handles primitive prop values", () => {
    const onApply = vi.fn();
    renderWithProviders(
      <PropsEditor
        story={{ ...story, args: { count: 3, enabled: true, note: null }, fieldsJson: "{}" }}
        override={null}
        onApply={onApply}
        onReset={vi.fn()}
        status="ready"
      />,
    );

    expect(screen.getByLabelText(/^count$/)).toHaveValue(3);
    expect(screen.getByLabelText(/^enabled$/)).toHaveValue("true");
    expect(screen.getByLabelText(/^note$/)).toHaveValue("");
    fireEvent.change(screen.getByLabelText(/^count$/), { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: "Apply props" }));

    expect(onApply).toHaveBeenCalledWith({ count: "", enabled: true, note: null });
  });

  it("shows the empty state for invalid schemas and stories without props", () => {
    renderWithProviders(
      <PropsEditor
        story={{ ...story, args: {}, fieldsJson: '{"fields":[null,{}, {"path":""}]}' }}
        override={null}
        onApply={vi.fn()}
        onReset={vi.fn()}
        status="ready"
      />,
    );

    expect(screen.getByText(en.workbench.component.noProps)).toBeInTheDocument();
  });

  it("includes displayed schema defaults when editing another nested property", () => {
    const onApply = vi.fn();
    renderWithProviders(
      <PropsEditor
        story={{
          ...story,
          args: {},
          fieldsJson: JSON.stringify({
            fields: [
              { path: "appearance.size", kind: "enum", options: ["small", "large"] },
              { path: "appearance.rounded", kind: "boolean", default: true },
            ],
          }),
        }}
        override={null}
        onApply={onApply}
        onReset={vi.fn()}
        status="ready"
      />,
    );

    expect(screen.getByLabelText(/^appearance\.size$/)).toHaveValue('"small"');
    expect(screen.getByLabelText(/^appearance\.rounded$/)).toHaveValue("true");
    fireEvent.change(screen.getByLabelText(/^appearance\.rounded$/), {
      target: { value: "false" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Apply props" }));

    expect(onApply).toHaveBeenCalledWith({ appearance: { size: "small", rounded: false } });
  });

  it("recovers from invalid JSON by retaining the latest valid values when a control changes", () => {
    const onApply = vi.fn();
    renderWithProviders(
      <PropsEditor
        story={story}
        override={null}
        onApply={onApply}
        onReset={vi.fn()}
        status="ready"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Edit JSON" }));
    fireEvent.change(screen.getByLabelText(en.workbench.component.propsJson), {
      target: { value: '{"disabled":' },
    });
    fireEvent.change(screen.getByLabelText(/^label$/), { target: { value: "Next" } });
    fireEvent.click(screen.getByRole("button", { name: "Apply props" }));

    expect(onApply).toHaveBeenCalledWith({
      disabled: false,
      label: "Next",
      appearance: { size: "small" },
    });
    expect(screen.queryByText(en.workbench.component.propsParse)).not.toBeInTheDocument();
  });

  it("resets the JSON editor when the selected story or override changes", () => {
    const { rerender } = renderWithProviders(
      <PropsEditor
        story={story}
        override={null}
        onApply={vi.fn()}
        onReset={vi.fn()}
        status="ready"
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Edit JSON" }));
    fireEvent.change(screen.getByLabelText(en.workbench.component.propsJson), {
      target: { value: "{}" },
    });

    rerender(
      <PropsEditor
        story={{ ...story, id: "alternate", args: { label: "Alternate" } }}
        override={{ label: "Override" }}
        onApply={vi.fn()}
        onReset={vi.fn()}
        status="ready"
      />,
    );

    expect(screen.queryByLabelText(en.workbench.component.propsJson)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Edit JSON" }));
    expect(screen.getByLabelText(en.workbench.component.propsJson)).toHaveValue(
      JSON.stringify({ label: "Override", appearance: { size: "small" } }, null, 2),
    );
  });
});
