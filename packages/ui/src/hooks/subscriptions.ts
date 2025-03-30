import { useCallback, useEffect, useState } from "react";
import { getCookie, removeCookie, setCookie } from "../utils/localStorage.js";
import { PubSub } from "../utils/pubsub.js";

/**
 * Tracks if the site should display in left-handed mode.
 */
export function useIsLeftHanded() {
    const [isLeftHanded, setIsLeftHanded] = useState<boolean>(getCookie("IsLeftHanded"));
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("isLeftHanded", (data) => {
            setIsLeftHanded(data);
        });
        return unsubscribe;
    }, []);

    return isLeftHanded;
}

export function useShowBotWarning() {
    const [showBotWarning, setShowBotWarning] = useState<boolean>(getCookie("ShowBotWarning"));
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("showBotWarning", (data) => {
            setShowBotWarning(data);
        });
        return unsubscribe;
    }, []);

    const handleUpdateShowBotWarning = useCallback(function handleUpdateShowBotWarning(showWarning: boolean | null | undefined) {
        if (typeof showWarning !== "boolean") {
            removeCookie("ShowBotWarning");
            PubSub.get().publish("showBotWarning", true);
        } else {
            setCookie("ShowBotWarning", showWarning);
            PubSub.get().publish("showBotWarning", showWarning);
        }
    }, []);

    return { handleUpdateShowBotWarning, showBotWarning };
}
