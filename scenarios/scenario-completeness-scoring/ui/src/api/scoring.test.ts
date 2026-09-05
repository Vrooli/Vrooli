import { afterEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    getScore: vi.fn(),
    getScoreTrend: vi.fn(),
    listScores: vi.fn(),
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

  it("fetchScoreTrend wraps GetScoreTrend with a bounded default limit", async () => {
    const { fetchScoreTrend } = await import("./scoring");
    mocks.client.getScoreTrend.mockResolvedValueOnce({ scenario: "cli-health", snapshots: [] });

    await fetchScoreTrend("cli-health");

    expect(mocks.client.getScoreTrend).toHaveBeenCalledWith({ scenario: "cli-health", limit: 12 });
  });

  it("fetchScores reads the priority-sorted persisted fleet page", async () => {
    const { fetchScores } = await import("./scoring");
    mocks.client.listScores.mockResolvedValueOnce({ scores: [], nextPageToken: "" });

    await fetchScores({ pageToken: "10", pageSize: 5 });

    expect(mocks.client.listScores).toHaveBeenCalledWith({
      sortBy: 5,
      order: 2,
      pageSize: 5,
      pageToken: "10",
    });
  });
});
