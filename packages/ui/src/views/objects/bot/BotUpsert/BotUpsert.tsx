import { BotCreateInput, botTranslationValidation, BotUpdateInput, botValidation, DUMMY_ID, endpointGetUser, endpointPostBot, endpointPutBot, Session, User } from "@local/shared";
import { InputAdornment, Slider, Stack, TextField, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { BotFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { BotIcon, HandleIcon } from "icons";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { findBotData } from "utils/botUtils";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { BotShape, shapeBot } from "utils/shape/models/bot";
import { BotUpsertProps } from "../types";

const botInitialValues = (
    session: Session | undefined,
    existing?: Partial<User> | BotShape | null | undefined,
): BotShape => {
    const { creativity, verbosity, translations } = findBotData(getUserLanguages(session)[0], existing);

    return {
        __typename: "User" as const,
        id: DUMMY_ID,
        creativity,
        isPrivate: false,
        name: "",
        verbosity,
        ...existing,
        isBot: true,
        translations,
    };
};

const transformBotValues = (session: Session | undefined, values: BotShape, existing: BotShape, isCreate: boolean) =>
    isCreate ? shapeBot.create(values) : shapeBot.update(botInitialValues(session, existing), values);

const validateBotValues = async (session: Session | undefined, values: BotShape, existing: BotShape, isCreate: boolean) => {
    const transformedValues = transformBotValues(session, values, existing, isCreate);
    const validationSchema = botValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

const BotForm = forwardRef<BaseFormRef | undefined, BotFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
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
        validationSchema: botTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
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
                        sx={{ marginBottom: 4 }}
                    />
                    <ProfilePictureInput
                        onBannerImageChange={(newPicture) => props.setFieldValue("bannerImage", newPicture)}
                        onProfileImageChange={(newPicture) => props.setFieldValue("profileImage", newPicture)}
                        name="profileImage"
                        profile={{ ...values }}
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
                        />
                        <Field
                            fullWidth
                            autoComplete="name"
                            name="name"
                            label={t("Name")}
                            placeholder={t("NamePlaceholder")}
                            as={TextField}
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <BotIcon />
                                    </InputAdornment>
                                ),
                            }}
                            error={props.touched.name && Boolean(props.errors.name)}
                            helperText={props.touched.name && props.errors.name}
                        />
                        <Field
                            fullWidth
                            autoComplete="handle"
                            name="handle"
                            label={t("Handle")}
                            placeholder={t("HandlePlaceholder")}
                            as={TextField}
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <HandleIcon />
                                    </InputAdornment>
                                ),
                            }}
                            error={props.touched.handle && Boolean(props.errors.handle)}
                            helperText={props.touched.handle && props.errors.handle}
                        />
                        <TranslatedRichInput
                            language={language}
                            maxChars={2048}
                            minRows={4}
                            name="bio"
                            placeholder={t("Bio")}
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
                            label={t("StartMessage")}
                            language={language}
                            name="startMessage"
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
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});

export const BotUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: BotUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<User, BotShape>({
        ...endpointGetUser,
        objectType: "User",
        overrideObject,
        transform: (data) => botInitialValues(session, data),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<User, BotCreateInput, BotUpdateInput>({
        display,
        endpointCreate: endpointPostBot,
        endpointUpdate: endpointPutBot,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="bot-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateBot" : "UpdateBot")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<BotCreateInput | BotUpdateInput, User>({
                        fetch,
                        inputs: transformBotValues(session, values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateBotValues(session, values, existing, isCreate)}
            >
                {(formik) =>
                    <BotForm
                        display={display}
                        isCreate={isCreate}
                        isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                        isOpen={true}
                        onCancel={handleCancel}
                        onClose={handleClose}
                        ref={formRef}
                        {...formik}
                    />
                }
            </Formik>
        </MaybeLargeDialog>
    );
};
