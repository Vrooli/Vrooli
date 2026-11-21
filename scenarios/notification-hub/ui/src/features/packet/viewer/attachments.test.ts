import { describe, expect, it } from "vitest";

import { makeAttachment } from "../mocks/askFactory";
import { findAttachmentByPath, resolveAttachmentRef } from "./attachments";

const attachments = [
  makeAttachment({ name: "plan.md", sourcePath: "/home/op/work/docs/plan.md" }),
  makeAttachment({ name: "work-readme", sourcePath: "/home/op/work/README.md" }),
  makeAttachment({ name: "other-readme", sourcePath: "/home/op/other/README.md" }),
  makeAttachment({ name: "renders", folder: true, sourcePath: "/home/op/work/renders", entries: [{ path: "walk/01.png", previewKind: "image", mediaType: "image/png" }] }),
];

describe("findAttachmentByPath", () => {
  it("matches an attachment name, its original path, and a file inside a folder", () => {
    expect(findAttachmentByPath(attachments, "plan.md")).toEqual({ name: "plan.md", path: "" });
    expect(findAttachmentByPath(attachments, "/home/op/work/docs/plan.md")).toEqual({ name: "plan.md", path: "" });
    expect(findAttachmentByPath(attachments, "renders/walk/01.png")).toEqual({ name: "renders", path: "walk/01.png" });
    expect(findAttachmentByPath(attachments, "/home/op/work/renders/walk/01.png")).toEqual({ name: "renders", path: "walk/01.png" });
  });

  it("ignores a line suffix and file:// scheme", () => {
    expect(findAttachmentByPath(attachments, "file:///home/op/work/docs/plan.md:12")).toEqual({ name: "plan.md", path: "" });
  });

  it("matches a relative tail only when it is unambiguous", () => {
    expect(findAttachmentByPath(attachments, "docs/plan.md")).toEqual({ name: "plan.md", path: "" });
    // Two attachments came from files called README.md on the requester's host: no guess.
    expect(findAttachmentByPath(attachments, "README.md")).toBeNull();
    expect(findAttachmentByPath(attachments, "work/README.md")).toEqual({ name: "work-readme", path: "" });
  });

  it("returns null for a path that is not attached", () => {
    expect(findAttachmentByPath(attachments, "/etc/passwd")).toBeNull();
    expect(findAttachmentByPath(attachments, "renders/nope.png")).toBeNull();
    expect(findAttachmentByPath(attachments, "")).toBeNull();
  });
});

describe("resolveAttachmentRef", () => {
  it("resolves a choice's attachment reference to its file and kind", () => {
    expect(resolveAttachmentRef(attachments, "renders/walk/01.png")).toMatchObject({ target: { name: "renders", path: "walk/01.png" }, kind: "image" });
    expect(resolveAttachmentRef(attachments, "missing.png")).toBeNull();
  });
});
