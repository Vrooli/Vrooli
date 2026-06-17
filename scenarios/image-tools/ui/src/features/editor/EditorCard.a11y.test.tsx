/**
 * EditorCard accessibility regression tests.
 *
 * Editor owns its discovery query and run mutation, so the a11y waits and
 * mocks live with the feature instead of leaking into shell-level a11y
 * tests.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { makeRunOpImageResult } from "./mocks/factories";
import { makeOpsMocks } from "./mocks/ops";
import { EditorCard } from "./EditorCard";

vi.mock("../../api/ops", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/ops")>();
  return { ...actual, ...makeOpsMocks() };
});

describe("EditorCard accessibility", () => {
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
    const { container } = renderWithProviders(<EditorCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.paramsForm)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the result state without axe violations", async () => {
    const { runOp } = await import("../../api/ops");
    vi.mocked(runOp).mockResolvedValueOnce(makeRunOpImageResult());

    const user = userEvent.setup();
    const { container } = renderWithProviders(<EditorCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.runButton)).toBeInTheDocument();
    });

    await user.upload(
      screen.getByTestId(selectors.editor.fileInput),
      new File(["bytes"], "in.png", { type: "image/png" }),
    );
    await user.click(screen.getByTestId(selectors.editor.runButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.resultImage)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the error state without axe violations", async () => {
    const { listOperations } = await import("../../api/ops");
    vi.mocked(listOperations).mockRejectedValueOnce(new Error("ops down"));

    const { container } = renderWithProviders(<EditorCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.editor.error)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });
});
