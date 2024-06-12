import { useCallback, useEffect, useState } from "react";
import { getCookie, setCookie } from "utils/cookies";
import { PubSub, SideMenuPub } from "utils/pubsub";

/** Hook for simplifying a side menu's pub/sub events */
export const useSideMenu = ({
    id,
    idPrefix,
    isMobile,
}: {
    id: SideMenuPub["id"],
    idPrefix?: string,
    isMobile?: boolean,
}) => {
    const defaultOpenState = getCookie("SideMenuState", `${idPrefix}${id}`);
    const [isOpen, setIsOpen] = useState<boolean>(isMobile ? false : defaultOpenState);
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("sideMenu", (data) => {
            if (data.id !== id || data.idPrefix !== idPrefix) return;
            setIsOpen(data.isOpen);
            setCookie("SideMenuState", data.isOpen, `${idPrefix}${id}`);
        });
        return unsubscribe;
    }, [id, idPrefix]);

    const close = useCallback(() => {
        setIsOpen(false);
        setCookie("SideMenuState", false, `${idPrefix}${id}`);
        PubSub.get().publish("sideMenu", { id, idPrefix, isOpen: false });
    }, [id, idPrefix]);

    return { isOpen, setIsOpen, close } as const;
};
