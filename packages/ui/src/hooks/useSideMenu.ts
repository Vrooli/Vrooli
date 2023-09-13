import { useCallback, useEffect, useState } from "react";
import { getSideMenuState, setSideMenuState } from "utils/cookies";
import { PubSub, SideMenuPub } from "utils/pubsub";

/** Hook for simplifying a side menu's pub/sub events */
export const useSideMenu = (
    id: SideMenuPub["id"],
    isMobile: boolean,
) => {
    const defaultOpenState: boolean = getSideMenuState(id, false);
    const [isOpen, setIsOpen] = useState<boolean>(isMobile ? false : defaultOpenState);
    useEffect(() => {
        const sideMenuSub = PubSub.get().subscribeSideMenu((data) => {
            if (data.id !== id) return;
            setIsOpen(data.isOpen);
            setSideMenuState(id, data.isOpen);
        });
        return (() => {
            PubSub.get().unsubscribe(sideMenuSub);
        });
    }, [id]);

    const close = useCallback(() => {
        setIsOpen(false);
        setSideMenuState(id, false);
        PubSub.get().publishSideMenu({ id, isOpen: false });
    }, [id]);

    return {
        isOpen,
        setIsOpen,
        close,
    } as const;
};
