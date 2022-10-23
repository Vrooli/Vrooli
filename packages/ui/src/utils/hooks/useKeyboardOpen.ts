import { useEffect, useState } from 'react';

/**
 * Hook to detect if the keyboard is open or not. Useful for hiding fixed elements when the keyboard is open.
 * @param minKeyboardHeight The minimum height of the keyboard to consider it open. Defaults to 300.
 * @returns Whether the keyboard is open or not.
 */
export const useKeyboardOpen = (minKeyboardHeight: number = 300): boolean => {
    const [isKeyboardOpen, setIsKeyboardOpen] = useState<boolean>(false);

    useEffect(() => {
        const listener = () => {
            if (!window.visualViewport) return;
            // Calculate difference in height between window and visual viewport. 
            const newState = (window.visualViewport.height - window.innerHeight) > minKeyboardHeight;
            // const newState = window.screen.height - minKeyboardHeight > window.visualViewport.height;
            if (isKeyboardOpen !== newState) {
                setIsKeyboardOpen(newState);
            }
        };
        window.visualViewport?.addEventListener('resize', listener);
        return () => {
            window.visualViewport?.removeEventListener('resize', listener);
        };
    }, [isKeyboardOpen, minKeyboardHeight]);

    return isKeyboardOpen;
};