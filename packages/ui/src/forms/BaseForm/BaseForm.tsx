import { FullPageSpinner } from "components/FullPageSpinner/FullPageSpinner"
import { BaseFormProps } from "forms/types"

/**
 * Form tag used to wrap any form. Has a fallback loading state.
 */
export const BaseForm = ({
    children,
    isLoading = false,
    onSubmit,
    style,
}: BaseFormProps) => {

    return (
        <form onSubmit={onSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            ...(style ?? {})
        }}
        >
            {isLoading ? <FullPageSpinner /> : children}
        </form>
    )
}