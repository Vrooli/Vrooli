import { userTranslationValidation } from "@local/shared";
import { TextField } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { Field } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormSection } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
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

    // // Handle handles
    // const [handlesField, , handlesHelpers] = useField("handle");
    // const [findHandles, { data: handlesData, loading: handlesLoading }] = useLazyFetch<FindHandlesInput, string[]>(endpointPostWalletHandles);
    // const [handles, setHandles] = useState<string[]>([]);
    // const fetchHandles = useCallback(() => {
    //     if (numVerifiedWallets > 0) {
    //         findHandles({}); // Intentionally empty
    //     } else {
    //         PubSub.get().publishSnack({ messageKey: "NoVerifiedWallets", severity: "Error" });
    //     }
    // }, [numVerifiedWallets, findHandles]);
    // useEffect(() => {
    //     if (handlesData) {
    //         setHandles(handlesData);
    //     }
    // }, [handlesData]);

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={500}
            >
                <ProfilePictureInput
                    onChange={(newPicture) => props.setFieldValue("profileImage", newPicture)}
                    name="profileImage"
                    profile={{ __typename: "User" }}
                    zIndex={zIndex}
                />
                <FormSection sx={{ marginTop: 2 }}>
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                        zIndex={zIndex}
                    />
                    {/* <Stack direction="row" spacing={0}>
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
                    </Stack> */}
                    <Field fullWidth name="name" label={t("Name")} as={TextField} />
                    <TranslatedMarkdownInput
                        language={language}
                        maxChars={2048}
                        minRows={4}
                        name="bio"
                        placeholder={t("Bio")}
                        zIndex={zIndex}
                    />
                </FormSection>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
                zIndex={zIndex}
            />
        </>
    );
};
