import { useEffect, useState } from "react";
import { getCookieIsLeftHanded } from "../cookies";
import { PubSub } from "../pubsub";
export const useIsLeftHanded = () => {
    const [isLeftHanded, setIsLeftHanded] = useState(getCookieIsLeftHanded(false));
    useEffect(() => {
        const isLeftHandedSub = PubSub.get().subscribeIsLeftHanded((data) => {
            setIsLeftHanded(data);
        });
        return (() => {
            PubSub.get().unsubscribe(isLeftHandedSub);
        });
    }, []);
    return isLeftHanded;
};
//# sourceMappingURL=useIsLeftHanded.js.map