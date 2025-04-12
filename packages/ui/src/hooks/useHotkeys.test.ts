import { renderHook } from "@testing-library/react";
import { expect } from "chai";
import sinon from "sinon";
import { useHotkeys } from "./useHotkeys.js";

describe("useHotkeys", () => {
    let mockCallback: sinon.SinonSpy;

    beforeEach(() => {
        mockCallback = sinon.spy();
    });

    afterEach(() => {
        sinon.restore();
    });

    const scenarios = [
        // Single Modifiers with 'a'
        { keys: ["a"], ctrlKey: true, shiftKey: false, altKey: false },
        { keys: ["a"], ctrlKey: false, shiftKey: true, altKey: false },
        { keys: ["a"], ctrlKey: false, shiftKey: false, altKey: true },
        // Double Modifiers with 'a'
        { keys: ["a"], ctrlKey: true, shiftKey: true, altKey: false },
        { keys: ["a"], ctrlKey: true, shiftKey: false, altKey: true },
        { keys: ["a"], ctrlKey: false, shiftKey: true, altKey: true },
        // Triple Modifiers with 'a'
        { keys: ["a"], ctrlKey: true, shiftKey: true, altKey: true },
        // Single Modifiers with 'b'
        { keys: ["b"], ctrlKey: true, shiftKey: false, altKey: false },
        { keys: ["b"], ctrlKey: false, shiftKey: true, altKey: false },
        { keys: ["b"], ctrlKey: false, shiftKey: false, altKey: true },
        // Double Modifiers with 'b'
        { keys: ["b"], ctrlKey: true, shiftKey: true, altKey: false },
        { keys: ["b"], ctrlKey: true, shiftKey: false, altKey: true },
        { keys: ["b"], ctrlKey: false, shiftKey: true, altKey: true },
        // Triple Modifiers with 'b'
        { keys: ["b"], ctrlKey: true, shiftKey: true, altKey: true },
        // Different keys without modifiers
        { keys: ["Enter"], ctrlKey: false, shiftKey: false, altKey: false },
        { keys: ["Escape"], ctrlKey: false, shiftKey: false, altKey: false },
        // Different keys with single modifier
        { keys: ["Enter"], ctrlKey: true, shiftKey: false, altKey: false },
        { keys: ["Escape"], ctrlKey: false, shiftKey: true, altKey: false },
        // Different keys with multiple modifiers
        { keys: ["Enter"], ctrlKey: true, shiftKey: true, altKey: false },
        { keys: ["Escape"], ctrlKey: false, shiftKey: true, altKey: true },
    ];

    scenarios.forEach(({ keys, ctrlKey, shiftKey, altKey }) => {
        it(`calls callback for key "${keys[0]}" with ctrlKey: ${ctrlKey}, shiftKey: ${shiftKey}, and altKey: ${altKey}`, () => {
            const hotkeys = [{ keys, ctrlKey, shiftKey, altKey, callback: mockCallback }];

            renderHook(() => useHotkeys(hotkeys, true));

            document.dispatchEvent(new KeyboardEvent("keydown", { key: keys[0], ctrlKey, shiftKey, altKey }));

            expect(mockCallback.calledOnce).to.be.true;
        });

        it(`does not call callback for incorrect key with ctrlKey: ${ctrlKey}, shiftKey: ${shiftKey}, and altKey: ${altKey}`, () => {
            const hotkeys = [{ keys, ctrlKey, shiftKey, altKey, callback: mockCallback }];

            renderHook(() => useHotkeys(hotkeys, true));

            // Dispatch with a different key
            document.dispatchEvent(new KeyboardEvent("keydown", { key: "wrongKey", ctrlKey, shiftKey, altKey }));

            expect(mockCallback.called).to.be.false;
        });
    });

    it("does not add event listener when condition is false", () => {
        const hotkeys = [{ keys: ["a"], ctrlKey: true, callback: mockCallback }];

        // Render the hook with the condition set to false
        renderHook(() => useHotkeys(hotkeys, false));

        // Attempt to trigger the hotkey
        document.dispatchEvent(new KeyboardEvent("keydown", { key: "a", ctrlKey: true }));

        // The callback should not have been called since the event listener should not have been added
        expect(mockCallback.called).to.be.false;
    });

    it("attaches and detaches event listener from the specified targetRef", () => {
        // Create a mock element with mocked addEventListener and removeEventListener
        const mockElement = {
            addEventListener: sinon.spy(),
            removeEventListener: sinon.spy(),
        };
        const targetRef = { current: mockElement };
        const hotkeys = [{ keys: ["a"], callback: mockCallback }];

        const { unmount } = renderHook(() => useHotkeys(hotkeys, true, targetRef as any));

        expect(mockElement.addEventListener.calledOnce).to.be.true;
        expect(mockElement.addEventListener.firstCall.args[0]).to.equal("keydown");
        expect(typeof mockElement.addEventListener.firstCall.args[1]).to.equal("function");

        unmount();

        expect(mockElement.removeEventListener.calledOnce).to.be.true;
        expect(mockElement.removeEventListener.firstCall.args[0]).to.equal("keydown");
        expect(typeof mockElement.removeEventListener.firstCall.args[1]).to.equal("function");
    });

    describe("requirePrecedingWhitespace functionality", () => {
        let inputElement: HTMLInputElement;
        let contentEditableElement: HTMLDivElement;

        beforeEach(() => {
            // Create an input element
            inputElement = document.createElement("input");
            inputElement.type = "text";
            document.body.appendChild(inputElement);

            // Create a contenteditable div
            contentEditableElement = document.createElement("div");
            contentEditableElement.contentEditable = "true";
            document.body.appendChild(contentEditableElement);
        });

        afterEach(() => {
            document.body.removeChild(inputElement);
            document.body.removeChild(contentEditableElement);
        });

        it("triggers hotkey when preceding character is whitespace in input element", () => {
            const hotkeys = [{ keys: ["a"], requirePrecedingWhitespace: true, callback: mockCallback }];
            renderHook(() => useHotkeys(hotkeys, true));

            // Set input value and cursor position after a space
            inputElement.value = "test space";
            inputElement.selectionStart = 5;  // Position after the space
            inputElement.selectionEnd = 5;
            inputElement.focus();

            // Simulate keydown
            const event = new KeyboardEvent("keydown", { key: "a", bubbles: true });
            inputElement.dispatchEvent(event);
            expect(mockCallback.calledOnce).to.be.true;
        });

        it("does not trigger hotkey when preceding character is not whitespace in input element", () => {
            const hotkeys = [{ keys: ["a"], requirePrecedingWhitespace: true, callback: mockCallback }];
            renderHook(() => useHotkeys(hotkeys, true));

            // Set input value and cursor position after a non-space character
            inputElement.value = "test";
            inputElement.selectionStart = 4;
            inputElement.selectionEnd = 4;
            inputElement.focus();

            // Simulate keydown
            inputElement.dispatchEvent(new KeyboardEvent("keydown", { key: "a", bubbles: true }));
            expect(mockCallback.called).to.be.false;
        });

        it("triggers hotkey at start of input", () => {
            const hotkeys = [{ keys: ["a"], requirePrecedingWhitespace: true, callback: mockCallback }];
            renderHook(() => useHotkeys(hotkeys, true));

            // Set cursor at start of input
            inputElement.value = "test";
            inputElement.selectionStart = 0;
            inputElement.selectionEnd = 0;
            inputElement.focus();

            // Simulate keydown
            inputElement.dispatchEvent(new KeyboardEvent("keydown", { key: "a", bubbles: true }));
            expect(mockCallback.calledOnce).to.be.true;
        });

        it("triggers hotkey when preceding character is newline in contenteditable", () => {
            const hotkeys = [{ keys: ["a"], requirePrecedingWhitespace: true, callback: mockCallback }];
            renderHook(() => useHotkeys(hotkeys, true));

            // Set content and selection after newline
            contentEditableElement.innerHTML = "test<br>";

            // Position cursor right after the BR element
            const range = document.createRange();
            const sel = window.getSelection();

            // Set range to be right after the BR element
            range.setStart(contentEditableElement, 2); // Position after both text node and BR
            range.collapse(true);

            // Focus first, then set selection
            contentEditableElement.focus();
            sel?.removeAllRanges();
            sel?.addRange(range);

            // Simulate keydown
            const event = new KeyboardEvent("keydown", {
                key: "a",
                bubbles: true,
                composed: true, // Ensure the event can cross shadow DOM boundaries
            });
            contentEditableElement.dispatchEvent(event);
            expect(mockCallback.calledOnce).to.be.true;
        });

        it("does not trigger hotkey when preceding character is not whitespace in contenteditable", () => {
            const hotkeys = [{ keys: ["a"], requirePrecedingWhitespace: true, callback: mockCallback }];
            renderHook(() => useHotkeys(hotkeys, true));

            // Set content and selection after non-whitespace
            contentEditableElement.textContent = "test";
            const range = document.createRange();
            const sel = window.getSelection();
            range.setStartAfter(contentEditableElement.firstChild!);
            range.collapse(true);
            sel?.removeAllRanges();
            sel?.addRange(range);
            contentEditableElement.focus();

            // Simulate keydown
            contentEditableElement.dispatchEvent(new KeyboardEvent("keydown", { key: "a", bubbles: true }));
            expect(mockCallback.called).to.be.false;
        });
    });

    describe("preventDefault functionality", () => {
        let mockEvent: KeyboardEvent;

        beforeEach(() => {
            // Create a keyboard event
            mockEvent = new KeyboardEvent("keydown", {
                key: "a",
                bubbles: true,
                cancelable: true, // Make sure the event is cancelable for preventDefault
            });
            // Spy on preventDefault after event creation
            sinon.spy(mockEvent, "preventDefault");
        });

        it("calls preventDefault by default", () => {
            const hotkeys = [{ keys: ["a"], callback: mockCallback }];
            renderHook(() => useHotkeys(hotkeys, true));

            document.dispatchEvent(mockEvent);

            expect(mockCallback.calledOnce).to.be.true;
            expect((mockEvent.preventDefault as sinon.SinonSpy).calledOnce).to.be.true;
        });

        it("calls preventDefault when explicitly set to true", () => {
            const hotkeys = [{ keys: ["a"], preventDefault: true, callback: mockCallback }];
            renderHook(() => useHotkeys(hotkeys, true));

            document.dispatchEvent(mockEvent);

            expect(mockCallback.calledOnce).to.be.true;
            expect((mockEvent.preventDefault as sinon.SinonSpy).calledOnce).to.be.true;
        });

        it("does not call preventDefault when set to false", () => {
            const hotkeys = [{ keys: ["a"], preventDefault: false, callback: mockCallback }];
            renderHook(() => useHotkeys(hotkeys, true));

            document.dispatchEvent(mockEvent);

            expect(mockCallback.calledOnce).to.be.true;
            expect((mockEvent.preventDefault as sinon.SinonSpy).called).to.be.false;
        });

        it("respects preventDefault setting for multiple hotkeys", () => {
            const mockCallbackTwo = sinon.spy();
            const hotkeys = [
                { keys: ["a"], preventDefault: false, callback: mockCallback },
                { keys: ["b"], callback: mockCallbackTwo },
            ];
            renderHook(() => useHotkeys(hotkeys, true));

            // Test first hotkey (preventDefault: false)
            document.dispatchEvent(mockEvent);
            expect(mockCallback.calledOnce).to.be.true;
            expect((mockEvent.preventDefault as sinon.SinonSpy).called).to.be.false;

            // Test second hotkey (default preventDefault behavior)
            const eventTwo = new KeyboardEvent("keydown", {
                key: "b",
                bubbles: true,
                cancelable: true,
            });
            sinon.spy(eventTwo, "preventDefault");
            document.dispatchEvent(eventTwo);
            expect(mockCallbackTwo.calledOnce).to.be.true;
            expect((eventTwo.preventDefault as sinon.SinonSpy).calledOnce).to.be.true;
        });
    });
});
