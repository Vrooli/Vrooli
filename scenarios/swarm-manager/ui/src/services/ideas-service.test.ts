import { describe, it, expect, vi, beforeEach } from "vitest";
import { createIdeasService, type IIdeasService } from "./ideas-service";
import type { IApiClient } from "../lib/api-client";
import type { Idea } from "../types";

/**
 * Ideas Service Tests
 *
 * These tests demonstrate the seam pattern - by injecting a mock API client,
 * we can test the service layer in isolation without needing to mock HTTP.
 *
 * This is cleaner than module-level mocking because:
 * 1. Tests explicitly show their dependencies
 * 2. No magic mocking of import paths
 * 3. Easy to test with different client configurations
 */

// [REQ:MOD-P0-001] Test ideas service seam
describe("Ideas Service", () => {
  let mockApiClient: IApiClient;
  let service: IIdeasService;

  beforeEach(() => {
    // Create a mock API client - the seam allows easy substitution
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };

    // Inject the mock client into the service
    service = createIdeasService(mockApiClient);
  });

  describe("list", () => {
    it("fetches ideas from the correct endpoint", async () => {
      const mockIdeas: Idea[] = [
        {
          name: "test-idea",
          title: "Test Idea",
          description: "A test",
          status: "backlog",
          priority: 1,
          tags: [],
          created: "2026-01-28T00:00:00Z",
          updated: "2026-01-28T00:00:00Z",
        },
      ];
      vi.mocked(mockApiClient.get).mockResolvedValue({ ideas: mockIdeas });

      const result = await service.list();

      expect(mockApiClient.get).toHaveBeenCalledWith("/ideas");
      expect(result).toEqual(mockIdeas);
    });
  });

  describe("get", () => {
    it("fetches a single idea by name", async () => {
      const mockIdea: Idea = {
        name: "my-idea",
        title: "My Idea",
        description: "Details",
        status: "ready",
        priority: 2,
        tags: ["test"],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
      };
      vi.mocked(mockApiClient.get).mockResolvedValue({ idea: mockIdea });

      const result = await service.get("my-idea");

      expect(mockApiClient.get).toHaveBeenCalledWith("/ideas/my-idea");
      expect(result).toEqual(mockIdea);
    });
  });

  describe("create", () => {
    it("posts a new idea", async () => {
      const newIdea = {
        name: "new-idea",
        title: "New Idea",
        description: "A new idea",
        status: "backlog" as const,
        priority: 1,
        tags: ["new"],
      };
      const createdIdea: Idea = {
        ...newIdea,
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
      };
      vi.mocked(mockApiClient.post).mockResolvedValue({ idea: createdIdea });

      const result = await service.create(newIdea);

      expect(mockApiClient.post).toHaveBeenCalledWith("/ideas", {
        name: "new-idea",
        title: "New Idea",
        description: "A new idea",
        priority: 1,
        tags: ["new"],
      });
      expect(result).toEqual(createdIdea);
    });
  });

  describe("update", () => {
    it("puts an updated idea", async () => {
      const updates = {
        title: "Updated Title",
        description: "Original description",
        status: "backlog" as const,
        priority: 1,
        tags: [],
      };
      const updatedIdea: Idea = {
        name: "my-idea",
        title: "Updated Title",
        description: "Original description",
        status: "backlog",
        priority: 1,
        tags: [],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T01:00:00Z",
      };
      vi.mocked(mockApiClient.put).mockResolvedValue({ idea: updatedIdea });

      const result = await service.update("my-idea", updates);

      expect(mockApiClient.put).toHaveBeenCalledWith("/ideas/my-idea", updates);
      expect(result).toEqual(updatedIdea);
    });
  });

  describe("delete", () => {
    it("deletes an idea by name", async () => {
      vi.mocked(mockApiClient.delete).mockResolvedValue(undefined);

      await service.delete("old-idea");

      expect(mockApiClient.delete).toHaveBeenCalledWith("/ideas/old-idea");
    });
  });

  // [REQ:REQ-P0-004] Test file operations
  describe("getFiles", () => {
    it("fetches files for an idea", async () => {
      const mockFiles = [
        {
          name: "spec.json",
          path: "spec.json",
          type: "file" as const,
          size: 256,
        },
        {
          name: "research",
          path: "research",
          type: "directory" as const,
          children: [
            {
              name: "notes.md",
              path: "research/notes.md",
              type: "file" as const,
              size: 1024,
            },
          ],
        },
      ];
      vi.mocked(mockApiClient.get).mockResolvedValue({
        files: [
          {
            name: "spec.json",
            path: "spec.json",
            type: "file",
            size: "256",
          },
          {
            name: "research",
            path: "research",
            type: "directory",
            children: [
              {
                name: "notes.md",
                path: "research/notes.md",
                type: "file",
                size: "1024",
              },
            ],
          },
        ],
      });

      const result = await service.getFiles("my-idea");

      expect(mockApiClient.get).toHaveBeenCalledWith("/ideas/my-idea/files");
      expect(result).toEqual(mockFiles);
      expect(result).toHaveLength(2);
    });

    it("returns empty array when idea has no files", async () => {
      vi.mocked(mockApiClient.get).mockResolvedValue({ files: [] });

      const result = await service.getFiles("empty-idea");

      expect(mockApiClient.get).toHaveBeenCalledWith("/ideas/empty-idea/files");
      expect(result).toEqual([]);
    });
  });

  // [REQ:REQ-P0-004] Test file content retrieval
  describe("getFileContent", () => {
    it("fetches file content with text response type", async () => {
      const mockContent = "# Research Notes\n\nThis is test content.";
      vi.mocked(mockApiClient.get).mockResolvedValue(mockContent);

      const result = await service.getFileContent("my-idea", "research/notes.md");

      expect(mockApiClient.get).toHaveBeenCalledWith(
        "/ideas/my-idea/files/research/notes.md",
        { responseType: "text" }
      );
      expect(result).toBe(mockContent);
    });

    it("handles nested file paths correctly", async () => {
      vi.mocked(mockApiClient.get).mockResolvedValue("content");

      await service.getFileContent("my-idea", "docs/api/reference.md");

      expect(mockApiClient.get).toHaveBeenCalledWith(
        "/ideas/my-idea/files/docs/api/reference.md",
        { responseType: "text" }
      );
    });
  });

  // [REQ:REQ-P0-004] Test file upload
  describe("uploadFile", () => {
    it("uploads a file with FormData", async () => {
      const mockFile = new File(["test content"], "test.txt", { type: "text/plain" });
      const mockResponse = {
        file: {
          name: "test.txt",
          path: "test.txt",
          type: "file" as const,
          size: "12",
        },
      };
      vi.mocked(mockApiClient.post).mockResolvedValue(mockResponse);

      const result = await service.uploadFile("my-idea", mockFile);

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "/ideas/my-idea/files",
        expect.any(FormData),
        { headers: {} }
      );
      expect(result).toEqual({
        name: "test.txt",
        path: "test.txt",
        type: "file",
        size: 12,
      });
    });

    it("includes path in FormData when specified", async () => {
      const mockFile = new File(["test content"], "notes.md", { type: "text/markdown" });
      const mockResponse = {
        file: {
          name: "notes.md",
          path: "research/notes.md",
          type: "file" as const,
          size: "12",
        },
      };
      vi.mocked(mockApiClient.post).mockResolvedValue(mockResponse);

      const result = await service.uploadFile("my-idea", mockFile, "research");

      // Verify FormData contains both file and path
      const postCall = vi.mocked(mockApiClient.post).mock.calls[0];
      expect(postCall).toBeDefined();
      const formData = postCall?.[1] as FormData;
      expect(formData.has("file")).toBe(true);
      expect(formData.get("path")).toBe("research");
      expect(result.path).toBe("research/notes.md");
    });
  });

  // [REQ:REQ-P0-005] Test queue functionality
  describe("queue", () => {
    it("queues an idea with default generator operation", async () => {
      const mockResponse = {
        idea: {
          name: "my-idea",
          title: "My Idea",
          description: "Test",
          status: "queued" as const,
          priority: 1,
          tags: [],
          created: "2026-01-28T00:00:00Z",
          updated: "2026-01-28T01:00:00Z",
        },
        task_id: "task-12345",
      };
      vi.mocked(mockApiClient.post).mockResolvedValue(mockResponse);

      const result = await service.queue("my-idea");

      expect(mockApiClient.post).toHaveBeenCalledWith("/ideas/my-idea/queue", {
        operation: "generator",
      });
      expect(result.taskId).toBe("task-12345");
      expect(result.idea.status).toBe("queued");
    });

    it("queues an idea with improver operation", async () => {
      const mockResponse = {
        idea: {
          name: "my-idea",
          title: "My Idea",
          description: "Test",
          status: "queued" as const,
          priority: 1,
          tags: [],
          created: "2026-01-28T00:00:00Z",
          updated: "2026-01-28T01:00:00Z",
        },
        task_id: "task-67890",
      };
      vi.mocked(mockApiClient.post).mockResolvedValue(mockResponse);

      const result = await service.queue("my-idea", "improver");

      expect(mockApiClient.post).toHaveBeenCalledWith("/ideas/my-idea/queue", {
        operation: "improver",
      });
      expect(result.taskId).toBe("task-67890");
    });
  });

  describe("research", () => {
    it("posts a research request and returns the run metadata", async () => {
      const mockResponse = {
        taskId: "task-abc",
        runId: "run-123",
        baseUrl: "http://localhost:5555",
        created: "2026-01-28T02:00:00Z",
      };
      vi.mocked(mockApiClient.post).mockResolvedValue(mockResponse);

      const result = await service.research("my-idea", { prompt: "Focus on feasibility" });

      expect(mockApiClient.post).toHaveBeenCalledWith("/ideas/my-idea/research", {
        prompt: "Focus on feasibility",
      });
      expect(result).toEqual(mockResponse);
    });

    it("throws when research response is invalid", async () => {
      vi.mocked(mockApiClient.post).mockResolvedValue({ run_id: "missing" });

      await expect(service.research("my-idea")).rejects.toThrow("Invalid research response");
    });
  });

  describe("error handling", () => {
    it("propagates API errors from list", async () => {
      const error = new Error("Network error");
      vi.mocked(mockApiClient.get).mockRejectedValue(error);

      await expect(service.list()).rejects.toThrow("Network error");
    });

    it("propagates API errors from get", async () => {
      const error = new Error("Not found");
      vi.mocked(mockApiClient.get).mockRejectedValue(error);

      await expect(service.get("missing-idea")).rejects.toThrow("Not found");
    });

    it("propagates API errors from create", async () => {
      const error = new Error("Conflict");
      vi.mocked(mockApiClient.post).mockRejectedValue(error);

      await expect(
        service.create({
          name: "test",
          title: "Test",
          description: "",
          status: "backlog",
          priority: 1,
          tags: [],
        })
      ).rejects.toThrow("Conflict");
    });

    it("propagates API errors from update", async () => {
      const error = new Error("Not found");
      vi.mocked(mockApiClient.put).mockRejectedValue(error);

      await expect(
        service.update("missing", {
          title: "New",
          description: "Updated description",
          status: "ready",
          priority: 2,
          tags: [],
        })
      ).rejects.toThrow("Not found");
    });

    it("propagates API errors from delete", async () => {
      const error = new Error("Server error");
      vi.mocked(mockApiClient.delete).mockRejectedValue(error);

      await expect(service.delete("test")).rejects.toThrow("Server error");
    });

    it("propagates API errors from queue", async () => {
      const error = new Error("Bad request");
      vi.mocked(mockApiClient.post).mockRejectedValue(error);

      await expect(service.queue("completed-idea")).rejects.toThrow("Bad request");
    });
  });
});
