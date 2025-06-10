import { type AITaskInfo } from "@vrooli/shared";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { isTaskStale, STALE_TASK_THRESHOLD_MS } from "./chatTools.js";

// Mock the chatTools module to inject our mocked Date
vi.mock("./chatTools.js", async () => {
    const actual = await vi.importActual("./chatTools.js") as any;
    
    return {
        ...actual,
        isTaskStale: (taskInfo: AITaskInfo): boolean => {
            if (!["Running", "Canceling"].includes(taskInfo.status)) {
                return false;
            }
            if (!taskInfo.lastUpdated) {
                return false;
            }
            const lastUpdated = new Date(taskInfo.lastUpdated);
            const now = new Date(); // This will use the mocked time
            const diff = now.getTime() - lastUpdated.getTime();
            return diff > actual.STALE_TASK_THRESHOLD_MS;
        },
    };
});

describe("isTaskStale", () => {
    const mockNow = new Date("2024-01-01T12:00:00.000Z");
    
    beforeEach(() => {
        // Mock Date constructor to return a consistent time
        vi.useFakeTimers();
        vi.setSystemTime(mockNow);
    });
    
    afterEach(() => {
        vi.useRealTimers();
    });

    it("should return false for a task with a status that is not \"Running\" or \"Canceling\"", () => {
        const taskInfo = { 
            status: "Completed", 
            lastUpdated: mockNow.toISOString(),
            label: "Test Task",
            task: "Start",
            taskId: "task-123",
            properties: null
        } as AITaskInfo;
        expect(isTaskStale(taskInfo)).toBe(false);
    });

    it("should return false for a task without a lastUpdated timestamp", () => {
        const taskInfo = { 
            status: "Running",
            label: "Test Task",
            task: "Start",
            taskId: "task-123",
            properties: null
        } as any;
        expect(isTaskStale(taskInfo)).toBe(false);
    });

    it("should return true for a \"Running\" task that is stale", () => {
        const staleTime = new Date(mockNow.getTime() - STALE_TASK_THRESHOLD_MS - 1 * 60 * 1000).toISOString(); // 1 minutes after stale
        const taskInfo = { 
            status: "Running", 
            lastUpdated: staleTime,
            label: "Test Task",
            task: "Start",
            taskId: "task-123",
            properties: null
        } as AITaskInfo;
        expect(isTaskStale(taskInfo)).toBe(true);
    });

    it("should return false for a \"Running\" task that is not stale", () => {
        const notStaleTime = new Date(mockNow.getTime() - STALE_TASK_THRESHOLD_MS + 1 * 60 * 1000).toISOString(); // 1 minutes before stale
        const taskInfo = { 
            status: "Running", 
            lastUpdated: notStaleTime,
            label: "Test Task",
            task: "Start",
            taskId: "task-123",
            properties: null
        } as AITaskInfo;
        expect(isTaskStale(taskInfo)).toBe(false);
    });

    it("should return true for a \"Canceling\" task that is stale", () => {
        const staleTime = new Date(mockNow.getTime() - STALE_TASK_THRESHOLD_MS - 1 * 60 * 1000).toISOString(); // 1 minutes after stale
        const taskInfo = { 
            status: "Canceling", 
            lastUpdated: staleTime,
            label: "Test Task",
            task: "Start",
            taskId: "task-123",
            properties: null
        } as AITaskInfo;
        expect(isTaskStale(taskInfo)).toBe(true);
    });

    it("should return false for a \"Canceling\" task that is not stale", () => {
        const notStaleTime = new Date(mockNow.getTime() - STALE_TASK_THRESHOLD_MS + 1 * 60 * 1000).toISOString(); // 1 minutes before stale
        const taskInfo = { 
            status: "Canceling", 
            lastUpdated: notStaleTime,
            label: "Test Task",
            task: "Start",
            taskId: "task-123",
            properties: null
        } as AITaskInfo;
        expect(isTaskStale(taskInfo)).toBe(false);
    });
});
