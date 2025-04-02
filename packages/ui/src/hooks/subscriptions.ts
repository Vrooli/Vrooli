import { useEffect, useState } from "react";
import { getCookie } from "../utils/localStorage.js";
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
