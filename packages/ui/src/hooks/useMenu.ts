import { useCallback, useEffect, useState } from "react";
import { getCookie, setCookie } from "../utils/localStorage.js";
import { MenuPayloads, PubSub } from "../utils/pubsub.js";

type UseMenuParams<ID extends keyof MenuPayloads> = {
    /** The id of the side menu */
    id: ID;
    /** Whether the app is in mobile mode, which changes the initial behavior of the side menu */
    isMobile?: boolean;
    /** Callback to perform additional logic when a pub/sub event is received */
    onEvent?: (data: MenuPayloads[ID]) => unknown;
}

/** Hook for simplifying a menu's pub/sub events */
export function useMenu<ID extends keyof MenuPayloads>({
    id,
    isMobile,
    onEvent,
}: UseMenuParams<ID>) {
    const defaultOpenState = getCookie("MenuState", id);
    const [isOpen, setIsOpen] = useState<boolean>(isMobile ? false : defaultOpenState);
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("menu", (data) => {
            // Special case for closing all menus
            if (data.id === "all" && data.isOpen === false) {
                setIsOpen(false);
                setCookie("MenuState", false, id);
                return;
            }
            if (data.id !== id) return;
            setIsOpen(data.isOpen);
            setCookie("MenuState", data.isOpen, id);
            onEvent?.(data as MenuPayloads[ID]);
        });
        return unsubscribe;
    }, [id, onEvent]);

    const close = useCallback(() => {
        setIsOpen(false);
        setCookie("MenuState", false, id);
        PubSub.get().publish("menu", { id, isOpen: false });
    }, [id]);

    return { isOpen, setIsOpen, close } as const;
}
