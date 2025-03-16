import { LlmTaskInfo } from "@local/shared";
import { expect } from "chai";
import { isTaskStale, STALE_TASK_THRESHOLD_MS } from "./chatTools.js";

describe("isTaskStale", () => {
    const now = new Date();

    it("should return false for a task with a status that is not \"Running\" or \"Canceling\"", () => {
        const taskInfo = { status: "Completed", lastUpdated: now.toISOString() } as LlmTaskInfo;
        expect(isTaskStale(taskInfo)).to.equal(false);
    });

    it("should return false for a task without a lastUpdated timestamp", () => {
        const taskInfo = { status: "Running" } as LlmTaskInfo;
        expect(isTaskStale(taskInfo)).to.equal(false);
    });

    it("should return true for a \"Running\" task that is stale", () => {
        const staleTime = new Date(now.getTime() - STALE_TASK_THRESHOLD_MS - 1 * 60 * 1000).toISOString(); // 1 minutes after stale
        const taskInfo = { status: "Running", lastUpdated: staleTime } as LlmTaskInfo;
        expect(isTaskStale(taskInfo)).to.equal(true);
    });

    it("should return false for a \"Running\" task that is not stale", () => {
        const notStaleTime = new Date(now.getTime() - STALE_TASK_THRESHOLD_MS + 1 * 60 * 1000).toISOString(); // 1 minutes before stale
        const taskInfo = { status: "Running", lastUpdated: notStaleTime } as LlmTaskInfo;
        expect(isTaskStale(taskInfo)).to.equal(false);
    });

    it("should return true for a \"Canceling\" task that is stale", () => {
        const staleTime = new Date(now.getTime() - STALE_TASK_THRESHOLD_MS - 1 * 60 * 1000).toISOString(); // 1 minutes after stale
        const taskInfo = { status: "Canceling", lastUpdated: staleTime } as LlmTaskInfo;
        expect(isTaskStale(taskInfo)).to.equal(true);
    });

    it("should return false for a \"Canceling\" task that is not stale", () => {
        const notStaleTime = new Date(now.getTime() - STALE_TASK_THRESHOLD_MS + 1 * 60 * 1000).toISOString(); // 1 minutes before stale
        const taskInfo = { status: "Canceling", lastUpdated: notStaleTime } as LlmTaskInfo;
        expect(isTaskStale(taskInfo)).to.equal(false);
    });
});
