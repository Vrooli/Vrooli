import { useCallback, useState } from "react";

/**
 * Stores the state in the browser history,
 * making the state reusable across refreshes, navigation
 * and even closing and reopening the window!
 *
 * @param key The key to store it in history
 * @param defaultTo A default value if nothing exists in history
 */
export function useHistoryState<T>(key: string, defaultTo: T) {
    const [state, rawSetState] = useState<T>(() => {
        const value = window.history.state && window.history.state[key];
        return value || defaultTo;
    });

    const setState = useCallback(function setStateCallback(update: T | ((prevState: T) => T)) {
        const newValue = update instanceof Function ? update(state) : update;
        window.history.replaceState(
            { ...window.history.state, [key]: newValue },
            document.title,
        );
        rawSetState(newValue);
    }, [state, key]);

    return [state, setState] as const;
}
