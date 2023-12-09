import { BotCreateInput, botTranslationValidation, BotUpdateInput, botValidation, DUMMY_ID, endpointGetUser, endpointPostBot, endpointPutBot, noopSubmit, Session, User } from "@local/shared";
import { InputAdornment, Slider, Stack, Typography } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CheckboxInput } from "components/inputs/CheckboxInput/CheckboxInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextInput } from "components/inputs/TranslatedTextInput/TranslatedTextInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { BotIcon, CommentIcon, HandleIcon, HeartFilledIcon, KeyPhrasesIcon, LearnIcon, OrganizationIcon, PersonaIcon, RoutineValidIcon } from "icons";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { findBotData } from "utils/botUtils";
import { getYou } from "utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { BotShape, shapeBot } from "utils/shape/models/bot";
import { BotFormProps, BotUpsertProps } from "../types";

const botInitialValues = (
    session: Session | undefined,
    existing?: Partial<User> | BotShape | null | undefined,
): BotShape => {
    const { creativity, verbosity, translations } = findBotData(getUserLanguages(session)[0], existing);

    return {
        __typename: "User" as const,
        id: DUMMY_ID,
        creativity,
        isBotDepictingPerson: false,
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

const BotForm = ({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: BotFormProps) => {
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

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<User>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "User",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<User, BotCreateInput, BotUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostBot,
        endpointUpdate: endpointPutBot,
    });
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "User" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<BotCreateInput | BotUpdateInput, User>({
        disabled,
        existing,
        fetch,
        inputs: transformBotValues(session, values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="bot-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateBot" : "UpdateBot")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
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
                            as={TextInput}
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
                            name="isBotDepictingPerson"
                            label={t("BotDepictPersonAsk")}
                            component={CheckboxInput}
                        />
                        <Field
                            fullWidth
                            autoComplete="handle"
                            name="handle"
                            label={t("Handle")}
                            placeholder={t("HandlePlaceholder")}
                            as={TextInput}
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
                        <TranslatedTextInput
                            fullWidth
                            label={t("Occupation")}
                            placeholder={t("OccupationPlaceholderBot")}
                            language={language}
                            name="occupation"
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <OrganizationIcon />
                                    </InputAdornment>
                                ),
                            }}
                        />
                        <TranslatedTextInput
                            fullWidth
                            label={t("Persona")}
                            placeholder={t("PersonaPlaceholderBot")}
                            language={language}
                            name="persona"
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <PersonaIcon />
                                    </InputAdornment>
                                ),
                            }}
                        />
                        <TranslatedTextInput
                            fullWidth
                            label={t("StartMessage")}
                            placeholder={t("StartMessagePlaceholder")}
                            language={language}
                            name="startMessage"
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <CommentIcon />
                                    </InputAdornment>
                                ),
                            }}
                        />
                        <TranslatedTextInput
                            fullWidth
                            label={t("Tone")}
                            placeholder={t("TonePlaceholderBot")}
                            language={language}
                            name="tone"
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <RoutineValidIcon />
                                    </InputAdornment>
                                ),
                            }}
                        />
                        <TranslatedTextInput
                            fullWidth
                            label={t("KeyPhrases")}
                            placeholder={t("KeyPhrasesPlaceholderBot")}
                            language={language}
                            name="keyPhrases"
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <KeyPhrasesIcon />
                                    </InputAdornment>
                                ),
                            }}
                        />
                        <TranslatedTextInput
                            fullWidth
                            label={t("DomainKnowledge")}
                            placeholder={t("DomainKnowledgePlaceholderBot")}
                            language={language}
                            name="domainKnowledge"
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <LearnIcon />
                                    </InputAdornment>
                                ),
                            }}
                        />
                        <TranslatedTextInput
                            fullWidth
                            label={t("Bias")}
                            placeholder={t("BiasPlaceholderBot")}
                            language={language}
                            name="bias"
                            InputProps={{
                                startAdornment: (
                                    <InputAdornment position="start">
                                        <HeartFilledIcon />
                                    </InputAdornment>
                                ),
                            }}
                        />
                        <Stack>
                            <Typography id="creativity-slider" gutterBottom>
                                {t("CreativityPlaceholder")}
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
                                {t("VerbosityPlaceholder")}
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
                                        label: t("Short"),
                                    },
                                    {
                                        value: 1,
                                        label: t("Long"),
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
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const BotUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: BotUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<User, BotShape>({
        ...endpointGetUser,
        isCreate,
        objectType: "User",
        overrideObject,
        transform: (data) => botInitialValues(session, data),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateBotValues(session, values, existing, isCreate)}
        >
            {(formik) =>
                <BotForm
                    disabled={!(isCreate || canUpdate)}
                    existing={existing}
                    handleUpdate={setExisting}
                    isCreate={isCreate}
                    isReadLoading={isReadLoading}
                    isOpen={isOpen}
                    {...props}
                    {...formik}
                />
            }
        </Formik>
    );
};
