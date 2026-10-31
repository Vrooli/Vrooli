/**
 * ApplyCard tests — focused on the apply-card surface only. Renders <ApplyCard />
 * directly so failures point at apply-feature behaviour, not shell composition.
 * Follows the canonical mock-builder pattern.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeApplyAction, makeApplyResponse, makeSkipReason } from "./mocks/factories";
import { makeApplyMocks } from "./mocks/apply";

vi.mock("../../api/apply", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/apply")>();
  return { ...actual, ...makeApplyMocks() };
});

import { ApplyCard } from "./ApplyCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("ApplyCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("keeps the preview button disabled until both inputs are filled", async () => {
    renderWithProviders(<ApplyCard />);
    const button = screen.getByTestId(selectors.apply.previewButton);
    expect(button).toBeDisabled();

    const user = userEvent.setup();
    await user.type(screen.getByTestId(selectors.apply.brandInput), "b1");
    expect(button).toBeDisabled();
    await user.type(screen.getByTestId(selectors.apply.scenarioInput), "web-console");
    expect(button).toBeEnabled();
  });

  it("previews the plan and renders applied + skipped rows", async () => {
    const { previewApply } = await import("../../api/apply");
    vi.mocked(previewApply).mockResolvedValueOnce(
      makeApplyResponse({
        scenario: "web-console",
        brandVersion: 4,
        applied: [makeApplyAction({ element: "colors", file: "ui/src/styles/brand.css", type: "css" })],
        skipped: [makeSkipReason({ element: "logo", reason: "no logo asset" })],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ApplyCard />);
    await user.type(screen.getByTestId(selectors.apply.brandInput), "b1");
    await user.type(screen.getByTestId(selectors.apply.scenarioInput), "web-console");
    await user.click(screen.getByTestId(selectors.apply.previewButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.apply.results)).toBeInTheDocument();
    });
    expect(vi.mocked(previewApply).mock.calls[0]?.[0]).toMatchObject({
      brandId: "b1",
      scenarioName: "web-console",
    });
    expect(screen.getByTestId(selectors.apply.appliedList).textContent).toContain("ui/src/styles/brand.css");
    expect(screen.getByTestId(selectors.apply.skippedList).textContent).toContain("logo");
    expect(screen.getByTestId(selectors.apply.summary).textContent).toContain("web-console");
  });

  it("shows the empty state when nothing would change", async () => {
    const { previewApply } = await import("../../api/apply");
    vi.mocked(previewApply).mockResolvedValueOnce(makeApplyResponse({ applied: [], skipped: [] }));

    const user = userEvent.setup();
    renderWithProviders(<ApplyCard />);
    await user.type(screen.getByTestId(selectors.apply.brandInput), "b1");
    await user.type(screen.getByTestId(selectors.apply.scenarioInput), "web-console");
    await user.click(screen.getByTestId(selectors.apply.previewButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.apply.empty)).toBeInTheDocument();
    });
  });
});
