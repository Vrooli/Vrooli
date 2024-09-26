import { useCallback, useEffect, useState } from "react";
import { getCookie, setCookie } from "utils/localStorage";
import { PubSub, SideMenuPayloads, SideMenuPub } from "utils/pubsub";

const COOKIE_PREFIX = "SideMenuState";

type UseSideMenuParams<ID extends SideMenuPub["id"]> = {
    /** The id of the side menu */
    id: ID;
    /** Whether the app is in mobile mode, which changes the initial behavior of the side menu */
    isMobile: boolean;
    /** Callback to perform additional logic when a pub/sub event is received */
    onEvent?: (data: SideMenuPayloads[ID]) => unknown;
}

/** Hook for simplifying a side menu's pub/sub events */
export function useSideMenu<ID extends SideMenuPub["id"]>({
    id,
    isMobile,
    onEvent,
}: UseSideMenuParams<ID>) {
    const defaultOpenState = getCookie(COOKIE_PREFIX, id);
    const [isOpen, setIsOpen] = useState<boolean>(isMobile ? false : defaultOpenState);
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("sideMenu", (data) => {
            if (data.id !== id) return;
            setIsOpen(data.isOpen);
            setCookie(COOKIE_PREFIX, data.isOpen, id);
            onEvent?.(data as SideMenuPayloads[ID]);
        });
        return unsubscribe;
    }, [id, onEvent]);

    const close = useCallback(() => {
        setIsOpen(false);
        setCookie(COOKIE_PREFIX, false, id);
        PubSub.get().publish("sideMenu", { id, isOpen: false });
    }, [id]);

    return { isOpen, setIsOpen, close } as const;
}
