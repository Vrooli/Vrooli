/**
 * Workspace × Create integration. Drives the real <Workspace /> with an
 * injected CreateClient: switch to Create mode, prompt, generate, and assert
 * the variation grid fills — then "send to canvas" adopts the chosen variation
 * as a new document (resetting to Edit mode with the image on the canvas). The
 * full Stage-3 wiring end to end, without the network.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeOpsMocks } from "./mocks/ops";
import { makeAIMocks, makeCreateClient } from "./mocks/ai";

vi.mock("../../api/ops", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ops")>();
  return { ...actual, ...makeOpsMocks() };
});

vi.mock("../../api/ai", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ai")>();
  return { ...actual, ...makeAIMocks() };
});

import { Workspace } from "./Workspace";

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

describe("Workspace create flow", () => {
  it("generates variations and sends one to the canvas as a new document", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Workspace createClient={makeCreateClient()} />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.modeSwitcher)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.workspace.modeOption({ mode: "create" })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.create.panel)).toBeInTheDocument();
    });

    await user.type(screen.getByTestId(selectors.workspace.create.prompt), "a serene lake");
    await user.click(screen.getByTestId(selectors.workspace.create.run));

    // The generated variation lands in the result grid…
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.workspace.createVariation({ index: 1 })),
      ).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workspace.create.succeeded)).toBeInTheDocument();

    // …and "send to canvas" adopts it as a new document in Edit mode.
    await user.click(screen.getByTestId(selectors.workspace.createSend({ index: 1 })));
    await waitFor(() => {
      expect(screen.queryByTestId(selectors.workspace.create.panel)).not.toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.workspace.canvas.image)).toBeInTheDocument();
  });
});
