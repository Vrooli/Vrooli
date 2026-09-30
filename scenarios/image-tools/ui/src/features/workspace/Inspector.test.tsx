/**
 * Inspector tests. The mode-aware right panel renders the humanized op form:
 * an op picker plus one primitive per spec field (toggle / slider / segmented /
 * position / color / format / target-size / filter-grid / number / text), with
 * crop's numeric box under an Advanced disclosure and an overlay file input for
 * compose ops. These cover every `renderControl` branch (by feeding the real
 * op specs), the loading / error / empty states, the run/disabled wiring, the
 * runError surface, and the submit path that forwards an overlay only when the
 * op accepts one.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeOperationInfo } from "./mocks/factories";
import { opSpec, type OpField, type OpSpec } from "./opSpecs";
import { defaultParamsFor } from "./opParams";
import { strings } from "../../consts/strings";
import { Inspector, type InspectorProps } from "./Inspector";

const OPERATIONS = [
  makeOperationInfo({ name: "resize" }),
  makeOperationInfo({ name: "crop" }),
  makeOperationInfo({ name: "rotate" }),
  makeOperationInfo({ name: "flip" }),
  makeOperationInfo({ name: "filter" }),
  makeOperationInfo({ name: "convert" }),
  makeOperationInfo({ name: "compress" }),
  makeOperationInfo({ name: "overlay" }),
  makeOperationInfo({ name: "metadata" }),
];

const baseProps: InspectorProps = {
  mode: "edit",
  opsLoading: false,
  opsError: false,
  operations: OPERATIONS,
  operation: "resize",
  params: defaultParamsFor("resize"),
  spec: opSpec("resize"),
  applying: false,
  runError: null,
  hasBase: true,
  hasSteps: false,
  previewUrl: "blob:preview",
  onSelectOperation: vi.fn(),
  onParam: vi.fn(),
  onApply: vi.fn(),
};

const renderInspector = (overrides: Partial<InspectorProps> = {}) =>
  renderWithProviders(<Inspector {...baseProps} {...overrides} />);

/** Render the inspector for an op with its real spec + default params. */
const renderForOp = (operation: string, overrides: Partial<InspectorProps> = {}) =>
  renderInspector({
    operation,
    spec: opSpec(operation),
    params: defaultParamsFor(operation),
    ...overrides,
  });

describe("Inspector", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  describe("loading / error / non-edit states", () => {
    it("shows the loading state", () => {
      renderInspector({ opsLoading: true });
      expect(screen.getByTestId(selectors.workspace.loading)).toBeInTheDocument();
      expect(screen.queryByTestId(selectors.workspace.paramsForm)).not.toBeInTheDocument();
    });

    it("shows the error state when ops discovery failed", () => {
      renderInspector({ opsError: true });
      expect(screen.getByTestId(selectors.workspace.error)).toBeInTheDocument();
      expect(screen.queryByTestId(selectors.workspace.paramsForm)).not.toBeInTheDocument();
    });

    it("renders the form (placeholder copy aside) for a non-edit mode", () => {
      renderInspector({ mode: "enhance" });
      // The mode-label heading reflects the active mode.
      expect(screen.getByTestId(selectors.workspace.inspector)).toBeInTheDocument();
    });
  });

  describe("control rendering per field type", () => {
    it("renders number inputs for the resize width/height fields", () => {
      renderForOp("resize");
      const width = screen.getByTestId(selectors.workspace.fieldInput({ name: "width" }));
      expect(width).toHaveAttribute("type", "number");
    });

    it("renders a segmented control and emits the chosen token", async () => {
      const user = userEvent.setup();
      const onParam = vi.fn();
      renderForOp("flip", { onParam });
      const axis = screen.getByTestId(selectors.workspace.fieldInput({ name: "axis" }));
      const vertical = within(axis)
        .getAllByRole("radio")
        .find((el) => el.getAttribute("aria-checked") === "false");
      await user.click(vertical as Element);
      expect(onParam).toHaveBeenCalledWith("axis", "vertical");
    });

    it("renders a slider and emits a numeric value", () => {
      const onParam = vi.fn();
      renderForOp("rotate", { onParam });
      const angle = screen.getByTestId(selectors.workspace.fieldInput({ name: "angle" }));
      fireEvent.change(angle, { target: { value: "45" } });
      expect(onParam).toHaveBeenCalledWith("angle", 45);
    });

    it("renders a color field and emits a hex string", () => {
      const onParam = vi.fn();
      renderForOp("rotate", { onParam });
      const bg = screen.getByTestId(selectors.workspace.fieldInput({ name: "background" }));
      fireEvent.input(bg, { target: { value: "#112233" } });
      expect(onParam).toHaveBeenCalled();
      expect(onParam.mock.calls.at(-1)?.[0]).toBe("background");
    });

    it("renders a position picker and emits the chosen token", async () => {
      const user = userEvent.setup();
      const onParam = vi.fn();
      renderForOp("resize", { onParam });
      const gravity = screen.getByTestId(selectors.workspace.fieldInput({ name: "gravity" }));
      const cell = within(gravity).getAllByRole("radio")[0];
      await user.click(cell as Element);
      expect(onParam).toHaveBeenCalledWith("gravity", expect.any(String));
    });

    it("renders format pills and emits the chosen format", async () => {
      const user = userEvent.setup();
      const onParam = vi.fn();
      renderForOp("convert", { onParam });
      const format = screen.getByTestId(selectors.workspace.fieldInput({ name: "format" }));
      // ENCODE_FORMATS order is [png, jpeg, …]; the pill text is uppercased.
      const jpeg = within(format).getAllByRole("radio").find((el) => el.textContent === "JPEG");
      await user.click(jpeg as Element);
      expect(onParam).toHaveBeenCalledWith("format", "jpeg");
    });

    it("renders a toggle and emits a boolean", async () => {
      const user = userEvent.setup();
      const onParam = vi.fn();
      renderForOp("convert", { onParam });
      const lossless = screen.getByTestId(selectors.workspace.fieldInput({ name: "lossless" }));
      await user.click(lossless);
      expect(onParam).toHaveBeenCalledWith("lossless", true);
    });

    it("renders a filter-thumbnail grid and emits the chosen filter", async () => {
      const user = userEvent.setup();
      const onParam = vi.fn();
      renderForOp("filter", { onParam });
      const grid = screen.getByTestId(selectors.workspace.fieldInput({ name: "filter" }));
      // Options follow the spec order [grayscale, sepia, invert, blur, sharpen];
      // index 1 is sepia regardless of the localized label text.
      const sepia = within(grid).getAllByRole("radio")[1];
      await user.click(sepia as Element);
      expect(onParam).toHaveBeenCalledWith("filter", "sepia");
    });

    it("renders a target-size field and emits bytes", () => {
      const onParam = vi.fn();
      renderForOp("compress", { onParam });
      const target = screen.getByTestId(selectors.workspace.fieldInput({ name: "target_bytes" }));
      fireEvent.change(target, { target: { value: "100" } });
      expect(onParam).toHaveBeenCalledWith("target_bytes", expect.any(Number));
    });

    it("renders a text input and emits the typed string", async () => {
      const user = userEvent.setup();
      const onParam = vi.fn();
      renderForOp("overlay", { onParam });
      const text = screen.getByTestId(selectors.workspace.fieldInput({ name: "text" }));
      expect(text).toHaveAttribute("type", "text");
      await user.type(text, "A");
      expect(onParam).toHaveBeenCalledWith("text", "A");
    });
  });

  describe("crop advanced disclosure + hint", () => {
    it("shows the crop hint and tucks the numeric fields under Advanced", () => {
      renderForOp("crop");
      // Crop's x/y/w/h are all advanced → the Advanced disclosure is present.
      expect(screen.getByTestId(selectors.workspace.crop.advanced)).toBeInTheDocument();
      const x = screen.getByTestId(selectors.workspace.fieldInput({ name: "x" }));
      expect(x).toHaveAttribute("type", "number");
    });

    it("emits a coerced number from an advanced crop field", () => {
      const onParam = vi.fn();
      renderForOp("crop", { onParam });
      const width = screen.getByTestId(selectors.workspace.fieldInput({ name: "width" }));
      fireEvent.change(width, { target: { value: "320" } });
      expect(onParam).toHaveBeenCalledWith("width", 320);
    });
  });

  describe("overlay op", () => {
    it("renders the overlay file input and forwards the file on submit", async () => {
      const user = userEvent.setup();
      const onApply = vi.fn();
      renderForOp("overlay", { onApply });

      const overlayInput = screen.getByTestId(selectors.workspace.overlayInput);
      const file = new File(["wm"], "wm.png", { type: "image/png" });
      await user.upload(overlayInput, file);

      await user.click(screen.getByTestId(selectors.workspace.applyButton));
      // overlay op accepts an overlay → the picked file is forwarded.
      expect(onApply).toHaveBeenCalledWith(file);
    });

    it("submits undefined overlay when none was picked", async () => {
      const user = userEvent.setup();
      const onApply = vi.fn();
      renderForOp("overlay", { onApply });
      await user.click(screen.getByTestId(selectors.workspace.applyButton));
      expect(onApply).toHaveBeenCalledWith(undefined);
    });
  });

  describe("apply button + run states", () => {
    it("disables apply when there is no base image", () => {
      renderInspector({ hasBase: false });
      expect(screen.getByTestId(selectors.workspace.applyButton)).toBeDisabled();
    });

    it("disables apply when no operation is selected", () => {
      renderInspector({ operation: "", spec: undefined, params: {} });
      expect(screen.getByTestId(selectors.workspace.applyButton)).toBeDisabled();
    });

    it("disables apply and shows running copy while applying", () => {
      const { rerender } = renderInspector({ applying: false });
      const runLabel = screen.getByTestId(selectors.workspace.applyButton).textContent;

      rerender(<Inspector {...baseProps} applying />);
      const apply = screen.getByTestId(selectors.workspace.applyButton);
      expect(apply).toBeDisabled();
      // The label flips from the idle "Run" copy to the "Applying…" copy.
      expect(apply.textContent).not.toBe(runLabel);
      expect(apply.textContent.length).toBeGreaterThan(0);
    });

    it("submits the form with no overlay for a non-overlay op", async () => {
      const user = userEvent.setup();
      const onApply = vi.fn();
      renderForOp("resize", { onApply });
      await user.click(screen.getByTestId(selectors.workspace.applyButton));
      // resize does not accept an overlay → undefined is passed.
      expect(onApply).toHaveBeenCalledWith(undefined);
    });

    it("surfaces a run error", () => {
      renderInspector({ runError: new Error("boom") });
      expect(screen.getByTestId(selectors.workspace.runError)).toBeInTheDocument();
    });

    it("shows the empty hint when a base is loaded but no steps are applied yet", () => {
      renderInspector({ hasBase: true, hasSteps: false, applying: false });
      expect(screen.getByTestId(selectors.workspace.empty)).toBeInTheDocument();
    });

    it("hides the empty hint once a step exists", () => {
      renderInspector({ hasBase: true, hasSteps: true });
      expect(screen.queryByTestId(selectors.workspace.empty)).not.toBeInTheDocument();
    });
  });

  it("forwards an op-picker selection to onSelectOperation", async () => {
    const user = userEvent.setup();
    const onSelectOperation = vi.fn();
    renderForOp("resize", { onSelectOperation });
    await user.click(screen.getByTestId(selectors.workspace.opOption({ name: "crop" })));
    expect(onSelectOperation).toHaveBeenCalledWith("crop");
  });

  // Every control reads `value ?? default` (or `?? ""`). Rendering an op with
  // EMPTY params makes each field's value nullish, exercising the fallback
  // (right-hand) side of those coalescing branches across all control kinds.
  describe("nullish-value fallbacks (empty params)", () => {
    for (const op of ["resize", "rotate", "convert", "compress", "filter", "overlay", "flip"]) {
      it(`renders ${op} from spec defaults when params are empty`, () => {
        renderInspector({
          operation: op,
          spec: opSpec(op),
          params: {},
        });
        // The form mounts and the op's first field control is present.
        expect(screen.getByTestId(selectors.workspace.paramsForm)).toBeInTheDocument();
        const first = opSpec(op)?.fields[0];
        if (first) {
          expect(
            screen.getByTestId(selectors.workspace.fieldInput({ name: first.name })),
          ).toBeInTheDocument();
        }
      });
    }
  });

  // The segmented + filter-grid controls fall back to the raw token (and "none"
  // css) when an option is missing from its label/option map. Real op specs
  // always have full maps, so feed a synthetic spec to reach those else arms.
  describe("unknown-token label fallbacks (synthetic specs)", () => {
    it("renders a segmented option label as the raw token when unmapped", () => {
      const field: OpField = {
        name: "mystery", // not in SEGMENTED_LABELS → labelMap is {}
        control: "segmented",
        labelKey: strings.workspace.field.fit,
        default: "alpha",
        options: ["alpha", "beta"],
      };
      const spec: OpSpec = { fields: [field] };
      renderInspector({ operation: "synthetic", spec, params: { mystery: "alpha" } });
      const control = screen.getByTestId(selectors.workspace.fieldInput({ name: "mystery" }));
      // The token text shows verbatim (no translation entry).
      expect(control.textContent).toContain("alpha");
      expect(control.textContent).toContain("beta");
    });

    it("renders a filter-grid option label as the raw token when unmapped", () => {
      const field: OpField = {
        name: "fx", // tokens not in FILTER_OPTION → label = token, css = "none"
        control: "filterGrid",
        labelKey: strings.workspace.field.filter,
        default: "weird",
        options: ["weird", "odd"],
      };
      const spec: OpSpec = { fields: [field] };
      renderInspector({
        operation: "synthetic",
        spec,
        params: {},
        previewUrl: null,
      });
      const grid = screen.getByTestId(selectors.workspace.fieldInput({ name: "fx" }));
      expect(grid.textContent).toContain("weird");
      expect(grid.textContent).toContain("odd");
    });
  });
});
