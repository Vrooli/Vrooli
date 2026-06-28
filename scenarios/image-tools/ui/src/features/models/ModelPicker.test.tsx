/**
 * ModelPicker tests — the host-aware model menu behind every AI action. The
 * `UseModelPicker` lifecycle is a hand-built fake, so the overlay's row
 * rendering, per-ready-state action button, host line, manual-steps expander,
 * footer, and select→onSelect+onClose wiring are exercised in isolation (the
 * lifecycle itself is covered by useModelPicker.test). Queries are testId-only
 * (eslint forbids string-literal Testing Library queries).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeBackendReadiness, makeCandidateModel, makeHostSummary, makeModel } from "./mocks/factories";
import { ModelPicker } from "./ModelPicker";
import type { UseModelPicker } from "./useModelPicker";

const fakePicker = (overrides: Partial<UseModelPicker> = {}): UseModelPicker => ({
  operation: "upscale",
  candidates: [],
  host: makeHostSummary(),
  selectedId: "",
  selectedReason: "",
  loading: false,
  error: null,
  busyId: "",
  rowError: {},
  refresh: vi.fn(),
  installModel: vi.fn(),
  installBackend: vi.fn(),
  enable: vi.fn(),
  ...overrides,
});

const candidate = (id: string, readyState: string, extra: Parameters<typeof makeCandidateModel>[0] = {}) =>
  makeCandidateModel({
    model: makeModel({ id, name: `Model ${id}`, tier: "default" }),
    readyState,
    ...extra,
  });

const renderPicker = (picker: UseModelPicker, props: Partial<Parameters<typeof ModelPicker>[0]> = {}) =>
  renderWithProviders(
    <ModelPicker
      open
      onClose={props.onClose ?? vi.fn()}
      operation="upscale"
      operationLabel="Upscale"
      picker={picker}
      value={props.value ?? ""}
      onSelect={props.onSelect ?? vi.fn()}
    />,
  );

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ModelPicker", () => {
  it("renders nothing when closed", () => {
    renderWithProviders(
      <ModelPicker
        open={false}
        onClose={vi.fn()}
        operation="upscale"
        operationLabel="Upscale"
        picker={fakePicker()}
        value=""
        onSelect={vi.fn()}
      />,
    );
    expect(screen.queryByTestId(selectors.models.picker.sheet)).not.toBeInTheDocument();
  });

  it("renders a row per candidate and the host hardware line", () => {
    renderPicker(fakePicker({ candidates: [candidate("a", "ready"), candidate("b", "disabled")] }));
    expect(screen.getByTestId(selectors.models.pickerRow({ id: "a" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.models.pickerRow({ id: "b" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.models.picker.host)).toBeInTheDocument();
  });

  it("shows the via-workflow support chip + caveat banner for a derived candidate", () => {
    renderPicker(
      fakePicker({
        operation: "inpaint",
        candidates: [
          candidate("base-sdxl", "derived_pipeline_unproven", {
            support: "derived",
            technique: "diffusers-inpaint",
            caveat: "derived: a base checkpoint inpaints via the standard pipeline",
          }),
        ],
      }),
    );
    const banner = screen.getByTestId(selectors.models.pickerCaveat({ id: "base-sdxl" }));
    expect(banner).toHaveTextContent("standard pipeline");
    // The unproven derived row is shown but offers no select button.
    expect(screen.queryByTestId(selectors.models.pickerSelect({ id: "base-sdxl" }))).not.toBeInTheDocument();
  });

  it("labels a proven derived candidate's select button 'Use anyway' (an informed opt-in past the native default)", () => {
    renderPicker(
      fakePicker({
        operation: "inpaint",
        candidates: [
          candidate("base-sdxl", "ready", {
            support: "derived",
            technique: "diffusers-inpaint",
            caveat: "derived: a base checkpoint inpaints via the standard pipeline",
          }),
        ],
      }),
    );
    const select = screen.getByTestId(selectors.models.pickerSelect({ id: "base-sdxl" }));
    expect(select).toHaveTextContent("Use anyway");
  });

  it("shows the loading state while the first load is in flight", () => {
    renderPicker(fakePicker({ loading: true, candidates: [] }));
    expect(screen.getByTestId(selectors.models.picker.loading)).toBeInTheDocument();
  });

  it("shows the error state when the load failed", () => {
    renderPicker(fakePicker({ error: "models down", candidates: [] }));
    expect(screen.getByTestId(selectors.models.picker.error)).toHaveTextContent("models down");
  });

  it("offers a Select button for a ready row and calls onSelect + onClose on click", async () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderPicker(fakePicker({ candidates: [candidate("a", "ready")] }), { onSelect, onClose });

    await user.click(screen.getByTestId(selectors.models.pickerSelect({ id: "a" })));
    expect(onSelect).toHaveBeenCalledWith("a");
    expect(onClose).toHaveBeenCalled();
  });

  it("marks the active row as in-use instead of offering a select button", () => {
    renderPicker(fakePicker({ candidates: [candidate("a", "ready")], selectedId: "a" }));
    expect(screen.getByTestId(selectors.models.pickerInUse({ id: "a" }))).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.models.pickerSelect({ id: "a" }))).not.toBeInTheDocument();
  });

  it("honors an explicit value override for the active row", () => {
    renderPicker(fakePicker({ candidates: [candidate("a", "ready")], selectedId: "b" }), { value: "a" });
    expect(screen.getByTestId(selectors.models.pickerInUse({ id: "a" }))).toBeInTheDocument();
  });

  it("offers an install-model button that drives installModel", async () => {
    const installModel = vi.fn();
    const user = userEvent.setup();
    renderPicker(fakePicker({ candidates: [candidate("a", "needs_model_install")], installModel }));
    await user.click(screen.getByTestId(selectors.models.pickerInstallModel({ id: "a" })));
    expect(installModel).toHaveBeenCalledWith("a");
  });

  it("offers an install-backend button that drives installBackend with the host tool", async () => {
    const installBackend = vi.fn();
    const user = userEvent.setup();
    renderPicker(
      fakePicker({
        candidates: [
          candidate("a", "needs_backend", {
            backend: makeBackendReadiness({ hostTool: "realesrgan-ncnn-vulkan" }),
          }),
        ],
        installBackend,
      }),
    );
    await user.click(screen.getByTestId(selectors.models.pickerInstallBackend({ id: "a" })));
    expect(installBackend).toHaveBeenCalledWith("realesrgan-ncnn-vulkan", "a");
  });

  it("offers an enable button for a disabled model that drives enable", async () => {
    const enable = vi.fn();
    const user = userEvent.setup();
    renderPicker(fakePicker({ candidates: [candidate("a", "disabled")], enable }));
    await user.click(screen.getByTestId(selectors.models.pickerEnable({ id: "a" })));
    expect(enable).toHaveBeenCalledWith("a");
  });

  it("offers a manual-steps toggle that expands the manual hint panel", async () => {
    const user = userEvent.setup();
    renderPicker(
      fakePicker({
        candidates: [
          candidate("a", "needs_backend_manual", {
            backend: makeBackendReadiness({ manualHint: "brew install realesrgan" }),
          }),
        ],
      }),
    );
    expect(screen.queryByTestId(selectors.models.pickerManual({ id: "a" }))).not.toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.models.pickerManualToggle({ id: "a" })));
    const manual = screen.getByTestId(selectors.models.pickerManual({ id: "a" }));
    expect(manual).toHaveTextContent("brew install realesrgan");
  });

  it("renders a secondary install-backend affordance for a needs_both auto candidate", () => {
    renderPicker(
      fakePicker({
        candidates: [
          candidate("a", "needs_both", {
            backend: makeBackendReadiness({ hostTool: "sd-tool", installTier: "auto" }),
          }),
        ],
      }),
    );
    // The primary action is install-model; the secondary is the backend install.
    expect(screen.getByTestId(selectors.models.pickerInstallModel({ id: "a" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.models.pickerInstallBackend({ id: "a" }))).toBeInTheDocument();
  });

  it("shows a busy spinner row (no action button) while a row install is in flight", () => {
    renderPicker(
      fakePicker({ candidates: [candidate("a", "needs_model_install")], busyId: "a" }),
    );
    expect(screen.queryByTestId(selectors.models.pickerInstallModel({ id: "a" }))).not.toBeInTheDocument();
  });

  it("surfaces a per-row error message", () => {
    renderPicker(
      fakePicker({
        candidates: [candidate("a", "needs_model_install")],
        rowError: { a: "disk full" },
      }),
    );
    expect(screen.getByTestId(selectors.models.pickerRowError({ id: "a" }))).toHaveTextContent(
      "disk full",
    );
  });

  it("renders no action button for an insufficient (un-runnable) candidate but keeps the row", () => {
    renderPicker(fakePicker({ candidates: [candidate("a", "insufficient")] }));
    const row = screen.getByTestId(selectors.models.pickerRow({ id: "a" }));
    expect(within(row).queryByTestId(selectors.models.pickerSelect({ id: "a" }))).not.toBeInTheDocument();
    expect(within(row).queryByTestId(selectors.models.pickerEnable({ id: "a" }))).not.toBeInTheDocument();
  });

  it("renders a 'no alternatives' footer for a single candidate and a count footer for many", () => {
    const { unmount } = renderPicker(fakePicker({ candidates: [candidate("a", "ready")] }));
    expect(screen.getByTestId(selectors.models.picker.footer)).toBeInTheDocument();
    unmount();

    renderPicker(fakePicker({ candidates: [candidate("a", "ready"), candidate("b", "ready")] }));
    expect(screen.getByTestId(selectors.models.picker.footer)).toBeInTheDocument();
  });
});
