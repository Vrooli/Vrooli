import { useEffect } from "react";

type HotkeyConfig = {
    keys: string[],
    ctrlKey?: boolean,
    altKey?: boolean,
    shiftKey?: boolean,
    callback: () => void
};

export const useHotkeys = (hotkeys: HotkeyConfig[], condition = true, targetRef: React.RefObject<HTMLElement> | null = null) => {
    useEffect(() => {
        // Sort hotkeys by number of conditions to prioritize more specific hotkeys
        const sortedHotkeys = [...hotkeys].sort((a, b) => {
            const conditionsA = [a.ctrlKey, a.altKey, a.shiftKey].filter(Boolean).length;
            const conditionsB = [b.ctrlKey, b.altKey, b.shiftKey].filter(Boolean).length;
            return conditionsB - conditionsA;
        });

        const handleKeyDown = (e: Event) => {
            // Check if the event is a KeyboardEvent
            if (e instanceof KeyboardEvent) {
                for (const hotkey of sortedHotkeys) {
                    const keysMatch = hotkey.keys.includes(e.key);
                    const ctrlMatch = hotkey.ctrlKey === undefined || hotkey.ctrlKey === e.ctrlKey;
                    const altMatch = hotkey.altKey === undefined || hotkey.altKey === e.altKey;
                    const shiftMatch = hotkey.shiftKey === undefined || hotkey.shiftKey === e.shiftKey;

                    if (keysMatch && ctrlMatch && altMatch && shiftMatch) {
                        e.preventDefault();
                        hotkey.callback();
                        break; // Stop checking further once a match is found
                    }
                }
            }
        };

        const targetElement = targetRef && targetRef.current ? targetRef.current : document;

        if (condition) {
            targetElement.addEventListener("keydown", handleKeyDown);
            return () => {
                targetElement.removeEventListener("keydown", handleKeyDown);
            };
        }
    }, [hotkeys, condition, targetRef]);
};
