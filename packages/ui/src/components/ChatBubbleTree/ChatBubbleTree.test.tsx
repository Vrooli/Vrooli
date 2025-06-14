/* eslint-disable @typescript-eslint/ban-ts-comment */
import { cleanup, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import React from "react";
import { render, screen } from "../../__test/testUtils.js";
import { NavigationArrows, ScrollToBottomButton, ChatBubbleTree } from "./ChatBubbleTree.js";
import { MessageTree } from "../../hooks/messages.js";
import { generatePK, type ChatMessageShape, type ChatSocketEventPayloads } from "@vrooli/shared";

describe("NavigationArrows Component", () => {
    const handleIndexChangeMock = vi.fn();
    
    afterEach(() => {
        handleIndexChangeMock.mockClear();
    });

    it("should not render if there are less than 2 siblings", () => {
        const { container } = render(<NavigationArrows activeIndex={0} numSiblings={1} onIndexChange={handleIndexChangeMock} />);
        expect(container.firstChild).toBeNull();
    });

    it("should handle navigation between siblings", () => {
        const { rerender } = render(<NavigationArrows activeIndex={0} numSiblings={3} onIndexChange={handleIndexChangeMock} />);
        const buttons = screen.getAllByRole("button");
        
        // Test initial state
        expect(buttons[0].hasAttribute('disabled')).toBe(true);
        expect(buttons[1].hasAttribute('disabled')).toBe(false);
        expect(screen.getByText("1/3")).toBeTruthy();
        
        // Click next
        fireEvent.click(buttons[1]);
        expect(handleIndexChangeMock).toHaveBeenCalledWith(1);
        
        // Update to middle position
        rerender(<NavigationArrows activeIndex={1} numSiblings={3} onIndexChange={handleIndexChangeMock} />);
        expect(screen.getByText("2/3")).toBeTruthy();
        
        // Recheck buttons
        const updatedButtons = screen.getAllByRole("button");
        expect(updatedButtons[0].hasAttribute('disabled')).toBe(false);
        expect(updatedButtons[1].hasAttribute('disabled')).toBe(false);
        
        // Click previous
        handleIndexChangeMock.mockClear();
        fireEvent.click(updatedButtons[0]);
        expect(handleIndexChangeMock).toHaveBeenCalledWith(0);
        
        // Update to last position
        rerender(<NavigationArrows activeIndex={2} numSiblings={3} onIndexChange={handleIndexChangeMock} />);
        const finalButtons = screen.getAllByRole("button");
        expect(finalButtons[0].hasAttribute('disabled')).toBe(false);
        expect(finalButtons[1].hasAttribute('disabled')).toBe(true);
    });
});

describe("ScrollToBottomButton", () => {
    let containerElement: HTMLDivElement;
    let scrollMock: ReturnType<typeof vi.fn>;

    beforeEach(() => {
        // Create a mock container element with minimal setup
        containerElement = document.createElement("div");
        scrollMock = vi.fn((options) => {
            if (options?.top !== undefined) {
                containerElement.scrollTop = options.top;
            }
        });
        
        // Define all properties at once for better performance
        Object.defineProperties(containerElement, {
            scrollHeight: { value: 1000, writable: true, configurable: true },
            scrollTop: { value: 0, writable: true, configurable: true },
            clientHeight: { value: 500, writable: true, configurable: true },
            scroll: { value: scrollMock, writable: true, configurable: true }
        });
    });

    afterEach(() => {
        cleanup();
    });

    it("renders and handles scroll events correctly", () => {
        const containerRef = { current: containerElement };
        render(<ScrollToBottomButton containerRef={containerRef} />);
        
        const button = screen.getByRole("button");
        expect(button).toBeTruthy();
        
        // Initial state - not at bottom
        expect(button.classList.contains("hide-scroll-button")).toBe(false);
        
        // Scroll to middle
        containerElement.scrollTop = 300;
        fireEvent.scroll(containerElement);
        expect(button.classList.contains("hide-scroll-button")).toBe(false);
        
        // Scroll close to bottom
        containerElement.scrollTop = 901;
        fireEvent.scroll(containerElement);
        expect(button.classList.contains("hide-scroll-button")).toBe(true);
        
        // Click button to scroll to bottom
        fireEvent.click(button);
        expect(scrollMock).toHaveBeenCalledWith({ top: 1000, behavior: "smooth" });
        expect(containerElement.scrollTop).toBe(1000);
    });

    it("cleans up properly on unmount", () => {
        const removeEventListenerSpy = vi.spyOn(containerElement, "removeEventListener");
        const containerRef = { current: containerElement };
        
        const { unmount } = render(<ScrollToBottomButton containerRef={containerRef} />);
        unmount();
        
        expect(removeEventListenerSpy).toHaveBeenCalledWith("scroll", expect.any(Function));
        removeEventListenerSpy.mockRestore();
    });
});

describe("ChatBubbleTree Streaming", () => {
    it("should display streaming message with accumulated text", async () => {
        const tree = new MessageTree<ChatMessageShape>();
        const branches = {};
        const setBranches = vi.fn();
        const handleEdit = vi.fn();
        const handleRegenerateResponse = vi.fn();
        const handleReply = vi.fn();
        const handleRetry = vi.fn();
        const removeMessages = vi.fn();

        const botId = generatePK().toString();
        
        // Start with no stream
        const { rerender } = render(
            <ChatBubbleTree
                tree={tree}
                branches={branches}
                setBranches={setBranches}
                handleEdit={handleEdit}
                handleRegenerateResponse={handleRegenerateResponse}
                handleReply={handleReply}
                handleRetry={handleRetry}
                removeMessages={removeMessages}
                isBotOnlyChat={true}
                isEditingMessage={false}
                isReplyingToMessage={false}
                messageStream={null}
            />,
        );
        
        // No message should be displayed
        expect(screen.queryByText("Hello")).toBeNull();
        
        // Stream first chunk
        rerender(
            <ChatBubbleTree
                tree={tree}
                branches={branches}
                setBranches={setBranches}
                handleEdit={handleEdit}
                handleRegenerateResponse={handleRegenerateResponse}
                handleReply={handleReply}
                handleRetry={handleRetry}
                removeMessages={removeMessages}
                isBotOnlyChat={true}
                isEditingMessage={false}
                isReplyingToMessage={false}
                messageStream={{
                    __type: "stream",
                    chunk: "Hello",
                    botId,
                } as ChatSocketEventPayloads["responseStream"]}
            />,
        );
        
        // Should display first chunk
        await waitFor(() => {
            expect(screen.getByText("Hello")).toBeTruthy();
        });
        
        // Stream second chunk
        rerender(
            <ChatBubbleTree
                tree={tree}
                branches={branches}
                setBranches={setBranches}
                handleEdit={handleEdit}
                handleRegenerateResponse={handleRegenerateResponse}
                handleReply={handleReply}
                handleRetry={handleRetry}
                removeMessages={removeMessages}
                isBotOnlyChat={true}
                isEditingMessage={false}
                isReplyingToMessage={false}
                messageStream={{
                    __type: "stream",
                    chunk: "World",
                    botId,
                } as ChatSocketEventPayloads["responseStream"]}
            />,
        );
        
        // Should display accumulated text
        await waitFor(() => {
            expect(screen.getByText("Hello World")).toBeTruthy();
        });
        
        // End stream with final message
        rerender(
            <ChatBubbleTree
                tree={tree}
                branches={branches}
                setBranches={setBranches}
                handleEdit={handleEdit}
                handleRegenerateResponse={handleRegenerateResponse}
                handleReply={handleReply}
                handleRetry={handleRetry}
                removeMessages={removeMessages}
                isBotOnlyChat={true}
                isEditingMessage={false}
                isReplyingToMessage={false}
                messageStream={{
                    __type: "end",
                    finalMessage: "Hello World!",
                    botId,
                } as ChatSocketEventPayloads["responseStream"]}
            />,
        );
        
        // Should display final message
        await waitFor(() => {
            expect(screen.getByText("Hello World!")).toBeTruthy();
        });
    });
});