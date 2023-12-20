import { useEffect, useState } from "react";
import { getDeviceInfo } from "utils/display/device";

/**
 * Hook to detect if the keyboard is open or not. 
 * Useful for hiding fixed elements (e.g. BottomNav).
 */
export const useKeyboardOpen = (): boolean => {
    const [isKeyboardOpen, setIsKeyboardOpen] = useState(false);

    useEffect(() => {
        // Function to handle focus event
        const handleFocus = (event: FocusEvent) => {
            const target = event.target as HTMLElement;
            const isTextBox = target?.role === "textbox" || target?.tagName === "TEXTAREA";
            setIsKeyboardOpen(isTextBox && getDeviceInfo().isMobile);
        };

        // Function to handle blur event
        const handleBlur = () => {
            setIsKeyboardOpen(false);
        };

        // Add event listeners
        window.addEventListener("focusin", handleFocus);
        window.addEventListener("focusout", handleBlur);

        // Cleanup
        return () => {
            window.removeEventListener("focusin", handleFocus);
            window.removeEventListener("focusout", handleBlur);
        };
    }, []);

    return isKeyboardOpen;
};
