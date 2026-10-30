import { describe, expect, it } from "vitest";
import {
  attachmentKind,
  attachmentKindForName,
  extensionOf,
  formatFileSize,
  isBlockedAttachment,
  isLoopbackHostname,
  nativeChooserPlatform,
  planAttachSources,
  type AttachEnvironment,
} from "./attachments";

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

const IPHONE_UA = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1";
const IPAD_DESKTOP_UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15";
const ANDROID_UA = "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Mobile Safari/537.36";
const LINUX_UA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36";

function env(userAgent: string, hostname = "console.example.dev", maxTouchPoints = 0): AttachEnvironment {
  return { userAgent, hostname, maxTouchPoints };
}

describe("attachmentKindForName", () => {
  it("classifies host files by extension alone", () => {
    expect(attachmentKindForName("shot.PNG")).toBe("image");
    expect(attachmentKindForName("clip.mov")).toBe("video");
    expect(attachmentKindForName("PRD.md")).toBe("file");
  });
});

describe("isLoopbackHostname", () => {
  it("recognises every loopback spelling", () => {
    for (const host of ["localhost", "LOCALHOST", "app.localhost", "127.0.0.1", "127.8.9.10", "::1", "[::1]"]) {
      expect(isLoopbackHostname(host)).toBe(true);
    }
  });

  it("treats tunnel and LAN hosts as another machine", () => {
    for (const host of ["console.example.dev", "192.168.1.20", "10.0.0.2", "localhost.example.com", "127.0.0.1.nip.io", ""]) {
      expect(isLoopbackHostname(host)).toBe(false);
    }
  });
});

describe("nativeChooserPlatform", () => {
  it("detects the platforms whose OS offers camera + library + files", () => {
    expect(nativeChooserPlatform(env(IPHONE_UA, "x", 5))).toBe("iphone");
    expect(nativeChooserPlatform(env(ANDROID_UA, "x", 5))).toBe("android");
  });

  it("tells an iPad in desktop mode from a Mac by touch", () => {
    expect(nativeChooserPlatform(env(IPAD_DESKTOP_UA, "x", 5))).toBe("ipad");
    expect(nativeChooserPlatform(env(IPAD_DESKTOP_UA, "x", 0))).toBeNull();
  });

  it("leaves desktop browsers to the in-app menu", () => {
    expect(nativeChooserPlatform(env(LINUX_UA))).toBeNull();
  });
});

describe("planAttachSources", () => {
  const both = { hostAvailable: true, cameraAvailable: true };

  it("on a phone through the tunnel: the phone's own picker, then the host", () => {
    expect(planAttachSources(env(IPHONE_UA, "console.example.dev", 5), both)).toEqual(["device", "host"]);
    expect(planAttachSources(env(ANDROID_UA, "console.example.dev", 5), both)).toEqual(["device", "host"]);
  });

  it("on a remote desktop browser: upload, the in-app camera, then the host", () => {
    expect(planAttachSources(env(LINUX_UA), both)).toEqual(["upload", "camera", "host"]);
  });

  it("drops the in-app camera when there is no camera", () => {
    expect(planAttachSources(env(LINUX_UA), { hostAvailable: true, cameraAvailable: false })).toEqual(["upload", "host"]);
  });

  it("on the host itself: browsing by path comes first", () => {
    expect(planAttachSources(env(LINUX_UA, "localhost"), both)).toEqual(["host", "upload", "camera"]);
  });

  it("offers this device only when there is no session to browse", () => {
    expect(planAttachSources(env(IPHONE_UA, "x", 5), { hostAvailable: false, cameraAvailable: true })).toEqual(["device"]);
    expect(planAttachSources(env(LINUX_UA, "localhost"), { hostAvailable: false, cameraAvailable: false })).toEqual(["upload"]);
  });
});
