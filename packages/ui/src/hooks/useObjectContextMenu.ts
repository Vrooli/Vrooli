import { type ListObject } from "@local/shared";
import { useCallback, useState } from "react";

/** Hook for providing context menu logic for object lists */
export function useObjectContextMenu() {
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const [object, setObject] = useState<ListObject | null>(null);
    const handleContextMenu = useCallback((target: EventTarget, object: ListObject | null) => {
        if (!object) return;
        setAnchorEl(target as HTMLElement);
        setObject(object);
    }, []);
    const closeContextMenu = useCallback(() => {
        setAnchorEl(null);
        // Don't remove object, since dialogs opened from context menu may need it
    }, []);

    return {
        anchorEl,
        closeContextMenu,
        handleContextMenu,
        object,
        setObject,
    };
}
