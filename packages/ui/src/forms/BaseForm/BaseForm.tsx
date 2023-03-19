import { exists } from "@shared/utils"
import { FullPageSpinner } from "components/FullPageSpinner/FullPageSpinner"
import { Form } from "formik"
import { BaseFormProps } from "forms/types"
import { forwardRef, useCallback, useEffect, useImperativeHandle } from "react"
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload"

export type BaseFormRef = {
    handleClose: (onClose: () => void, closeAnyway?: boolean) => void
}

export const BaseForm = forwardRef<BaseFormRef, BaseFormProps>(({
    children,
    dirty,
    isLoading = false,
    promptBeforeUnload = true,
    style
}, ref) => {
    // Check for valid props
    useEffect(() => {
        if (promptBeforeUnload && !exists(dirty)) {
            console.warn('BaseForm: promptBeforeUnload is true but dirty is not defined. This will cause the prompt to never appear.');
        }
    }, [promptBeforeUnload, dirty])

    // Alert user if they try to close/refresh the tab with unsaved changes
    usePromptBeforeUnload({ shouldPrompt: promptBeforeUnload && dirty })
    // Alert user if they try to close the dialog (if the form is in one) with unsaved changes
    const handleClose = useCallback((onClose: () => void, closeAnyway?: boolean) => {
        if (dirty && closeAnyway !== true) {
            if (window.confirm('You have unsaved changes. Are you sure you want to close?')) {
                onClose()
            }
        } else {
            onClose()
        }
    }, [dirty])
    useImperativeHandle(ref, () => ({ handleClose }))

    return (
        <Form style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            ...(style ?? {})
        }}>
            {isLoading ? <FullPageSpinner /> : children}
        </Form>
    )
})