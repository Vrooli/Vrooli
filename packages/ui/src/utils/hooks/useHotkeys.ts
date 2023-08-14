import { useEffect } from "react";

type HotkeyConfig = {
    keys: string[],
    ctrlKey?: boolean,
    altKey?: boolean,
    shiftKey?: boolean,
    callback: () => void
};

export const useHotkeys = (hotkeys: HotkeyConfig[], condition = true) => {
    useEffect(() => {
        // Sort hotkeys by number of conditions to prioritize more specific hotkeys
        const sortedHotkeys = [...hotkeys].sort((a, b) => {
            const conditionsA = [a.ctrlKey, a.altKey, a.shiftKey].filter(Boolean).length;
            const conditionsB = [b.ctrlKey, b.altKey, b.shiftKey].filter(Boolean).length;
            return conditionsB - conditionsA;
        });

        const handleKeyDown = (e: KeyboardEvent) => {
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
        };

        if (condition) {
            document.addEventListener("keydown", handleKeyDown);
            return () => {
                document.removeEventListener("keydown", handleKeyDown);
            };
        }
    }, [hotkeys, condition]);
};
