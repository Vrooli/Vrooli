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
});
