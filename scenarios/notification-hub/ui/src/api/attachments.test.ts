import { beforeEach, describe, expect, it, vi } from "vitest";

import { attachmentUrl, fetchAttachment } from "./attachments";
import { sessionFetch } from "./notifications";

vi.mock("./notifications", () => ({ sessionFetch: vi.fn() }));

describe("attachmentUrl", () => {
  it("addresses one stored copy by ask and attachment name", () => {
    expect(attachmentUrl("ask-1", "plan.md")).toMatch(/\/asks\/ask-1\/attachments\/plan\.md$/);
  });

  it("encodes names so a space or slash cannot change the route", () => {
    const url = attachmentUrl("ask 1", "my plan/draft.md");
    expect(url).toContain("/asks/ask%201/attachments/my%20plan%2Fdraft.md");
  });

  it("addresses a file inside a folder attachment with an encoded ?path=", () => {
    expect(attachmentUrl("ask-1", "docs", "notes/a b.md")).toMatch(/\/attachments\/docs\?path=notes%2Fa%20b\.md$/);
  });
});

describe("fetchAttachment", () => {
  beforeEach(() => vi.mocked(sessionFetch).mockReset());

  it("returns the response through the session fetch, passing the abort signal", async () => {
    const response = new Response("bytes", { status: 200 });
    vi.mocked(sessionFetch).mockResolvedValue(response);
    const controller = new AbortController();
    await expect(fetchAttachment("ask-1", "plan.md", "", controller.signal)).resolves.toBe(response);
    expect(sessionFetch).toHaveBeenCalledWith(expect.stringContaining("/asks/ask-1/attachments/plan.md"), { signal: controller.signal });
  });

  it("names the attachment and the status when the Hub cannot serve it", async () => {
    vi.mocked(sessionFetch).mockResolvedValue(new Response("", { status: 404 }));
    await expect(fetchAttachment("ask-1", "gone.md")).rejects.toThrow("attachment gone.md unavailable (404)");
  });
});
