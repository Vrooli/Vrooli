import { AutoFillInput, AutoFillResult, BotCreateInput, botTranslationValidation, BotUpdateInput, botValidation, DUMMY_ID, endpointGetAutoFill, endpointGetUser, endpointPostBot, endpointPutBot, LINKS, LlmTask, noopSubmit, Session, User } from "@local/shared";
import { Divider, IconButton, InputAdornment, Slider, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper, useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { SearchExistingButton } from "components/buttons/SearchExistingButton/SearchExistingButton";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CheckboxInput } from "components/inputs/CheckboxInput/CheckboxInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { TextInput, TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { BotIcon, CommentIcon, HandleIcon, HeartFilledIcon, KeyPhrasesIcon, LearnIcon, MagicIcon, PersonaIcon, RoutineValidIcon, TeamIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { AVAILABLE_MODELS, findBotData, LlmModel } from "utils/botUtils";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { SearchPageTabOption } from "utils/search/objectToSearch";
import { validateAndGetYupErrors } from "utils/shape/general";
import { BotShape, BotTranslationShape, shapeBot } from "utils/shape/models/bot";
import { BotFormProps, BotUpsertProps } from "../types";

function botInitialValues(
    session: Session | undefined,
    existing?: Partial<User> | BotShape | null | undefined,
): BotShape {
    const { creativity, verbosity, model, translations } = findBotData(getUserLanguages(session)[0], existing);

    return {
        __typename: "User" as const,
        id: DUMMY_ID,
        creativity,
        isBotDepictingPerson: false,
        isPrivate: false,
        model,
        name: "",
        verbosity,
        ...existing,
        isBot: true,
        translations,
    };
}

function transformBotValues(session: Session | undefined, values: BotShape, existing: BotShape, isCreate: boolean) {
    return isCreate ? shapeBot.create(values) : shapeBot.update(botInitialValues(session, existing), values);
}

async function validateBotValues(session: Session | undefined, values: BotShape, existing: BotShape, isCreate: boolean) {
    const transformedValues = transformBotValues(session, values, existing, isCreate);
    const validationSchema = botValidation[isCreate ? "create" : "update"]({ env: process.env.NODE_ENV });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export function FeatureSlider({
    disabled,
    id,
    labelLeft,
    labelRight,
    setValue,
    title,
    value,
}: {
    disabled?: boolean,
    id: string,
    labelLeft: string,
    labelRight: string,
    setValue: (value: number) => unknown,
    title: string,
    value: number,
}) {
    const min = 0.1;
    const max = 1;
    const step = 0.1;

    return (
        <Stack>
            <Typography id={id} gutterBottom>
                {title}
            </Typography>
            <Slider
                aria-labelledby={id}
                disabled={disabled}
                value={value}
                onChange={(_, value) => {
                    if (Array.isArray(value) && value.length > 0) {
                        setValue(value[0]);
                    } else if (typeof value === "number") {
                        setValue(value);
                    } else {
                        setValue(min);
                    }
                }}
                valueLabelDisplay="auto"
                min={min}
                max={max}
                step={step}
                marks={[
                    { value: min, label: labelLeft },
                    { value: max, label: labelRight },
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
    );
}

type InputMode = "default" | "custom";

// const InputToggle = ({
//     canChangeInputMode,
//     inputMode,
//     onInputModeChange,
// }: {
//     canChangeInputMode: boolean,
//     inputMode: InputMode,
//     onInputModeChange: (inputMode: InputMode) => unknown,
// }) => {
//     if (!canChangeInputMode) return null;
//     return (
//         <Box sx={{ marginBottom: "10px", display: "flex", gap: 1 }}>
//             <Button variant={inputMode === "default" ? "contained" : "outlined"} color="primary" onClick={() => onInputModeChange("default")}>
//                 Default View
//             </Button>
//             <Button variant={inputMode === "custom" ? "contained" : "outlined"} color="primary" onClick={() => onInputModeChange("custom")}>
//                 Custom View
//             </Button>
//         </Box>
//     )
// };

function BotForm({
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
}: BotFormProps) {
    const session = useContext(SessionContext);
    const { credits } = useMemo(() => getCurrentUser(session), [session]);
    const { palette } = useTheme();
    const { t } = useTranslation();

    // const [inputMode, setInputMode] = useState<InputMode>("default");
    // const changeInputMode = (inputMode: InputMode) => {
    //     setInputMode(inputMode);
    // }

    const [modelField, , modelHelpers] = useField<string>("model");
    const [model, setModel] = useState<LlmModel | null>(null);
    useEffect(() => {
        const availableModel = AVAILABLE_MODELS.find(m => m.value === modelField.value);
        setModel(availableModel ?? null);
    }, [modelField.value]);

    const [creativityField, , creativityHelpers] = useField<number>("creativity");
    const [verbosityField, , verbosityHelpers] = useField<number>("verbosity");

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
        validationSchema: botTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const { handleCancel, handleCompleted } = useUpsertActions<User>({
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
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "User" });

    const onSubmit = useSubmitHelper<BotCreateInput | BotUpdateInput, User>({
        disabled,
        existing,
        fetch,
        inputs: transformBotValues(session, values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const [getAutoFill, { loading: isLoadingAutoFill }] = useLazyFetch<AutoFillInput, AutoFillResult>(endpointGetAutoFill);
    const autoFill = useCallback(() => {
        const defaultTranslation = { ...(values.translations?.length ? values.translations[0] : {}) } as Partial<BotTranslationShape> & { __typename?: string };
        delete defaultTranslation.id;
        delete defaultTranslation.language;
        delete defaultTranslation.__typename;
        const existingData = {
            ...defaultTranslation,
            name: values.name,
        };
        Object.entries(existingData).forEach(([key, value]) => {
            if (typeof value === "string" && value.trim() === "") {
                delete existingData[key];
            }
        });
        fetchLazyWrapper<AutoFillInput, AutoFillResult>({
            fetch: getAutoFill,
            inputs: {
                task: isCreate ? LlmTask.BotAdd : LlmTask.BotUpdate,
                data: existingData,
            },
            onSuccess: ({ data }) => {
                console.log("got autofill response", data);
                const originalValues = { ...values };
                const { name, ...rest } = data;
                const updatedValues = {
                    ...values,
                    name: name ?? values.name,
                    translations: values.translations?.length ? [{
                        ...values.translations[0],
                        ...rest,
                    }, ...values.translations.slice(1)] : [],
                };
                handleUpdate(updatedValues);
                PubSub.get().publish("snack", { message: "Form auto-filled", buttonKey: "Undo", buttonClicked: () => { handleUpdate(originalValues); }, severity: "Success", autoHideDuration: 15000 });
            },
        });
    }, [getAutoFill, handleUpdate, isCreate, values]);

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || isLoadingAutoFill || props.isSubmitting, [isCreateLoading, isLoadingAutoFill, isReadLoading, isUpdateLoading, props.isSubmitting]);

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
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.User}"&isBot=true`}
                text="Search existing bots"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
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
                        <FormSection sx={{ overflowX: "hidden" }}>
                            <Field
                                fullWidth
                                autoComplete="name"
                                autoFocus
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
                                isRequired={false}
                                language={language}
                                maxChars={2048}
                                minRows={4}
                                name="bio"
                                placeholder={t("Bio")}
                            />
                            <TranslatedTextInput
                                fullWidth
                                isRequired={false}
                                label={t("StartMessage")}
                                placeholder={t("StartMessagePlaceholder")}
                                language={language}
                                name="startingMessage"
                                InputProps={{
                                    startAdornment: (
                                        <InputAdornment position="start">
                                            <CommentIcon />
                                        </InputAdornment>
                                    ),
                                }}
                            />
                            <LanguageInput
                                currentLanguage={language}
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                languages={languages}
                                sx={{ flexDirection: "row-reverse" }}
                            />
                        </FormSection>
                    </ContentCollapse>
                    <Divider />
                    <ContentCollapse title="Personality" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <FormSection sx={{
                            overflowX: "hidden",
                        }}>
                            {/* <InputToggle
                            canChangeInputMode={true}
                            inputMode={inputMode}
                            onInputModeChange={changeInputMode}
                        /> */}
                            <TranslatedTextInput
                                fullWidth
                                isRequired={false}
                                label={t("Occupation")}
                                placeholder={t("OccupationPlaceholderBot")}
                                language={language}
                                name="occupation"
                                InputProps={{
                                    startAdornment: (
                                        <InputAdornment position="start">
                                            <TeamIcon />
                                        </InputAdornment>
                                    ),
                                }}
                            />
                            <TranslatedTextInput
                                fullWidth
                                isRequired={false}
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
                                isRequired={false}
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
                                isRequired={false}
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
                                isRequired={false}
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
                                isRequired={false}
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
                            <FeatureSlider
                                id="creativity-slider"
                                labelLeft={t("Low")}
                                labelRight={t("High")}
                                setValue={creativityHelpers.setValue}
                                title={t("CreativityPlaceholder")}
                                value={creativityField.value}
                            />
                            <FeatureSlider
                                id="verbosity-slider"
                                labelLeft={t("Low")}
                                labelRight={t("High")}
                                setValue={verbosityHelpers.setValue}
                                title={t("VerbosityPlaceholder")}
                                value={verbosityField.value}
                            />
                        </FormSection>
                    </ContentCollapse>
                    <Divider />
                    <ContentCollapse title={t("Model", { count: 1 })} titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <SelectorBase
                            name="model"
                            options={AVAILABLE_MODELS}
                            getOptionLabel={(r) => r.name}
                            getOptionDescription={(r) => r.description}
                            fullWidth={true}
                            inputAriaLabel="Mode"
                            label={t("Model", { count: 1 })}
                            onChange={(newModel) => {
                                modelHelpers.setValue(newModel.value);
                            }}
                            value={model}
                        />
                    </ContentCollapse>
                </FormContainer>
            </BaseForm >
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
                sideActionButtons={credits && BigInt(credits) > 0 ? (
                    <Tooltip title={t("AutoFill")} placement="top">
                        <IconButton
                            aria-label={t("AutoFill")}
                            disabled={isLoadingAutoFill}
                            onClick={autoFill}
                            sx={{ background: palette.secondary.main }}
                        >
                            <MagicIcon fill={palette.secondary.contrastText} width="36px" height="36px" />
                        </IconButton>
                    </Tooltip>
                ) : null}
            />
        </MaybeLargeDialog >
    );
}

export function BotUpsert({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: BotUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<User, BotShape>({
        ...endpointGetUser,
        isCreate,
        objectType: "User",
        overrideObject,
        transform: (data) => botInitialValues(session, data),
    });

    async function validateValues(values: BotShape) {
        return await validateBotValues(session, values, existing, isCreate);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) =>
                <BotForm
                    disabled={!(isCreate || permissions.canUpdate)}
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
}
