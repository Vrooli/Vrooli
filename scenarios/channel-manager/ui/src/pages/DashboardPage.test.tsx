import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DashboardPage } from "./DashboardPage";
import { renderWithProviders } from "../test-utils";
import { overview } from "../api/channelManager";

vi.mock("../api/channelManager", () => ({
  createIdentity: vi.fn().mockResolvedValue({}),
  overview: vi.fn().mockResolvedValue({ identities: {}, actions: {} }),
  startProgram: vi.fn().mockResolvedValue({ status: "warming" }),
  enqueueAction: vi.fn().mockResolvedValue({ id: "action-1" }),
  completeAction: vi.fn().mockResolvedValue({ status: "succeeded" }),
  recordObservation: vi.fn().mockResolvedValue({ flag: null }),
}));

describe("DashboardPage", () => {
  it("guides a credential-free keyboard manual completion and observation", async () => {
    renderWithProviders(<DashboardPage />);
    fireEvent.change(screen.getByLabelText("New identity ID"), { target: { value: "x-1" } });
    fireEvent.click(screen.getByRole("button", { name: "Create and start X warming" }));
    await waitFor(() => expect(screen.getByTestId("operator-identity-ready")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Queue manual engagement" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Record manual completion" })).not.toBeDisabled());
    fireEvent.change(screen.getByLabelText("Completion evidence"), { target: { value: "https://example.test/proof" } });
    fireEvent.click(screen.getByRole("button", { name: "Record manual completion" }));
    await waitFor(() => expect(screen.getAllByRole("status").some((node) => node.textContent?.includes("evidence"))).toBe(true));
    fireEvent.change(screen.getByLabelText("Reach or impressions"), { target: { value: "120" } });
    fireEvent.click(screen.getByRole("button", { name: "Record observation" }));
    await waitFor(() => expect(screen.getByText(/evidence, not a claim/i)).toBeInTheDocument());
  });

  it("renders roster, due work, provenance, flags, and a purposeful empty state", async () => {
    vi.mocked(overview).mockResolvedValueOnce({
      identities: { "x-brand": { id: "x-brand", platform_id: "x", purpose: "brand", environment_ref: "env", vault_ref: "vault://ref", status: "warming", lane_grants: ["main"] } },
      actions: { "a-1": { id: "a-1", identity_id: "x-brand", kind: "engage", window: "2026-07-28T12:00:00Z", status: "scheduled", rolled_count: 2 } },
      programs: { "x-conservative": { id: "x-conservative", platform_id: "x", provenance: { confidence: "speculative", source_kind: "operator-practice", revisit_trigger: "five completed runs" } } },
      flags: { "x-brand": [{ message: "Reach measurement needs review" }] },
    });
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByLabelText("x-brand: warming")).toBeInTheDocument();
    expect(screen.getByText(/rolled count 2/i)).toBeInTheDocument();
    expect(screen.getByText(/operator-practice/i)).toBeInTheDocument();
    expect(screen.getByText(/Reach measurement needs review/i)).toBeInTheDocument();
  });
});
