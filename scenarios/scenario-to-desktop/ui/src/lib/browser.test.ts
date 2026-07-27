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
    vi.mocked(navigator.clipboard.writeText).mockRejectedValue(
      new Error("Permission denied"),
    );

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
  const windowOpenMock = vi.fn(() => null as Window | null);
  let originalLocation: Location;
  let originalOpen: typeof window.open;

  beforeEach(() => {
    originalOpen = window.open;
    windowOpenMock.mockClear();
    window.open = windowOpenMock as unknown as typeof window.open;
    originalLocation = window.location;
    // Use Object.defineProperty for location mock since it's not deletable
    Object.defineProperty(window, "location", {
      value: { href: "" },
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    window.open = originalOpen;
    windowOpenMock.mockClear();
    Object.defineProperty(window, "location", {
      value: originalLocation,
      writable: true,
      configurable: true,
    });
  });

  it("opens URL in new window by default", () => {
    triggerDownload({ url: "http://example.com/file.zip" });

    expect(windowOpenMock).toHaveBeenCalledWith(
      "http://example.com/file.zip",
      "_blank",
    );
  });

  it("opens URL in new window when newWindow is true", () => {
    triggerDownload({ url: "http://example.com/file.zip", newWindow: true });

    expect(windowOpenMock).toHaveBeenCalledWith(
      "http://example.com/file.zip",
      "_blank",
    );
  });

  it("navigates current window when newWindow is false", () => {
    triggerDownload({ url: "http://example.com/file.zip", newWindow: false });

    expect(window.location.href).toBe("http://example.com/file.zip");
    expect(windowOpenMock).not.toHaveBeenCalled();
  });
});

describe("triggerBlobDownload", () => {
  let originalCreateObjectURL: typeof URL.createObjectURL;
  let originalRevokeObjectURL: typeof URL.revokeObjectURL;
  let originalCreateElement: typeof document.createElement;
  let originalAppendChild: typeof document.body.appendChild;
  let originalRemoveChild: typeof document.body.removeChild;
  let mockLink: {
    href: string;
    download: string;
    click: ReturnType<typeof vi.fn>;
  };
  let createdUrls: Blob[];
  let revokedUrls: string[];

  beforeEach(() => {
    // Store originals and mock URL methods
    originalCreateObjectURL = URL.createObjectURL;
    originalRevokeObjectURL = URL.revokeObjectURL;
    originalCreateElement = document.createElement.bind(document);
    originalAppendChild = document.body.appendChild.bind(document.body);
    originalRemoveChild = document.body.removeChild.bind(document.body);
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

    document.createElement = vi
      .fn()
      .mockReturnValue(mockLink) as typeof document.createElement;
    document.body.appendChild = vi
      .fn()
      .mockReturnValue(mockLink) as typeof document.body.appendChild;
    document.body.removeChild = vi
      .fn()
      .mockReturnValue(mockLink) as typeof document.body.removeChild;
  });

  afterEach(() => {
    // Restore originals
    URL.createObjectURL = originalCreateObjectURL;
    URL.revokeObjectURL = originalRevokeObjectURL;
    document.createElement = originalCreateElement;
    document.body.appendChild = originalAppendChild;
    document.body.removeChild = originalRemoveChild;
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

    expect(document.body.appendChild).toHaveBeenCalled();
    expect(mockLink.click).toHaveBeenCalled();
    expect(document.body.removeChild).toHaveBeenCalled();
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
