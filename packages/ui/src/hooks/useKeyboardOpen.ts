import { useEffect, useState } from "react";
import { getDeviceInfo } from "../utils/display/device.js";

/**
 * Hook to detect if the keyboard is open or not. 
 * Useful for hiding fixed elements (e.g. BottomNav).
 */
export function useKeyboardOpen(): boolean {
    const [isKeyboardOpen, setIsKeyboardOpen] = useState(false);

    useEffect(() => {
        // Function to handle focus event
        function handleFocus(event: FocusEvent) {
            const target = event.target as HTMLElement;
            const isTextBox = target?.role === "textbox" || target?.tagName === "TEXTAREA";
            setIsKeyboardOpen(isTextBox && getDeviceInfo().isMobile);
        }

        // Function to handle blur event
        function handleBlur() {
            setIsKeyboardOpen(false);
        }

        // Add event listeners
        window.addEventListener("focusin", handleFocus);
        window.addEventListener("focusout", handleBlur);

        // Cleanup
        return () => {
            window.removeEventListener("focusin", handleFocus);
            window.removeEventListener("focusout", handleBlur);
        };
    }, []);

    //TODO this fixes initial state for pages like NoteCrud, but then you have to press "Submit" or "Cancel" twice
    // // Check initial state
    // useEffect(() => {
    //     const activeElement = document.activeElement as HTMLElement | null;
    //     if (!activeElement) return;
    //     const isTextBox = activeElement?.role === "textbox" || activeElement?.tagName === "TEXTAREA";
    //     setIsKeyboardOpen(isTextBox && getDeviceInfo().isMobile);
    // }, []);

    return isKeyboardOpen;
}
