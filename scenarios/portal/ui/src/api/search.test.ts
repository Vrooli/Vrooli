import { describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({
  suggest: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => client),
}));

import { suggest } from "./search";

describe("api/search Connect wrappers", () => {
  it("projects default suggest input into the generated request", async () => {
    client.suggest.mockResolvedValueOnce({ hits: [] });

    await expect(suggest({ query: "portal" })).resolves.toEqual({ hits: [] });

    expect(client.suggest).toHaveBeenCalledWith({
      query: "portal",
      types: [],
      limit: 5,
      group: "",
    });
  });

  it("preserves explicit suggest filters", async () => {
    client.suggest.mockResolvedValueOnce({ hits: [{ title: "Skill" }] });

    await suggest({ query: "skill", types: ["skill"], limit: 2, group: "docs" });

    expect(client.suggest).toHaveBeenCalledWith({
      query: "skill",
      types: ["skill"],
      limit: 2,
      group: "docs",
    });
  });
});
