import { uuid, type ChatMessage, type ChatParticipant, type ChatShape, type LlmTaskInfo, type Session } from "@vrooli/shared";
import { expect } from "chai";
import { fullPreferences, getCookieTasksForChat, setCookie, upsertCookieTaskForChat } from "../utils/localStorage.js";
import { processLlmTasks, processMessages, processParticipantsUpdates, processResponseStream, processTypingUpdates } from "./useSocketChat.js";

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
        processMessages({ added: messages, updated: [], removed: [] }, addMessages, editMessage, removeMessages);
        expect(addMessages).toHaveBeenCalledWith([
            { taskId: "msg1", content: "Hello", status: "sent" },
            { taskId: "msg2", content: "World", status: "sent" },
        ]);
        expect(removeMessages).not.toHaveBeenCalled();
        expect(editMessage).not.toHaveBeenCalled();
    });

    it("should handle updated messages", () => {
        const messages = [
            { taskId: "msg1", content: "Updated Hello" },
            { taskId: "msg2", content: "Updated World" },
        ] as unknown as ChatMessage[];
        processMessages({ added: [], updated: messages, removed: [] }, addMessages, editMessage, removeMessages);
        messages.forEach(message => {
            expect(editMessage).toHaveBeenCalledWith({ ...message, status: "sent" });
        });
        expect(addMessages).not.toHaveBeenCalled();
        expect(removeMessages).not.toHaveBeenCalled();
    });

    it("should handle removed messages", () => {
        const messages = ["msg1", "msg2"];
        processMessages({ added: [], updated: [], removed: messages }, addMessages, editMessage, removeMessages);
        expect(removeMessages).toHaveBeenCalledWith(messages);
        expect(addMessages).not.toHaveBeenCalled();
        expect(editMessage).not.toHaveBeenCalled();
    });

    it("should handle combinations of added, updated, and removed messages", () => {
        const added = [{ taskId: "msg3", content: "New" }] as unknown as ChatMessage[];
        const updated = [{ taskId: "msg2", content: "Updated World" }] as unknown as ChatMessage[];
        const removed = ["msg1"];

        processMessages({ added, updated, removed }, addMessages, editMessage, removeMessages);
        expect(addMessages).toHaveBeenCalledWith([{ taskId: "msg3", content: "New", status: "sent" }]);
        expect(removeMessages).toHaveBeenCalledWith(removed);
        updated.forEach(message => {
            expect(editMessage).toHaveBeenCalledWith({ ...message, status: "sent" });
        });
    });

    it("should do nothing if no messages are added, updated, or removed", () => {
        processMessages({ added: [], updated: [], removed: [] }, addMessages, editMessage, removeMessages);
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

    beforeEach(() => {
        jest.clearAllMocks();
        global.localStorage.clear();
        setCookie("Preferences", fullPreferences);
    });
    after(() => {
        global.localStorage.clear();
        jest.restoreAllMocks();
    });

    it("should add joining participants to the chat", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [{ user: { id: "user2" }, id: "participant2" }] as Omit<ChatParticipant, "chat">[];
        const leaving = [];

        processParticipantsUpdates({ joining, leaving }, participants, chat, setParticipants);

        expect(setParticipants).toHaveBeenCalledWith([...participants, ...joining]);
        // expect(getCookieMatchingChat(participants.map(p => p.user.id))).toEqual(chat.id); //TODO localstorage in these test suites not working properly for some reason
    });

    it("should remove leaving participants from the chat", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
            { user: { id: "user2" }, id: "participant2" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [];
        const leaving = ["user1"];

        processParticipantsUpdates({ joining, leaving }, participants, chat, setParticipants);

        expect(setParticipants).toHaveBeenCalledWith(participants.filter(p => p.user.id !== "user1"));
        // expect(getCookieMatchingChat(participants.map(p => p.user.id))).toEqual(['user2']);
    });

    it("should handle empty joining and leaving arrays", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [];
        const leaving = [];

        processParticipantsUpdates({ joining, leaving }, participants, chat, setParticipants);

        expect(setParticipants).not.toHaveBeenCalled();
        // expect(getCookieMatchingChat(participants.map(p => p.user.id))).toEqual(['user1']);
    });

    it("should not call setParticipants if no changes are made - test1", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [{ user: { id: "user1" }, id: "participant1" }] as Omit<ChatParticipant, "chat">[]; // user1 is already in participants
        const leaving = [];

        processParticipantsUpdates({ joining, leaving }, participants, chat, setParticipants);

        expect(setParticipants).not.toHaveBeenCalled();
        // expect(getCookieMatchingChat(participants.map(p => p.user.id))).toEqual(['user1']);
    });

    it("should not call setParticipants if no changes are made - test2", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [];
        const leaving = ["user2"]; // user2 is not in participants

        processParticipantsUpdates({ joining, leaving }, participants, chat, setParticipants);

        expect(setParticipants).not.toHaveBeenCalled();
        // expect(getCookieMatchingChat(participants.map(p => p.user.id))).toEqual(['user1']);
    });

    it("should not call setParticipants if no changes are made - test3", () => {
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [{ user: { id: "user2" }, id: "participant2" }] as Omit<ChatParticipant, "chat">[];
        const leaving = ["user2"]; // Same one as in joining

        processParticipantsUpdates({ joining, leaving }, participants, chat, setParticipants);

        expect(setParticipants).not.toHaveBeenCalled();
        // expect(getCookieMatchingChat(participants.map(p => p.user.id))).toEqual(['user1']);
    });
});

describe("processLlmTasks", () => {
    beforeEach(() => {
        jest.clearAllMocks();
        global.localStorage.clear();
        setCookie("Preferences", fullPreferences);
    });
    after(() => {
        global.localStorage.clear();
        jest.restoreAllMocks();
    });

    it("should handle empty tasks and updates", () => {
        processLlmTasks({ tasks: [], updates: [] }, "chat1");

        const storedTasks = getCookieTasksForChat("chat1");
        expect(storedTasks?.inactiveTasks).to.be.undefined;
    });

    it("should process full tasks", () => {
        const tasks = [
            { taskId: "task1", status: "Running" },
            { taskId: "task2", status: "Suggested" },
        ] as LlmTaskInfo[];
        upsertCookieTaskForChat("chat1", tasks[0]);
        upsertCookieTaskForChat("chat1", tasks[1]);
        processLlmTasks({ tasks, updates: [] }, "chat1");

        const storedTasks = getCookieTasksForChat("chat1");
        expect(storedTasks?.inactiveTasks).toEqual(tasks.reverse());
    });

    it("should ignore partial tasks when tasks haven't been stored yet", () => {
        const updates = [
            { taskId: "task3", status: "Completed" },
            { taskId: "task4", status: "Failed" },
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks: [], updates }, "chat2");

        const storedTasks = getCookieTasksForChat("chat2");
        expect(storedTasks?.inactiveTasks).toEqual([]);
    });

    it("should apply partial tasks to existing tasks", () => {
        const tasks = [
            { taskId: "task1", status: "Running" },
            { taskId: "task2", status: "Suggested" },
        ] as LlmTaskInfo[];
        processLlmTasks({ tasks, updates: [] }, "chat1");

        const updates = [
            { taskId: "task1", status: "Completed" },
            { taskId: "task2", status: "Failed" },
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks: [], updates }, "chat1");

        const storedTasks = getCookieTasksForChat("chat1");
        expect(storedTasks?.inactiveTasks).toEqual([
            { taskId: "task2", status: "Failed", lastUpdated: expect.any(String) },
            { taskId: "task1", status: "Completed", lastUpdated: expect.any(String) },
        ]);
    });

    it("should use valid `lastUpdated` date strings for partial tasks", () => {
        const tasks = [
            { taskId: "task1", status: "Running" },
            { taskId: "task2", status: "Suggested" },
        ] as LlmTaskInfo[];
        processLlmTasks({ tasks, updates: [] }, "chat1");

        const updates = [
            { taskId: "task1", status: "Completed" },
            { taskId: "task2", status: "Failed" },
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks: [], updates }, "chat1");

        const storedTasks = getCookieTasksForChat("chat1");
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        storedTasks!.inactiveTasks.forEach(task => {
            // Make sure `lastUpdated` can be parsed as a valid date
            expect(new Date(task.lastUpdated).toString()).not.to.equal("Invalid Date");
        });
    });

    it("should process full tasks and partial tasks at the same time", () => {
        const tasks = [
            { taskId: "task1", status: "Running" },
            { taskId: "task2", status: "Suggested" },
        ] as LlmTaskInfo[];
        const updates = [
            { taskId: "task1", status: "Completed" }, // Same ID as in `tasks`, so should overwrite
            { taskId: "task4", status: "Failed" }, // New ID, so should skip (since it wasn't stored yet)
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks, updates }, "chat1");

        const storedTasks = getCookieTasksForChat("chat1");
        expect(storedTasks?.inactiveTasks).toEqual([
            { taskId: "task2", status: "Suggested" }, // Wasn't updated, so no `lastUpdated` change
            { taskId: "task1", status: "Completed", lastUpdated: expect.any(String) },
        ]);
    });

    it("should process tasks for 2 different chats", () => {
        const chat1Tasks = [
            { taskId: "task1", status: "Running" },
            { taskId: "task2", status: "Suggested" },
        ] as LlmTaskInfo[];
        const chat2Tasks = [
            { taskId: "task3", status: "Running" },
            { taskId: "task4", status: "Suggested" },
        ] as LlmTaskInfo[];
        processLlmTasks({ tasks: chat1Tasks, updates: [] }, "chat1");
        processLlmTasks({ tasks: chat2Tasks, updates: [] }, "chat2");

        const chat1StoredTasks = getCookieTasksForChat("chat1");
        const chat2StoredTasks = getCookieTasksForChat("chat2");
        expect(chat1StoredTasks?.inactiveTasks).toEqual(chat1Tasks.reverse());
        expect(chat2StoredTasks?.inactiveTasks).toEqual(chat2Tasks.reverse());
    });

    it("should ignore tasks without an ID", () => {
        const tasks = [
            { status: "Running" },
            { status: "Suggested" },
        ] as LlmTaskInfo[];
        processLlmTasks({ tasks, updates: [] }, "chat1");

        const storedTasks = getCookieTasksForChat("chat1");
        expect(storedTasks?.inactiveTasks).to.be.undefined;
    });

    it("should ignore updates without an ID", () => {
        const updates = [
            { status: "Completed" },
            { status: "Failed" },
        ] as Partial<LlmTaskInfo>[];
        processLlmTasks({ tasks: [], updates }, "chat1");

        const storedTasks = getCookieTasksForChat("chat1");
        expect(storedTasks?.inactiveTasks).toEqual([]);
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
        expect(messageStreamRef.current.message).to.equal("Hello World");
    });

    it("should clear message and update botId on new bot message", () => {
        messageStreamRef.current = { __type: "stream", botId: "bot123", message: "Hello" };
        processResponseStream({ __type: "stream", botId: "bot456", message: "New message" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current).toEqual({ __type: "stream", botId: "bot456", message: "New message" });
    });

    it("should handle error type by continuing to append message", () => {
        processResponseStream({ __type: "error", botId: "bot123", message: "Error occurred" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current.message).to.equal("Error occurred");
    });

    it("should clear the stream on end type", () => {
        console.log = jest.fn();
        processResponseStream({ __type: "end", botId: "bot123", message: "Complete" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current).to.be.null;
        expect(setMessageStream).toHaveBeenCalledWith(null);
    });

    it("should not call update functions when no changes are made", () => {
        processResponseStream({ __type: "end", botId: "bot123", message: "Complete" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(setMessageStream).toHaveBeenCalledWith(null);
        expect(throttledSetMessageStream).not.toHaveBeenCalled();
    });
});
