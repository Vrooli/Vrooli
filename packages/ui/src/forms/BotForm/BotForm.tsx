import { botTranslationValidation, botValidation, DUMMY_ID, Session, User } from "@local/shared";
import { Slider, Stack, TextField, Typography } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { Field, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { BotFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { BotShape, BotTranslationShape, shapeBot } from "utils/shape/models/bot";

export const botInitialValues = (
    session: Session | undefined,
    existing?: User | null | undefined,
): BotShape => {
    // Try to parse stringified settings
    let settings: Record<string, string | number> = {};
    try {
        settings = JSON.parse(existing?.botSettings ?? "{}");
    } catch (error) {
        console.error("Failed to parse settings", error);
    }
    // Inject string settings into translations
    const translations: BotTranslationShape[] = existing?.translations?.map((translation) => ({
        ...settings.translations?.[translation.language],
        bias: "",
        domainKnowledge: "",
        keyPhrases: "",
        occupation: "",
        persona: "",
        tone: "",
        ...translation,
    })) ?? [{
        __typename: "UserTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        bio: "",
        bias: "",
        domainKnowledge: "",
        keyPhrases: "",
        occupation: "",
        persona: "",
        tone: "",
    }];
    return {
        __typename: "User" as const,
        id: DUMMY_ID,
        creativity: 0.5,
        isPrivate: false,
        name: "Bot Name",
        verbosity: 0.5,
        ...existing,
        isBot: true,
        translations,
    };
};

export function transformBotValues(session: Session | undefined, values: BotShape, existing?: User | null | undefined) {
    return existing === undefined
        ? shapeBot.create(values)
        : shapeBot.update(botInitialValues(session, existing), values);
}

export const validateBotValues = async (session: Session | undefined, values: BotShape, existing?: User | null | undefined) => {
    const transformedValues = transformBotValues(session, values, existing);
    const validationSchema = botValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const BotForm = forwardRef<BaseFormRef | undefined, BotFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const [creativityField, , creativityHelpers] = useField("creativity");
    const [verbosityField, , verbosityHelpers] = useField("verbosity");

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
        fields: ["description", "name"],
        validationSchema: botTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={700}
                ref={ref}
            >
                <FormContainer>
                    <RelationshipList
                        isEditing={true}
                        objectType={"User"}
                        zIndex={zIndex}
                        sx={{ marginBottom: 4 }}
                    />
                    <FormSection sx={{
                        overflowX: "hidden",
                    }}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                            zIndex={zIndex + 1}
                        />
                        <Field
                            fullWidth
                            name="name"
                            label={t("Name")}
                            as={TextField}
                        />
                        <TranslatedMarkdownInput
                            language={language}
                            maxChars={2048}
                            minRows={4}
                            name="bio"
                            placeholder={t("Bio")}
                            zIndex={zIndex}
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Occupation")}
                            language={language}
                            name="occupation"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Persona")}
                            language={language}
                            name="persona"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Tone")}
                            language={language}
                            name="tone"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("KeyPhrases")}
                            language={language}
                            name="keyPhrases"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("DomainKnowledge")}
                            language={language}
                            name="domainKnowledge"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Bias")}
                            language={language}
                            name="bias"
                        />
                        <Stack>
                            <Typography id="creativity-slider" gutterBottom>
                                {t("Creativity")}
                            </Typography>
                            <Slider
                                aria-labelledby="creativity-slider"
                                value={creativityField.value}
                                onChange={(_, value) => creativityHelpers.setValue(value)}
                                valueLabelDisplay="auto"
                                min={0.1}
                                max={1}
                                step={0.1}
                                marks={[
                                    {
                                        value: 0.1,
                                        label: t("Low"),
                                    },
                                    {
                                        value: 1,
                                        label: t("High"),
                                    },
                                ]}
                                sx={{
                                    "& .MuiSlider-markLabel": {
                                        "&[data-index=\"0\"]": {
                                            marginLeft: 2,
                                        },
                                        "&[data-index=\"1\"]": {
                                            marginLeft: -2,
                                        },
                                    },
                                }}
                            />
                        </Stack>
                        <Stack>
                            <Typography id="verbosity-slider" gutterBottom>
                                {t("Verbosity")}
                            </Typography>
                            <Slider
                                aria-labelledby="verbosity-slider"
                                value={verbosityField.value}
                                onChange={(_, value) => verbosityHelpers.setValue(value)}
                                valueLabelDisplay="auto"
                                min={0.1}
                                max={1}
                                step={0.1}
                                marks={[
                                    {
                                        value: 0.1,
                                        label: t("Low"),
                                    },
                                    {
                                        value: 1,
                                        label: t("High"),
                                    },
                                ]}
                                sx={{
                                    "& .MuiSlider-markLabel": {
                                        "&[data-index=\"0\"]": {
                                            marginLeft: 2,
                                        },
                                        "&[data-index=\"1\"]": {
                                            marginLeft: -2,
                                        },
                                    },
                                }}
                            />
                        </Stack>
                    </FormSection>
                </FormContainer>
                <GridSubmitButtons
                    display={display}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                    zIndex={zIndex}
                />
            </BaseForm>
        </>
    );
});
