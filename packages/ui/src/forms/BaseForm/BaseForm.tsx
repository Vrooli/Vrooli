import { exists } from "@shared/utils"
import { FullPageSpinner } from "components/FullPageSpinner/FullPageSpinner"
import { Form } from "formik"
import { BaseFormProps } from "forms/types"
import { useEffect } from "react"
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload"

/**
 * Form tag used to wrap any form. Has a fallback loading state.
 */
export const BaseForm = ({
    children,
    dirty,
    isLoading = false,
    promptBeforeUnload = true,
    style,
}: BaseFormProps) => {

    useEffect(() => {
        if (promptBeforeUnload && !exists(dirty)) {
            console.warn('BaseForm: promptBeforeUnload is true but dirty is not defined. This will cause the prompt to never appear.');
        }
    }, [promptBeforeUnload, dirty])
    // Alert user if they try to leave/refresh with unsaved changes
    usePromptBeforeUnload({ shouldPrompt: promptBeforeUnload && dirty })

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
}