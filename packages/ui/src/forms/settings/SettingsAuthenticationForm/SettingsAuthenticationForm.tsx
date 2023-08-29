import { TextField } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { PasswordTextField } from "components/inputs/PasswordTextField/PasswordTextField";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
import { FormSection } from "styles";
import { SettingsAuthenticationFormProps } from "../types";

export const SettingsAuthenticationForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsAuthenticationFormProps) => {
    const { t } = useTranslation();

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
            >
                {/* Hidden username input because some password managers require it */}
                <TextField
                    name="username"
                    autoComplete="username"
                    sx={{ display: "none" }}
                />
                <FormSection>
                    <PasswordTextField
                        fullWidth
                        name="currentPassword"
                        label={t("PasswordCurrent")}
                        autoComplete="current-password"
                    />
                    <PasswordTextField
                        fullWidth
                        name="newPassword"
                        label={t("PasswordNew")}
                        autoComplete="new-password"
                    />
                    <PasswordTextField
                        fullWidth
                        name="newPasswordConfirmation"
                        autoComplete="new-password"
                        label={t("PasswordNewConfirm")}
                    />
                </FormSection>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
};
