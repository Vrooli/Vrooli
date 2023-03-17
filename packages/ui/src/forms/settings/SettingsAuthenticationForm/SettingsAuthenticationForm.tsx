import { Grid, TextField } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { PasswordTextField } from "components/inputs/PasswordTextField/PasswordTextField";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
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
        <BaseForm
            dirty={dirty}
            isLoading={isLoading}
            style={{
                width: { xs: '100%', md: 'min(100%, 700px)' },
                margin: 'auto',
                display: 'block',
            }}
        >

            {/* Hidden username input because some password managers require it */}
            <TextField
                name="username"
                autoComplete="username"
                sx={{ display: 'none' }}
            />
            <Grid container spacing={1}>
                <Grid item xs={12}>
                    <PasswordTextField
                        fullWidth
                        name="currentPassword"
                        label={t('PasswordCurrent')}
                        autoComplete="current-password"
                    />
                </Grid>
                <Grid item xs={12}>
                    <PasswordTextField
                        fullWidth
                        id="newPassword"
                        name="newPassword"
                        label={t('PasswordNew')}
                        autoComplete="new-password"
                    />
                </Grid>
                <Grid item xs={12}>
                    <PasswordTextField
                        fullWidth
                        id="newPasswordConfirmation"
                        name="newPasswordConfirmation"
                        autoComplete="new-password"
                        label={t('PasswordNewConfirmation')}
                    />
                </Grid>
            </Grid>
            <GridSubmitButtons
                display={display}
                errors={props.errors}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </BaseForm>
    )
}