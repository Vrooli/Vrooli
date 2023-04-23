import { useState } from "react";
export function useHistoryState(key, defaultTo) {
    const [state, rawSetState] = useState(() => {
        const value = window.history.state && window.history.state[key];
        return value || defaultTo;
    });
    function setState(value) {
        window.history.replaceState({ ...window.history.state, [key]: value }, document.title);
        rawSetState(value);
    }
    return [state, setState];
}
//# sourceMappingURL=useHistoryState.js.map