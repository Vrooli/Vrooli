// AI_CHECK: TEST_COVERAGE=1,TEST_QUALITY=1 | LAST: 2025-06-19
import { describe, it, expect, beforeEach, vi } from "vitest";
import { useDrawerStore, type DrawerView } from "./drawerStore.js";
import type { ChatConfigObject } from "@vrooli/shared";

describe("useDrawerStore", () => {
    beforeEach(() => {
        // Reset store to initial state before each test
        useDrawerStore.setState({
            rightDrawer: {
                view: "chat",
            },
        });
    });

    it("initializes with default state", () => {
        const state = useDrawerStore.getState();
        
        expect(state.rightDrawer).toEqual({
            view: "chat",
        });
        expect(typeof state.setRightDrawer).toBe("function");
        expect(typeof state.resetRightDrawer).toBe("function");
    });

    it("setRightDrawer updates drawer data partially", () => {
        const { setRightDrawer } = useDrawerStore.getState();
        
        setRightDrawer({ chatId: "test-chat-123" });
        
        const state = useDrawerStore.getState();
        expect(state.rightDrawer).toEqual({
            view: "chat",
            chatId: "test-chat-123",
        });
    });

    it("setRightDrawer merges with existing data", () => {
        const { setRightDrawer } = useDrawerStore.getState();
        
        // Set initial data
        setRightDrawer({ 
            chatId: "test-chat-123",
            view: "swarmDetail" as DrawerView,
        });
        
        // Update partially
        setRightDrawer({ swarmStatus: "running" });
        
        const state = useDrawerStore.getState();
        expect(state.rightDrawer).toEqual({
            view: "swarmDetail",
            chatId: "test-chat-123",
            swarmStatus: "running",
        });
    });

    it("setRightDrawer updates view correctly", () => {
        const { setRightDrawer } = useDrawerStore.getState();
        
        setRightDrawer({ view: "swarmDetail" });
        
        const state = useDrawerStore.getState();
        expect(state.rightDrawer.view).toBe("swarmDetail");
    });

    it("setRightDrawer handles swarm configuration", () => {
        const { setRightDrawer } = useDrawerStore.getState();
        
        const mockSwarmConfig: ChatConfigObject = {
            id: "swarm-123",
            __typename: "ChatConfig",
        } as ChatConfigObject;
        
        setRightDrawer({ 
            view: "swarmDetail",
            swarmConfig: mockSwarmConfig,
            swarmStatus: "idle",
        });
        
        const state = useDrawerStore.getState();
        expect(state.rightDrawer.view).toBe("swarmDetail");
        expect(state.rightDrawer.swarmConfig).toEqual(mockSwarmConfig);
        expect(state.rightDrawer.swarmStatus).toBe("idle");
    });

    it("setRightDrawer handles callback functions", () => {
        const { setRightDrawer } = useDrawerStore.getState();
        
        const mockOnStart = vi.fn();
        const mockOnPause = vi.fn();
        const mockOnResume = vi.fn();
        const mockOnStop = vi.fn();
        const mockOnApprove = vi.fn();
        const mockOnReject = vi.fn();
        
        setRightDrawer({
            onStart: mockOnStart,
            onPause: mockOnPause,
            onResume: mockOnResume,
            onStop: mockOnStop,
            onApproveToolCall: mockOnApprove,
            onRejectToolCall: mockOnReject,
        });
        
        const state = useDrawerStore.getState();
        expect(state.rightDrawer.onStart).toBe(mockOnStart);
        expect(state.rightDrawer.onPause).toBe(mockOnPause);
        expect(state.rightDrawer.onResume).toBe(mockOnResume);
        expect(state.rightDrawer.onStop).toBe(mockOnStop);
        expect(state.rightDrawer.onApproveToolCall).toBe(mockOnApprove);
        expect(state.rightDrawer.onRejectToolCall).toBe(mockOnReject);
    });

    it("resetRightDrawer restores default state", () => {
        const { setRightDrawer, resetRightDrawer } = useDrawerStore.getState();
        
        // First, modify the state
        setRightDrawer({
            view: "swarmDetail",
            chatId: "test-chat",
            swarmStatus: "running",
            onStart: vi.fn(),
        });
        
        // Verify state was changed
        let state = useDrawerStore.getState();
        expect(state.rightDrawer.view).toBe("swarmDetail");
        expect(state.rightDrawer.chatId).toBe("test-chat");
        
        // Reset to default
        resetRightDrawer();
        
        // Verify state was reset
        state = useDrawerStore.getState();
        expect(state.rightDrawer).toEqual({
            view: "chat",
        });
    });

    it("handles null values correctly", () => {
        const { setRightDrawer } = useDrawerStore.getState();
        
        setRightDrawer({ 
            chatId: null,
            swarmConfig: null,
        });
        
        const state = useDrawerStore.getState();
        expect(state.rightDrawer.chatId).toBeNull();
        expect(state.rightDrawer.swarmConfig).toBeNull();
    });

    it("handles multiple updates correctly", () => {
        const { setRightDrawer } = useDrawerStore.getState();
        
        // First update
        setRightDrawer({ view: "swarmDetail" });
        
        // Second update
        setRightDrawer({ chatId: "chat-123" });
        
        // Third update
        setRightDrawer({ swarmStatus: "running" });
        
        const state = useDrawerStore.getState();
        expect(state.rightDrawer).toEqual({
            view: "swarmDetail",
            chatId: "chat-123",
            swarmStatus: "running",
        });
    });

    it("handles overwriting existing values", () => {
        const { setRightDrawer } = useDrawerStore.getState();
        
        // Set initial value
        setRightDrawer({ chatId: "old-chat" });
        
        // Overwrite with new value
        setRightDrawer({ chatId: "new-chat" });
        
        const state = useDrawerStore.getState();
        expect(state.rightDrawer.chatId).toBe("new-chat");
    });

    it("notifies subscribers when store state changes", () => {
        let callbackCount = 0;
        let latestState: any = null;
        
        // Subscribe to store changes
        const unsubscribe = useDrawerStore.subscribe((state) => {
            callbackCount++;
            latestState = state;
        });
        
        const { setRightDrawer } = useDrawerStore.getState();
        
        // Make changes
        setRightDrawer({ chatId: "test-123" });
        setRightDrawer({ view: "swarmDetail" });
        
        expect(callbackCount).toBe(2);
        expect(latestState.rightDrawer.chatId).toBe("test-123");
        expect(latestState.rightDrawer.view).toBe("swarmDetail");
        
        unsubscribe();
    });
});
