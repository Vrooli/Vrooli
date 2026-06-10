import { afterEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    getScore: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

describe("api/scoring", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("exports the generated Connect client", async () => {
    const { scoringClient } = await import("./scoring");

    await scoringClient.getScore({ scenario: "web-search" });

    expect(mocks.client.getScore).toHaveBeenCalledWith({ scenario: "web-search" });
  });

  it("fetchScore wraps GetScore with the scenario name", async () => {
    const { fetchScore } = await import("./scoring");
    mocks.client.getScore.mockResolvedValueOnce({ scenario: "cli-health" });

    const result = await fetchScore("cli-health");

    expect(mocks.client.getScore).toHaveBeenCalledWith({ scenario: "cli-health" });
    expect(result).toEqual({ scenario: "cli-health" });
  });
});
