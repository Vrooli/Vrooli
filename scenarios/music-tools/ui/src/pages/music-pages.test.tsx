import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { AuditionPage } from "./AuditionPage";
import { CompositionPage } from "./CompositionPage";
import { PoolPage } from "./PoolPage";
import { StylesPage } from "./StylesPage";

const take = { id: "take-1", style_id: "launch-trap", blob_ref: "/audio.wav", provenance: { seed: 4, applied_rung: "offload-dit" } };
const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  fetchMock.mockImplementation((url: string) => {
    if (url.includes("styles")) return Promise.resolve({ ok: true, json: async () => [{ id: "launch-trap", name: "Launch Trap", caption: "dark trap" }] });
    if (url.includes("pool/status")) return Promise.resolve({ ok: true, json: async () => ({ available: 2, reserved: 1, target_depth: 10, replenish_below: 3 }) });
    if (url.includes("pool/draw")) return Promise.resolve({ ok: true, json: async () => ({ take }) });
    if (url.includes("compose")) return Promise.resolve({ ok: true, json: async () => ({ job_id: "job-1" }) });
    if (url.includes("takes")) return Promise.resolve({ ok: true, json: async () => ({ takes: [take] }) });
    return Promise.resolve({ ok: true, json: async () => ({}) });
  });
  vi.stubGlobal("fetch", fetchMock);
});

describe("music feature pages", () => {
  it("renders styles and fetched style data", async () => { renderWithProviders(<StylesPage />); expect(await screen.findByText("Launch Trap")).toBeInTheDocument(); });
  it("renders audition peers with in-place playback", async () => { renderWithProviders(<AuditionPage />); expect(await screen.findByLabelText("Play take-1")).toBeInTheDocument(); });
  it("renders pool depth", async () => { renderWithProviders(<PoolPage />); expect(await screen.findByText(/2 available/)).toBeInTheDocument(); });
  it("queues composition and draws a take", async () => {
    renderWithProviders(<CompositionPage />);
    fireEvent.click(screen.getByRole("button", { name: "Compose" }));
    fireEvent.click(screen.getByRole("button", { name: "Draw take" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    expect(screen.getByText(/seed 4/)).toBeInTheDocument();
  });
});
