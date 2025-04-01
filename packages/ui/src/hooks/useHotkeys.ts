import { useEffect } from "react";

type HotkeyConfig = {
    keys: string[],
    ctrlKey?: boolean,
    altKey?: boolean,
    shiftKey?: boolean,
    requirePrecedingWhitespace?: boolean,
    callback: () => unknown
};

export function useHotkeys(
    hotkeys: HotkeyConfig[],
    condition = true,
    targetRef: React.RefObject<HTMLElement> | null = null,
) {
    useEffect(() => {
        // Sort hotkeys by number of conditions to prioritize more specific hotkeys
        const sortedHotkeys = [...hotkeys].sort((a, b) => {
            const conditionsA = [a.ctrlKey, a.altKey, a.shiftKey].filter(Boolean).length;
            const conditionsB = [b.ctrlKey, b.altKey, b.shiftKey].filter(Boolean).length;
            return conditionsB - conditionsA;
        });

        function checkPrecedingWhitespace(element: HTMLElement): boolean {
            // Handle different types of input elements
            if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement) {
                const cursorPosition = element.selectionStart;
                if (cursorPosition === null) return false;
                const text = element.value;
                // If at start of text, consider it valid
                if (cursorPosition === 0) return true;
                // Check if previous character is whitespace
                return /[\s\n]/.test(text[cursorPosition - 1]);
            } else if (element.isContentEditable || element.contentEditable === "true") {
                const selection = window.getSelection();
                if (!selection || !selection.rangeCount) return false;

                const range = selection.getRangeAt(0);

                // If we're at the start of the element
                if (range.startContainer === element && range.startOffset === 0) {
                    return true;
                }

                // If we're right after a BR element
                if (range.startContainer === element) {
                    const childNodes = Array.from(element.childNodes);
                    const prevNode = childNodes[range.startOffset - 1];
                    if (prevNode && prevNode.nodeName === "BR") {
                        return true;
                    }
                }

                // Get the text content up to the cursor
                const preCaretRange = range.cloneRange();
                preCaretRange.selectNodeContents(element);
                preCaretRange.setEnd(range.startContainer, range.startOffset);
                const text = preCaretRange.toString();

                // If at start of text, consider it valid
                if (text.length === 0) return true;

                // Check if previous character is whitespace
                return /[\s\n]/.test(text[text.length - 1]);
            }

            console.error("[useHotkeys.checkPrecedingWhitespace] Unsupported element type");
            return false;
        }

        function handleKeyDown(e: Event) {
            // Check if the event is a KeyboardEvent
            if (e instanceof KeyboardEvent) {
                for (const hotkey of sortedHotkeys) {
                    const keysMatch = hotkey.keys.includes(e.key);
                    const ctrlMatch = hotkey.ctrlKey === undefined || hotkey.ctrlKey === e.ctrlKey;
                    const altMatch = hotkey.altKey === undefined || hotkey.altKey === e.altKey;
                    const shiftMatch = hotkey.shiftKey === undefined || hotkey.shiftKey === e.shiftKey;

                    // Check for preceding whitespace if required
                    const whitespaceMatch = !hotkey.requirePrecedingWhitespace || (
                        e.target instanceof HTMLElement && checkPrecedingWhitespace(e.target)
                    );

                    if (keysMatch && ctrlMatch && altMatch && shiftMatch && whitespaceMatch) {
                        e.preventDefault();
                        hotkey.callback();
                        break; // Stop checking further once a match is found
                    }
                }
            }
        }

        const targetElement = targetRef && targetRef.current ? targetRef.current : document;

        if (condition) {
            targetElement.addEventListener("keydown", handleKeyDown);
            return () => {
                targetElement.removeEventListener("keydown", handleKeyDown);
            };
        }
    }, [hotkeys, condition, targetRef]);
}
