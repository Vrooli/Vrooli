/**
 * Workspace accessibility regression tests. The form, in-canvas result, and
 * error states must each pass axe. The Workspace owns its discovery query and
 * op runner, so the a11y waits + mocks live with the feature.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeRunOpImageResult } from "./mocks/factories";
import { makeOpsMocks } from "./mocks/ops";

vi.mock("../../api/ops", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ops")>();
  return { ...actual, ...makeOpsMocks() };
});

import { Workspace } from "./Workspace";
import type { WorkspaceRunner } from "./useWorkspace";

const imageRunner: WorkspaceRunner = () =>
  Promise.resolve({
    kind: "image",
    result: makeRunOpImageResult(),
    outputFile: new File(["x"], "out.png", { type: "image/png" }),
  });

describe("Workspace accessibility", () => {
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

  it("renders the form state without axe violations", async () => {
    const { container } = renderWithProviders(<Workspace />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.paramsForm)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the result state without axe violations", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<Workspace runner={imageRunner} />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.applyButton)).toBeInTheDocument();
    });

    await user.upload(
      screen.getByTestId(selectors.workspace.fileInput),
      new File(["bytes"], "in.png", { type: "image/png" }),
    );
    await user.click(screen.getByTestId(selectors.workspace.applyButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.canvas.image)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the error state without axe violations", async () => {
    const { listOperations } = await import("../../api/ops");
    vi.mocked(listOperations).mockRejectedValueOnce(new Error("ops down"));

    const { container } = renderWithProviders(<Workspace />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.error)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });
});
