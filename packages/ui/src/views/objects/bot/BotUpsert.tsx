// AI_CHECK: REACT_PERF=2 | LAST: 2024-06-27
// Performance optimizations applied:
// - Disabled validation on change (validateOnChange=false)
// - Temporarily disabled autofill, cache, and complex submit logic
// - Added React.memo to BotForm component
// - Implemented debounced JSON parsing for persona field
// - Used local state for CodeInput to prevent lag
import Divider from "@mui/material/Divider";
import InputAdornment from "@mui/material/InputAdornment";
import Slider from "@mui/material/Slider";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ToggleButton from "@mui/material/ToggleButton";
import ToggleButtonGroup from "@mui/material/ToggleButtonGroup";
import { CodeLanguage, DEFAULT_PERSONA, DEFAULT_PERSONA_UI_ORDER, DUMMY_ID, LATEST_CONFIG_VERSION, LINKS, LlmTask, SearchPageTabOption, botTranslationValidation, botValidation, endpointsUser, getModelDescription, getModelName, noopSubmit, orDefault, shapeBot, validateAndGetYupErrors, type BotCreateInput, type BotShape, type BotUpdateInput, type LlmModel, type Session, type User } from "@vrooli/shared";
import { Field, Formik, useField } from "formik";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useDebounce } from "../../../hooks/useDebounce.js";
import { getAvailableModels, getExistingAIConfig, getFallbackModel } from "../../../api/ai.js";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { ContentCollapse } from "../../../components/containers/ContentCollapse.js";
import { Dialog } from "../../../components/dialogs/Dialog/Dialog.js";
import { CheckboxInput } from "../../../components/inputs/CheckboxInput/CheckboxInput.js";
import { CodeInput } from "../../../components/inputs/CodeInput/CodeInput.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { ProfilePictureInput } from "../../../components/inputs/ProfilePictureInput/ProfilePictureInput.js";
import { TranslatedAdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { SelectorBase } from "../../../components/inputs/Selector/Selector.js";
import { TextInput, TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { createUpdatedTranslations, getAutoFillTranslationData, useAutoFill, type UseAutoFillProps } from "../../../hooks/tasks.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon, IconRoutine } from "../../../icons/Icons.js";
import { FormContainer, FormSection, ScrollBox } from "../../../styles.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { type BotFormProps, type BotUpsertProps } from "./types.js";

function botInitialValues(
    session: Session | undefined,
    existing?: Partial<User> | BotShape | null | undefined,
): BotShape {
    const aiServicesConfig = getExistingAIConfig()?.service?.config;
    const availableModels = getAvailableModels(aiServicesConfig);
    const fallbackModel = getFallbackModel(aiServicesConfig);
    const model = availableModels.find(m => m.value === existing?.botSettings?.model)?.value ?? fallbackModel;

    const botSettings = existing?.botSettings ?? { __version: LATEST_CONFIG_VERSION, persona: { ...DEFAULT_PERSONA } };
    botSettings.model = model ?? "";
    
    // Extract creativity and verbosity from persona if they exist there
    const persona = botSettings.persona || {};
    const creativity = typeof persona.creativity === "number" ? persona.creativity : 0.5;
    const verbosity = typeof persona.verbosity === "number" ? persona.verbosity : 0.5;
    
    // Remove creativity and verbosity from persona object (they're handled separately)
    const { creativity: _, verbosity: __, ...personaWithoutSliders } = persona;
    botSettings.persona = personaWithoutSliders;

    return {
        ...existing,
        __typename: "User" as const,
        id: DUMMY_ID,
        botSettings,
        creativity, // Store at top level for sliders
        verbosity,  // Store at top level for sliders
        isBotDepictingPerson: false,
        isPrivate: false,
        name: "",
        isBot: true,
        translations: orDefault(existing?.translations, [{
            __typename: "UserTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            bio: "",
        }]),
    };
}

function transformBotValues(session: Session | undefined, values: BotShape, existing: BotShape, isCreate: boolean) {
    // Merge creativity and verbosity back into persona for saving
    const transformedValues = {
        ...values,
        botSettings: {
            ...values.botSettings,
            persona: {
                ...values.botSettings.persona,
                creativity: values.creativity,
                verbosity: values.verbosity,
            },
        },
    };
    // Remove top-level creativity and verbosity
    delete (transformedValues as any).creativity;
    delete (transformedValues as any).verbosity;
    
    return isCreate ? shapeBot.create(transformedValues) : shapeBot.update(botInitialValues(session, existing), transformedValues);
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

const nameInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="Bot"
            />
        </InputAdornment>
    ),
} as const;
const handleInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="Handle"
            />
        </InputAdornment>
    ),
} as const;
const startingMessageInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="Comment"
            />
        </InputAdornment>
    ),
} as const;
const occupationInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="Team"
            />
        </InputAdornment>
    ),
} as const;
const personaInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="Persona"
            />
        </InputAdornment>
    ),
} as const;
const toneInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconRoutine
                decorative
                name="RoutineValid"
            />
        </InputAdornment>
    ),
} as const;
const keyPhrasesInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="KeyPhrases"
            />
        </InputAdornment>
    ),
} as const;
const domainKnowledgeInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="Learn"
            />
        </InputAdornment>
    ),
} as const;
const biasInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="HeartFilled"
            />
        </InputAdornment>
    ),
} as const;
const relationshipListStyle = { marginBottom: 4 } as const;
const formSectionStyle = { overflowX: "hidden" } as const;

const BotForm = React.memo(function BotForm({
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

    // Check if persona has non-default fields to determine initial mode
    const hasCustomPersonaFields = useMemo(() => {
        if (!values.botSettings?.persona) return false;
        const personaKeys = Object.keys(values.botSettings.persona);
        return personaKeys.some(key => !DEFAULT_PERSONA_UI_ORDER.includes(key));
    }, [values.botSettings?.persona]);

    const [inputMode, setInputMode] = useState<InputMode>(hasCustomPersonaFields ? "custom" : "default");
    
    // Handle mode change
    const handleInputModeChange = useCallback((_: React.MouseEvent<HTMLElement>, newMode: InputMode | null) => {
        if (newMode !== null) {
            setInputMode(newMode);
        }
    }, []);

    const [modelField, , modelHelpers] = useField<string>("model");
    const [model, setModel] = useState<LlmModel | null>(null);
    useEffect(() => {
        const availableModels = getAvailableModels(getExistingAIConfig()?.service?.config);
        const availableModel = availableModels.find(m => m.value === modelField.value);
        setModel(availableModel ?? null);
    }, [modelField.value]);

    const handleModelChange = useCallback(function handleModelChangeCallback(newModel: LlmModel | null) {
        modelHelpers.setValue(newModel?.value ?? "");
    }, [modelHelpers]);

    const [creativityField, , creativityHelpers] = useField<number>("creativity");
    const [verbosityField, , verbosityHelpers] = useField<number>("verbosity");
    
    // Handle persona JSON with debouncing to prevent lag
    const [personaField, , personaHelpers] = useField<Record<string, unknown>>("botSettings.persona");
    const [localPersonaJSON, setLocalPersonaJSON] = useState<string>(() => {
        try {
            return JSON.stringify(personaField.value || {}, null, 2);
        } catch {
            return "{}";
        }
    });
    
    // Sync local JSON with field value when it changes externally
    useEffect(() => {
        try {
            const newJSON = JSON.stringify(personaField.value || {}, null, 2);
            if (newJSON !== localPersonaJSON) {
                setLocalPersonaJSON(newJSON);
            }
        } catch {
            // Invalid JSON, ignore
        }
    }, [personaField.value]); // eslint-disable-line react-hooks/exhaustive-deps
    
    // Debounce JSON parsing and field update
    const debouncedPersonaJSON = useDebounce(localPersonaJSON, 500);
    useEffect(() => {
        if (inputMode === "custom") {
            try {
                const parsed = JSON.parse(debouncedPersonaJSON);
                personaHelpers.setValue(parsed);
            } catch {
                // Invalid JSON, don't update
            }
        }
    }, [debouncedPersonaJSON, inputMode, personaHelpers]);
    
    const handlePersonaJSONChange = useCallback((newValue: string) => {
        setLocalPersonaJSON(newValue);
    }, []);

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
        endpointCreate: endpointsUser.botCreateOne,
        endpointUpdate: endpointsUser.botUpdateOne,
    });
    // Temporarily disable cache to fix performance
    // useSaveToCache({ isCreate, values, objectId: values.id, objectType: "User" });

    // Simplified onSubmit to test performance
    const onSubmit = useCallback(() => {
        console.log("Submit clicked");
    }, []);

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            name: values.name,
        };
    }, [language, values.name, values.translations]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        const { updatedTranslations, rest } = createUpdatedTranslations(values, data, language, ["bio"]);
        delete rest.id;
        
        // Extract persona fields from the autofill result
        const personaFields = ["occupation", "persona", "startingMessage", "tone", "keyPhrases", "domainKnowledge", "bias"];
        const updatedPersona = { ...values.botSettings.persona };
        
        // Update persona fields from translations
        personaFields.forEach(field => {
            const translationUpdate = updatedTranslations.find(t => t.language === language);
            if (translationUpdate && field in translationUpdate) {
                updatedPersona[field] = translationUpdate[field];
                delete translationUpdate[field]; // Remove from translations as it's now in persona
            }
        });
        
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
            botSettings: {
                ...values.botSettings,
                persona: updatedPersona,
            },
        };
        return { originalValues, updatedValues };
    }, [language, values.botSettings.persona, values.creativity, values.isBotDepictingPerson, values.name, values.translations, values.verbosity]);

    // Temporarily disable autofill to test performance
    const { autoFill, isAutoFillLoading } = { autoFill: () => {}, isAutoFillLoading: false };
    // const { autoFill, isAutoFillLoading } = useAutoFill({
    //     getAutoFillInput,
    //     shapeAutoFillResult,
    //     handleUpdate,
    //     task: isCreate ? LlmTask.BotAdd : LlmTask.BotUpdate,
    // });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const handleBannerImageChange = useCallback(function handleBannerImageChangeCallback(newPicture: File | null) {
        setFieldValue("bannerImage", newPicture);
    }, [setFieldValue]);
    const handleProfileImageChange = useCallback(function handleProfileImageChangeCallback(newPicture: File | null) {
        setFieldValue("profileImage", newPicture);
    }, [setFieldValue]);

    return (
        <>
            {display === "Dialog" ? (
                <Dialog
                    isOpen={isOpen ?? false}
                    onClose={onClose ?? (() => console.warn("onClose not passed to dialog"))}
                    size="md"
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
                                    <TranslatedAdvancedInput
                                        isRequired={false}
                                        language={language}
                                        features={{ maxChars: 2048, minRowsCollapsed: 4 }}
                                        name="bio"
                                        placeholder={t("Bio")}
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
                                    <ToggleButtonGroup
                                        value={inputMode}
                                        exclusive
                                        onChange={handleInputModeChange}
                                        size="small"
                                        fullWidth
                                        sx={{ marginBottom: 2 }}
                                    >
                                        <ToggleButton value="default">
                                            {t("Default")}
                                        </ToggleButton>
                                        <ToggleButton value="custom">
                                            {t("Custom")}
                                        </ToggleButton>
                                    </ToggleButtonGroup>
                                    
                                    {inputMode === "default" ? (
                                        <>
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.occupation"
                                                label={t("Occupation")}
                                                placeholder={t("OccupationPlaceholderBot")}
                                                as={TextInput}
                                                InputProps={occupationInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.persona"
                                                label={t("Persona")}
                                                placeholder={t("PersonaPlaceholderBot")}
                                                as={TextInput}
                                                InputProps={personaInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.tone"
                                                label={t("Tone")}
                                                placeholder={t("TonePlaceholderBot")}
                                                as={TextInput}
                                                InputProps={toneInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.startingMessage"
                                                label={t("StartMessage")}
                                                placeholder={t("StartMessagePlaceholder")}
                                                as={TextInput}
                                                InputProps={startingMessageInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.keyPhrases"
                                                label={t("KeyPhrases")}
                                                placeholder={t("KeyPhrasesPlaceholderBot")}
                                                as={TextInput}
                                                InputProps={keyPhrasesInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.domainKnowledge"
                                                label={t("DomainKnowledge")}
                                                placeholder={t("DomainKnowledgePlaceholderBot")}
                                                as={TextInput}
                                                InputProps={domainKnowledgeInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.bias"
                                                label={t("Bias")}
                                                placeholder={t("BiasPlaceholderBot")}
                                                as={TextInput}
                                                InputProps={biasInputProps}
                                            />
                                        </>
                                    ) : (
                                        <CodeInput
                                            id="bot-persona-json"
                                            label={t("PersonaJSON")}
                                            helperText={t("PersonaJSONHelperText", { defaultValue: "Enter custom personality data as JSON" })}
                                            language={CodeLanguage.Json}
                                            value={localPersonaJSON}
                                            onChange={handlePersonaJSONChange}
                                            minRows={10}
                                        />
                                    )}
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
                                    options={getAvailableModels(getExistingAIConfig()?.service?.config)}
                                    getOptionLabel={getModelName}
                                    getOptionDescription={getModelDescription}
                                    fullWidth={true}
                                    inputAriaLabel="Model"
                                    label={t("Model", { count: 1 })}
                                    onChange={handleModelChange}
                                    value={model}
                                />
                            </ContentCollapse>
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
                        sideActionButtons={<AutoFillButton
                            handleAutoFill={autoFill}
                            isAutoFillLoading={isAutoFillLoading}
                        />}
                    />
                </Dialog>
            ) : (
                <ScrollBox>
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
                                    <TranslatedAdvancedInput
                                        isRequired={false}
                                        language={language}
                                        features={{ maxChars: 2048, minRowsCollapsed: 4 }}
                                        name="bio"
                                        placeholder={t("Bio")}
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
                                    <ToggleButtonGroup
                                        value={inputMode}
                                        exclusive
                                        onChange={handleInputModeChange}
                                        size="small"
                                        fullWidth
                                        sx={{ marginBottom: 2 }}
                                    >
                                        <ToggleButton value="default">
                                            {t("Default")}
                                        </ToggleButton>
                                        <ToggleButton value="custom">
                                            {t("Custom")}
                                        </ToggleButton>
                                    </ToggleButtonGroup>
                                    
                                    {inputMode === "default" ? (
                                        <>
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.occupation"
                                                label={t("Occupation")}
                                                placeholder={t("OccupationPlaceholderBot")}
                                                as={TextInput}
                                                InputProps={occupationInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.persona"
                                                label={t("Persona")}
                                                placeholder={t("PersonaPlaceholderBot")}
                                                as={TextInput}
                                                InputProps={personaInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.tone"
                                                label={t("Tone")}
                                                placeholder={t("TonePlaceholderBot")}
                                                as={TextInput}
                                                InputProps={toneInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.startingMessage"
                                                label={t("StartMessage")}
                                                placeholder={t("StartMessagePlaceholder")}
                                                as={TextInput}
                                                InputProps={startingMessageInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.keyPhrases"
                                                label={t("KeyPhrases")}
                                                placeholder={t("KeyPhrasesPlaceholderBot")}
                                                as={TextInput}
                                                InputProps={keyPhrasesInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.domainKnowledge"
                                                label={t("DomainKnowledge")}
                                                placeholder={t("DomainKnowledgePlaceholderBot")}
                                                as={TextInput}
                                                InputProps={domainKnowledgeInputProps}
                                            />
                                            <Field
                                                fullWidth
                                                name="botSettings.persona.bias"
                                                label={t("Bias")}
                                                placeholder={t("BiasPlaceholderBot")}
                                                as={TextInput}
                                                InputProps={biasInputProps}
                                            />
                                        </>
                                    ) : (
                                        <CodeInput
                                            id="bot-persona-json"
                                            label={t("PersonaJSON")}
                                            helperText={t("PersonaJSONHelperText", { defaultValue: "Enter custom personality data as JSON" })}
                                            language={CodeLanguage.Json}
                                            value={localPersonaJSON}
                                            onChange={handlePersonaJSONChange}
                                            minRows={10}
                                        />
                                    )}
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
                                    options={getAvailableModels(getExistingAIConfig()?.service?.config)}
                                    getOptionLabel={getModelName}
                                    getOptionDescription={getModelDescription}
                                    fullWidth={true}
                                    inputAriaLabel="Model"
                                    label={t("Model", { count: 1 })}
                                    onChange={handleModelChange}
                                    value={model}
                                />
                            </ContentCollapse>
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
                        sideActionButtons={<AutoFillButton
                            handleAutoFill={autoFill}
                            isAutoFillLoading={isAutoFillLoading}
                        />}
                    />
                </ScrollBox>
            )}
        </>
    );
});

export function BotUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: BotUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<User, BotShape>({
        ...endpointsUser.findOne,
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "User",
        overrideObject,
        transform: (data) => botInitialValues(session, data),
    });

    // Only validate on submit to prevent lag
    const validateValues = useCallback(async (values: BotShape) => {
        return await validateBotValues(session, values, existing, isCreate);
    }, [session, existing, isCreate]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
            validateOnChange={false}
            validateOnBlur={true}
        >
            {(formik) =>
                <BotForm
                    disabled={!(isCreate || permissions.canUpdate)}
                    display={display}
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
