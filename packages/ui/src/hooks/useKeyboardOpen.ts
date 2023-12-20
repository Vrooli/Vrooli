import { useEffect, useState } from "react";

/**
 * Hook to detect if the keyboard is open or not. 
 * Useful for hiding fixed elements (e.g. BottomNav).
 */
export const useKeyboardOpen = (): boolean => {
    const [isKeyboardOpen, setIsKeyboardOpen] = useState(false);

    useEffect(() => {
        // Function to handle focus event
        const handleFocus = () => {
            setIsKeyboardOpen(true);
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
