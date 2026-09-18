import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";

const getProfile = vi.hoisted(() => vi.fn());
const saveProfileDraft = vi.hoisted(() => vi.fn());
const applyProfile = vi.hoisted(() => vi.fn());
vi.mock("../../api/profile", () => ({ getProfile, saveProfileDraft, applyProfile }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));

import { OnboardingPage } from "./OnboardingPage";

describe("OnboardingPage", () => {
  beforeEach(() => { vi.clearAllMocks(); getProfile.mockResolvedValue(undefined); saveProfileDraft.mockResolvedValue({}); applyProfile.mockResolvedValue({ profile: {} }); });

  it("preserves a draft while moving through steps", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Make your plan fit your life" })).toBeInTheDocument());
    await user.click(screen.getByLabelText("vegan"));
    await user.click(screen.getByRole("button", { name: "Save and next" }));
    await waitFor(() => expect(screen.getByText("2. Allergies")).toBeInTheDocument());
    expect(saveProfileDraft).toHaveBeenCalledWith("w1", expect.stringContaining("vegan"));
    await user.click(screen.getByRole("button", { name: "Back" }));
    expect(screen.getByLabelText("vegan")).toBeChecked();
  });

  it("allows zero appliances and applies the final profile", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OnboardingPage />);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Make your plan fit your life" })).toBeInTheDocument());
    for (let index = 0; index < 3; index += 1) await user.click(screen.getByRole("button", { name: "Save and next" }));
    await user.click(screen.getByRole("button", { name: "Apply profile" }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Active profile applied"));
    expect(applyProfile).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", appliances: [] }));
  });
});
