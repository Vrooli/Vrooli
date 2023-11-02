import { useCallback, useEffect } from "react";

interface useConfirmBeforeLeaveProps {
    handleCancel?: () => unknown,
    shouldPrompt: boolean,
}

export const useConfirmBeforeLeave = ({ handleCancel, shouldPrompt }: useConfirmBeforeLeaveProps) => {

    /** Handle closing by button or backdrop click */
    const handleClose = useCallback(() => {
        if (shouldPrompt && window.confirm("You have unsaved changes. Are you sure you want to close?")) {
            handleCancel?.();
        } else {
            handleCancel?.();
        }
    }, [handleCancel, shouldPrompt]);

    useEffect(() => {
        if (shouldPrompt) {
            localStorage.setItem("blockNavigation", "true");
        } else {
            localStorage.removeItem("blockNavigation");
        }
    }, [shouldPrompt]);

    useEffect(() => {
        const handleBeforeUnload = (e: BeforeUnloadEvent) => {
            if (shouldPrompt) {
                e.preventDefault();
                e.returnValue = "";
            }
        };
        window.addEventListener("beforeunload", handleBeforeUnload);
        return () => window.removeEventListener("beforeunload", handleBeforeUnload);
    }, [shouldPrompt]);

    return { handleClose };
};
