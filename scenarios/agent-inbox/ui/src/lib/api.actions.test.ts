/**
 * Tests for API client - selectBranch, approveToolCall, rejectToolCall, uploadAttachment
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  selectBranch,
  approveToolCall,
  rejectToolCall,
  uploadAttachment,
} from "./api";
import { mockFetch, createMockResponse } from "./api.test.helpers";

describe("selectBranch", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  it("selects branch", async () => {
    mockFetch.mockResolvedValueOnce(
      createMockResponse({ active_leaf_message_id: "msg-2" })
    );

    const result = await selectBranch("chat-1", "msg-2");

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/chats/chat-1/messages/msg-2/select",
      expect.objectContaining({
        method: "POST",
      })
    );
    expect(result.active_leaf_message_id).toBe("msg-2");
  });
});

describe("approveToolCall", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  it("approves tool call", async () => {
    const mockResult = {
      success: true,
      tool_result: { id: "call-1", tool_name: "test", status: "completed" },
      pending_approvals: [],
      auto_continued: false,
    };

    mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

    const result = await approveToolCall("call-1", "chat-1");

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/tool-calls/call-1/approve?chat_id=chat-1",
      expect.objectContaining({
        method: "POST",
      })
    );
    expect(result).toEqual(mockResult);
  });
});

describe("rejectToolCall", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  it("rejects tool call with reason", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse(null, { status: 204 }));

    await rejectToolCall("call-1", "chat-1", "Not authorized");

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/tool-calls/call-1/reject?chat_id=chat-1",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ reason: "Not authorized" }),
      })
    );
  });
});

describe("uploadAttachment", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  it("uploads file", async () => {
    const mockUploadResponse = {
      id: "attach-1",
      file_name: "test.png",
      content_type: "image/png",
      file_size: 1024,
      storage_path: "/uploads/test.png",
      url: "/api/v1/uploads/test.png",
    };

    mockFetch.mockResolvedValueOnce(createMockResponse(mockUploadResponse));

    const file = new File(["test"], "test.png", { type: "image/png" });
    const result = await uploadAttachment(file);

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/attachments/upload",
      expect.objectContaining({
        method: "POST",
      })
    );
    expect(result).toEqual(mockUploadResponse);
  });

  it("throws on 413 payload too large", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse("", { status: 413, ok: false }));

    const file = new File(["test"], "large.png", { type: "image/png" });

    await expect(uploadAttachment(file)).rejects.toThrow("File is too large");
  });

  it("throws on 415 unsupported media type", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse("", { status: 415, ok: false }));

    const file = new File(["test"], "file.exe", { type: "application/octet-stream" });

    await expect(uploadAttachment(file)).rejects.toThrow("File type not supported");
  });
});
