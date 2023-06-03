import { FindHandlesInput, userTranslationValidation } from "@local/shared";
import { Grid, TextField, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { Field, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
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
    zIndex,
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
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["bio"],
        validationSchema: userTranslationValidation.update({}),
    });

    // Handle handles
    const [handlesField, , handlesHelpers] = useField("handle");
    const [findHandles, { data: handlesData, loading: handlesLoading }] = useLazyFetch<FindHandlesInput, string[]>("/wallet/handles");
    const [handles, setHandles] = useState<string[]>([]);
    const fetchHandles = useCallback(() => {
        if (numVerifiedWallets > 0) {
            findHandles({}); // Intentionally empty
        } else {
            PubSub.get().publishSnack({ messageKey: "NoVerifiedWallets", severity: "Error" });
        }
    }, [numVerifiedWallets, findHandles]);
    useEffect(() => {
        if (handlesData) {
            setHandles(handlesData);
        }
    }, [handlesData]);

    return (
        <BaseForm
            dirty={dirty}
            isLoading={isLoading}
            style={{
                width: { xs: "100%", md: "min(100%, 700px)" },
                margin: "auto",
                display: "block",
            }}
        >
            <Grid container spacing={2} sx={{
                paddingBottom: 4,
                paddingLeft: 2,
                paddingRight: 2,
                paddingTop: 2,
            }}>
                <Grid item xs={12}>
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                        zIndex={zIndex}
                    />
                </Grid>
                {/* <Grid item xs={12}>
                    <Stack direction="row" spacing={0}>
                        <Autocomplete
                            disablePortal
                            id="ada-handle-select"
                            loading={handlesLoading}
                            options={handles}
                            onOpen={fetchHandles}
                            onChange={(_, value) => { handlesHelpers.setValue(value); }}
                            renderInput={(params) => <TextField
                                {...params}
                                label="(ADA) Handle"
                                sx={{
                                    "& .MuiInputBase-root": {
                                        borderRadius: "5px 0 0 5px",
                                    },
                                }}
                            />}
                            value={handlesField.value}
                            sx={{
                                width: "100%",
                            }}
                        />
                        <ColorIconButton
                            aria-label='fetch-handles'
                            background={palette.secondary.main}
                            onClick={fetchHandles}
                            sx={{ borderRadius: "0 5px 5px 0" }}
                        >
                            <RefreshIcon />
                        </ColorIconButton>
                    </Stack>
                </Grid> */}
                <Grid item xs={12}>
                    <Field fullWidth name="name" label={t("Name")} as={TextField} />
                </Grid>
                <Grid item xs={12}>
                    <TranslatedMarkdownInput
                        language={language}
                        minRows={4}
                        name="bio"
                        zIndex={zIndex}
                    />
                </Grid>
            </Grid>
            <GridSubmitButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </BaseForm>
    );
};
