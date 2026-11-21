import { describe, expect, it } from "vitest";

import { makeAttachment } from "../mocks/askFactory";
import { packetManifest } from "./manifest";

describe("packetManifest", () => {
  const attachments = [
    makeAttachment({ name: "brief.md", sizeBytes: 12 }),
    makeAttachment({ name: "page.html", explainer: true }),
    makeAttachment({ name: "renders", folder: true, entries: [{ path: "walk/01.png", previewKind: "image", mediaType: "image/png" }, { path: "notes.txt" }] }),
  ];

  it("lays files and folder entries out as one virtual tree on the Hub's content route", () => {
    const files = packetManifest("ask 1", attachments);

    expect(files.map((file) => file.path)).toEqual(["brief.md", "renders/walk/01.png", "renders/notes.txt"]);
    expect(files[0]).toMatchObject({ url: expect.stringMatching(/\/asks\/ask%201\/attachments\/brief\.md$/), sizeBytes: 12 });
    expect(files[1]?.url).toMatch(/\/attachments\/renders\?path=walk%2F01\.png$/);
    expect(files[1]?.kind).toBe("image");
  });

  it("leaves the sealed explainer page out", () => {
    expect(packetManifest("a", attachments).some((file) => file.path === "page.html")).toBe(false);
  });
});
