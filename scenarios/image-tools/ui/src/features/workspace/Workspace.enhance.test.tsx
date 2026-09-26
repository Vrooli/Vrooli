/**
 * Workspace × Enhance integration. Drives the real <Workspace /> with an
 * injected EnhanceClient: switch to Enhance mode, load an image, run a one-tap
 * enhancement, and assert the async result composes into the same history and
 * auto-engages the before/after compare — the full Stage-2 wiring end to end,
 * without the network.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeOpsMocks } from "./mocks/ops";
import { makeAIMocks, makeEnhanceClient } from "./mocks/ai";

vi.mock("../../api/ops", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ops")>();
  return { ...actual, ...makeOpsMocks() };
});

vi.mock("../../api/ai", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ai")>();
  return { ...actual, ...makeAIMocks() };
});

import { Workspace } from "./Workspace";

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

describe("Workspace enhance flow", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn(() => "blob:fake"),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("runs an enhancement and composes the result into history with before/after", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Workspace enhanceClient={makeEnhanceClient()} />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.modeSwitcher)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.modeOption({ mode: "enhance" })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.enhance.panel)).toBeInTheDocument();
    });

    await user.upload(screen.getByTestId(selectors.workspace.fileInput), PNG);
    await user.click(screen.getByTestId(selectors.workspace.enhance.run));

    // The async result lands as the first non-destructive history step…
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.workspace.historyStep({ index: 1 })),
      ).toBeInTheDocument();
    });
    // …and the canvas auto-engages the before/after compare.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.compare.root)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workspace.enhance.succeeded)).toBeInTheDocument();
  });
});
