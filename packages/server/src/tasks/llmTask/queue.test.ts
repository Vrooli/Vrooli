import { MINUTES_1_MS } from "@local/shared";
import Bull from "../../__mocks__/bull";
import { LlmTaskProcessPayload, changeLlmTaskStatus, processLlmTask, setupLlmTaskQueue } from "./queue";

describe("processLlmTask", () => {
    let llmTaskQueue;

    beforeAll(async () => {
        llmTaskQueue = new Bull("command");
        await setupLlmTaskQueue();
    });

    beforeEach(() => {
        Bull.resetMock();
    });

    it("enqueues an llm task", async () => {
        const testTask = {
            chatId: "chatId123",
            language: "en",
            taskInfo: {
                id: "task123",
                task: "test",
                properties: {},
            },
            userData: {
                id: "user123",
            },
        };

        processLlmTask(testTask as LlmTaskProcessPayload);

        // Check if the add function was called
        expect(llmTaskQueue.add).toHaveBeenCalled();

        // Extract the actual data passed to the mock
        const actualData = llmTaskQueue.add.mock.calls[0][0];

        // Validate that the data in the queue matches the provided payload 
        // plus a status of "scheduled"
        expect(actualData).toMatchObject({ ...testTask, status: "scheduled" });

        // Additionally, check if the jobId and timeout options are set correctly
        const options = llmTaskQueue.add.mock.calls[0][1];
        expect(options).toEqual({ jobId: testTask.taskInfo.id, timeout: MINUTES_1_MS });
    });
});

describe("changeLlmTaskStatus", () => {
    let llmTaskQueue;

    beforeAll(async () => {
        llmTaskQueue = new Bull("command");
        await setupLlmTaskQueue();
    });

    beforeEach(() => {
        Bull.resetMock();
    });

    test("should reschedule a failed task", async () => {
        // Add a mock job with status "failed"
        const mockJob = {
            id: "jobId123",
            data: { userData: { id: "userId456" }, status: "failed" },
            getState: jest.fn().mockResolvedValue("failed"),
            update: jest.fn(),
        };
        Bull.__addMockJob(mockJob);

        const result = await changeLlmTaskStatus("jobId123", "scheduled", "userId456");

        expect(result).toEqual({ success: true, message: "Task rescheduled." });
        expect(mockJob.update).toHaveBeenCalledWith({ ...mockJob.data, status: "scheduled" });
        expect(llmTaskQueue.add).toHaveBeenCalledWith(mockJob.data);
    });

    test("should cancel a task that exists", async () => {
        // Add a mock job with status "scheduled"
        const mockJob = {
            id: "jobId123",
            data: { userData: { id: "userId456" }, status: "scheduled" },
            update: jest.fn(),
            remove: jest.fn(),
        };
        Bull.__addMockJob(mockJob);

        const result = await changeLlmTaskStatus("jobId123", "canceled", "userId456");

        expect(result).toEqual({ success: true, message: "Task canceled." });
        expect(mockJob.update).toHaveBeenCalledWith({ ...mockJob.data, status: "canceled" });
        expect(mockJob.remove).toHaveBeenCalled();
    });

    test("should consider it a success if trying to cancel a task that does not exist", async () => {
        const result = await changeLlmTaskStatus("jobId123", "canceled", "userId123");

        expect(result).toEqual({ success: true, message: "Task not found but considered canceled." });
    });

    test("should fail to reschedule a task if not in the correct state", async () => {
        // Add a mock job with status "completed"
        const mockJob = {
            id: "jobId123",
            data: { userData: { id: "userId456" }, status: "completed" },
            getState: jest.fn().mockResolvedValue("completed"),
        };
        Bull.__addMockJob(mockJob);

        const result = await changeLlmTaskStatus("jobId123", "scheduled", "userId456");

        expect(result).toEqual({ success: false, message: "LLM task with jobId jobId123 cannot be rescheduled from state completed." });
    });

    test("should not allow unauthorized user to change task status", async () => {
        // Add a mock job with status "scheduled"
        const mockJob = {
            id: "jobId123",
            data: { userData: { id: "userId456" }, status: "scheduled" },
        };
        Bull.__addMockJob(mockJob);

        const result = await changeLlmTaskStatus("jobId123", "canceled", "unauthorizedUserId");

        expect(result).toEqual({ success: false, message: "User unauthorizedUserId is not authorized to change the status of job jobId123." });
    });
});
