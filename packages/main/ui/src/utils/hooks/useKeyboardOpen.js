import { useEffect, useState } from "react";
export const useKeyboardOpen = (minKeyboardHeight = 300) => {
    const [isKeyboardOpen, setIsKeyboardOpen] = useState(false);
    useEffect(() => {
        const listener = () => {
            if (!window.visualViewport)
                return;
            const newState = (window.visualViewport.height - window.innerHeight) > minKeyboardHeight;
            if (isKeyboardOpen !== newState) {
                setIsKeyboardOpen(newState);
            }
        };
        window.visualViewport?.addEventListener("resize", listener);
        return () => {
            window.visualViewport?.removeEventListener("resize", listener);
        };
    }, [isKeyboardOpen, minKeyboardHeight]);
    return isKeyboardOpen;
};
//# sourceMappingURL=useKeyboardOpen.js.map