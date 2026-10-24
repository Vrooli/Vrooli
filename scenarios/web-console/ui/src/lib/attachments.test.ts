import { describe, expect, it } from "vitest";
import { attachmentKind, extensionOf, formatFileSize, isBlockedAttachment } from "./attachments";

function file(name: string, type = ""): File {
  return new File(["x"], name, { type });
}

describe("extensionOf", () => {
  it("returns the lower-case extension without the dot", () => {
    expect(extensionOf("Report.PDF")).toBe("pdf");
    expect(extensionOf("archive.tar.gz")).toBe("gz");
    expect(extensionOf("noext")).toBe("");
    expect(extensionOf(".hidden")).toBe("");
    expect(extensionOf("trailing.")).toBe("");
  });
});

describe("isBlockedAttachment", () => {
  it("blocks native executables and installers regardless of case", () => {
    expect(isBlockedAttachment(file("evil.exe"))).toBe(true);
    expect(isBlockedAttachment(file("LIB.SO"))).toBe(true);
    expect(isBlockedAttachment(file("app.jar"))).toBe(true);
  });

  it("allows ordinary files", () => {
    expect(isBlockedAttachment(file("notes.md"))).toBe(false);
    expect(isBlockedAttachment(file("clip.mp4"))).toBe(false);
    expect(isBlockedAttachment(file("data.bin"))).toBe(false);
  });
});

describe("attachmentKind", () => {
  it("prefers the MIME type", () => {
    expect(attachmentKind(file("x", "image/png"))).toBe("image");
    expect(attachmentKind(file("x", "video/mp4"))).toBe("video");
    expect(attachmentKind(file("x", "audio/mpeg"))).toBe("audio");
  });

  it("falls back to the extension when the browser omits the type", () => {
    expect(attachmentKind(file("vector.svg"))).toBe("image");
    expect(attachmentKind(file("clip.webm"))).toBe("video");
    expect(attachmentKind(file("voice.flac"))).toBe("audio");
    expect(attachmentKind(file("archive.zip"))).toBe("file");
  });
});

describe("formatFileSize", () => {
  it("formats binary units", () => {
    expect(formatFileSize(512)).toBe("512 B");
    expect(formatFileSize(25 * 1024 * 1024)).toBe("25 MiB");
    expect(formatFileSize(1536)).toBe("1.5 KiB");
  });
});
