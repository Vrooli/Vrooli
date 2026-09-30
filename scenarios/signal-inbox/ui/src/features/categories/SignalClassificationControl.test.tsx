import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

const api = vi.hoisted(() => ({
  listCategories: vi.fn(),
  getClassification: vi.fn(),
  confirmClassification: vi.fn(),
}));

vi.mock("../../api/categories", () => ({ categoriesClient: api }));

import { SignalClassificationControl } from "./SignalClassificationControl";

describe("SignalClassificationControl [REQ:SIG-P0-005]", () => {
  beforeEach(() => {
    api.listCategories.mockResolvedValue({ categories: [{ id: "category-a", name: "Research" }, { id: "category-b", name: "Build" }] });
    api.getClassification.mockResolvedValue({ classification: { signalId: "signal-1", proposedCategoryId: "category-a", proposedConfidence: 0.82, confirmedCategoryId: "", reason: "model route" } });
    api.confirmClassification.mockResolvedValue({});
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a proposal as advisory and appends the selected operator decision", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SignalClassificationControl signalID="signal-1" />);

    expect(await screen.findByText(new RegExp(`${strings.categories.proposed}.*${strings.categories.confidence}`))).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText(strings.categories.categoryFor), "category-b");
    await user.click(screen.getByRole("button", { name: strings.categories.confirm }));

    await waitFor(() => expect(api.confirmClassification).toHaveBeenCalledWith({ signalId: "signal-1", categoryId: "category-b" }));
  });

  it("does not pretend a missing classification is a confirmed category", async () => {
    api.getClassification.mockRejectedValue(new Error("not found"));
    renderWithProviders(<SignalClassificationControl signalID="signal-1" />);

    expect(await screen.findByText(strings.categories.classificationPending)).toBeInTheDocument();
  });
});
