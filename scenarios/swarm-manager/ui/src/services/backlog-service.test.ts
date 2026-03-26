import { describe, it, expect, vi, beforeEach } from "vitest";
import { createBacklogService, type IBacklogService } from "./backlog-service";
import type { IApiClient } from "../lib/api-client";
import type { BacklogItem } from "../types";

/**
 * Backlog Service Tests
 *
 * These tests demonstrate the seam pattern - by injecting a mock API client,
 * we can test the service layer in isolation without needing to mock HTTP.
 */

// [REQ:MOD-P0-001] Test backlog service seam
describe("Backlog Service", () => {
  let mockApiClient: IApiClient;
  let service: IBacklogService;

  beforeEach(() => {
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };

    service = createBacklogService(mockApiClient);
  });

  describe("list", () => {
    it("fetches backlog items from the correct endpoint", async () => {
      const mockItems: BacklogItem[] = [
        {
          name: "test-idea",
          title: "Test Idea",
          description: "A test",
          status: "backlog",
          priority: 1,
          tags: [],
          created: "2026-01-28T00:00:00Z",
          updated: "2026-01-28T00:00:00Z",
          kind: "idea",
        },
      ];
      vi.mocked(mockApiClient.get).mockResolvedValue({ items: mockItems });

      const result = await service.list();

      expect(mockApiClient.get).toHaveBeenCalledWith("/backlog");
      expect(result).toEqual(mockItems);
    });
  });

  describe("get", () => {
    it("fetches a single backlog item", async () => {
      const mockItem: BacklogItem = {
        name: "my-idea",
        title: "My Idea",
        description: "Details",
        status: "ready",
        priority: 2,
        tags: ["test"],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
        kind: "idea",
      };
      vi.mocked(mockApiClient.get).mockResolvedValue({ item: mockItem });

      const result = await service.get("idea", "my-idea");

      expect(mockApiClient.get).toHaveBeenCalledWith("/backlog/idea/my-idea");
      expect(result).toEqual(mockItem);
    });
  });

  describe("create", () => {
    it("posts a new backlog item", async () => {
      const newItem = {
        name: "new-idea",
        title: "New Idea",
        description: "A new idea",
        status: "backlog" as const,
        priority: 1,
        tags: ["new"],
        kind: "idea" as const,
      };
      const createdItem: BacklogItem = {
        ...newItem,
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
      };
      vi.mocked(mockApiClient.post).mockResolvedValue({ item: createdItem });

      const result = await service.create(newItem);

      expect(mockApiClient.post).toHaveBeenCalledWith("/backlog", {
        name: "new-idea",
        title: "New Idea",
        description: "A new idea",
        priority: 1,
        tags: ["new"],
        kind: "idea",
        depends_on: [],
        acceptance_allow: [],
        acceptance_deny: [],
      });
      expect(result).toEqual(createdItem);
    });
  });

  describe("update", () => {
    it("puts an updated backlog item", async () => {
      const updates = {
        title: "Updated Title",
        description: "Original description",
        status: "backlog" as const,
        priority: 1,
        tags: [],
        researchTarget: undefined,
      };
      const updatedItem: BacklogItem = {
        name: "my-idea",
        title: "Updated Title",
        description: "Original description",
        status: "backlog",
        priority: 1,
        tags: [],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T01:00:00Z",
        kind: "idea",
      };
      vi.mocked(mockApiClient.put).mockResolvedValue({ item: updatedItem });

      const result = await service.update("idea", "my-idea", updates);

      expect(mockApiClient.put).toHaveBeenCalledWith(
        "/backlog/idea/my-idea",
        expect.objectContaining({
          title: "Updated Title",
          description: "Original description",
          status: "backlog",
          priority: 1,
          tags: [],
        })
      );
      expect(result).toEqual(updatedItem);
    });
  });

  describe("delete", () => {
    it("deletes a backlog item", async () => {
      vi.mocked(mockApiClient.delete).mockResolvedValue(undefined);

      await service.delete("idea", "old-idea");

      expect(mockApiClient.delete).toHaveBeenCalledWith("/backlog/idea/old-idea");
    });
  });

  describe("file operations", () => {
    it("renames a file via patch endpoint", async () => {
      vi.mocked(mockApiClient.patch).mockResolvedValue({
        file: {
          name: "notes-renamed.md",
          path: "notes-renamed.md",
          type: "file",
          size: 42,
        },
      });

      const result = await service.renameFile("idea", "my-idea", "notes.md", "notes-renamed.md");

      expect(mockApiClient.patch).toHaveBeenCalledWith(
        "/backlog/idea/my-idea/files",
        expect.objectContaining({
          operation: "rename",
          source_path: "notes.md",
          destination_path: "notes-renamed.md",
        })
      );
      expect(result.file?.path).toBe("notes-renamed.md");
    });

    it("deletes a file via patch endpoint", async () => {
      vi.mocked(mockApiClient.patch).mockResolvedValue({
        deleted_path: "notes.md",
      });

      const result = await service.deleteFile("idea", "my-idea", "notes.md");

      expect(mockApiClient.patch).toHaveBeenCalledWith(
        "/backlog/idea/my-idea/files",
        expect.objectContaining({
          operation: "delete",
          source_path: "notes.md",
        })
      );
      expect(result.deletedPath).toBe("notes.md");
    });
  });

  describe("workshopSave", () => {
    it("calls the correct endpoint with round data", async () => {
      vi.mocked(mockApiClient.post).mockResolvedValue({
        file: { name: "round-001.json", path: "workshop/round-001.json", type: "file", size: 200 },
        auto_advance: { triggered: false, reason: "ready" },
      });

      const result = await service.workshopSave("idea", "my-idea", 1, '{"round":1}');

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "/backlog/idea/my-idea/workshop/save",
        { round_number: 1, content: '{"round":1}' }
      );
      expect(result.file.name).toBe("round-001.json");
      expect(result.autoAdvance.triggered).toBe(false);
      expect(result.autoAdvance.reason).toBe("ready");
    });

    it("returns auto-advance data when triggered", async () => {
      vi.mocked(mockApiClient.post).mockResolvedValue({
        file: { name: "round-002.json", path: "workshop/round-002.json", type: "file", size: 300 },
        auto_advance: { triggered: true, run_id: "run-123", task_id: "task-456", reason: "not_ready" },
      });

      const result = await service.workshopSave("idea", "my-idea", 2, '{"round":2}');

      expect(result.autoAdvance.triggered).toBe(true);
      expect(result.autoAdvance.runId).toBe("run-123");
      expect(result.autoAdvance.taskId).toBe("task-456");
      expect(result.autoAdvance.reason).toBe("not_ready");
    });

  });
});
