import { ChatMessage, ChatParticipant, ChatShape, LlmTaskInfo, Session, uuid } from "@local/shared";
import { getCookieTasksForMessage, setCookie } from "../utils/cookies";
import { processLlmTasks, processMessages, processParticipantsUpdates, processResponseStream, processTypingUpdates } from "./useSocketChat";

describe("processMessages", () => {
    const addMessages = jest.fn();
    const removeMessages = jest.fn();
    const editMessage = jest.fn();

    beforeEach(() => {
        jest.clearAllMocks();
    });

    it("should handle added messages", () => {
        const messages = [
            { taskId: "msg1", content: "Hello" },
            { taskId: "msg2", content: "World" },
        ] as unknown as ChatMessage[];
        processMessages({ added: messages, deleted: [], edited: [] }, addMessages, removeMessages, editMessage);
        expect(addMessages).toHaveBeenCalledWith([
            { taskId: "msg1", content: "Hello", status: "sent" },
            { taskId: "msg2", content: "World", status: "sent" },
        ]);
        expect(removeMessages).not.toHaveBeenCalled();
        expect(editMessage).not.toHaveBeenCalled();
    });

    it("should handle deleted messages", () => {
        const messages = ["msg1", "msg2"];
        processMessages({ added: [], deleted: messages, edited: [] }, addMessages, removeMessages, editMessage);
        expect(removeMessages).toHaveBeenCalledWith(messages);
        expect(addMessages).not.toHaveBeenCalled();
        expect(editMessage).not.toHaveBeenCalled();
    });

    it("should handle edited messages", () => {
        const messages = [
            { taskId: "msg1", content: "Updated Hello" },
            { taskId: "msg2", content: "Updated World" },
        ] as unknown as ChatMessage[];
        processMessages({ added: [], deleted: [], edited: messages }, addMessages, removeMessages, editMessage);
        messages.forEach(message => {
            expect(editMessage).toHaveBeenCalledWith({ ...message, status: "sent" });
        });
        expect(addMessages).not.toHaveBeenCalled();
        expect(removeMessages).not.toHaveBeenCalled();
    });

    it("should handle combinations of added, deleted, and edited messages", () => {
        const added = [{ taskId: "msg3", content: "New" }] as unknown as ChatMessage[];
        const deleted = ["msg1"];
        const edited = [{ taskId: "msg2", content: "Updated World" }] as unknown as ChatMessage[];

        processMessages({ added, deleted, edited }, addMessages, removeMessages, editMessage);
        expect(addMessages).toHaveBeenCalledWith([{ taskId: "msg3", content: "New", status: "sent" }]);
        expect(removeMessages).toHaveBeenCalledWith(deleted);
        edited.forEach(message => {
            expect(editMessage).toHaveBeenCalledWith({ ...message, status: "sent" });
        });
    });

    it("should do nothing if no messages are added, deleted, or edited", () => {
        processMessages({ added: [], deleted: [], edited: [] }, addMessages, removeMessages, editMessage);
        expect(addMessages).not.toHaveBeenCalled();
        expect(removeMessages).not.toHaveBeenCalled();
        expect(editMessage).not.toHaveBeenCalled();
    });
});

describe("processTypingUpdates", () => {
    const currentUserId = uuid();
    const session = {
        isLoggedIn: true,
        users: [{ id: currentUserId }],
    } as Session;
    const setUsersTyping = jest.fn();
    const participants = [
        { user: { id: "user1" } },
        { user: { id: "user2" } },
        { user: { id: "user3" } },
        { user: { id: currentUserId } }, // this is the current user
    ] as Omit<ChatParticipant, "chat">[];

    beforeEach(() => {
        jest.clearAllMocks();
    });

    it("should add participants who start typing, excluding the current user", () => {
        const starting = ["user1", "user2", currentUserId];
        const stopping = [];
        const usersTyping = [];

        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        expect(setUsersTyping).toHaveBeenCalledWith([
            { user: { id: "user1" } },
            { user: { id: "user2" } },
        ]);
    });

    it("should remove participants who stop typing", () => {
        const starting = [];
        const stopping = ["user1"];
        const usersTyping = [
            { user: { id: "user1" } },
            { user: { id: "user2" } },
        ] as Omit<ChatParticipant, "chat">[];

        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        expect(setUsersTyping).toHaveBeenCalledWith([
            { user: { id: "user2" } },
        ]);
    });

    it("should handle participants starting and stopping typing simultaneously", () => {
        const starting = ["user3"];
        const stopping = ["user1"];
        const usersTyping = [
            { user: { id: "user1" } },
            { user: { id: "user2" } },
        ] as Omit<ChatParticipant, "chat">[];

        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        expect(setUsersTyping).toHaveBeenCalledWith([
            { user: { id: "user2" } },
            { user: { id: "user3" } },
        ]);
    });

    it("should ignore ids not found in participants", () => {
        const starting = ["user4"]; // user4 is not in participants
        const stopping = [];
        const usersTyping = [];

        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        expect(setUsersTyping).not.toHaveBeenCalled();
    });

    it("should handle empty starting and stopping arrays", () => {
        const starting = [];
        const stopping = [];
        const usersTyping = [{ user: { id: "user1" } }] as Omit<ChatParticipant, "chat">[];

        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        expect(setUsersTyping).not.toHaveBeenCalled(); // No changes
    });

    it("should still work when session is undefined", () => {
        const starting = ["user1", currentUserId]; // Not actually current user, since no session
        const stopping = [];
        const usersTyping = [];

        // Passing undefined session
        processTypingUpdates({ starting, stopping }, usersTyping, participants, undefined, setUsersTyping);
        expect(setUsersTyping).toHaveBeenCalledWith([
            { user: { id: "user1" } },
            { user: { id: currentUserId } },
        ]);
    });
});

describe("processParticipantsUpdates", () => {
    const setParticipants = jest.fn();
    const chat = { id: "chat1" } as ChatShape;
    const task = "RunRoutine";

    beforeEach(() => {
        jest.clearAllMocks();
        global.localStorage.clear();
        setCookie("Preferences", {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        });
    });
    afterAll(() => {
        global.localStorage.clear();
        jest.restoreAllMocks();
    });

    it("should add joining participants to the chat", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [{ user: { id: "user2" }, id: "participant2" }] as Omit<ChatParticipant, "chat">[];
        const leaving = [];

        processParticipantsUpdates({ joining, leaving }, participants, chat, task, setParticipants);

        expect(setParticipants).toHaveBeenCalledWith([...participants, ...joining]);
        // expect(getCookieMatchingChat(participants.map(p => p.user.id), task)).toEqual(chat.id); //TODO localstorage in these test suites not working properly for some reason
    });

    it("should remove leaving participants from the chat", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
            { user: { id: "user2" }, id: "participant2" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [];
        const leaving = ["user1"];

        processParticipantsUpdates({ joining, leaving }, participants, chat, task, setParticipants);

        expect(setParticipants).toHaveBeenCalledWith(participants.filter(p => p.user.id !== "user1"));
        // expect(getCookieMatchingChat(participants.map(p => p.user.id), task)).toEqual(['user2']);
    });

    it("should handle empty joining and leaving arrays", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [];
        const leaving = [];

        processParticipantsUpdates({ joining, leaving }, participants, chat, task, setParticipants);

        expect(setParticipants).not.toHaveBeenCalled();
        // expect(getCookieMatchingChat(participants.map(p => p.user.id), task)).toEqual(['user1']);
    });

    it("should not call setParticipants if no changes are made - test1", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [{ user: { id: "user1" }, id: "participant1" }] as Omit<ChatParticipant, "chat">[]; // user1 is already in participants
        const leaving = [];

        processParticipantsUpdates({ joining, leaving }, participants, chat, task, setParticipants);

        expect(setParticipants).not.toHaveBeenCalled();
        // expect(getCookieMatchingChat(participants.map(p => p.user.id), task)).toEqual(['user1']);
    });

    it("should not call setParticipants if no changes are made - test2", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [];
        const leaving = ["user2"]; // user2 is not in participants

        processParticipantsUpdates({ joining, leaving }, participants, chat, task, setParticipants);

        expect(setParticipants).not.toHaveBeenCalled();
        // expect(getCookieMatchingChat(participants.map(p => p.user.id), task)).toEqual(['user1']);
    });

    test("should not call setParticipants if no changes are made - test3", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [{ user: { id: "user2" }, id: "participant2" }] as Omit<ChatParticipant, "chat">[];
        const leaving = ["user2"]; // Same one as in joining

        processParticipantsUpdates({ joining, leaving }, participants, chat, task, setParticipants);

        expect(setParticipants).not.toHaveBeenCalled();
        // expect(getCookieMatchingChat(participants.map(p => p.user.id), task)).toEqual(['user1']);
    });
});

describe("processLlmTasks", () => {
    const originalLocalStorage = global.localStorage;
    const updateTasksForMessage = jest.fn();
    let messageTasks: Record<string, LlmTaskInfo[]> = {};

    beforeEach(() => {
        jest.clearAllMocks();
        messageTasks = {};

        let store: Record<string, string> = {};
        const mockLocalStorage = {
            getItem: (key: string) => (key in store ? store[key] : null),
            setItem: (key: string, value: string) => (store[key] = value),
            removeItem: (key: string) => delete store[key],
            clear: () => (store = {}),
        };

        global.localStorage = mockLocalStorage as unknown as typeof global.localStorage;
        global.localStorage.clear();
        setCookie("Preferences", {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        });

    });
    afterEach(() => {
        global.localStorage = originalLocalStorage;
    });

    it("should handle empty tasks and updates", () => {
        processLlmTasks({ tasks: [], updates: [] }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).not.toHaveBeenCalled();
    });

    it("should process full tasks", () => {
        const tasks = [
            { taskId: "task1", messageId: "msg1", status: "Running" },
            { taskId: "task2", messageId: "msg1", status: "Suggested" },
        ] as LlmTaskInfo[];
        messageTasks["msg1"] = [...tasks];
        processLlmTasks({ tasks, updates: [] }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).toHaveBeenCalledWith("msg1", expect.any(Array));

        const storedTasks = getCookieTasksForMessage("msg1");
        expect(storedTasks).toEqual({ tasks });
    });

    it("should handle partial tasks when tasks haven't been stored yet", () => {
        const updates = [
            { taskId: "task3", messageId: "msg1", status: "Completed" },
            { taskId: "task4", messageId: "msg1", status: "Failed" },
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks: [], updates }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).toHaveBeenCalledWith("msg1", expect.any(Array));

        const storedTasks = getCookieTasksForMessage("msg1");
        expect(storedTasks).toEqual({
            tasks: updates.map(update => ({
                ...update,
                lastUpdated: expect.any(String), // Adds a `lastUpdated` date
            })),
        });
        storedTasks?.tasks?.forEach(task => {
            // Check that each task has a lastUpdated field that is a valid ISO date string
            const date = new Date(task.lastUpdated);
            expect(date.toISOString()).toBe(task.lastUpdated);
        });
    });

    it("should apply partial tasks to existing tasks", () => {
        const tasks = [
            { taskId: "task1", messageId: "msg1", status: "Running" },
            { taskId: "task2", messageId: "msg1", status: "Suggested" },
        ] as LlmTaskInfo[];
        messageTasks["msg1"] = [...tasks];
        processLlmTasks({ tasks, updates: [] }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).toHaveBeenCalledWith("msg1", expect.any(Array));

        const updates = [
            { taskId: "task1", messageId: "msg1", status: "Completed" },
            { taskId: "task2", messageId: "msg1", status: "Failed" },
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks: [], updates }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).toHaveBeenCalledWith("msg1", expect.any(Array));

        const storedTasks = getCookieTasksForMessage("msg1");
        expect(storedTasks).toEqual({
            tasks: [
                { taskId: "task1", messageId: "msg1", status: "Completed", lastUpdated: expect.any(String) },
                { taskId: "task2", messageId: "msg1", status: "Failed", lastUpdated: expect.any(String) },
            ],
        });
    });

    it("should process full tasks and partial tasks at the same time", () => {
        const tasks = [
            { taskId: "task1", messageId: "msg1", status: "Running" },
            { taskId: "task2", messageId: "msg2", status: "Suggested" },
        ] as LlmTaskInfo[];
        const updates = [
            { taskId: "task1", messageId: "msg1", status: "Completed" }, // Same ID as in `tasks`, so should overwrite
            { taskId: "task4", messageId: "msg2", status: "Failed" }, // New ID, so should skip (since it wasn't stored yet)
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks, updates }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).toHaveBeenCalledWith("msg1", expect.any(Array));
        expect(updateTasksForMessage).toHaveBeenCalledWith("msg2", expect.any(Array));

        const storedTasks1 = getCookieTasksForMessage("msg1");
        expect(storedTasks1).toEqual({ tasks: [{ taskId: "task1", messageId: "msg1", status: "Completed", lastUpdated: expect.any(String) }] });

        const storedTasks2 = getCookieTasksForMessage("msg2");
        expect(storedTasks2).toEqual({
            tasks: [
                { taskId: "task2", messageId: "msg2", status: "Suggested" }, // Wasn't updated, so no `lastUpdated` change
                { taskId: "task4", messageId: "msg2", status: "Failed", lastUpdated: expect.any(String) },
            ],
        });
    });

    it("should ignore tasks without a messageId if they can't be found in messageTasks", () => {
        const tasks = [
            { taskId: "task1", status: "Running" },
            { taskId: "task2", status: "Suggested" },
        ] as LlmTaskInfo[];
        processLlmTasks({ tasks, updates: [] }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).not.toHaveBeenCalled();
    });

    it("should still ignore tasks without a messageId if they can be found in messageTasks - we want full data for new tasks", () => {
        const tasks = [
            { taskId: "task1", status: "Running" },
            { taskId: "task2", status: "Suggested" },
        ] as LlmTaskInfo[];
        messageTasks["msg1"] = [{ taskId: "task1", messageId: "msg1", status: "Running" }] as LlmTaskInfo[];
        processLlmTasks({ tasks, updates: [] }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).not.toHaveBeenCalled();
    });

    it("should ignore tasks without an ID", () => {
        const tasks = [
            { messageId: "msg1", status: "Running" },
            { messageId: "msg1", status: "Suggested" },
        ] as LlmTaskInfo[];
        processLlmTasks({ tasks, updates: [] }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).not.toHaveBeenCalled();
    });

    it("should ignore updated tasks without a messageId if they cannot be found in messageTasks", () => {
        const updates = [
            { taskId: "task1", status: "Completed" },
            { taskId: "task2", status: "Failed" },
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks: [], updates }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).not.toHaveBeenCalled();
    });

    it("should not ignore updated tasks without a messageId if they can be found in messageTasks", () => {
        const updates = [
            { taskId: "task1", status: "Completed" },
            { taskId: "task2", status: "Failed" },
        ] as Partial<LlmTaskInfo>[];
        messageTasks["msg1"] = [{ taskId: "task1", messageId: "msg1", status: "Running" }] as LlmTaskInfo[];
        processLlmTasks({ tasks: [], updates }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).toHaveBeenCalledWith("msg1", expect.any(Array));
    });

    it("should ignore updates without an ID", () => {
        const updates = [
            { messageId: "msg1", status: "Completed" },
            { messageId: "msg1", status: "Failed" },
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks: [], updates }, messageTasks, updateTasksForMessage);
        expect(updateTasksForMessage).not.toHaveBeenCalled();
    });
});

describe("processResponseStream", () => {
    let messageStreamRef;
    const setMessageStream = jest.fn();
    const throttledSetMessageStream = jest.fn();

    beforeEach(() => {
        jest.clearAllMocks();
        messageStreamRef = { current: null };
    });

    it("should initialize and update stream for new messages", () => {
        processResponseStream({ __type: "stream", botId: "bot123", message: "Hello" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current).toEqual({ __type: "stream", botId: "bot123", message: "Hello" });
        expect(throttledSetMessageStream).toHaveBeenCalledWith({ __type: "stream", botId: "bot123", message: "Hello" });
    });

    it("should append to the existing message on stream type", () => {
        messageStreamRef.current = { __type: "stream", botId: "bot123", message: "Hello" };
        processResponseStream({ __type: "stream", botId: "bot123", message: " World" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current.message).toBe("Hello World");
    });

    it("should clear message and update botId on new bot message", () => {
        messageStreamRef.current = { __type: "stream", botId: "bot123", message: "Hello" };
        processResponseStream({ __type: "stream", botId: "bot456", message: "New message" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current).toEqual({ __type: "stream", botId: "bot456", message: "New message" });
    });

    it("should handle error type by continuing to append message", () => {
        processResponseStream({ __type: "error", botId: "bot123", message: "Error occurred" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current.message).toBe("Error occurred");
    });

    it("should clear the stream on end type", () => {
        console.log = jest.fn();
        processResponseStream({ __type: "end", botId: "bot123", message: "Complete" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current).toBeNull();
        expect(setMessageStream).toHaveBeenCalledWith(null);
    });

    it("should not call update functions when no changes are made", () => {
        processResponseStream({ __type: "end", botId: "bot123", message: "Complete" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(setMessageStream).toHaveBeenCalledWith(null);
        expect(throttledSetMessageStream).not.toHaveBeenCalled();
    });
});
