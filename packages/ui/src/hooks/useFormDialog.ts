import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { useCallback, useRef } from "react";

export const useFormDialog = ({
    handleCancel,
}: {
    handleCancel: () => unknown,
}) => {
    const formRef = useRef<BaseFormRef>();

    /** Shows confirmation dialog when closing a dirty dialog */
    const handleClose = useCallback((_?: unknown, reason?: "backdropClick" | "escapeKeyDown") => {
        // Confirm dialog is dirty and closed by clicking outside
        formRef.current?.handleClose(handleCancel, reason !== "backdropClick");
    }, [handleCancel]);

    return { formRef, handleClose };
};
