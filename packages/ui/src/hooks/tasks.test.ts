/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect, beforeAll, beforeEach, afterEach, vi } from "vitest";

// Mock the device module before importing anything that uses it
vi.mock("../utils/display/device.js");
vi.mock("../utils/display/chatTools.js", () => ({
    isTaskStale: vi.fn(),
    taskToTaskInfo: vi.fn(),
    STALE_TASK_THRESHOLD_MS: 600000,
}));

import { renderHook } from "@testing-library/react";
import { LlmTask, TaskStatus, nanoid, noop, type LlmTaskInfo } from "@vrooli/shared";
import { act } from "react";
import { PubSub as PubSubMock } from "../utils/__mocks__/pubsub.js";
import { taskToTaskInfo } from "../utils/display/chatTools.js";
import { PubSub } from "../utils/pubsub.js";
import { DEFAULT_ACTIVE_TASK, DEFAULT_ACTIVE_TASK_ID, createUpdatedTranslations, generateContextLabel, getAutoFillTranslationData, useChatTasks, type UseChatTaskReturn } from "./tasks.js";

describe("getAutoFillTranslationData", () => {
    it("should return correct translation data omitting specific fields", () => {
        const values = {
            translations: [
                { language: "en", content: "Hello", id: "123", __typename: "Translation", otherField: "value" },
            ],
        };
        const language = "en";

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const result = getAutoFillTranslationData(values, language) as any;

        expect(result).toEqual({ content: "Hello", otherField: "value" });
        expect(result.id).toBeUndefined();
        expect(result.language).toBeUndefined();
        expect(result.__typename).toBeUndefined();
    });

    it("should return any translation data if the language does not match", () => {
        const values = {
            translations: [
                { language: "fr", content: "Bonjour", id: "124", __typename: "Translation" },
            ],
        };
        const language = "en";

        const result = getAutoFillTranslationData(values, language);

        expect(result).toEqual({ content: "Bonjour" });
    });

    it("should handle null translations gracefully", () => {
        const values = {
            translations: null,
        };
        const language = "en";

        const result = getAutoFillTranslationData(values, language);

        expect(result).toEqual({});
    });

    it("should handle an empty translations array", () => {
        const values = {
            translations: [],
        };
        const language = "en";

        const result = getAutoFillTranslationData(values, language);

        expect(result).toEqual({});
    });

    it("should omit the specified fields even if they are not present", () => {
        const values = {
            translations: [
                { language: "en", content: "Hello", id: "123" }, // __typename is not present
            ],
        };
        const language = "en";

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const result = getAutoFillTranslationData(values, language) as any;

        expect(result).toEqual({ content: "Hello" });
        expect(result.id).toBeUndefined();
        expect(result.language).toBeUndefined();
        expect(result.__typename).toBeUndefined(); // Ensuring it handles missing fields properly
    });
});

describe("createUpdatedTranslations", () => {
    it("should return empty translations if initial translations are undefined", () => {
        const values = {};
        const autofillData = { name: "New Name", description: "New Description" };
        const language = "en";
        const translatedFields = ["description"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations).toEqual([]);
        expect(result.rest).toEqual({ name: "New Name" });
    });

    it("should update the correct translation based on language", () => {
        const values = {
            translations: [
                { language: "en", content: "Old Content", id: "1" },
                { language: "fr", content: "Contenu", id: "2" },
            ],
        };
        const autofillData = { content: "New Content" };
        const language = "en";
        const translatedFields = ["content"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations).toEqual([
            { language: "en", content: "New Content", id: "1" },
            { language: "fr", content: "Contenu", id: "2" },
        ]);
        expect(result.rest).toEqual({});
    });

    it("should handle non-existent language by updating the first translation", () => {
        const values = {
            translations: [
                { language: "es", content: "Contenido", id: "1" },
            ],
        };
        const autofillData = { content: "New Content" };
        const language = "de"; // German is not present
        const translatedFields = ["content"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations[0]).toEqual({
            language: "es", content: "New Content", id: "1",
        });
    });

    it("should separate translated and non-translated fields appropriately", () => {
        const values = {
            translations: [
                { language: "en", content: "Old Content", id: "1" },
            ],
        };
        const autofillData = { content: "New Content", name: "New Name" };
        const language = "en";
        const translatedFields = ["content"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations[0]).toEqual({
            language: "en", content: "New Content", id: "1",
        });
        expect(result.rest).toEqual({ name: "New Name" });
    });

    it("should return empty updatedTranslations if translations array is empty", () => {
        const values = { translations: [] };
        const autofillData = { content: "New Content" };
        const language = "en";
        const translatedFields = ["content"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations).toEqual([]);
        expect(result.rest).toEqual({});
    });

    it("should only update specified translated fields", () => {
        const values = {
            translations: [
                { language: "en", title: "Old Title", content: "Old Content", id: "1" },
            ],
        };
        const autofillData = { title: "New Title", content: "New Content", extra: "Extra Info" };
        const language = "en";
        const translatedFields = ["title"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations[0]).toEqual({
            language: "en", title: "New Title", content: "Old Content", id: "1",
        });
        expect(result.rest).toEqual({ content: "New Content", extra: "Extra Info" });
    });
});

describe("generateContextLabel", () => {
    const CONTEXT_LABEL_LENGTH = 20; // Change to match the actual value in the function

    it("should return the entire string if it is shorter than the limit", () => {
        const input = "Short string";
        const expected = "Short string";
        expect(generateContextLabel(input)).toBe(expected);
    });

    it("should truncate the string and add an ellipsis if it is longer than the limit", () => {
        const input = "This is a very long string that should be truncated.";
        const expected = input.slice(0, CONTEXT_LABEL_LENGTH - 1) + "…";
        expect(generateContextLabel(input)).toBe(expected);
    });

    it("should return the stringified number if it is shorter than the limit", () => {
        const input = 12345;
        const expected = "12345";
        expect(generateContextLabel(input)).toBe(expected);
    });

    it("should truncate and add an ellipsis to the stringified number if it is longer than the limit", () => {
        // eslint-disable-next-line @typescript-eslint/no-loss-of-precision
        const input = -123456789012345678901234567890.2343243;
        const expected = input.toString().slice(0, CONTEXT_LABEL_LENGTH - 1) + "…";
        expect(generateContextLabel(input)).toBe(expected);
    });

    it("should return the stringified boolean", () => {
        expect(generateContextLabel(true)).toBe("true");
        expect(generateContextLabel(false)).toBe("false");
    });

    it("should label arrays as \"Array\"", () => {
        expect(generateContextLabel([1, 2, 3])).toBe("Array");
    });

    it("should label objects as \"Object\"", () => {
        expect(generateContextLabel({ key: "value" })).toBe("Object");
    });

    it("should handle null values correctly", () => {
        expect(generateContextLabel(null)).toBe("Value");
    });

    it("should handle undefined values correctly", () => {
        expect(generateContextLabel(undefined)).toBe("Value");
    });

    it("should handle function values correctly", () => {
        expect(generateContextLabel(noop)).toBe("Value");
    });

    it("handles empty strings correctly", () => {
        expect(generateContextLabel("")).toBe("");
    });

    it("handles empty objects correctly", () => {
        expect(generateContextLabel({})).toBe("Object");
    });

    it("handles empty arrays correctly", () => {
        expect(generateContextLabel([])).toBe("Array");
    });
});

const task1 = {
    taskId: "Task1",
    label: "Task 1",
    task: "Task1",
} as unknown as LlmTaskInfo;
const task2 = {
    taskId: "Task2",
    label: "Task 2",
    task: "Task2",
} as unknown as LlmTaskInfo;
const task3 = {
    taskId: "Task3",
    label: "Task 3",
    task: "Task3",
} as unknown as LlmTaskInfo;
const task4 = {
    taskId: "Task4",
    label: "Task 4",
    task: "Task4",
} as unknown as LlmTaskInfo;
const task5 = {
    taskId: "Task5",
    label: "Task 5",
    task: "Task5",
} as unknown as LlmTaskInfo;

/**
 * Checks that the total number of contexts is correct
 */
function validateTotalContexts(data: UseChatTaskReturn, expected: number) {
    const totalContexts = Object.keys(data.contexts).reduce((acc, key) => acc + data.contexts[key].length, 0);
    // @ts-ignore: expect-message
    expect(totalContexts, `Total contexts is incorrect. Expected ${expected}, got ${totalContexts}`).toBe(expected);
}

/**
 * Checks that the contexts for a task are present and in the correct order
 */
function validateContextsForTask(data: UseChatTaskReturn, taskId: string, labels: string[]) {
    const contexts = data.contexts[taskId];
    expect(contexts).toHaveLength(labels.length);
    for (let i = 0; i < labels.length; i++) {
        // @ts-ignore: expect-message
        expect(contexts[i].label, `Context ${i} for task ${taskId} has incorrect label`).toBe(labels[i]);
    }
}

describe("useChatTasks", () => {

    beforeAll(() => {
        // Mock console.error to avoid cluttering test output
        console.error = vi.fn();
    });

    let originalPubSubMethods;
    beforeEach(() => {
        vi.useFakeTimers();
        originalPubSubMethods = { ...PubSub };
        Object.assign(PubSub, PubSubMock);
        PubSubMock.resetMock();
        vi.clearAllMocks();
    });

    afterEach(() => {
        Object.assign(PubSub, originalPubSubMethods);
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    describe("initial state", () => {
        it("activeTask should initially be the start task", () => {
            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));
            expect(result.current.activeTask.task).toBe(LlmTask.Start);
        });
        it("contexts should initially have no data", () => {
            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));
            validateTotalContexts(result.current, 0);
        });
        it("inactiveTasks should initially be an empty array", () => {
            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));
            expect(result.current.inactiveTasks).toEqual([]);
        });
    });

    describe("clears data on chat ID change", () => {
        it("clears contexts", async () => {
            const { result, rerender } = renderHook((props: { chatId: string }) => useChatTasks(props), { initialProps: { chatId: "main" } });

            // Add context data first
            await act(async () => {
                PubSub.get().publish("chatTask", {
                    chatId: "main",
                    contexts: {
                        add: {
                            behavior: "add",
                            connect: {
                                __type: "taskType",
                                data: LlmTask.Start,
                            },
                            value: [{
                                id: nanoid(),
                                label: "Boop",
                                data: "Boop",
                            }, {
                                id: nanoid(),
                                label: "Beep",
                                data: "Beep",
                            }],
                        },
                    },
                });
            });

            // Make sure the data was added
            validateTotalContexts(result.current, 2);

            // Change the chat ID
            rerender({ chatId: "new" });

            // Make sure the data was cleared
            validateTotalContexts(result.current, 0);
        });
        it("clears active task", async () => {
            const { result, rerender } = renderHook((props: { chatId: string }) => useChatTasks(props), { initialProps: { chatId: "main" } });

            // Add active task first
            await act(async () => {
                PubSub.get().publish("chatTask", {
                    chatId: "main",
                    tasks: {
                        add: {
                            active: {
                                behavior: "force",
                                value: { ...task1 },
                            },
                        },
                    },
                });
            });

            // Make sure the task was added
            expect(result.current.activeTask).toEqual(task1);

            // Change the chat ID
            rerender({ chatId: "new" });

            // Make sure the task was cleared
            expect(result.current.activeTask.task).toBe(LlmTask.Start);
        });
        it("clears inactive tasks", async () => {
            const { result, rerender } = renderHook((props: { chatId: string }) => useChatTasks(props), { initialProps: { chatId: "main" } });

            // Add inactive tasks first
            await act(async () => {
                PubSub.get().publish("chatTask", {
                    chatId: "main",
                    tasks: {
                        add: {
                            inactive: {
                                behavior: "replaceAll",
                                value: [task1, task2],
                            },
                        },
                    },
                });
            });

            // Make sure the tasks were added
            expect(result.current.inactiveTasks).toEqual([task1, task2]);

            // Change the chat ID
            rerender({ chatId: "new" });

            // Make sure the tasks were cleared
            expect(result.current.inactiveTasks).toEqual([]);
        });
    });

    describe("handling PubSub events", () => {
        describe("contexts", () => {
            describe("add", () => {
                describe("behavior", () => {
                    it("ignore if no matching task", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.ApiFind,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: "Boop",
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 0);
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                    it("replace with no existing data", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "replace",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: "Boop",
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop"]);
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                    it("add with no existing data", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: -420.69,
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop"]);
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                    it("replace with existing data", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Old value 1",
                                            data: { key: "old value 1" },
                                        }, {
                                            id: nanoid(),
                                            label: "Old value 2",
                                            data: { key: "old value 2" },
                                        }, {
                                            id: nanoid(),
                                            label: "Old value 3",
                                            data: { key: "old value 3" },
                                        }],
                                    },
                                },
                            });
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "replace",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: "Boop",
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop"]);
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                    it("add with existing data", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Old value 1",
                                            data: { key: "old value 1" },
                                        }, {
                                            id: nanoid(),
                                            label: "Old value 2",
                                            data: { key: "old value 2" },
                                        }, {
                                            id: nanoid(),
                                            label: "Old value 3",
                                            data: { key: "old value 3" },
                                        }],
                                    },
                                },
                            });
                        });
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: "Boop",
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 4);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop", "Old value 1", "Old value 2", "Old value 3"]);
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                });
                describe("connect", () => {
                    it("taskId", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        // Add a task first
                        const task = taskToTaskInfo(LlmTask.RoutineFind);
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                tasks: {
                                    add: {
                                        inactive: {
                                            behavior: "replaceTaskId",
                                            value: [task],
                                        },
                                    },
                                },
                            });
                        });

                        // Add context data for the task
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskId",
                                            data: task.taskId,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: "Boop",
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, task.taskId, ["Boop"]);
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([task]);
                    });
                    it("taskType", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        // Add a task first
                        const task = taskToTaskInfo(LlmTask.RoutineFind);
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                tasks: {
                                    add: {
                                        active: {
                                            behavior: "force",
                                            value: task,
                                        },
                                    },
                                },
                            });
                        });

                        // Add context data for the task
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: task.task,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: "Boop",
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, task.taskId, ["Boop"]);
                        expect(result.current.activeTask.task).toBe(task.task);
                        expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                    });
                    describe("contextId", () => {
                        it("did not previously exist", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            // Add a task first
                            const task = taskToTaskInfo(LlmTask.ScheduleAdd);
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            active: {
                                                behavior: "force",
                                                value: task,
                                            },
                                        },
                                    },
                                });
                            });

                            // Add context by context ID
                            const contexts = [{
                                id: nanoid(),
                                label: "Boop",
                                data: "Boop",
                            }];
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    contexts: {
                                        add: {
                                            behavior: "add",
                                            connect: {
                                                __type: "contextId",
                                                data: contexts[0].id,
                                            },
                                            value: contexts,
                                        },
                                    },
                                });
                            });

                            validateTotalContexts(result.current, 1);
                            validateContextsForTask(result.current, task.taskId, ["Boop"]);
                            expect(result.current.activeTask.task).toBe(task.task);
                            expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                        });
                        it("did previously exist", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            // Add a task first
                            const task = taskToTaskInfo(LlmTask.ScheduleAdd);
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            active: {
                                                behavior: "force",
                                                value: task,
                                            },
                                        },
                                    },
                                });
                            });

                            // Add context by context ID
                            const originalContexts = [{
                                id: nanoid(),
                                label: "Boop",
                                data: "Boop",
                            }];
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    contexts: {
                                        add: {
                                            behavior: "add",
                                            connect: {
                                                __type: "contextId",
                                                data: originalContexts[0].id,
                                            },
                                            value: originalContexts,
                                        },
                                    },
                                });
                            });

                            // Add to same context ID, but with different data
                            const newContexts = [{
                                id: originalContexts[0].id,
                                label: "Beep",
                                data: "Beep",
                            }];
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    contexts: {
                                        add: {
                                            behavior: "add",
                                            connect: {
                                                __type: "contextId",
                                                data: newContexts[0].id,
                                            },
                                            value: newContexts,
                                        },
                                    },
                                });
                            });

                            validateTotalContexts(result.current, 1);
                            expect(result.current.contexts[task.taskId]).toEqual(newContexts);
                            expect(result.current.activeTask.task).toBe(task.task);
                            expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                        });
                    });
                });
                describe("value", () => {
                    it("boolean", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: true,
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop"]);
                        expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID][0].data).toBe(true);
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                    it("number", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: -420.69,
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop"]);
                        expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID][0].data).toBe(-420.69);
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                    it("string", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: "Boop",
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop"]);
                        expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID][0].data).toBe("Boop");
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                    it("object", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: { key: "value" },
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop"]);
                        expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID][0].data).toEqual({ key: "value" });
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                    it("array", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: [1, 2, 3],
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop"]);
                        expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID][0].data).toEqual([1, 2, 3]);
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                    it("null", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: {
                                            __type: "taskType",
                                            data: LlmTask.Start,
                                        },
                                        value: [{
                                            id: nanoid(),
                                            label: "Boop",
                                            data: null,
                                        }],
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        validateContextsForTask(result.current, DEFAULT_ACTIVE_TASK_ID, ["Boop"]);
                        expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID][0].data).toBeNull();
                        expect(result.current.activeTask.task).toBe(LlmTask.Start);
                        expect(result.current.inactiveTasks).toEqual([]);
                    });
                });
            });
            describe("update", () => {
                it("all values existing", async () => {
                    const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                    // Add task and existing data first
                    const task = taskToTaskInfo(LlmTask.ScheduleUpdate);
                    const originalContexts = [{
                        id: nanoid(),
                        label: "Old value 1",
                        data: { key: "old value 1" },
                    }, {
                        id: nanoid(),
                        label: "Old value 2",
                        data: { key: "old value 2" },
                    }, {
                        id: nanoid(),
                        label: "Old value 3",
                        data: { key: "old value 3" },
                    }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                add: {
                                    inactive: {
                                        behavior: "replaceAll",
                                        value: [task],
                                    },
                                },
                            },
                        });
                    });
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                add: {
                                    behavior: "add",
                                    connect: {
                                        __type: "taskId",
                                        data: task.taskId,
                                    },
                                    value: originalContexts,
                                },
                            },
                        });
                    });

                    validateTotalContexts(result.current, 3);
                    expect(result.current.contexts[task.taskId]).toEqual(originalContexts);
                    expect(result.current.activeTask.task).toBe(LlmTask.Start);
                    expect(result.current.inactiveTasks).toEqual([task]);

                    // Update some data
                    const updateContextData = [{
                        id: originalContexts[1].id,
                        label: "New value 2",
                        data: "hello",
                    }, {
                        id: originalContexts[2].id,
                        label: "New value 3",
                        // "data" skipped to test that it keeps the original data
                    }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                update: updateContextData,
                            },
                        });
                    });

                    validateTotalContexts(result.current, 3);
                    const expectedContexts = [{
                        id: originalContexts[0].id,
                        label: "Old value 1",
                        data: { key: "old value 1" },
                    }, {
                        id: originalContexts[1].id,
                        label: "New value 2",
                        data: "hello",
                    }, {
                        id: originalContexts[2].id,
                        label: "New value 3",
                        data: { key: "old value 3" },
                    }];
                    expect(result.current.contexts[task.taskId]).toEqual(expectedContexts);
                    expect(result.current.activeTask.task).toBe(LlmTask.Start);
                    expect(result.current.inactiveTasks).toEqual([task]);
                });
                it("some values existing", async () => {
                    const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                    // Add and existing data first
                    const originalContexts = [{
                        id: nanoid(),
                        label: "Old value 1",
                        data: { key: "old value 1" },
                    }, {
                        id: nanoid(),
                        label: "Old value 2",
                        data: { key: "old value 2" },
                    }, {
                        id: nanoid(),
                        label: "Old value 3",
                        data: { key: "old value 3" },
                    }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                add: {
                                    behavior: "add",
                                    connect: {
                                        __type: "taskType",
                                        data: LlmTask.Start,
                                    },
                                    value: originalContexts,
                                },
                            },
                        });
                    });

                    validateTotalContexts(result.current, 3);
                    expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID]).toEqual(originalContexts);
                    expect(result.current.activeTask.task).toBe(LlmTask.Start);
                    expect(result.current.inactiveTasks).toEqual([]);

                    // Update one value and try to update one that doesn't exist
                    const updateContextData = [{
                        id: originalContexts[1].id,
                        label: "New value 2",
                        data: "hello",
                    }, {
                        id: nanoid(),
                        label: "New value 3",
                        data: 2319,
                    }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                update: updateContextData,
                            },
                        });
                    });

                    validateTotalContexts(result.current, 3);
                    const expectedContexts = [{
                        id: originalContexts[0].id,
                        label: "Old value 1",
                        data: { key: "old value 1" },
                    }, {
                        id: originalContexts[1].id,
                        label: "New value 2",
                        data: "hello",
                    }, {
                        id: originalContexts[2].id,
                        label: "Old value 3",
                        data: { key: "old value 3" },
                    }];
                    expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID]).toEqual(expectedContexts);
                    expect(result.current.activeTask.task).toBe(LlmTask.Start);
                    expect(result.current.inactiveTasks).toEqual([]);
                });
            });
            describe("remove", () => {
                it("by task type", async () => {
                    const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                    // Add contexts to a task type first
                    const originalContexts = [{
                        id: nanoid(),
                        label: "Old value 1",
                        data: { key: "old value 1" },
                    }, {
                        id: nanoid(),
                        label: "Old value 2",
                        data: { key: "old value 2" },
                    }, {
                        id: nanoid(),
                        label: "Old value 3",
                        data: { key: "old value 3" },
                    }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                add: {
                                    behavior: "add",
                                    connect: { __type: "taskType", data: LlmTask.Start },
                                    value: originalContexts,
                                },
                            },
                        });
                    });
                    // Add contexts to other task types to make sure they don't get removed
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                add: {
                                    inactive: {
                                        behavior: "onlyIfNoTaskId",
                                        value: [task1, task2],
                                    },
                                },
                            },
                        });
                    });
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                add: {
                                    behavior: "add",
                                    connect: { __type: "taskType", data: task1.task },
                                    value: [{
                                        id: nanoid(),
                                        label: "Value 1",
                                        data: "Data 1",
                                    }, {
                                        id: nanoid(),
                                        label: "Value 2",
                                        data: "Data 2",
                                    }],
                                },
                            },
                        });
                    });

                    validateTotalContexts(result.current, originalContexts.length + 2);
                    expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID]).toEqual(originalContexts);
                    expect(result.current.activeTask.task).toBe(LlmTask.Start);
                    expect(result.current.inactiveTasks).toEqual([task1, task2]);

                    // Remove context by task type
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                remove: [{ __type: "taskType", data: LlmTask.Start }],
                            },
                        });
                    });

                    validateTotalContexts(result.current, 2);
                    expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID]).toBeUndefined();
                    expect(result.current.contexts[task1.task].length).toBe(2);
                    expect(result.current.activeTask.task).toBe(LlmTask.Start);
                    expect(result.current.inactiveTasks).toEqual([task1, task2]);
                });

                it("by task ID", async () => {
                    const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                    // Add contexts to a task type first
                    const originalContexts = [{
                        id: nanoid(),
                        label: "Old value 1",
                        data: { key: "old value 1" },
                    }, {
                        id: nanoid(),
                        label: "Old value 2",
                        data: { key: "old value 2" },
                    }, {
                        id: nanoid(),
                        label: "Old value 3",
                        data: { key: "old value 3" },
                    }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                add: {
                                    behavior: "add",
                                    connect: { __type: "taskType", data: LlmTask.Start },
                                    value: originalContexts,
                                },
                            },
                        });
                    });
                    // Add contexts to other task types to make sure they don't get removed
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                add: {
                                    inactive: {
                                        behavior: "onlyIfNoTaskId",
                                        value: [task1, task2],
                                    },
                                },
                            },
                        });
                    });
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                add: {
                                    behavior: "add",
                                    connect: { __type: "taskType", data: task1.task },
                                    value: [{
                                        id: nanoid(),
                                        label: "Value 1",
                                        data: "Data 1",
                                    }, {
                                        id: nanoid(),
                                        label: "Value 2",
                                        data: "Data 2",
                                    }],
                                },
                            },
                        });
                    });

                    validateTotalContexts(result.current, originalContexts.length + 2);
                    expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID]).toEqual(originalContexts);
                    expect(result.current.activeTask.task).toBe(LlmTask.Start);
                    expect(result.current.inactiveTasks).toEqual([task1, task2]);

                    // Remove context by task ID
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            contexts: {
                                remove: [{ __type: "taskId", data: DEFAULT_ACTIVE_TASK_ID }],
                            },
                        });
                    });

                    validateTotalContexts(result.current, 2);
                    expect(result.current.contexts[DEFAULT_ACTIVE_TASK_ID]).toBeUndefined();
                    expect(result.current.contexts[task1.task].length).toBe(2);
                    expect(result.current.activeTask.task).toBe(LlmTask.Start);
                    expect(result.current.inactiveTasks).toEqual([task1, task2]);
                });

                describe("by context ID", () => {
                    it("all values existing", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        // Add contexts to a task type first
                        const originalContexts = [{
                            id: nanoid(),
                            label: "Value 1",
                            data: "Data 1",
                        }, {
                            id: nanoid(),
                            label: "Value 2",
                            data: "Data 2",
                        }];
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                tasks: {
                                    add: {
                                        active: {
                                            behavior: "onlyReplaceStart",
                                            value: task1,
                                        },
                                    },
                                },
                            });
                        });
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: { __type: "taskType", data: task1.task },
                                        value: originalContexts,
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 2);
                        expect(result.current.contexts[task1.taskId]).toEqual(originalContexts);
                        expect(result.current.activeTask.task).toBe(task1.task);
                        expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);

                        // Remove context by context ID
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    remove: [{ __type: "contextId", data: originalContexts[0].id }],
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        expect(result.current.contexts[task1.taskId]).toEqual([originalContexts[1]]);
                        expect(result.current.activeTask.task).toBe(task1.task);
                        expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                    });
                    it("some values existing", async () => {
                        const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                        // Add contexts to a task type first
                        const originalContexts = [{
                            id: nanoid(),
                            label: "Value 1",
                            data: "Data 1",
                        }, {
                            id: nanoid(),
                            label: "Value 2",
                            data: "Data 2",
                        }];
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                tasks: {
                                    add: {
                                        active: {
                                            behavior: "onlyReplaceStart",
                                            value: task1,
                                        },
                                    },
                                },
                            });
                        });
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    add: {
                                        behavior: "add",
                                        connect: { __type: "taskType", data: task1.task },
                                        value: originalContexts,
                                    },
                                },
                            });
                        });

                        validateTotalContexts(result.current, 2);
                        expect(result.current.contexts[task1.taskId]).toEqual(originalContexts);
                        expect(result.current.activeTask.task).toBe(task1.task);
                        expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);

                        // Remove context by context ID
                        await act(async () => {
                            PubSub.get().publish("chatTask", {
                                chatId: "main",
                                contexts: {
                                    remove: [
                                        { __type: "contextId", data: originalContexts[0].id }, // Exists
                                        { __type: "contextId", data: nanoid() }, // Doesn't exist
                                    ],
                                },
                            });
                        });

                        validateTotalContexts(result.current, 1);
                        expect(result.current.contexts[task1.taskId]).toEqual([originalContexts[1]]);
                        expect(result.current.activeTask.task).toBe(task1.task);
                        expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                    });
                });
            });
        });
        describe("tasks", () => {
            describe("add", () => {
                describe("active", () => {
                    describe("behavior", () => {
                        it("force with no existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            active: {
                                                behavior: "force",
                                                value: { ...task1 },
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(task1);
                            expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                        });
                        it("onlyReplaceStart with no existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            active: {
                                                behavior: "onlyReplaceStart",
                                                value: { ...task3 },
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(task3);
                            expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                        });
                        it("onlyReplaceDifferentTaskType with no existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            active: {
                                                behavior: "onlyReplaceDifferentTaskType",
                                                value: { ...task2 },
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(task2);
                            expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                        });
                        it("force with existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            active: {
                                                behavior: "force",
                                                value: { ...task1 },
                                            },
                                        },
                                    },
                                });
                            });
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            active: {
                                                behavior: "force",
                                                value: { ...task2 },
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(task2);
                            expect(result.current.inactiveTasks).toEqual([task1, DEFAULT_ACTIVE_TASK]);
                        });
                        it("onlyReplaceStart with existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            active: {
                                                behavior: "force",
                                                value: { ...task1 }, // Now the active task isn't Llm.Start, so the next change should not replace it
                                            },
                                        },
                                    },
                                });
                            });
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            active: {
                                                behavior: "onlyReplaceStart",
                                                value: { ...task3 },
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(task1);
                            expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                        });
                        describe("onlyReplaceDifferentTaskType with existing data", () => {
                            it("existing data is same type", async () => {
                                const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                                const originalActiveTask = { ...task1 };
                                await act(async () => {
                                    PubSub.get().publish("chatTask", {
                                        chatId: "main",
                                        tasks: {
                                            add: {
                                                active: {
                                                    behavior: "force",
                                                    value: { ...originalActiveTask },
                                                },
                                            },
                                        },
                                    });
                                });
                                const updatedActiveTask = { ...task1, taskId: nanoid() }; // Same task type, so it shouldn't update
                                await act(async () => {
                                    PubSub.get().publish("chatTask", {
                                        chatId: "main",
                                        tasks: {
                                            add: {
                                                active: {
                                                    behavior: "onlyReplaceDifferentTaskType",
                                                    value: { ...updatedActiveTask },
                                                },
                                            },
                                        },
                                    });
                                });

                                expect(result.current.activeTask).toEqual(originalActiveTask);
                                expect(result.current.inactiveTasks).toEqual([DEFAULT_ACTIVE_TASK]);
                            });
                            it("existing data is different type", async () => {
                                const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                                const originalActiveTask = { ...task1 };
                                await act(async () => {
                                    PubSub.get().publish("chatTask", {
                                        chatId: "main",
                                        tasks: {
                                            add: {
                                                active: {
                                                    behavior: "force",
                                                    value: { ...originalActiveTask },
                                                },
                                            },
                                        },
                                    });
                                });
                                const updatedActiveTask = { ...task2 }; // Different task type, so it should update
                                await act(async () => {
                                    PubSub.get().publish("chatTask", {
                                        chatId: "main",
                                        tasks: {
                                            add: {
                                                active: {
                                                    behavior: "onlyReplaceDifferentTaskType",
                                                    value: { ...updatedActiveTask },
                                                },
                                            },
                                        },
                                    });
                                });

                                expect(result.current.activeTask).toEqual(updatedActiveTask);
                                expect(result.current.inactiveTasks).toEqual([originalActiveTask, DEFAULT_ACTIVE_TASK]);
                            });
                        });
                    });
                });
                describe("inactive", () => {
                    describe("behavior", () => {
                        it("onlyIfNoTaskType with no existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "onlyIfNoTaskType",
                                                value: [task1],
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual([task1]);
                        });
                        it("onlyIfNoTaskId with no existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "onlyIfNoTaskId",
                                                value: [task2],
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual([task2]);
                        });
                        it("replaceAll with no existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "replaceAll",
                                                value: [task3],
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual([task3]);
                        });
                        it("replaceTaskType with no existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "replaceTaskType",
                                                value: [task1],
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual([task1]);
                        });
                        it("replaceTaskId with no existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "replaceTaskId",
                                                value: [task4],
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual([task4]);
                        });
                        it("onlyIfNoTaskType with existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            const originalInactiveTasks = [{ ...task1 }, { ...task2 }];
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "onlyIfNoTaskType",
                                                value: originalInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });
                            const newInactiveTasks = [{ ...task2, taskId: nanoid() }, { ...task3, taskId: nanoid() }]; // task2 should be skipped, but still effect the order of the array
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "onlyIfNoTaskType",
                                                value: newInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual([originalInactiveTasks[1], newInactiveTasks[1], originalInactiveTasks[0]]);
                        });
                        it("onlyIfNoTaskId with existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            const originalInactiveTasks = [{ ...task1 }, { ...task2 }];
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "onlyIfNoTaskId",
                                                value: originalInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });
                            const newInactiveTasks = [{ ...task1 }, { ...task2, taskId: nanoid() }]; // task1 has same ID as existing task, so it should be skipped
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "onlyIfNoTaskId",
                                                value: newInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual([originalInactiveTasks[0], newInactiveTasks[1], originalInactiveTasks[1]]);
                        });
                        it("replaceAll with existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            const originalInactiveTasks = [{ ...task1 }, { ...task2 }];
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "replaceAll",
                                                value: originalInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });
                            const newInactiveTasks = [{ ...task3 }, { ...task4 }];
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "replaceAll",
                                                value: newInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual(newInactiveTasks);
                        });
                        it("replaceTaskType with existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            const originalInactiveTasks = [{ ...task1 }, { ...task2 }];
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "replaceAll",
                                                value: originalInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });
                            const newInactiveTasks = [{ ...task2, taskId: nanoid() }, { ...task3, taskId: nanoid() }]; // task2 should replace the existing task2
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "replaceTaskType",
                                                value: newInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual([...newInactiveTasks, originalInactiveTasks[0]]);
                        });
                        it("replaceTaskId with existing data", async () => {
                            const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                            const originalInactiveTasks = [{ ...task1 }, { ...task2 }];
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "replaceAll",
                                                value: originalInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });
                            const newInactiveTasks = [{ ...task1 }, { ...task4, id: nanoid() }]; // task1 has same ID as existing task1, so it should replace it
                            await act(async () => {
                                PubSub.get().publish("chatTask", {
                                    chatId: "main",
                                    tasks: {
                                        add: {
                                            inactive: {
                                                behavior: "replaceTaskId",
                                                value: newInactiveTasks,
                                            },
                                        },
                                    },
                                });
                            });

                            expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                            expect(result.current.inactiveTasks).toEqual([...newInactiveTasks, originalInactiveTasks[1]]);
                        });
                    });
                });
            });
            describe("update", () => {
                it("all values existing", async () => {
                    const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                    const originalActiveTask = { ...task1 };
                    const originalInactiveTasks = [{ ...task2 }, { ...task3 }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                add: {
                                    active: {
                                        behavior: "force",
                                        value: { ...originalActiveTask },
                                    },
                                    inactive: {
                                        behavior: "replaceAll",
                                        value: originalInactiveTasks,
                                    },
                                },
                            },
                        });
                    });

                    expect(result.current.activeTask).toEqual(originalActiveTask);
                    expect(result.current.inactiveTasks).toEqual(originalInactiveTasks);

                    const updateTasksData = [{ ...originalActiveTask, status: TaskStatus.Running }, { ...task3, status: TaskStatus.Completed }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                update: updateTasksData,
                            },
                        });
                    });

                    expect(result.current.activeTask).toEqual({ ...originalActiveTask, status: TaskStatus.Running });
                    expect(result.current.inactiveTasks).toEqual([{ ...task2 }, { ...task3, status: TaskStatus.Completed }]);
                });
                it("some values existing", async () => {
                    const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                    const originalActiveTask = { ...task1 };
                    const originalInactiveTasks = [{ ...task2 }, { ...task3 }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                add: {
                                    active: {
                                        behavior: "force",
                                        value: { ...originalActiveTask },
                                    },
                                    inactive: {
                                        behavior: "replaceAll",
                                        value: originalInactiveTasks,
                                    },
                                },
                            },
                        });
                    });

                    expect(result.current.activeTask).toEqual(originalActiveTask);
                    expect(result.current.inactiveTasks).toEqual(originalInactiveTasks);

                    const updateTasksData = [
                        { ...originalActiveTask, status: TaskStatus.Running },
                        { ...task3, status: TaskStatus.Completed },
                        { ...task4 }, // Doesn't exist, so it should be skipped
                    ];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                update: updateTasksData,
                            },
                        });
                    });

                    expect(result.current.activeTask).toEqual({ ...originalActiveTask, status: TaskStatus.Running });
                    expect(result.current.inactiveTasks).toEqual([{ ...task2 }, { ...task3, status: TaskStatus.Completed }]);
                });
            });
            describe("remove", () => {
                it("by task type", async () => {
                    const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                    const originalActiveTask = { ...task1 };
                    const originalInactiveTasks = [{ ...task2 }, { ...task3 }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                add: {
                                    active: {
                                        behavior: "force",
                                        value: { ...originalActiveTask },
                                    },
                                    inactive: {
                                        behavior: "replaceAll",
                                        value: originalInactiveTasks,
                                    },
                                },
                            },
                        });
                    });

                    expect(result.current.activeTask).toEqual(originalActiveTask);
                    expect(result.current.inactiveTasks).toEqual(originalInactiveTasks);

                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                remove: [{
                                    __type: "taskType",
                                    data: originalActiveTask.task,
                                }, {
                                    __type: "taskType",
                                    data: task2.task,
                                }],
                            },
                        });
                    });

                    expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                    expect(result.current.inactiveTasks).toEqual([task3]);
                });

                it("by task ID", async () => {
                    const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

                    const originalActiveTask = { ...task1 };
                    const originalInactiveTasks = [{ ...task2 }, { ...task3 }];
                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                add: {
                                    active: {
                                        behavior: "force",
                                        value: { ...originalActiveTask },
                                    },
                                    inactive: {
                                        behavior: "replaceAll",
                                        value: originalInactiveTasks,
                                    },
                                },
                            },
                        });
                    });

                    expect(result.current.activeTask).toEqual(originalActiveTask);
                    expect(result.current.inactiveTasks).toEqual(originalInactiveTasks);

                    await act(async () => {
                        PubSub.get().publish("chatTask", {
                            chatId: "main",
                            tasks: {
                                remove: [{
                                    __type: "taskId",
                                    data: originalActiveTask.taskId,
                                }, {
                                    __type: "taskId",
                                    data: task2.taskId,
                                }],
                            },
                        });
                    });

                    expect(result.current.activeTask).toEqual(DEFAULT_ACTIVE_TASK);
                    expect(result.current.inactiveTasks).toEqual([task3]);
                });
            });
        });
    });

    // describe("task selection", () => {
    //     describe("selectTask", () => {
    //         it("should set the active task if it was previously null", async () => {
    //             const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

    //             await act(async () => {
    //                 result.current.selectTask({ ...task1 });
    //             });

    //             expect(result.current.activeTask).toEqual(task1);
    //         });

    //         it("should replace the active task if it was previously set", async () => {
    //             const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

    //             // Set an initial task
    //             act(() => {
    //                 result.current.selectTask({ ...task1 });
    //             });

    //             // Replace it with a new task
    //             await act(async () => {
    //                 result.current.selectTask({ ...task2 });
    //             });

    //             expect(result.current.activeTask).toEqual(task2);
    //         });

    //         it("should remove the task from suggestedTasks", async () => {
    //             const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

    //             // Add tasks to suggestedTasks
    //             act(() => {
    //                 PubSub.get().publish("chatTask", { chatId: "main", addSuggestedTasks: { behavior: "replace", value: [{ ...task3 }, { ...task4 }] } });
    //             });

    //             // Select the task to remove it from suggestedTasks
    //             await act(async () => {
    //                 result.current.selectTask({ ...task3 });
    //             });

    //             expect(result.current.suggestedTasks).toEqual([task4]);
    //         });
    //     });

    //     describe("unselectTask", () => {
    //         it("should clear the active task", async () => {
    //             const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

    //             // Set an active task
    //             act(() => {
    //                 result.current.selectTask({ ...task3 });
    //             });

    //             // Clear the active task
    //             await act(async () => {
    //                 result.current.unselectTask();
    //             });

    //             expect(result.current.activeTask).toBeNull();
    //         });

    //         it("should add the previous active task back to suggestedTasks", async () => {
    //             const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

    //             // Set an active task
    //             act(() => {
    //                 result.current.selectTask({ ...task2 });
    //             });

    //             // Clear the active task and add it back to suggestedTasks
    //             await act(async () => {
    //                 result.current.unselectTask();
    //             });

    //             expect(result.current.suggestedTasks.some(t => t.taskId === task2.taskId)).to.be.ok;
    //         });
    //     });
    // });

    // describe("computed properties", () => {
    //     describe("activeContexts", () => {
    //         it("should return contexts for the active task", () => {
    //             const { result } = renderHook(() => useChatTasks({ chatId: "main" }));

    //             // Mock setting an active task and its associated context
    //             act(() => {
    //                 result.current.selectTask({ ...task2 });
    //             });

    //             act(() => {
    //                 PubSub.get().publish("chatTask", {
    //                     chatId: "main",
    //                     addContext: {
    //                         behavior: "replace",
    //                         linkedWithTask: task2.task as LlmTask,
    //                         value: { key: "activeTaskContext" },
    //                     },
    //                 });
    //             });

    //             const expected = {
    //                 id: expect.any(String),
    //                 label: "Object",
    //                 value: { key: "activeTaskContext" },
    //             };
    //             expect(result.current.activeContexts[0]).toEqual(expected);
    //         });

    //         it("should return general contexts if there is no active task", () => {
    //             const { result } = renderHook(() => useChatTasks({ chatId: "main" }));
    //             const generalContext = { key: "generalContext" };

    //             // Mock setting a general context without an active task
    //             act(() => {
    //                 PubSub.get().publish("chatTask", {
    //                     chatId: "main",
    //                     addContext: {
    //                         behavior: "replace",
    //                         value: generalContext,
    //                     },
    //                 });
    //             });

    //             const expected = {
    //                 id: expect.any(String),
    //                 label: "Object",
    //                 value: generalContext,
    //             };
    //             expect(result.current.activeContexts[0]).toEqual(expected);
    //         });
    //     });
    // });
});

// describe("sendMessageWithContext", () => {
//     it("should skip context collection if no active task is set", async () => {
//         const onSendMessage = vi.fn();
//         const { result } = renderHook(() => useChatTasks({ chatId: "main", onSendMessage }));

//         await act(async () => {
//             await result.current.sendMessageWithContext("Hello");
//         });

//         expect(onSendMessage).toHaveBeenCalledWith("Hello");
//         expect(onSendMessage).toHaveBeenCalledTimes(1); // Only called once with no context
//     });

//     it("should resolve with the correct context if data is available", async () => {
//         const onSendMessage = vi.fn();
//         const { result } = renderHook(() => useChatTasks({ chatId: "main", onSendMessage }));
//         const mockContext = { key: "value" };

//         act(() => {
//             result.current.selectTask({ ...task2 });
//         });

//         let timesSubscribeCalled = 0;
//         PubSub.get().subscribe("requestTaskContext", (data) => {
//             if (data.__type !== "request") return;
//             timesSubscribeCalled++;
//             PubSub.get().publish("requestTaskContext", { __type: "response", chatId: "main", task: task2.task as LlmTask, context: mockContext });
//         })

//         act(() => {
//             result.current.sendMessageWithContext("Hello");
//         });

//         await act(async () => {
//             vi.advanceTimersByTime(250);
//         });

//         expect(onSendMessage).toHaveBeenCalledWith("Hello", mockContext);
//         expect(onSendMessage).toHaveBeenCalledTimes(1);
//     });

//     it("should reject with a timeout error if no data is available within the timeout", async () => {
//         const onSendMessage = vi.fn();
//         const { result } = renderHook(() => useChatTasks({ chatId: "main", onSendMessage }));

//         act(() => {
//             result.current.selectTask({ ...task1 });
//         });

//         act(() => {
//             result.current.sendMessageWithContext("Hello");
//         });

//         // Fast-forward until all timers have been executed
//         await act(async () => {
//             vi.advanceTimersByTime(2_000);
//         });

//         expect(onSendMessage).toHaveBeenCalledWith("Hello");  // Context failed to collect, send message without it
//         vi.useRealTimers();
//     });

//     it("should ignore context collection if id does not match", async () => {
//         const onSendMessage = vi.fn();
//         const { result } = renderHook(() => useChatTasks({ chatId: "main", onSendMessage }));

//         act(() => {
//             result.current.selectTask({ ...task1 });
//         });

//         let timesSubscribeCalled = 0;
//         PubSub.get().subscribe("requestTaskContext", (data) => {
//             if (data.__type !== "request") return;
//             timesSubscribeCalled++;
//             PubSub.get().publish("requestTaskContext", { __type: "response", id: "other", task: task1.task as LlmTask, context: { key: "value" } });
//         });

//         act(() => {
//             result.current.sendMessageWithContext("Hello");
//         });

//         // Fast-forward until all timers have been executed
//         await act(async () => {
//             vi.advanceTimersByTime(2_000);
//         });

//         expect(onSendMessage).toHaveBeenCalledWith("Hello");  // No context collected since ID does not match
//         expect(timesSubscribeCalled).toBe(1); // Still called, but ignored
//     });

//     it("should ignore context collection if task does not match", async () => {
//         const onSendMessage = vi.fn();
//         const { result } = renderHook(() => useChatTasks({ chatId: "main", onSendMessage }));

//         act(() => {
//             result.current.selectTask({ ...task1 });
//         });

//         let timesSubscribeCalled = 0;
//         PubSub.get().subscribe("requestTaskContext", (data) => {
//             if (data.__type !== "request") return;
//             timesSubscribeCalled++;
//             PubSub.get().publish("requestTaskContext", { __type: "response", chatId: "main", task: task2.task as LlmTask, context: { key: "value" } });
//         });

//         act(() => {
//             result.current.sendMessageWithContext("Hello");
//         });

//         // Fast-forward until all timers have been executed
//         await act(async () => {
//             vi.advanceTimersByTime(2_000);
//         });

//         expect(onSendMessage).toHaveBeenCalledWith("Hello");  // No context collected since the task does not match
//         expect(timesSubscribeCalled).toBe(1); // Still called, but ignored
//     });

//     it("should ignore context collection if __type does not match", async () => {
//         const onSendMessage = vi.fn();
//         const { result } = renderHook(() => useChatTasks({ chatId: "main", onSendMessage }));

//         act(() => {
//             result.current.selectTask({ ...task1 });
//         });

//         let timesSubscribeCalled = 0;
//         PubSub.get().subscribe("requestTaskContext", (data) => {
//             // Since we're purposely returning the wrong __type here and don't want to cause
//             // an infinite loop, we'll stop after the first call
//             timesSubscribeCalled++;
//             if (timesSubscribeCalled > 1) return;
//             PubSub.get().publish("requestTaskContext", { __type: "request", chatId: "main", task: task1.task as LlmTask });
//         });

//         act(() => {
//             result.current.sendMessageWithContext("Hello");
//         });

//         // Fast-forward until all timers have been executed
//         await act(async () => {
//             vi.advanceTimersByTime(2_000);
//         });

//         expect(onSendMessage).toHaveBeenCalledWith("Hello");  // No context collected since the __type does not match
//     });

//     it("should send message with context if activeTask is set", async () => {
//         const onSendMessage = vi.fn();
//         const { result } = renderHook(() => useChatTasks({ chatId: "main", onSendMessage }));
//         const contextData = { key: "value" };

//         // Set an active task
//         act(() => {
//             result.current.selectTask({ ...task1 });
//         });

//         // Simulate response for context collection
//         PubSub.get().subscribe("requestTaskContext", (data) => {
//             if (data.__type !== "request" || data.task !== task1.task) return;
//             PubSub.get().publish("requestTaskContext", { __type: "response", chatId: "main", task: task1.task as LlmTask, context: contextData });
//         });

//         // Trigger the sendMessageWithContext function
//         act(() => {
//             result.current.sendMessageWithContext("Hello");
//         });

//         await act(async () => {
//             vi.advanceTimersByTime(250); // Advance time to simulate async context collection
//         });

//         expect(onSendMessage).toHaveBeenCalledWith("Hello", contextData);
//     });

//     it("should send message without context if activeTask is null", async () => {
//         const onSendMessage = vi.fn();
//         const { result } = renderHook(() => useChatTasks({ chatId: "main", onSendMessage }));

//         // No active task is set
//         await act(async () => {
//             await result.current.sendMessageWithContext("Hello");
//         });

//         expect(onSendMessage).toHaveBeenCalledWith("Hello");
//         expect(onSendMessage).toHaveBeenCalledTimes(1);
//     });

//     it("should handle errors in context collection", async () => {
//         const onSendMessage = vi.fn();
//         const { result } = renderHook(() => useChatTasks({ chatId: "main", onSendMessage }));

//         // Set an active task
//         act(() => {
//             result.current.selectTask({ ...task1 });
//         });

//         // Simulate a situation where context collection fails (no response triggered)
//         act(() => {
//             result.current.sendMessageWithContext("Hello");
//         });

//         // Fast-forward until after the timeout for context collection has passed
//         await act(async () => {
//             vi.advanceTimersByTime(2_000);
//         });

//         expect(onSendMessage).toHaveBeenCalledWith("Hello");  // Should send message without context due to failure
//         expect(onSendMessage).toHaveBeenCalledTimes(1);
//     });
// });
