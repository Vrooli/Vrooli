import { useCallback, useEffect, useState } from "react";
import { getCookieSideMenuState, setCookieSideMenuState } from "utils/cookies";
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
    const defaultOpenState: boolean = getCookieSideMenuState(`${idPrefix}${id}`, false);
    const [isOpen, setIsOpen] = useState<boolean>(isMobile ? false : defaultOpenState);
    useEffect(() => {
        const sideMenuSub = PubSub.get().subscribeSideMenu((data) => {
            if (data.id !== id || data.idPrefix !== idPrefix) return;
            setIsOpen(data.isOpen);
            setCookieSideMenuState(`${idPrefix}${id}`, data.isOpen);
        });
        return (() => {
            PubSub.get().unsubscribe(sideMenuSub);
        });
    }, [id, idPrefix]);

    const close = useCallback(() => {
        setIsOpen(false);
        setCookieSideMenuState(`${idPrefix}${id}`, false);
        PubSub.get().publishSideMenu({ id, idPrefix, isOpen: false });
    }, [id, idPrefix]);

    return { isOpen, setIsOpen, close } as const;
};
