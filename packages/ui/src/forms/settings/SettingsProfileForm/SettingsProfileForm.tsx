import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
import { SettingsProfileFormProps } from "../types";

export const SettingsProfileForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsProfileFormProps) => {
    const { t } = useTranslation();

    return (
        <BaseForm
            dirty={dirty}
            isLoading={isLoading}
            style={{
                width: { xs: '100%', md: 'min(100%, 700px)' },
                margin: 'auto',
                display: 'block',
            }}
        >
        </BaseForm>
    )
}