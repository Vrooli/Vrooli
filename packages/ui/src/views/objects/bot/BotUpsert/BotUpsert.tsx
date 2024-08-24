import { AutoFillResult, AVAILABLE_MODELS, BotCreateInput, BotShape, botTranslationValidation, BotUpdateInput, botValidation, DUMMY_ID, endpointGetUser, endpointPostBot, endpointPutBot, findBotData, LINKS, LlmModel, LlmTask, noopSubmit, SearchPageTabOption, Session, shapeBot, User, validateAndGetYupErrors } from "@local/shared";
import { Divider, InputAdornment, Slider, Stack, Typography } from "@mui/material";
import { useSubmitHelper } from "api";
import { AutoFillButton } from "components/buttons/AutoFillButton/AutoFillButton";
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
import { createUpdatedTranslations, getAutoFillTranslationData, useAutoFill } from "hooks/useAutoFill";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { BotIcon, CommentIcon, HandleIcon, HeartFilledIcon, KeyPhrasesIcon, LearnIcon, PersonaIcon, RoutineValidIcon, TeamIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
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

const sliderStyle = {
    "& .MuiSlider-markLabel": {
        "&[data-index=\"0\"]": {
            marginLeft: 2,
        },
        "&[data-index=\"1\"]": {
            marginLeft: -2,
        },
    },
} as const;

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

    const handleChange = useCallback(function handleChangeMemo(_: unknown, value: number | number[]) {
        if (Array.isArray(value) && value.length > 0) {
            setValue(value[0]);
        } else if (typeof value === "number") {
            setValue(value);
        } else {
            setValue(min);
        }
    }, [setValue]);

    const marks = useMemo(function marksMemo() {
        return [
            { value: min, label: labelLeft },
            { value: max, label: labelRight },
        ];
    }, [labelLeft, labelRight]);

    return (
        <Stack>
            <Typography id={id} gutterBottom>
                {title}
            </Typography>
            <Slider
                aria-labelledby={id}
                disabled={disabled}
                value={value}
                onChange={handleChange}
                valueLabelDisplay="auto"
                min={min}
                max={max}
                step={step}
                marks={marks}
                sx={sliderStyle}
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

const nameInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <BotIcon />
        </InputAdornment>
    ),
} as const;
const handleInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <HandleIcon />
        </InputAdornment>
    ),
} as const;
const startingMessageInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <CommentIcon />
        </InputAdornment>
    ),
} as const;
const occupationInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <TeamIcon />
        </InputAdornment>
    ),
} as const;
const personaInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <PersonaIcon />
        </InputAdornment>
    ),
} as const;
const toneInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <RoutineValidIcon />
        </InputAdornment>
    ),
} as const;
const keyPhrasesInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <KeyPhrasesIcon />
        </InputAdornment>
    ),
} as const;
const domainKnowledgeInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <LearnIcon />
        </InputAdornment>
    ),
} as const;
const biasInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <HeartFilledIcon />
        </InputAdornment>
    ),
} as const;
const relationshipListStyle = { marginBottom: 4 } as const;
const formSectionStyle = { overflowX: "hidden" } as const;

function BotForm({
    disabled,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onClose,
    setFieldValue,
    values,
    ...props
}: BotFormProps) {
    const session = useContext(SessionContext);
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

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            name: values.name,
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback({ data }: AutoFillResult) {
        const originalValues = { ...values };
        const { updatedTranslations, rest } = createUpdatedTranslations(values, data, language, ["bio", "occupation", "persona", "startingMessage", "tone", "keyPhrases", "domainKnowledge", "bias"]);
        delete rest.id;
        const creativity = typeof rest.creativity === "number" ? rest.creativity : values.creativity;
        const isBotDepictingPerson = typeof rest.isBotDepictingPerson === "boolean" ? rest.isBotDepictingPerson : values.isBotDepictingPerson;
        const name = typeof rest.name === "string" ? rest.name : values.name;
        const verbosity = typeof rest.verbosity === "number" ? rest.verbosity : values.verbosity;
        const updatedValues = {
            ...values,
            creativity,
            isBotDepictingPerson,
            name,
            translations: updatedTranslations,
            verbosity,
        };
        return { originalValues, updatedValues };
    }, [language, values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.BotAdd : LlmTask.BotUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const handleBannerImageChange = useCallback(function handleBannerImageChangeCallback(newPicture: File | null) {
        setFieldValue("bannerImage", newPicture);
    }, [setFieldValue]);
    const handleProfileImageChange = useCallback(function handleProfileImageChangeCallback(newPicture: File | null) {
        setFieldValue("profileImage", newPicture);
    }, [setFieldValue]);

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
                            sx={relationshipListStyle}
                        />
                        <ProfilePictureInput
                            onBannerImageChange={handleBannerImageChange}
                            onProfileImageChange={handleProfileImageChange}
                            name="profileImage"
                            profile={values}
                        />
                        <FormSection sx={formSectionStyle}>
                            <Field
                                fullWidth
                                autoComplete="name"
                                name="name"
                                label={t("Name")}
                                placeholder={t("NamePlaceholder")}
                                as={TextInput}
                                InputProps={nameInputProps}
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
                                InputProps={handleInputProps}
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
                                InputProps={startingMessageInputProps}
                            />
                            <LanguageInput
                                currentLanguage={language}
                                flexDirection="row-reverse"
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                languages={languages}
                            />
                        </FormSection>
                    </ContentCollapse>
                    <Divider />
                    <ContentCollapse title="Personality" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <FormSection overflow="hidden">
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
                                InputProps={occupationInputProps}
                            />
                            <TranslatedTextInput
                                fullWidth
                                isRequired={false}
                                label={t("Persona")}
                                placeholder={t("PersonaPlaceholderBot")}
                                language={language}
                                name="persona"
                                InputProps={personaInputProps}
                            />
                            <TranslatedTextInput
                                fullWidth
                                isRequired={false}
                                label={t("Tone")}
                                placeholder={t("TonePlaceholderBot")}
                                language={language}
                                name="tone"
                                InputProps={toneInputProps}
                            />
                            <TranslatedTextInput
                                fullWidth
                                isRequired={false}
                                label={t("KeyPhrases")}
                                placeholder={t("KeyPhrasesPlaceholderBot")}
                                language={language}
                                name="keyPhrases"
                                InputProps={keyPhrasesInputProps}
                            />
                            <TranslatedTextInput
                                fullWidth
                                isRequired={false}
                                label={t("DomainKnowledge")}
                                placeholder={t("DomainKnowledgePlaceholderBot")}
                                language={language}
                                name="domainKnowledge"
                                InputProps={domainKnowledgeInputProps}
                            />
                            <TranslatedTextInput
                                fullWidth
                                isRequired={false}
                                label={t("Bias")}
                                placeholder={t("BiasPlaceholderBot")}
                                language={language}
                                name="bias"
                                InputProps={biasInputProps}
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
                sideActionButtons={<AutoFillButton
                    handleAutoFill={autoFill}
                    isAutoFillLoading={isAutoFillLoading}
                />}
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
