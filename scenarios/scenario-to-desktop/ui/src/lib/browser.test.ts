/**
 * Tests for browser utility functions.
 * These tests mock browser APIs to verify the seam functions work correctly.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  writeToClipboard,
  readFileAsText,
  triggerDownload,
  triggerBlobDownload,
  browserMocks,
} from "./browser";

// ============================================================================
// Clipboard Operations
// ============================================================================

describe("writeToClipboard", () => {
  beforeEach(() => {
    // Reset clipboard mock
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn(),
      },
    });
  });

  it("returns success when clipboard write succeeds", async () => {
    vi.mocked(navigator.clipboard.writeText).mockResolvedValue(undefined);

    const result = await writeToClipboard("test content");

    expect(result.success).toBe(true);
    expect(result.error).toBeUndefined();
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("test content");
  });

  it("returns error when clipboard write fails", async () => {
    vi.mocked(navigator.clipboard.writeText).mockRejectedValue(new Error("Permission denied"));

    const result = await writeToClipboard("test content");

    expect(result.success).toBe(false);
    expect(result.error).toBe("Permission denied");
  });

  it("handles non-Error exceptions", async () => {
    vi.mocked(navigator.clipboard.writeText).mockRejectedValue("string error");

    const result = await writeToClipboard("test content");

    expect(result.success).toBe(false);
    expect(result.error).toBe("Failed to copy to clipboard");
  });
});

// ============================================================================
// File Operations
// ============================================================================

describe("readFileAsText", () => {
  it("returns content when file read succeeds", async () => {
    const mockFile = {
      text: vi.fn().mockResolvedValue("file content here"),
    } as unknown as File;

    const result = await readFileAsText(mockFile);

    expect(result.success).toBe(true);
    expect(result.content).toBe("file content here");
    expect(result.error).toBeUndefined();
  });

  it("returns error when file read fails", async () => {
    const mockFile = {
      text: vi.fn().mockRejectedValue(new Error("File not readable")),
    } as unknown as File;

    const result = await readFileAsText(mockFile);

    expect(result.success).toBe(false);
    expect(result.content).toBeUndefined();
    expect(result.error).toBe("File not readable");
  });

  it("handles non-Error exceptions", async () => {
    const mockFile = {
      text: vi.fn().mockRejectedValue("unknown failure"),
    } as unknown as File;

    const result = await readFileAsText(mockFile);

    expect(result.success).toBe(false);
    expect(result.error).toBe("Failed to read file");
  });
});

// ============================================================================
// Download Operations
// ============================================================================

describe("triggerDownload", () => {
  let windowOpenSpy: ReturnType<typeof vi.spyOn>;
  let originalLocation: Location;

  beforeEach(() => {
    windowOpenSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    originalLocation = window.location;
    // @ts-expect-error - mocking window.location
    delete window.location;
    window.location = { href: "" } as Location;
  });

  afterEach(() => {
    window.location = originalLocation;
  });

  it("opens URL in new window by default", () => {
    triggerDownload({ url: "http://example.com/file.zip" });

    expect(windowOpenSpy).toHaveBeenCalledWith("http://example.com/file.zip", "_blank");
  });

  it("opens URL in new window when newWindow is true", () => {
    triggerDownload({ url: "http://example.com/file.zip", newWindow: true });

    expect(windowOpenSpy).toHaveBeenCalledWith("http://example.com/file.zip", "_blank");
  });

  it("navigates current window when newWindow is false", () => {
    triggerDownload({ url: "http://example.com/file.zip", newWindow: false });

    expect(window.location.href).toBe("http://example.com/file.zip");
    expect(windowOpenSpy).not.toHaveBeenCalled();
  });
});

describe("triggerBlobDownload", () => {
  let originalCreateObjectURL: typeof URL.createObjectURL;
  let originalRevokeObjectURL: typeof URL.revokeObjectURL;
  let createElementSpy: ReturnType<typeof vi.spyOn>;
  let appendChildSpy: ReturnType<typeof vi.spyOn>;
  let removeChildSpy: ReturnType<typeof vi.spyOn>;
  let mockLink: { href: string; download: string; click: ReturnType<typeof vi.fn> };
  let createdUrls: Blob[];
  let revokedUrls: string[];

  beforeEach(() => {
    // Store originals and mock URL methods
    originalCreateObjectURL = URL.createObjectURL;
    originalRevokeObjectURL = URL.revokeObjectURL;
    createdUrls = [];
    revokedUrls = [];

    URL.createObjectURL = vi.fn((blob: Blob) => {
      createdUrls.push(blob);
      return "blob:test-url";
    });
    URL.revokeObjectURL = vi.fn((url: string) => {
      revokedUrls.push(url);
    });

    mockLink = {
      href: "",
      download: "",
      click: vi.fn(),
    };

    createElementSpy = vi.spyOn(document, "createElement").mockReturnValue(mockLink as unknown as HTMLElement);
    appendChildSpy = vi.spyOn(document.body, "appendChild").mockImplementation(() => mockLink as unknown as HTMLElement);
    removeChildSpy = vi.spyOn(document.body, "removeChild").mockImplementation(() => mockLink as unknown as HTMLElement);
  });

  afterEach(() => {
    // Restore originals
    URL.createObjectURL = originalCreateObjectURL;
    URL.revokeObjectURL = originalRevokeObjectURL;
    createElementSpy.mockRestore();
    appendChildSpy.mockRestore();
    removeChildSpy.mockRestore();
  });

  it("creates download link with correct attributes", () => {
    const blob = new Blob(["test content"], { type: "text/plain" });

    triggerBlobDownload(blob, "test-file.txt");

    expect(createdUrls).toHaveLength(1);
    expect(createdUrls[0]).toBe(blob);
    expect(mockLink.href).toBe("blob:test-url");
    expect(mockLink.download).toBe("test-file.txt");
  });

  it("triggers click and cleans up", () => {
    const blob = new Blob(["test content"], { type: "text/plain" });

    triggerBlobDownload(blob, "test-file.txt");

    expect(appendChildSpy).toHaveBeenCalled();
    expect(mockLink.click).toHaveBeenCalled();
    expect(removeChildSpy).toHaveBeenCalled();
    expect(revokedUrls).toContain("blob:test-url");
  });
});

// ============================================================================
// Testing Utilities
// ============================================================================

describe("browserMocks", () => {
  describe("createClipboardMock", () => {
    it("creates a mock clipboard that tracks writes", async () => {
      const clipboard = browserMocks.createClipboardMock();

      await clipboard.writeText("first");
      await clipboard.writeText("second");

      expect(clipboard.getWrites()).toEqual(["first", "second"]);
    });

    it("can clear writes", async () => {
      const clipboard = browserMocks.createClipboardMock();

      await clipboard.writeText("content");
      clipboard.clear();

      expect(clipboard.getWrites()).toEqual([]);
    });

    it("returns copies of writes array", async () => {
      const clipboard = browserMocks.createClipboardMock();

      await clipboard.writeText("test");
      const writes = clipboard.getWrites();
      writes.push("modified");

      expect(clipboard.getWrites()).toEqual(["test"]);
    });
  });

  describe("createFileReaderMock", () => {
    it("creates a mock file with specified content", async () => {
      const mockFile = browserMocks.createFileReaderMock("mock content");

      const content = await mockFile.text();

      expect(content).toBe("mock content");
    });
  });
});
