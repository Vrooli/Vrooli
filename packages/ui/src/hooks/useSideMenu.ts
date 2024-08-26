import { useCallback, useEffect, useState } from "react";
import { getCookie, setCookie } from "utils/cookies";
import { PubSub, SideMenuPub } from "utils/pubsub";

const COOKIE_PREFIX = "SideMenuState";

/** Hook for simplifying a side menu's pub/sub events */
export function useSideMenu({
    id,
    isMobile,
}: {
    id: SideMenuPub["id"],
    isMobile?: boolean,
}) {
    const defaultOpenState = getCookie(COOKIE_PREFIX, id);
    const [isOpen, setIsOpen] = useState<boolean>(isMobile ? false : defaultOpenState);
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("sideMenu", (data) => {
            if (data.id !== id) return;
            setIsOpen(data.isOpen);
            setCookie(COOKIE_PREFIX, data.isOpen, id);
        });
        return unsubscribe;
    }, [id]);

    const close = useCallback(() => {
        setIsOpen(false);
        setCookie(COOKIE_PREFIX, false, id);
        PubSub.get().publish("sideMenu", { id, isOpen: false });
    }, [id]);

    return { isOpen, setIsOpen, close } as const;
}
