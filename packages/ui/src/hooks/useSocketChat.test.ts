// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
import { generatePK, type ChatMessage, type ChatParticipant, type ChatShape, type LlmTaskInfo, type Session } from "@vrooli/shared";
import { describe, it, expect, afterAll, beforeEach, vi } from "vitest";
import { fullPreferences, getCookieTasksForChat, setCookie, upsertCookieTaskForChat } from "../utils/localStorage.js";
import { processLlmTasks, processMessages, processParticipantsUpdates, processResponseStream, processTypingUpdates } from "./useSocketChat.js";

describe("processMessages - Real-time chat message synchronization", () => {
    const addMessages = vi.fn();
    const removeMessages = vi.fn();
    const editMessage = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("displays new messages in the chat when other users send them", () => {
        // GIVEN: Other users send new messages
        const messages = [
            { taskId: "msg1", content: "Hello" },
            { taskId: "msg2", content: "World" },
        ] as unknown as ChatMessage[];
        
        // WHEN: The socket receives new messages
        processMessages({ added: messages, updated: [], removed: [] }, addMessages, editMessage, removeMessages);
        
        // THEN: Messages should be added to the chat UI with sent status
        expect(addMessages).toHaveBeenCalledWith([
            { taskId: "msg1", content: "Hello", status: "sent" },
            { taskId: "msg2", content: "World", status: "sent" },
        ]);
        expect(removeMessages).not.toHaveBeenCalled();
        expect(editMessage).not.toHaveBeenCalled();
    });

    it("updates existing messages when they are edited by the sender", () => {
        // GIVEN: Messages that have been edited
        const messages = [
            { taskId: "msg1", content: "Updated Hello" },
            { taskId: "msg2", content: "Updated World" },
        ] as unknown as ChatMessage[];
        
        // WHEN: Socket receives message updates
        processMessages({ added: [], updated: messages, removed: [] }, addMessages, editMessage, removeMessages);
        
        // THEN: Messages should be updated in the UI
        messages.forEach(message => {
            expect(editMessage).toHaveBeenCalledWith({ ...message, status: "sent" });
        });
        expect(addMessages).not.toHaveBeenCalled();
        expect(removeMessages).not.toHaveBeenCalled();
    });

    it("removes messages from chat when they are deleted", () => {
        // GIVEN: Messages that have been deleted
        const messages = ["msg1", "msg2"];
        
        // WHEN: Socket receives message deletions
        processMessages({ added: [], updated: [], removed: messages }, addMessages, editMessage, removeMessages);
        
        // THEN: Messages should be removed from the UI
        expect(removeMessages).toHaveBeenCalledWith(messages);
        expect(addMessages).not.toHaveBeenCalled();
        expect(editMessage).not.toHaveBeenCalled();
    });

    it("handles simultaneous message operations in a single update", () => {
        // GIVEN: Multiple simultaneous chat operations
        const added = [{ taskId: "msg3", content: "New" }] as unknown as ChatMessage[];
        const updated = [{ taskId: "msg2", content: "Updated World" }] as unknown as ChatMessage[];
        const removed = ["msg1"];

        // WHEN: Socket receives a complex update
        processMessages({ added, updated, removed }, addMessages, editMessage, removeMessages);
        
        // THEN: All operations should be applied correctly
        expect(addMessages).toHaveBeenCalledWith([{ taskId: "msg3", content: "New", status: "sent" }]);
        expect(removeMessages).toHaveBeenCalledWith(removed);
        updated.forEach(message => {
            expect(editMessage).toHaveBeenCalledWith({ ...message, status: "sent" });
        });
    });

    it("efficiently handles empty updates without unnecessary operations", () => {
        // GIVEN: An update with no actual changes
        // WHEN: Socket receives an empty update
        processMessages({ added: [], updated: [], removed: [] }, addMessages, editMessage, removeMessages);
        
        // THEN: No UI operations should be performed
        expect(addMessages).not.toHaveBeenCalled();
        expect(removeMessages).not.toHaveBeenCalled();
        expect(editMessage).not.toHaveBeenCalled();
    });
});

describe("processTypingUpdates - Real-time typing indicators", () => {
    const currentUserId = generatePK().toString();
    const session = {
        isLoggedIn: true,
        users: [{ id: currentUserId }],
    } as Session;
    const setUsersTyping = vi.fn();
    const participants = [
        { user: { id: "user1" } },
        { user: { id: "user2" } },
        { user: { id: "user3" } },
        { user: { id: currentUserId } }, // this is the current user
    ] as Omit<ChatParticipant, "chat">[];

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("shows typing indicators when other users start typing", () => {
        // GIVEN: Multiple users start typing (including current user)
        const starting = ["user1", "user2", currentUserId];
        const stopping = [];
        const usersTyping = [];

        // WHEN: Typing update is received
        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        
        // THEN: Only other users' typing indicators should be shown (not current user's)
        expect(setUsersTyping).toHaveBeenCalledWith([
            { user: { id: "user1" } },
            { user: { id: "user2" } },
        ]);
    });

    it("hides typing indicators when users stop typing", () => {
        // GIVEN: Users who were typing
        const starting = [];
        const stopping = ["user1"];
        const usersTyping = [
            { user: { id: "user1" } },
            { user: { id: "user2" } },
        ] as Omit<ChatParticipant, "chat">[];

        // WHEN: User1 stops typing
        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        
        // THEN: User1's typing indicator should be removed
        expect(setUsersTyping).toHaveBeenCalledWith([
            { user: { id: "user2" } },
        ]);
    });

    it("updates typing indicators correctly when users simultaneously start and stop typing", () => {
        // GIVEN: User1 is typing, user3 is not
        const starting = ["user3"];
        const stopping = ["user1"];
        const usersTyping = [
            { user: { id: "user1" } },
            { user: { id: "user2" } },
        ] as Omit<ChatParticipant, "chat">[];

        // WHEN: User3 starts typing while user1 stops
        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        
        // THEN: Typing list should be updated correctly
        expect(setUsersTyping).toHaveBeenCalledWith([
            { user: { id: "user2" } },
            { user: { id: "user3" } },
        ]);
    });

    it("ignores typing updates from users not in the chat", () => {
        // GIVEN: A user not in the participants list
        const starting = ["user4"]; // user4 is not in participants
        const stopping = [];
        const usersTyping = [];

        // WHEN: Non-participant tries to show typing
        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        
        // THEN: No typing indicator should be shown
        expect(setUsersTyping).not.toHaveBeenCalled();
    });

    it("avoids unnecessary updates when typing state hasn't changed", () => {
        // GIVEN: No changes in typing state
        const starting = [];
        const stopping = [];
        const usersTyping = [{ user: { id: "user1" } }] as Omit<ChatParticipant, "chat">[];

        // WHEN: Empty update is received
        processTypingUpdates({ starting, stopping }, usersTyping, participants, session, setUsersTyping);
        
        // THEN: No UI update should occur
        expect(setUsersTyping).not.toHaveBeenCalled();
    });

    it("handles anonymous users gracefully when no session exists", () => {
        // GIVEN: No authenticated session
        const starting = ["user1", currentUserId];
        const stopping = [];
        const usersTyping = [];

        // WHEN: Typing updates are received without session
        processTypingUpdates({ starting, stopping }, usersTyping, participants, undefined, setUsersTyping);
        
        // THEN: All users' typing indicators should be shown (no current user to exclude)
        expect(setUsersTyping).toHaveBeenCalledWith([
            { user: { id: "user1" } },
            { user: { id: currentUserId } },
        ]);
    });
});

describe("processParticipantsUpdates - Chat participant management", () => {
    const setParticipants = vi.fn();
    const chat = { id: "chat1" } as ChatShape;

    beforeEach(() => {
        vi.clearAllMocks();
        global.localStorage.clear();
        setCookie("Preferences", fullPreferences);
    });
    afterAll(() => {
        global.localStorage.clear();
        vi.restoreAllMocks();
    });

    it("adds new users to chat when they join the conversation", () => {
        // GIVEN: An existing chat with one participant
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [{ user: { id: "user2" }, id: "participant2" }] as Omit<ChatParticipant, "chat">[];
        const leaving = [];

        // WHEN: A new user joins the chat
        processParticipantsUpdates({ joining, leaving }, participants, chat, setParticipants);

        // THEN: The new user should be added to participants list
        expect(setParticipants).toHaveBeenCalledWith([...participants, ...joining]);
    });

    it("removes users from chat when they leave the conversation", () => {
        // GIVEN: A chat with multiple participants
        const participants = [
            { user: { id: "user1" }, id: "participant1" },
            { user: { id: "user2" }, id: "participant2" },
        ] as Omit<ChatParticipant, "chat">[];
        const joining = [];
        const leaving = ["user1"];

        // WHEN: User1 leaves the chat
        processParticipantsUpdates({ joining, leaving }, participants, chat, setParticipants);

        // THEN: User1 should be removed from participants list
        expect(setParticipants).toHaveBeenCalledWith(participants.filter(p => p.user.id !== "user1"));
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
        vi.clearAllMocks();
        global.localStorage.clear();
        setCookie("Preferences", fullPreferences);
    });
    afterAll(() => {
        global.localStorage.clear();
        vi.restoreAllMocks();
    });

    it("should handle empty tasks and updates", () => {
        processLlmTasks({ tasks: [], updates: [] }, "chat1");

        const storedTasks = getCookieTasksForChat("chat1");
        expect(storedTasks?.inactiveTasks).toBeUndefined();
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
            expect(new Date(task.lastUpdated).toString()).not.toBe("Invalid Date");
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
        expect(storedTasks?.inactiveTasks).toBeUndefined();
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
    const setMessageStream = vi.fn();
    const throttledSetMessageStream = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
        messageStreamRef = { current: null };
    });

    it("should initialize and update stream for new messages", () => {
        processResponseStream({ __type: "stream", botId: "bot123", chunk: "Hello" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current).toEqual({ __type: "stream", botId: "bot123", accumulatedMessage: "Hello" });
        expect(throttledSetMessageStream).toHaveBeenCalledWith({ __type: "stream", botId: "bot123", accumulatedMessage: "Hello" });
    });

    it("should append to the existing message on stream type", () => {
        messageStreamRef.current = { __type: "stream", botId: "bot123", accumulatedMessage: "Hello" };
        processResponseStream({ __type: "stream", botId: "bot123", chunk: " World" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current.accumulatedMessage).toBe("Hello World");
    });

    it("should clear message and update botId on new bot message", () => {
        messageStreamRef.current = { __type: "stream", botId: "bot123", accumulatedMessage: "Hello" };
        processResponseStream({ __type: "stream", botId: "bot456", chunk: "New message" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current).toEqual({ __type: "stream", botId: "bot456", accumulatedMessage: "New message" });
    });

    it("should handle error type by continuing to append message", () => {
        processResponseStream({ __type: "error", botId: "bot123", error: { message: "Error occurred" } }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current.error?.message).toBe("Error occurred");
    });

    it("should clear the stream on end type", () => {
        console.log = vi.fn();
        processResponseStream({ __type: "end", botId: "bot123" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(messageStreamRef.current).toBeNull();
        expect(setMessageStream).toHaveBeenCalledWith(null);
    });

    it("should not call update functions when no changes are made", () => {
        processResponseStream({ __type: "end", botId: "bot123" }, messageStreamRef, setMessageStream, throttledSetMessageStream);
        expect(setMessageStream).toHaveBeenCalledWith(null);
        expect(throttledSetMessageStream).not.toHaveBeenCalled();
    });
});
