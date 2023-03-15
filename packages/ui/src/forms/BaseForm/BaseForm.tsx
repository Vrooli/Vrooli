import { FullPageSpinner } from "components";
import { BaseFormProps } from "../types";

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