import { Autocomplete, Stack, TextField, useTheme } from "@mui/material";
import { FindHandlesInput } from "@shared/consts";
import { RefreshIcon } from "@shared/icons";
import { userTranslationValidation } from "@shared/validation";
import { useCustomLazyQuery } from "api";
import { walletFindHandles } from "api/generated/endpoints/wallet_findHandles";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { Field, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { SettingsProfileFormProps } from "../types";

export const SettingsProfileForm = ({
    display,
    dirty,
    isLoading,
    numVerifiedWallets,
    onCancel,
    values,
    ...props
}: SettingsProfileFormProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        onTranslationBlur,
        onTranslationChange,
        setLanguage,
        translations,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['bio'],
        validationSchema: userTranslationValidation.update({}),
    });

    console.log('translations', translations);

    // Handle handles
    const [handlesField, , handlesHelpers] = useField('handle');
    const [findHandles, { data: handlesData, loading: handlesLoading }] = useCustomLazyQuery<string[], FindHandlesInput>(walletFindHandles);
    const [handles, setHandles] = useState<string[]>([]);
    const fetchHandles = useCallback(() => {
        if (numVerifiedWallets > 0) {
            findHandles({ variables: {} }); // Intentionally empty
        } else {
            PubSub.get().publishSnack({ messageKey: 'NoVerifiedWallets', severity: 'Error' })
        }
    }, [numVerifiedWallets, findHandles]);
    useEffect(() => {
        if (handlesData) {
            setHandles(handlesData);
        }
    }, [handlesData])

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
            {/* <LanguageInput
                currentLanguage={language}
                handleAdd={handleAddLanguage}
                handleDelete={handleDeleteLanguage}
                handleCurrent={setLanguage}
                translations={translations}
                zIndex={200}
            /> */}
            <Stack direction="row" spacing={0}>
                <Autocomplete
                    disablePortal
                    id="ada-handle-select"
                    loading={handlesLoading}
                    options={handles}
                    onOpen={fetchHandles}
                    onChange={(_, value) => { handlesHelpers.setValue(value) }}
                    renderInput={(params) => <TextField
                        {...params}
                        label="(ADA) Handle"
                        sx={{
                            '& .MuiInputBase-root': {
                                borderRadius: '5px 0 0 5px',
                            }
                        }}
                    />}
                    value={handlesField.value}
                    sx={{
                        width: '100%',
                    }}
                />
                <ColorIconButton
                    aria-label='fetch-handles'
                    background={palette.secondary.main}
                    onClick={fetchHandles}
                    sx={{ borderRadius: '0 5px 5px 0' }}
                >
                    <RefreshIcon />
                </ColorIconButton>
            </Stack>
            <Field fullWidth name="name" label={t('Name')} as={TextField} />
            {/* <Field fullWidth name="bio" label={t('Bio')} as={TextField} /> */}
            <TranslatedTextField
                fullWidth
                name="bio"
                label={t('Bio')}
                language={language}
                multiline
                minRows={4}
                maxRows={10}
            />
            <GridSubmitButtons
                display={display}
                errors={{
                    ...props.errors,
                    ...translations.errorsWithTranslations
                }}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </BaseForm>
    )
}