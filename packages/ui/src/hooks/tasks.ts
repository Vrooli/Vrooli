import { AITaskInfo, CheckTaskStatusesInput, CheckTaskStatusesResult, DUMMY_ID, LlmTask, StartLlmTaskInput, TaskContextInfo, TaskType, VALYXA_ID, endpointGetCheckTaskStatuses, endpointPostStartLlmTask, getTranslation, uuid } from "@local/shared";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { ActiveChatContext } from "contexts";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { taskToTaskInfo } from "utils/display/chatTools";
import { getCookieTasksForChat, setCookieTasksForChat } from "utils/localStorage";
import { ChatTaskPub, ContextConnect, PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

export type UseAutoFillProps<FormShape = object> = {
    /**
     * @returns The input to send to the autofill endpoint, with 
     * its shape depending on the task
     */
    getAutoFillInput: () => TaskContextInfo["data"];
    /**
     * Shapes the autofill result into the form values to update 
     * the form with, while also returning a copy of the original 
     * form values for undoing the autofill
     * @param values The values returned from the autofill endpoint
     * @returns An object with the original form values and the 
     * values to update the form with
     */
    shapeAutoFillResult: (data: TaskContextInfo["data"]) => { originalValues: FormShape, updatedValues: FormShape };
    /**
     * Handles updating the form
     * @param values The values to update the form with
     */
    handleUpdate: (values: FormShape) => unknown;
    task: LlmTask;
}

const LOADING_SNACK_ID = "autofill-loading";
const SUCCESS_SNACK_DURATION_MS = 15_000;
const ERROR_SNACK_DURATION_MS = 10_000;

/**
 * A basic function to clean up the input object before sending it to the server. 
 * Since the input object comes from a form (and formik requires all fields to be
 * present), we can remove fields which represent no data (i.e. empty strings).
 * @param input The input object to clean
 * @returns The cleaned input object
 */
function cleanInput(input: TaskContextInfo["data"]): TaskContextInfo["data"] {
    if (!input || typeof input !== "object") return input;
    Object.entries(input).forEach(([key, value]) => {
        if (typeof value === "string" && value.trim() === "") {
            delete input[key];
        }
    });
    return input;
}

/**
 * Finds relevant autofill data from the current translation
 * @param values The current form values, where translation data is stored in 
 * a `translations` array
 * @params language The current language being edited
 * @returns The translation data to send to the autofill endpoint, notably with 
 * fields like `language` and `id` omitted
 */
export function getAutoFillTranslationData<
    Translation extends { language: string },
>(
    values: { translations?: Translation[] | null | undefined },
    language: string,
): object {
    const currentTranslation = { ...getTranslation(values, [language], true) } as Record<string, unknown>;
    delete currentTranslation.id;
    delete currentTranslation.language;
    delete currentTranslation.__typename;
    return currentTranslation;
}

/**
 * Creates updated translations based on autofill results and the current language.
 * @param values The current form values, where translation data is stored in
 * @param autofillData The data received from the autofill process, which contains translation 
 * and other data
 * @param language The current language being edited
 * @param translatedFields Fields that should be included in the updated translations
 * @returns An object containing the updated translations array and an object with the non-translated fields
 */
export function createUpdatedTranslations<
    Translation extends { language: string },
>(
    values: { translations?: Translation[] | null | undefined },
    autofillData: unknown,
    language: string,
    translatedFields: string[],
): { updatedTranslations: Translation[], rest: Record<string, unknown> } {
    const updatedTranslations: Translation[] = [];
    const updatedTranslationFields: Partial<Translation> = {};
    const rest: Record<string, unknown> = autofillData && typeof autofillData === "object" ? { ...autofillData } : {};

    // Extract translated fields from autofill data
    translatedFields.forEach(field => {
        if (autofillData && typeof autofillData === "object" && Object.prototype.hasOwnProperty.call(autofillData, field)) {
            updatedTranslationFields[field] = autofillData[field];
            delete rest[field];
        }
    });

    // We can only update translations if they existed in the first place
    if (!values.translations || !Array.isArray(values.translations) || values.translations.length === 0) {
        return { updatedTranslations, rest };
    }

    // Use the original translations to find the correct index to update
    let languageIndex = values.translations.findIndex(t => t.language === language);
    if (languageIndex < 0) {
        languageIndex = 0; // Fallback to first translation if the language doesn't exist
    }

    values.translations.forEach((translation, index) => {
        if (index === languageIndex) {
            // Merge autofill data into the current translation, excluding non-translated fields
            updatedTranslations.push({
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore This is in case the translation is missing a language
                language,
                ...translation,
                ...updatedTranslationFields,
                id: (translation as { id?: string }).id || DUMMY_ID,
            });
        } else {
            updatedTranslations.push(translation);
        }
    });

    return { updatedTranslations, rest };
}

/**
 * Creates a task context object for the autofill process
 * @param contextId The ID of the context
 * @param data The data to send to the server
 * @returns The task context object
 */
function createTaskContext(contextId: string, data: unknown): TaskContextInfo {
    return {
        id: contextId,
        label: "Form data",
        data,
    };
}

/**
 * Hook for handling AI autofill functionality in forms
 */
export function useAutoFill<T = object>({
    getAutoFillInput,
    shapeAutoFillResult,
    handleUpdate,
    task,
}: UseAutoFillProps<T>) {
    // Should always be associated with the main active chat
    const { chat, latestMessageId } = useContext(ActiveChatContext);

    /**
     * ID used when sending form data as a task context to the server. 
     * This is used to identify the task context when the server sends back the autofill result.
     */
    const contextId = useRef<string>(uuid());

    // Starts the autofill process in the server. Must listen to socket events for the result
    const [startLlmTask, { loading: isStartingLlmTask }] = useLazyFetch<StartLlmTaskInput, undefined>(endpointPostStartLlmTask);
    const autoFill = useCallback(function autoFillCallback() {
        const chatId = chat?.id;
        if (!chatId) {
            console.error("No chat ID found for autofill");
            return;
        }

        // Grab form data to send as context
        let data = getAutoFillInput();
        data = cleanInput(data);

        PubSub.get().publish("snack", {
            message: "Auto-filling form...",
            id: LOADING_SNACK_ID,
            severity: "Info",
            autoHideDuration: "persist",
        });

        fetchLazyWrapper<StartLlmTaskInput, undefined>({
            fetch: startLlmTask,
            inputs: {
                chatId,
                // Used to add messages to the LLM context
                parentId: latestMessageId,
                respondingBotId: VALYXA_ID,
                // Used to ensure that the task is only suggested, and not actually run. 
                // We only want what it'd look like to create/update this object, and let the user 
                // press the submit button to actually do it.
                shouldNotRunTasks: true,
                task,
                taskContexts: [createTaskContext(contextId.current, data)],
            },
            // TODO should be handled by socket event
            // onSuccess: (result) => {
            //     console.log("got autofill response", result);

            //     const { originalValues, updatedValues } = shapeAutoFillResult(result);
            //     handleUpdate(updatedValues);

            //     PubSub.get().publish("snack", {
            //         message: "Form auto-filled",
            //         buttonKey: "Undo",
            //         buttonClicked: () => { handleUpdate(originalValues); },
            //         severity: "Success",
            //         autoHideDuration: SUCCESS_SNACK_DURATION_MS,
            //         id: LOADING_SNACK_ID,
            //     });
            // },
            onError: (error) => {
                //TODO socket event should be able to send this too
                PubSub.get().publish("snack", {
                    message: "Failed to auto-fill form.",
                    severity: "Error",
                    autoHideDuration: ERROR_SNACK_DURATION_MS,
                    id: LOADING_SNACK_ID,
                    data: error,
                });
            },
            spinnerDelay: null, // Disables loading spinner
        });
    }, [chat?.id, getAutoFillInput, startLlmTask, latestMessageId, task]);

    // Suggest the autofill task to the main chat
    useEffect(function suggestTaskEffect() {
        const chatId = chat?.id;
        if (!chatId) {
            console.error("No chat ID found for autofill");
            return;
        }

        PubSub.get().publish("chatTask", {
            chatId,
            tasks: {
                add: {
                    inactive: {
                        behavior: "onlyIfNoTaskType",
                        value: [taskToTaskInfo(task)],
                    },
                },
            },
        });
    }, [chat?.id, task]);

    useEffect(function sendAutoFillDataEffect() {
        const chatId = chat?.id;
        if (!chatId) {
            console.error("No chat ID found for autofill");
            return;
        }

        const unsubscribe = PubSub.get().subscribe("requestTaskContext", (data) => {
            if (data.__type !== "request" || data.chatId !== chatId || data.task !== task) return;
            const context = getAutoFillInput();
            PubSub.get().publish("requestTaskContext", {
                __type: "response",
                chatId,
                task,
                context: {
                    id: uuid(), //TODO have way to connect to things like NoteCrud's RichInput component, so that we can override existing context
                    label: "Form data",
                    data: context,
                },
            });
        }, { source: "useAutoFill" });

        return unsubscribe;
    }, [chat?.id, getAutoFillInput, task]);

    const isAutoFillLoading = useMemo(() => isStartingLlmTask, [isStartingLlmTask]); //TODO track socket event too by updating useChatTasks to send event. Add event for listening to specific task updates?

    return {
        autoFill,
        isAutoFillLoading,
    };
}

const CONTEXT_LABEL_LENGTH = 20; // Set the maximum length of the label

/**
 * Generates a label for a given value of any type.
 * @param value The value for which to generate the label.
 * @returns A string representing the label.
 */
export function generateContextLabel(value: unknown): string {
    if (typeof value === "string") {
        return value.length > CONTEXT_LABEL_LENGTH ? `${value.substring(0, CONTEXT_LABEL_LENGTH - 1)}…` : value;
    } else if (typeof value === "number" || typeof value === "boolean") {
        const valueStr = String(value);
        return valueStr.length > CONTEXT_LABEL_LENGTH ? `${valueStr.substring(0, CONTEXT_LABEL_LENGTH - 1)}…` : valueStr;
    } else if (Array.isArray(value)) {
        return "Array"; // Or you can be more descriptive, like "Array of items"
    } else if (typeof value === "object" && value !== null) {
        return "Object"; // Or include more details, like the type of object if available
    }
    return "Value"; // Fallback for any other types like null, undefined, function, etc.
}

function getTaskId(
    connect: ContextConnect,
    activeTask: AITaskInfo,
    inactiveTasks: AITaskInfo[],
    contexts: { [taskId: string]: TaskContextInfo[] },
): string | null {
    if (connect.__type === "taskId") {
        return connect.data;
    } else if (connect.__type === "taskType") {
        let matchingTaskId: string | null = null;
        if (activeTask.task === connect.data) {
            matchingTaskId = activeTask.taskId;
        } else {
            matchingTaskId = inactiveTasks.find(task => task.task === connect.data)?.taskId || null;
        }
        return matchingTaskId;
    } else if (connect.__type === "contextId") {
        const taskKey = Object.entries(contexts).find(([, value]) => value.some(entry => entry.id === connect.data))?.[0];
        return taskKey || null;
    } else {
        console.error("Unknown context connect type:", (connect as { __type: string }).__type);
        return null;
    }
}

type UseChatTasksProps = {
    chatId: ChatTaskPub["chatId"] | null | undefined;
}

export type UseChatTaskReturn = {
    activeTask: AITaskInfo;
    contexts: { [taskId: string]: TaskContextInfo[] };
    inactiveTasks: AITaskInfo[];
    unselectTask: () => unknown;
    selectTask: (task: AITaskInfo) => unknown;
}

export const DEFAULT_ACTIVE_TASK_ID = "task-00000000-0000-0000-0000-000000000000";
export const DEFAULT_ACTIVE_TASK = {
    ...taskToTaskInfo(LlmTask.Start),
    taskId: DEFAULT_ACTIVE_TASK_ID,
};
export const DEFAULT_CONTEXTS = { [DEFAULT_ACTIVE_TASK_ID]: [] };
export const DEFAULT_INACTIVE_TASKS: AITaskInfo[] = [];

function initializeActiveTask(chatId: string | null | undefined) {
    if (!chatId) return DEFAULT_ACTIVE_TASK;

    const cachedData = getCookieTasksForChat(chatId);
    return cachedData?.activeTask || DEFAULT_ACTIVE_TASK;
}

function initializeContexts(chatId: string | null | undefined) {
    if (!chatId) return DEFAULT_CONTEXTS;

    const cachedData = getCookieTasksForChat(chatId);
    return cachedData?.contexts || DEFAULT_CONTEXTS;
}

function initializeInactiveTasks(chatId: string | null | undefined) {
    if (!chatId) return DEFAULT_INACTIVE_TASKS;

    const cachedData = getCookieTasksForChat(chatId);
    return cachedData?.inactiveTasks || DEFAULT_INACTIVE_TASKS;
}

/**
 * Collects and manages contexts and tasks for a chat, 
 * which are all used to configure the chat's response behavior
 */
export function useChatTasks({
    chatId,
}: UseChatTasksProps): UseChatTaskReturn {
    // The current task. Determines the actions the bot can take
    const [activeTask, setActiveTask] = useState<AITaskInfo>(() => initializeActiveTask(chatId));
    const activeTaskRef = useRef<AITaskInfo>(activeTask);
    const handleActiveTaskChange = useCallback((task: AITaskInfo) => {
        if (!chatId) return;
        // Update cache
        setCookieTasksForChat(chatId, { activeTask: task });
        // Update ref and state
        activeTaskRef.current = task;
        setActiveTask(task);
    }, [chatId]);

    // Information to provide in the context window of the responding bot(s)
    const [contexts, setContexts] = useState<{ [taskId: string]: TaskContextInfo[] }>(() => initializeContexts(chatId));
    const contextsRef = useRef<{ [taskId: string]: TaskContextInfo[] }>(contexts);
    const handleContextsChange = useCallback((newContexts: { [taskId: string]: TaskContextInfo[] }) => {
        if (!chatId) return;
        // Update cache
        setCookieTasksForChat(chatId, { contexts: newContexts });
        // Update ref and state
        contextsRef.current = newContexts;
        setContexts(newContexts);
    }, [chatId]);

    // Tasks that aren't active, but have been suggested, are running, or have run during the chat
    const [inactiveTasks, setInactiveTasks] = useState<AITaskInfo[]>(() => initializeInactiveTasks(chatId));
    const inactiveTasksRef = useRef<AITaskInfo[]>(inactiveTasks);
    const handleInactiveTasksChange = useCallback((tasks: AITaskInfo[]) => {
        if (!chatId) return;
        // Update cache
        setCookieTasksForChat(chatId, { inactiveTasks: tasks });
        // Update ref and state
        inactiveTasksRef.current = tasks;
        setInactiveTasks(tasks);
    }, [chatId]);

    useEffect(function subscribeToChangesEffect() {
        function handlePubSubEvent(data: ChatTaskPub) {
            if (data.chatId !== chatId) return;

            const updatedContexts = { ...contextsRef.current };
            let updatedActiveTask = { ...activeTaskRef.current };
            let updatedInactiveTasks = [...inactiveTasksRef.current];

            // Update tasks first, since contexts are tied to tasks and we may be adding new tasks and setting 
            // their contexts in the same PubSub event
            if (data.tasks) {
                const { add, update, remove } = data.tasks;

                if (add) {
                    const { active, inactive } = add;

                    if (active) {
                        if (
                            active.behavior === "force" ||
                            (active.behavior === "onlyReplaceStart" && activeTaskRef.current.task === LlmTask.Start) ||
                            (active.behavior === "onlyReplaceDifferentTaskType" && active.value?.task !== activeTaskRef.current.task)
                        ) {
                            // Add old active task to inactive tasks, if not already there
                            if (!inactiveTasksRef.current.some(task => task.taskId === activeTaskRef.current.taskId)) {
                                updatedInactiveTasks = [activeTaskRef.current, ...updatedInactiveTasks];
                            }
                            // Set new active task
                            updatedActiveTask = active.value || DEFAULT_ACTIVE_TASK;
                            // Remove new active task from inactive tasks
                            updatedInactiveTasks = updatedInactiveTasks.filter(task => task.taskId !== active.value?.taskId);
                        }
                    }
                    if (inactive) {
                        switch (inactive.behavior) {
                            case "onlyIfNoTaskType": {
                                for (let i = inactive.value.length - 1; i >= 0; i--) {
                                    const index = updatedInactiveTasks.findIndex(task => task.task === inactive.value[i].task);
                                    // If not found, add to the list
                                    if (index < 0) {
                                        updatedInactiveTasks = [inactive.value[i], ...updatedInactiveTasks];
                                    }
                                    // If found, move to the front of the list
                                    else {
                                        updatedInactiveTasks = [updatedInactiveTasks[index], ...updatedInactiveTasks.filter((_, idx) => idx !== index)];
                                    }
                                }
                                break;
                            }
                            case "onlyIfNoTaskId": {
                                for (let i = inactive.value.length - 1; i >= 0; i--) {
                                    const index = updatedInactiveTasks.findIndex(task => task.taskId === inactive.value[i].taskId);
                                    // If not found, add to the list
                                    if (index < 0) {
                                        updatedInactiveTasks = [inactive.value[i], ...updatedInactiveTasks];
                                    }
                                    // If found, move to the front of the list
                                    else {
                                        updatedInactiveTasks = [updatedInactiveTasks[index], ...updatedInactiveTasks.filter((_, idx) => idx !== index)];
                                    }
                                }
                                break;
                            }
                            case "replaceAll": {
                                updatedInactiveTasks = [...inactive.value];
                                break;
                            }
                            case "replaceTaskType": {
                                // Remove matching task types first
                                const typesAdding = inactive.value.map(task => task.task);
                                updatedInactiveTasks = updatedInactiveTasks.filter(task => !typesAdding.includes(task.task));
                                // Add new tasks
                                updatedInactiveTasks = [...inactive.value, ...updatedInactiveTasks];
                                break;
                            }
                            case "replaceTaskId": {
                                // Remove matching task IDs first
                                const idsAdding = inactive.value.map(task => task.taskId);
                                updatedInactiveTasks = updatedInactiveTasks.filter(task => !idsAdding.includes(task.taskId));
                                // Add new tasks
                                updatedInactiveTasks = [...inactive.value, ...updatedInactiveTasks];
                                break;
                            }
                        }

                        // Ensure that newly inactive tasks are not in the active task
                        if (activeTaskRef.current && updatedInactiveTasks.some(task => task.taskId === activeTaskRef.current.taskId)) {
                            updatedActiveTask = DEFAULT_ACTIVE_TASK;
                        }
                    }
                }

                if (update) {
                    for (const task of update) {
                        const inactiveTaskIndex = updatedInactiveTasks.findIndex(t => t.taskId === task.taskId);
                        if (inactiveTaskIndex >= 0) {
                            updatedInactiveTasks[inactiveTaskIndex] = { ...updatedInactiveTasks[inactiveTaskIndex], ...task };
                        }
                        if (updatedActiveTask && updatedActiveTask.taskId === task.taskId) {
                            updatedActiveTask = { ...updatedActiveTask, ...task };
                        }
                    }
                }

                if (remove) {
                    for (const taskConnect of remove) {
                        const taskIdToRemove = getTaskId(taskConnect, updatedActiveTask, updatedInactiveTasks, updatedContexts);
                        if (!taskIdToRemove) continue;

                        // Remove from active task if matches
                        if (updatedActiveTask && updatedActiveTask.taskId === taskIdToRemove) {
                            updatedActiveTask = DEFAULT_ACTIVE_TASK;
                        }

                        // Remove from inactive tasks
                        updatedInactiveTasks = updatedInactiveTasks.filter(task => task.taskId !== taskIdToRemove);

                        // Remove contexts tied to the task
                        delete updatedContexts[taskIdToRemove];
                    }
                }
            }

            if (data.contexts) {
                const { add, update, remove } = data.contexts;

                if (add) {
                    const { behavior, connect, value } = add;
                    let taskKey = getTaskId(connect, updatedActiveTask, updatedInactiveTasks, updatedContexts);

                    // If the task key doesn't exist and we're using connect.__type === "contextId", add to active task
                    if (!taskKey && connect.__type === "contextId") {
                        taskKey = activeTaskRef.current.taskId;
                    }

                    if (taskKey) {
                        const currentContext = updatedContexts[taskKey] || [];
                        if (behavior === "replace" || connect.__type === "contextId") {
                            updatedContexts[taskKey] = value;
                        } else if (behavior === "add") {
                            // Add to beginning of the list
                            updatedContexts[taskKey] = [...value, ...currentContext];
                        } else {
                            console.error("Unknown context behavior:", behavior);
                        }
                    }
                }

                if (update) {
                    for (const u of update) {
                        const taskKey = getTaskId({ __type: "contextId", data: u.id }, updatedActiveTask, updatedInactiveTasks, updatedContexts);
                        if (!taskKey) continue;

                        updatedContexts[taskKey] = updatedContexts[taskKey].map(entry => {
                            if (entry.id === u.id) {
                                return { ...entry, ...u };
                            }
                            return entry;
                        });
                    }
                }

                if (remove) {
                    for (const r of remove) {
                        const taskKey = getTaskId(r, updatedActiveTask, updatedInactiveTasks, updatedContexts);
                        if (!taskKey) continue;

                        if (r.__type === "contextId") {
                            updatedContexts[taskKey] = updatedContexts[taskKey].filter(entry => entry.id !== r.data);
                            if (updatedContexts[taskKey].length === 0) {
                                delete updatedContexts[taskKey];
                            }
                        } else {
                            delete updatedContexts[taskKey];
                        }
                    }
                }
            }

            handleContextsChange(updatedContexts);
            handleActiveTaskChange(updatedActiveTask);
            handleInactiveTasksChange(updatedInactiveTasks);
        }

        const unsubscribe = PubSub.get().subscribe("chatTask", handlePubSubEvent);
        return () => unsubscribe();
    }, [chatId, handleActiveTaskChange, handleContextsChange, handleInactiveTasksChange]);

    const selectTask = useCallback((task: AITaskInfo) => {
        // Move the current active task to inactive tasks if it's not the default task
        let updatedInactiveTasks = inactiveTasksRef.current;
        if (activeTaskRef.current && activeTaskRef.current.task !== DEFAULT_ACTIVE_TASK.task) {
            updatedInactiveTasks = [activeTaskRef.current, ...updatedInactiveTasks];
        }
        // Remove the selected task from inactive tasks if it's there
        updatedInactiveTasks = updatedInactiveTasks.filter(t => t.taskId !== task.taskId);
        // Update the inactive tasks and set the new active task
        handleInactiveTasksChange(updatedInactiveTasks);
        handleActiveTaskChange(task);
    }, [handleActiveTaskChange, handleInactiveTasksChange]);


    const unselectTask = useCallback(() => {
        const updatedInactiveTasks = [activeTaskRef.current, ...inactiveTasksRef.current.filter(t => t.taskId !== activeTaskRef.current.taskId)];
        handleInactiveTasksChange(updatedInactiveTasks);
        handleActiveTaskChange(DEFAULT_ACTIVE_TASK);
    }, [handleActiveTaskChange, handleInactiveTasksChange]);

    // Query for tasks which are in a running state, to see if they were completed when we were offline
    const [getTaskData, { data: fetchedTaskData, loading: isTaskLoading }] = useLazyFetch<CheckTaskStatusesInput, CheckTaskStatusesResult>(endpointGetCheckTaskStatuses);

    // Fetch task data when the chat ID changes
    const hasCheckedRunningTasks = useRef(false);
    useEffect(function fetchTaskDataEffect() {
        if (hasCheckedRunningTasks.current || process.env.NODE_ENV === "test") return;
        const taskIds = [activeTaskRef.current, ...inactiveTasks].filter(task => task && (task.status === "Running" || task.status === "Canceling")).map(task => (task as AITaskInfo).taskId);
        if (taskIds.length === 0) return;
        hasCheckedRunningTasks.current = true;
        getTaskData({ taskIds, taskType: TaskType.Llm });
    }, [chatId, getTaskData, inactiveTasks]);

    // Add fetched task data to the existing tasks
    useEffect(function addFetchedTaskDataEffect() {
        if (!fetchedTaskData || !chatId) return;
        let updatedActiveTask = { ...activeTaskRef.current };
        const updatedInactiveTasks = [...inactiveTasksRef.current];
        fetchedTaskData.statuses.forEach(({ id: taskId, status }) => {
            if (!status) return;
            if (activeTaskRef.current && activeTaskRef.current.taskId === taskId) {
                updatedActiveTask = { ...activeTaskRef.current, status, lastUpdated: new Date().toISOString() };
            } else {
                const taskIndex = updatedInactiveTasks.findIndex(task => task.taskId === taskId);
                if (taskIndex >= 0) {
                    updatedInactiveTasks[taskIndex] = { ...updatedInactiveTasks[taskIndex], status, lastUpdated: new Date().toISOString() };
                }
            }
        });
        handleActiveTaskChange(updatedActiveTask);
        handleInactiveTasksChange(updatedInactiveTasks);
    }, [chatId, fetchedTaskData, handleActiveTaskChange, handleInactiveTasksChange]);

    // Wehn ID changes, update everything
    useEffect(function refreshDataEffect() {
        handleActiveTaskChange(initializeActiveTask(chatId));
        handleContextsChange(initializeContexts(chatId));
        handleInactiveTasksChange(initializeInactiveTasks(chatId));
        hasCheckedRunningTasks.current = false;
    }, [chatId, handleActiveTaskChange, handleContextsChange, handleInactiveTasksChange]);

    return {
        activeTask,
        contexts,
        inactiveTasks,
        unselectTask,
        selectTask,
    };
}
