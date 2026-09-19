import { describe, expect, it, vi } from "vitest";
import type { IApiClient } from "../../lib/api-client";
import { createCrudMethods } from "./crud-service";

describe("backlog CRUD next-action reads", () => {
  it("splits visible-list next-action requests into server-bounded batches", async () => {
    const post = vi.fn(async (_url: string, body: { items: string[] }) => ({
      results: body.items.map((item) => ({
        item,
        action: {
          id: "author_plan",
          compact_label: "Plan",
          expanded_label: "Author plan",
          enabled: true,
          blockers: [],
        },
      })),
    }));
    const crud = createCrudMethods({ post } as unknown as IApiClient);
    const items = Array.from({ length: 101 }, (_, index) => ({ kind: "idea" as const, name: `item-${index}` }));

    const actions = await crud.getNextActions(items);

    expect(post).toHaveBeenCalledTimes(2);
    expect(post.mock.calls.map(([, body]) => (body as { items: string[] }).items)).toEqual([
      items.slice(0, 100).map(({ kind, name }) => `${kind}/${name}`),
      items.slice(100).map(({ kind, name }) => `${kind}/${name}`),
    ]);
    expect(Object.keys(actions)).toHaveLength(101);
  });
});
