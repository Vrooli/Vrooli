import { useEffect, useState } from "react";
import { getCookieIsLeftHanded } from "../cookies";
import { PubSub } from "../pubsub";

/**
 * Tracks if the site should display in left-handed mode.
 */
export const useIsLeftHanded = () => {
    const [isLeftHanded, setIsLeftHanded] = useState<boolean>(getCookieIsLeftHanded(false));
    useEffect(() => {
        // Handle isLeftHanded updates
        const isLeftHandedSub = PubSub.get().subscribeIsLeftHanded((data) => {
            setIsLeftHanded(data);
        });
        // On unmount, unsubscribe from all PubSub topics
        return (() => {
            PubSub.get().unsubscribe(isLeftHandedSub);
        });
    }, []);

    return isLeftHanded;
};
