import { useEffect } from "react";
export const usePromptBeforeUnload = ({ shouldPrompt = true }) => {
    useEffect(() => {
        const handleBeforeUnload = (e) => {
            if (shouldPrompt) {
                e.preventDefault();
                e.returnValue = "";
            }
        };
        window.addEventListener("beforeunload", handleBeforeUnload);
        return () => window.removeEventListener("beforeunload", handleBeforeUnload);
    }, [shouldPrompt]);
};
//# sourceMappingURL=usePromptBeforeUnload.js.map