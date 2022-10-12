import { useEffect } from "react";

interface UsePromptBeforeUnloadProps {
    shouldPrompt?: boolean;
}

/**
 * Prompt the user before unloading the page.
 */
export const usePromptBeforeUnload = ({ shouldPrompt = true }: UsePromptBeforeUnloadProps) => {
    useEffect(() => {
        const handleBeforeUnload = (e: BeforeUnloadEvent) => {
            if (shouldPrompt) {
                e.preventDefault()
                e.returnValue = ''
            }
        };
        window.addEventListener('beforeunload', handleBeforeUnload);
        return () => window.removeEventListener('beforeunload', handleBeforeUnload);
    }, [shouldPrompt]);
}