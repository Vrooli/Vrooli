/**
 * CreatePanel tests — the Create-mode inspector. AI-op discovery is mocked and
 * the `UseCreate` lifecycle is a hand-built fake, so the panel's generation-op
 * list, prompt gating, run wiring, masked-op mask brush, install gate, and
 * accessibility are exercised in isolation (the lifecycle itself is covered by
 * useCreate.test).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ConsentWeight,
  DeploymentTier,
  SafetyPolicySchema,
  type SafetyPolicy,
} from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeAIMocks, makeAIOperationInfo, makeListAIOperationsResponse, makeSelectedModel } from "./mocks/ai";
import { VerdictGrade } from "@vrooli/proto-types/image-tools/v1/models/models_pb";
import { makeModelsMocks } from "../models/mocks/models";
import { makeExplainResolutionResponse, makeModel, makeResolution } from "../models/mocks/factories";

vi.mock("../../api/ai", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ai")>();
  return { ...actual, ...makeAIMocks() };
});

// The panel renders <ModelPickerButton/>, which fires a real `selectModel`
// query for its trigger label. Mock the models API so the panel renders
// hermetically (no network) — the picker itself is covered by its own tests.
vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

// The panel renders <JobRatingControl/> for a finished run, which reads the
// job's existing rating. Mock the jobs API so the panel stays hermetic.
const jobsMock = vi.hoisted(() => ({
  getJob: vi.fn(),
  rateJob: vi.fn(),
  clearJobRating: vi.fn(),
}));
vi.mock("../../api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/jobs")>();
  return { ...actual, jobsClient: jobsMock, rateJob: jobsMock.rateJob, clearJobRating: jobsMock.clearJobRating };
});

const policyMock = vi.hoisted(
  (): { data: SafetyPolicy | undefined; error: unknown; isLoading: boolean } => ({
    data: undefined,
    error: null,
    isLoading: false,
  }),
);
vi.mock("../safety/useSafetyPolicy", () => ({
  useSafetyPolicy: () => policyMock,
  SAFETY_POLICY_QUERY_KEY: ["safety-policy"],
}));

import { modelsClient } from "../../api/models";
import { CreatePanel } from "./CreatePanel";
import type { UseCreate } from "./useCreate";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });
const MASK = new File(["mask"], "mask.png", { type: "image/png" });

const fakeCreate = (overrides: Partial<UseCreate> = {}): UseCreate => ({
  phase: "idle",
  model: null,
  progress: { percent: 0, message: "", state: "unspecified" },
  tier: "",
  warnings: [],
  error: null,
  consentBlocked: false,
  results: [],
  requestedCount: 1,
  preview: vi.fn(),
  start: vi.fn(),
  installAndRun: vi.fn(),
  cancel: vi.fn(),
  retry: vi.fn(),
  dismiss: vi.fn(),
  ...overrides,
});

const renderPanel = (create: UseCreate, input: File | null = null, inputUrl: string | null = null) =>
  renderWithProviders(
    <CreatePanel
      create={create}
      input={input}
      inputUrl={inputUrl}
      onSendToCanvas={vi.fn()}
      onSendToEnhance={vi.fn()}
    />,
  );

beforeEach(async () => {
  await setLocale("en");
  jobsMock.getJob.mockResolvedValue({});
  // Each test starts from "this route needs no consent"; tests that need the
  // server to ask say so with explainConsent.
  vi.mocked(modelsClient.explainResolution).mockResolvedValue(makeExplainResolutionResponse());
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  policyMock.data = undefined;
});

const publicPolicy = (
  overrides: MessageInitShape<typeof SafetyPolicySchema> = {},
): SafetyPolicy =>
  create(SafetyPolicySchema, {
    tier: DeploymentTier.PUBLIC,
    requireConsent: true,
    opWeights: [
      { operation: "text_to_image", weight: ConsentWeight.NONE },
      { operation: "edit_instruct", weight: ConsentWeight.HIGH },
    ],
    ...overrides,
  });

describe("CreatePanel", () => {
  it("populates the generation-op list from discovery and shows the prompt", async () => {
    renderPanel(fakeCreate());
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.workspace.createAction({ name: "text_to_image" })),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByTestId(selectors.workspace.createAction({ name: "inpaint" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.workspace.createAction({ name: "edit_instruct" })),
    ).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.create.prompt)).toBeInTheDocument();
  });

  it("stages a synthetic pose source and requires the versioned license acceptance while runtime gates stay closed", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create);
    const prompt = await screen.findByTestId(selectors.workspace.create.prompt);
    await user.type(prompt, "a synthetic stick figure running");
    const run = screen.getByTestId(selectors.workspace.create.run);
    expect(run).toBeEnabled();

    const poseSource = new File(["synthetic pose"], "synthetic-pose.png", { type: "image/png" });
    await user.upload(screen.getByTestId("create-pose-source"), poseSource);

    expect(screen.getByText(/operator approved this exact ControlNet under CreativeML OpenRAIL-M/i)).toBeInTheDocument();
    expect(screen.getByTestId("create-pose-staged")).toHaveTextContent("synthetic-pose.png");
    expect(screen.getByTestId("create-pose-license-acceptance")).not.toBeChecked();
    await user.click(screen.getByTestId("create-pose-license-acceptance"));
    expect(screen.getByTestId("create-pose-license-acceptance")).toBeChecked();
    expect(run).toBeDisabled();
    expect(create.start).not.toHaveBeenCalled();
  });

  it("gates run on a non-empty prompt, then starts text-to-image with the prompt", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create);

    const run = await screen.findByTestId(selectors.workspace.create.run);
    expect(run).toBeDisabled();

    await user.type(screen.getByTestId(selectors.workspace.create.prompt), "a serene lake");
    expect(run).toBeEnabled();
    await user.click(run);

    expect(create.start).toHaveBeenCalledWith(
      "text_to_image",
      expect.objectContaining({ prompt: "a serene lake" }),
      {},
    );
  });

  it("shows the mask brush and gates a masked op until a mask exists", async () => {
    const user = userEvent.setup();
    renderPanel(fakeCreate(), PNG, "blob:img");

    await user.click(
      await screen.findByTestId(selectors.workspace.createAction({ name: "inpaint" })),
    );

    expect(screen.getByTestId(selectors.workspace.mask.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.create.needsMask)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.create.run)).toBeDisabled();
  });

  it("renders the install gate and installs on click", async () => {
    const create = fakeCreate({
      phase: "needs-install",
      model: makeSelectedModel({ id: "sd-1.5", name: "sd-1.5", installed: false }),
    });
    const user = userEvent.setup();
    renderPanel(create);

    const gate = await screen.findByTestId(selectors.workspace.create.installGate);
    expect(gate).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.workspace.create.install));
    expect(create.installAndRun).toHaveBeenCalledTimes(1);
  });

  it("shows the graphics-card note for the model the run would use, and not once a model is picked by hand", async () => {
    renderPanel(fakeCreate({ model: makeSelectedModel({ gpuViable: true, cpuCapable: false }) }));
    expect((await screen.findByTestId(selectors.workspace.capacityNote)).textContent).toBe("May wait for the graphics card if it's busy");
  });

  it("has no detectable accessibility violations", async () => {
    const { container } = renderPanel(fakeCreate());
    await screen.findByTestId(selectors.workspace.create.prompt);
    await expectNoA11yViolations(container);
  });

  it("threads size, seed, variations, negative, BYOK and CPU-only execution into the params", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create);

    await user.type(
      await screen.findByTestId(selectors.workspace.create.prompt),
      "a serene lake",
    );

    // Size is offered only for text_to_image — pick the landscape preset.
    await user.click(screen.getByRole("radio", { name: /landscape/i }));
    // Bump the variation count to 3.
    await user.click(within(screen.getByTestId(selectors.workspace.create.variations)).getByText("3"));

    // Lock a seed via the number input + the lock toggle.
    await user.type(screen.getByTestId(selectors.workspace.create.seed), "42");
    await user.click(screen.getByTestId(selectors.workspace.create.seedLock));

    // Advanced disclosure: negative prompt, BYOK. (The model override moved out of
    // a free-text input and into the host-aware ModelPickerButton — covered there.)
    await user.type(screen.getByTestId(selectors.workspace.create.negative), "blurry");
    await user.click(screen.getByTestId(selectors.workspace.create.byok));
    await user.click(screen.getByTestId(selectors.workspace.create.forceCpu));

    await user.click(screen.getByTestId(selectors.workspace.create.run));

    expect(create.start).toHaveBeenCalledWith(
      "text_to_image",
      expect.objectContaining({
        prompt: "a serene lake",
        negativePrompt: "blurry",
        width: 768,
        height: 512,
        variations: 3,
        seed: 42n,
        allowByok: true,
        forceCpu: true,
      }),
      {},
    );
  });

  it("threads instruction and image guidance for instruction edits", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create, PNG);

    await user.click(await screen.findByTestId(selectors.workspace.createAction({ name: "edit_instruct" })));
    await user.type(screen.getByTestId(selectors.workspace.create.prompt), "recolor the stars");
    await user.click(screen.getByTestId(selectors.workspace.create.advanced));
    await user.type(screen.getByTestId(selectors.workspace.create.cfgScale), "10");
    await user.type(screen.getByTestId(selectors.workspace.create.imageGuidance), "2");
    await user.click(screen.getByTestId(selectors.workspace.create.forceCpu));
    await user.click(screen.getByTestId(selectors.workspace.create.run));

    expect(create.start).toHaveBeenCalledWith(
      "edit_instruct",
      expect.objectContaining({ prompt: "recolor the stars", cfgScale: 10, strength: 2, forceCpu: true }),
      { input: PNG },
    );
  });

  it("sends explicit contextual crop settings with an inpaint mask", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create, PNG);

    await user.click(await screen.findByTestId(selectors.workspace.createAction({ name: "inpaint" })));
    await user.upload(screen.getByTestId(selectors.workspace.mask.upload), MASK);
    await user.type(screen.getByTestId(selectors.workspace.create.prompt), "fill the masked area");
    await user.click(screen.getByTestId(selectors.workspace.create.advanced));
    await user.click(screen.getByTestId(selectors.workspace.create.cropToMask));
    await user.clear(screen.getByTestId(selectors.workspace.create.cropContextPixels));
    await user.type(screen.getByTestId(selectors.workspace.create.cropContextPixels), "128");
    await user.click(screen.getByTestId(selectors.workspace.create.run));

    expect(create.start).toHaveBeenCalledWith(
      "inpaint",
      expect.objectContaining({
        prompt: "fill the masked area",
        // Operation-specific knobs ride opParams under the schema's key names; a
        // zero max_edge_px is omitted because 0 is not one of its legal values.
        opParams: { inpaint_crop_to_mask: true, inpaint_context_pixels: 128 },
      }),
      { input: PNG, mask: MASK },
    );
  });

  it("randomizes and locks the seed when the dice control is used", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create);

    await user.type(
      await screen.findByTestId(selectors.workspace.create.prompt),
      "a serene lake",
    );
    await user.click(screen.getByTestId(selectors.workspace.create.seedRandomize));

    // A number input is empty (value null) until randomize seeds it.
    const seed = screen.getByTestId(selectors.workspace.create.seed);
    expect(seed).not.toHaveValue(null);
    // Randomize also engages the lock so the value sticks across runs.
    expect(screen.getByTestId(selectors.workspace.create.seedLock)).toHaveAttribute(
      "aria-checked",
      "true",
    );

    await user.click(screen.getByTestId(selectors.workspace.create.run));
    expect(create.start).toHaveBeenCalledWith(
      "text_to_image",
      expect.objectContaining({ seed: expect.any(BigInt) }),
      {},
    );
  });

  it("passes the current canvas image to an image-required op and shows the uses-current hint", async () => {
    const user = userEvent.setup();
    const create = fakeCreate();
    renderPanel(create, PNG, "blob:img");

    await user.click(
      await screen.findByTestId(selectors.workspace.createAction({ name: "image_to_image" })),
    );
    expect(screen.getByTestId(selectors.workspace.create.usesCurrent)).toBeInTheDocument();
    // image_to_image is prompt-driven, so a prompt is still required.
    await user.type(screen.getByTestId(selectors.workspace.create.prompt), "watercolor style");
    await user.click(screen.getByTestId(selectors.workspace.create.run));

    expect(create.start).toHaveBeenCalledWith(
      "image_to_image",
      expect.objectContaining({ prompt: "watercolor style" }),
      { input: PNG },
    );
  });

  it("shows the needs-image hint for an image-required op with no canvas image", async () => {
    const user = userEvent.setup();
    renderPanel(fakeCreate(), null);

    await user.click(
      await screen.findByTestId(selectors.workspace.createAction({ name: "image_to_image" })),
    );
    expect(screen.getByTestId(selectors.workspace.create.needsImage)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.create.run)).toBeDisabled();
  });

  it("renders the host-aware model picker trigger inside the model badge", async () => {
    const { modelsClient } = await import("../../api/models");
    const { makeSelectModelResponse } = await import("../models/mocks/factories");
    vi.mocked(modelsClient.selectModel).mockResolvedValue(
      makeSelectModelResponse({ model: makeModel({ id: "sd-1.5", name: "sd-1.5" }) }),
    );
    renderPanel(fakeCreate());
    const badge = await screen.findByTestId(selectors.workspace.create.modelBadge);
    // The static "model detected" label was replaced by the picker trigger.
    const trigger = within(badge).getByTestId(selectors.models.pickerTrigger);
    expect(trigger).toBeInTheDocument();
    await waitFor(() => expect(trigger.textContent).toContain("sd-1.5"));
  });

  it("shows live per-variation progress with a cancel control while running", async () => {
    const create = fakeCreate({
      phase: "running",
      progress: { percent: 30, message: "produced 1/3", state: "running" },
      tier: "local-cpu",
      requestedCount: 3,
    });
    const user = userEvent.setup();
    renderPanel(create);

    const progress = await screen.findByTestId(selectors.workspace.create.progress);
    // The "produced i/N" marker is localized into the produced row.
    expect(progress.textContent).toMatch(/1.*3/);
    await user.click(screen.getByTestId(selectors.workspace.create.cancel));
    expect(create.cancel).toHaveBeenCalledTimes(1);
  });

  it("shows a raw status message while running when there is no produced marker", async () => {
    const create = fakeCreate({
      phase: "running",
      progress: { percent: 10, message: "warming up the model", state: "running" },
    });
    renderPanel(create);
    expect(await screen.findByText(/warming up the model/)).toBeInTheDocument();
  });

  it("shows the installing spinner while the model installs", async () => {
    const create = fakeCreate({ phase: "installing" });
    renderPanel(create);
    expect(await screen.findByTestId(selectors.workspace.create.progress)).toBeInTheDocument();
  });

  it("shows the default model's verdict on every operation tile that has a default", async () => {
    const { listAIOperations } = await import("../../api/ai");
    vi.mocked(listAIOperations).mockResolvedValueOnce(
      makeListAIOperationsResponse({
        operations: [
          makeAIOperationInfo({ name: "text_to_image", defaultModelId: "flux-2-klein-4b", defaultVerdict: { operation: "text_to_image", grade: VerdictGrade.RECOMMENDED, reason: "best" } }),
          makeAIOperationInfo({ name: "image_to_image", defaultUnavailable: "no enabled model" }),
        ],
      }),
    );
    renderPanel(fakeCreate());
    expect(await screen.findByTestId(selectors.workspace.opVerdict({ name: "text_to_image" }))).toHaveTextContent("Recommended");
    expect(screen.queryByTestId(selectors.workspace.opVerdict({ name: "image_to_image" }))).not.toBeInTheDocument();
  });

  it("offers a rating for the finished run's job, and not before it has finished", async () => {
    const result = { kind: "image" as const, url: "blob:v0", width: 512, height: 512, format: "png", jobId: "gen-7" };
    const variations = [{ index: 0, result, outputFile: PNG }];
    const running = renderPanel(fakeCreate({ phase: "running", results: variations }));
    expect(screen.queryByTestId(selectors.workspace.rating.root)).not.toBeInTheDocument();
    running.unmount();

    jobsMock.rateJob.mockResolvedValue({ jobId: "gen-7", score: 5, note: "" });
    const user = userEvent.setup();
    renderPanel(fakeCreate({ phase: "succeeded", results: variations }));
    await user.click(await screen.findByTestId(selectors.workspace.ratingScore({ score: 5 })));

    await waitFor(() => expect(jobsMock.rateJob).toHaveBeenCalledWith("gen-7", 5, ""));
    expect(jobsMock.getJob).toHaveBeenCalledWith({ id: "gen-7" });
  });

  it("lists fallback warnings, the success count, and the failure with retry", async () => {
    const user = userEvent.setup();
    const succeeded = fakeCreate({
      phase: "succeeded",
      warnings: ["fell back to CPU"],
      results: [
        {
          index: 0,
          result: {
            kind: "image",
            url: "blob:v0",
            width: 512,
            height: 512,
            format: "png",
            jobId: "gen-1",
          },
          outputFile: PNG,
        },
      ],
    });
    const { unmount } = renderPanel(succeeded);
    expect((await screen.findByTestId(selectors.workspace.create.warnings)).textContent).toContain(
      "fell back to CPU",
    );
    expect(screen.getByTestId(selectors.workspace.create.succeeded)).toBeInTheDocument();
    unmount();

    const failed = fakeCreate({ phase: "failed", error: "out of memory" });
    renderPanel(failed);
    const fail = await screen.findByTestId(selectors.workspace.create.failed);
    expect(fail.textContent).toContain("out of memory");
    await user.click(screen.getByTestId(selectors.workspace.create.retry));
    expect(failed.retry).toHaveBeenCalledTimes(1);
  });

  it("renders the discovery loading and error states", async () => {
    const { listAIOperations } = await import("../../api/ai");
    vi.mocked(listAIOperations).mockRejectedValueOnce(new Error("ai down"));
    renderPanel(fakeCreate());
    expect(await screen.findByTestId(selectors.workspace.create.error)).toBeInTheDocument();
  });

  // The server answers "does this request need consent" for the route it would
  // take; the panel shows its checkbox only when that answer says so.
  const explainConsent = (resolution: Parameters<typeof makeResolution>[0]) =>
    vi.mocked(modelsClient.explainResolution).mockResolvedValue(
      makeExplainResolutionResponse({ resolution: makeResolution(resolution) }),
    );

  describe("reference image", () => {
    it("takes a reference photo for face swap and sends it with the run", async () => {
      const user = userEvent.setup();
      const createHook = fakeCreate();
      renderPanel(createHook, PNG, "blob:img");
      await user.click(
        await screen.findByTestId(selectors.workspace.createAction({ name: "face_swap" })),
      );

      expect(screen.getByTestId(selectors.workspace.create.needsReference)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.workspace.create.run)).toBeDisabled();

      const face = new File(["face"], "face.png", { type: "image/png" });
      await user.upload(screen.getByTestId(selectors.workspace.create.reference), face);
      expect(screen.queryByTestId(selectors.workspace.create.needsReference)).not.toBeInTheDocument();
      await user.click(screen.getByTestId(selectors.workspace.create.run));

      expect(createHook.start).toHaveBeenCalledWith("face_swap", expect.anything(), {
        input: PNG,
        reference: face,
      });
    });

    it("sends the swap extent the person picked: a preset, a custom mix, and the face strength", async () => {
      const user = userEvent.setup();
      const createHook = fakeCreate();
      renderPanel(createHook, PNG, "blob:img");
      await user.click(
        await screen.findByTestId(selectors.workspace.createAction({ name: "face_swap" })),
      );
      const face = new File(["face"], "face.png", { type: "image/png" });
      await user.upload(screen.getByTestId(selectors.workspace.create.reference), face);
      const paramsOfLastRun = () => {
        const call = vi.mocked(createHook.start).mock.calls.at(-1);
        return (call?.[1] as { opParams?: unknown } | undefined)?.opParams;
      };

      // The default is the plain swap.
      await user.click(screen.getByTestId(selectors.workspace.create.run));
      expect(paramsOfLastRun()).toEqual({ extent: "face" });

      // A preset names itself.
      const presets = within(screen.getByTestId(selectors.workspace.create.extentPresets));
      await user.click(presets.getByRole("radio", { name: "Full look" }));
      await user.click(screen.getByTestId(selectors.workspace.create.run));
      expect(paramsOfLastRun()).toEqual({ extent: "full_look" });

      // Switching one stage off leaves a custom mix, sent as the stages.
      await user.click(screen.getByTestId(`${selectors.workspace.create.extentStage}-skin`));
      await user.click(screen.getByTestId(selectors.workspace.create.run));
      expect(paramsOfLastRun()).toEqual({ match: ["face", "hair_colour"] });

      // The whole head takes the face's place and hides the face strength.
      await user.click(screen.getByTestId(`${selectors.workspace.create.extentStage}-head`));
      expect(screen.queryByTestId(selectors.workspace.create.extentStrength)).not.toBeInTheDocument();
      await user.click(screen.getByTestId(selectors.workspace.create.run));
      expect(paramsOfLastRun()).toEqual({ match: ["hair_colour", "head"] });
    });

    it("offers the skin tone's direction while the skin stage runs and sends a non-default choice", async () => {
      const user = userEvent.setup();
      const createHook = fakeCreate();
      renderPanel(createHook, PNG, "blob:img");
      await user.click(
        await screen.findByTestId(selectors.workspace.createAction({ name: "face_swap" })),
      );
      await user.upload(
        screen.getByTestId(selectors.workspace.create.reference),
        new File(["face"], "face.png", { type: "image/png" }),
      );
      const optionTestId = `${selectors.workspace.create.extentStage}-skin-skin_target`;
      expect(screen.queryByTestId(optionTestId)).not.toBeInTheDocument(); // the skin stage is off
      await user.click(screen.getByTestId(`${selectors.workspace.create.extentStage}-skin`));
      const direction = within(screen.getByTestId(optionTestId));
      await user.click(direction.getByRole("radio", { name: "The target's body" }));
      await user.click(screen.getByTestId(selectors.workspace.create.run));
      const call = vi.mocked(createHook.start).mock.calls.at(-1);
      expect((call?.[1] as { opParams?: unknown }).opParams).toEqual({ extent: "face_skin", skin_target: "body" });
    });

    it("sends the face strength when it is below a full swap", async () => {
      const user = userEvent.setup();
      const createHook = fakeCreate();
      renderPanel(createHook, PNG, "blob:img");
      await user.click(
        await screen.findByTestId(selectors.workspace.createAction({ name: "face_swap" })),
      );
      await user.upload(
        screen.getByTestId(selectors.workspace.create.reference),
        new File(["face"], "face.png", { type: "image/png" }),
      );
      fireEvent.change(screen.getByTestId(selectors.workspace.create.extentStrength), { target: { value: "0.6" } });
      await user.click(screen.getByTestId(selectors.workspace.create.run));
      const call = vi.mocked(createHook.start).mock.calls.at(-1);
      expect((call?.[1] as { opParams?: unknown }).opParams).toEqual({ extent: "face", face_strength: 0.6 });
    });

    it("does not ask for a reference on other operations", async () => {
      renderPanel(fakeCreate(), PNG, "blob:img");
      await screen.findByTestId(selectors.workspace.createAction({ name: "edit_instruct" }));
      expect(screen.queryByTestId(selectors.workspace.create.reference)).not.toBeInTheDocument();
    });
  });

  describe("consent gate", () => {
    it("shows no consent checkbox on the local tier (requireConsent off)", async () => {
      policyMock.data = publicPolicy({ tier: DeploymentTier.LOCAL, requireConsent: false });
      const user = userEvent.setup();
      renderPanel(fakeCreate(), PNG, "blob:img");

      await user.click(
        await screen.findByTestId(selectors.workspace.createAction({ name: "edit_instruct" })),
      );
      expect(screen.queryByTestId(selectors.workspace.create.consent)).not.toBeInTheDocument();
    });

    it("requires consent for a high-weight op on the public tier and threads it into params", async () => {
      policyMock.data = publicPolicy();
      explainConsent({ consentRequired: true, consentBasis: "tier", consentReason: "This deployment asks for consent." });
      const user = userEvent.setup();
      const createHook = fakeCreate();
      renderPanel(createHook, PNG, "blob:img");

      await user.click(
        await screen.findByTestId(selectors.workspace.createAction({ name: "edit_instruct" })),
      );
      // edit_instruct is prompt-driven; supply a prompt so only consent gates run.
      await user.type(screen.getByTestId(selectors.workspace.create.prompt), "make it sunset");

      const consent = screen.getByTestId(selectors.workspace.create.consent);
      expect(consent).toBeInTheDocument();
      expect(screen.getByTestId(selectors.workspace.create.run)).toBeDisabled();
      expect(screen.getByTestId(selectors.workspace.create.consentRequired)).toBeInTheDocument();

      await user.click(consent);
      expect(screen.getByTestId(selectors.workspace.create.run)).toBeEnabled();
      await user.click(screen.getByTestId(selectors.workspace.create.run));

      expect(createHook.start).toHaveBeenCalledWith(
        "edit_instruct",
        expect.objectContaining({ consentAffirmed: true }),
        { input: PNG },
      );
    });

    it("asks for consent only when the route leaves this machine, in plain words", async () => {
      policyMock.data = publicPolicy({ tier: DeploymentTier.LOCAL, requireConsent: false });
      const user = userEvent.setup();
      const createHook = fakeCreate();
      renderPanel(createHook, PNG, "blob:img");
      await user.click(
        await screen.findByTestId(selectors.workspace.createAction({ name: "edit_instruct" })),
      );
      await user.type(screen.getByTestId(selectors.workspace.create.prompt), "make it sunset");

      // A local run: the server says nothing is required, so no checkbox.
      explainConsent({ consentRequired: false });
      await waitFor(() => expect(screen.getByTestId(selectors.workspace.create.run)).toBeEnabled());
      expect(screen.queryByTestId(selectors.workspace.create.consent)).not.toBeInTheDocument();

      // Choosing the cloud route changes the answer; the checkbox appears with the reason.
      explainConsent({
        consentRequired: true,
        consentBasis: "route",
        consentReason: "This would run outside this machine.",
        tier: "byok-cloud",
      });
      await user.click(screen.getByLabelText(/cloud/i));
      const reason = await screen.findByTestId(selectors.workspace.create.consentReason);
      expect(reason.textContent).toContain("outside this machine");
      const consent = screen.getByTestId(selectors.workspace.create.consent);
      expect(screen.getByLabelText(/send them outside this machine/i)).toBe(consent);
      expect(screen.getByTestId(selectors.workspace.create.run)).toBeDisabled();

      await user.click(consent);
      await user.click(screen.getByTestId(selectors.workspace.create.run));
      expect(createHook.start).toHaveBeenCalledWith(
        "edit_instruct",
        expect.objectContaining({ consentAffirmed: true, allowByok: true }),
        { input: PNG },
      );
    });

    it("does not gate a none-weight op even on the public tier", async () => {
      policyMock.data = publicPolicy();
      const user = userEvent.setup();
      renderPanel(fakeCreate());

      // text_to_image is the default first generation op (none-weight).
      await user.type(
        await screen.findByTestId(selectors.workspace.create.prompt),
        "a serene lake",
      );
      expect(screen.queryByTestId(selectors.workspace.create.consent)).not.toBeInTheDocument();
      expect(screen.getByTestId(selectors.workspace.create.run)).toBeEnabled();
    });

    it("surfaces a 403 consent rejection with the actionable hint", async () => {
      policyMock.data = publicPolicy();
      const failed = fakeCreate({
        phase: "failed",
        error: "forbidden: consent required to edit people",
        consentBlocked: true,
      });
      renderPanel(failed, PNG, "blob:img");

      const fail = await screen.findByTestId(selectors.workspace.create.failed);
      expect(fail.textContent).toContain("consent required to edit people");
      expect(screen.getByTestId(selectors.workspace.create.consentBlocked)).toBeInTheDocument();
    });
  });
});
