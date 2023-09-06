import { exists } from "@local/shared";
import { Form } from "formik";
import { BaseFormProps } from "forms/types";
import { usePromptBeforeUnload } from "hooks/usePromptBeforeUnload";
import { forwardRef, useCallback, useEffect, useImperativeHandle } from "react";

export type BaseFormRef = {
    handleClose: (onClose: () => void, closeAnyway?: boolean) => void
}

export const BaseForm = forwardRef<BaseFormRef | undefined, BaseFormProps>(({
    children,
    dirty,
    display,
    isLoading = false,
    maxWidth,
    promptBeforeUnload = true,
    style,
}, ref) => {
    // Check for valid props
    useEffect(() => {
        if (promptBeforeUnload && !exists(dirty)) {
            console.warn("BaseForm: promptBeforeUnload is true but dirty is not defined. This will cause the prompt to never appear.");
        }
    }, [promptBeforeUnload, dirty]);

    // Alert user if they try to close/refresh the tab with unsaved changes.
    // NOTE: This only works for pages. Dialogs must implement their own logic.
    usePromptBeforeUnload({ shouldPrompt: promptBeforeUnload && dirty });
    const handleClose = useCallback((onClose: () => void, closeAnyway?: boolean) => {
        if (dirty && closeAnyway !== true) {
            if (window.confirm("You have unsaved changes. Are you sure you want to close?")) {
                onClose();
            }
        } else {
            onClose();
        }
    }, [dirty]);
    useImperativeHandle(ref, () => ({ handleClose }));

    return (
        <Form style={{
            display: "block",
            margin: "auto",
            alignItems: "center",
            justifyContent: "center",
            width: maxWidth ? `min(${maxWidth}px, 100vw - ${display === "page" ? "16px" : "64px"})` : "-webkit-fill-available",
            maxWidth: "100%",
            paddingBottom: "64px", // Make room for the submit buttons
            paddingLeft: display === "dialog" ? "env(safe-area-inset-left)" : undefined,
            paddingRight: display === "dialog" ? "env(safe-area-inset-right)" : undefined,
            ...(style ?? {}),
        }}>
            {/* When loading, display a dark overlay */}
            {isLoading && <div style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                height: "100%",
                backgroundColor: "rgba(0, 0, 0, 0.2)",
                zIndex: 1,
            }} />}
            {children}
        </Form>
    );
});
