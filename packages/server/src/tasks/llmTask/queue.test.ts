import { MINUTES_1_MS, TaskStatus } from "@local/shared";
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
            __process: "LlmTask",
            chatId: "chatId123",
            language: "en",
            status: TaskStatus.Scheduled,
            taskInfo: {
                task: "test",
                taskId: "task123",
                properties: {},
            } as any,
            userData: {
                id: "user123",
            },
        } as const;

        processLlmTask(testTask as LlmTaskProcessPayload);

        // Check if the add function was called
        expect(llmTaskQueue.add).toHaveBeenCalled();

        // Extract the actual data passed to the mock
        const actualData = llmTaskQueue.add.mock.calls[0][0];

        // Validate that the data in the queue matches the provided payload 
        // plus a status of "Scheduled"
        expect(actualData).toMatchObject({ ...testTask, status: "Scheduled" });

        // Additionally, check if the jobId and timeout options are set correctly
        const options = llmTaskQueue.add.mock.calls[0][1];
        expect(options).toEqual({ jobId: testTask.taskInfo.taskId, timeout: MINUTES_1_MS });
    });
});

describe("changeLlmTaskStatus", () => {
    beforeEach(() => {
        Bull.resetMock();
    });

    test("should update a failed task that exists", async () => {
        // Add a mock job with status "Failed"
        const mockJob = {
            id: "jobId123",
            data: { userData: { id: "userId456" }, status: "Failed" },
            getState: jest.fn().mockResolvedValue("failed"),
            update: jest.fn(),
        };
        Bull.__addMockJob(mockJob);

        const result = await changeLlmTaskStatus("jobId123", "Scheduled", "userId456");

        expect(result).toEqual({ __typename: "Success" as const, success: true });
        expect(mockJob.update).toHaveBeenCalledWith({ ...mockJob.data, status: "Scheduled" });
    });

    test("should consider it a success when updating to 'Failed' and the task doesn't exist", async () => {
        const result = await changeLlmTaskStatus("jobId123", "Failed", "userId456");

        expect(result).toEqual({ __typename: "Success" as const, success: true });
    });

    test("should consider it a success when updating to 'Completed' and the task doesn't exist", async () => {
        const result = await changeLlmTaskStatus("jobId123", "Completed", "userId456");

        expect(result).toEqual({ __typename: "Success" as const, success: true });
    });

    test("should consider it a success when updating to 'Suggested' and the task doesn't exist", async () => {
        const result = await changeLlmTaskStatus("jobId123", "Suggested", "userId456");

        expect(result).toEqual({ __typename: "Success" as const, success: true });
    });

    test("should consider it a fail when updating to 'Running' and the task doesn't exist", async () => {
        const result = await changeLlmTaskStatus("jobId123", "Running", "userId456");

        expect(result).toEqual({ __typename: "Success" as const, success: false });
    });

    test("should update a scheduled task that exists", async () => {
        // Add a mock job with status "Scheduled"
        const mockJob = {
            id: "jobId123",
            data: { userData: { id: "userId456" }, status: "Scheduled" },
            update: jest.fn(),
        };
        Bull.__addMockJob(mockJob);

        const result = await changeLlmTaskStatus("jobId123", "Running", "userId456");

        expect(result).toEqual({ __typename: "Success" as const, success: true });
        expect(mockJob.update).toHaveBeenCalledWith({ ...mockJob.data, status: "Running" });
    });

    test("should fail to reschedule a task if not in the correct state", async () => {
        // Add a mock job with status "Completed"
        const mockJob = {
            id: "jobId123",
            data: { userData: { id: "userId456" }, status: "Completed" },
            getState: jest.fn().mockResolvedValue("completed"),
        };
        Bull.__addMockJob(mockJob);

        const result = await changeLlmTaskStatus("jobId123", "Scheduled", "userId456");

        expect(result).toEqual({ __typename: "Success" as const, success: false });
    });

    test("should not allow unauthorized user to change task status", async () => {
        // Add a mock job with status "Scheduled"
        const mockJob = {
            id: "jobId123",
            data: { userData: { id: "userId456" }, status: "Scheduled" },
        };
        Bull.__addMockJob(mockJob);

        const result = await changeLlmTaskStatus("jobId123", "Running", "unauthorizedUserId");

        expect(result).toEqual({ __typename: "Success" as const, success: false });
    });
});
