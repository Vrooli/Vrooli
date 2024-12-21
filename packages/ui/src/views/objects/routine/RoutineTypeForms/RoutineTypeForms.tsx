import { BotShape, BotStyle, BotStyleOption, CodeLanguage, CodeVersionShape, CodeVersionTranslationShape, ConfigCallData, ConfigCallDataGenerate, DEFAULT_MODEL, FormSchema, FormStructureType, LlmModel, RunProject, RunRoutine, RunStatus, User, getAvailableModels, getModelDescription, getModelName, getTranslation, noop, uuid, uuidValidate } from "@local/shared";
import { Box, Button, Card, Divider, Grid, Typography, styled, useTheme } from "@mui/material";
import { getExistingAIConfig } from "api/ai";
import { LoadableButton } from "components/buttons/LoadableButton/LoadableButton";
import { RunPickerMenu } from "components/buttons/RunButton/RunButton";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { CodeInputBase } from "components/inputs/CodeInput/CodeInput";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { FormTip } from "components/inputs/form/FormTip/FormTip";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts";
import { FormView } from "forms/FormView/FormView";
import { usePopover } from "hooks/usePopover";
import { AddIcon, ApiIcon, BotIcon, CancelIcon, LockIcon, MagicIcon, MinusIcon, OpenInNewIcon, PlayIcon, SaveIcon, SmartContractIcon, SuccessIcon, TerminalIcon } from "icons";
import { memo, useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ProfileAvatar } from "styles";
import { PartialWithType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { ELEMENT_IDS } from "utils/consts";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor } from "utils/display/listTools";
import { getCookiePartialData } from "utils/localStorage";
import { openObject } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";

const PREFIX_INPUT = "input";
const PREFIX_OUTPUT = "output";

/**
 * The context in which the routine form is displayed. 
 * For most purposes, non-edit modes should be the same.
 */
type RoutineFormDisplayType = "edit" | "view" | "run";

export type RoutineFormPropsBase = {
    configCallData: ConfigCallData;
    disabled: boolean;
    display: RoutineFormDisplayType;
    /**
     * Clears the run when viewing a single-step routine. Useful for creating a new run 
     * without having to refresh the page.
     */
    handleClearRun: () => unknown;
    /**
     * Completes the routine. Used when not in a multi-step form.
     */
    handleCompleteStep: (event: React.MouseEvent | React.TouchEvent) => unknown;
    /** 
     * Performs the action associated with the routine 
     * (e.g. runs sandboxed code, calls API
     */
    handleRunStep: (event: React.MouseEvent | React.TouchEvent) => unknown;
    /**
     * True if there are errors preventing the step from being completed.
     */
    hasErrors: boolean;
    /**
     * True if the submit button is disabled, either from 
     * not being in a multi-step form, from the form being invalid, 
     * or from the routine being deleted/private/etc.
     */
    isCompleteStepDisabled: boolean;
    /**
     * True if we are in a multi-step routine.
     */
    isPartOfMultiStepRoutine: boolean;
    /**
     * True if the run button is disabled, either from 
     * an inapplicable display type, the form being invalid, 
     * or from the routine being deleted/private/etc.
     */
    isRunStepDisabled: boolean;
    /**
     * True if we are currently running the action associated with the routine, 
     * which happens asynchronously.
     */
    isRunningStep: boolean;
    onConfigCallDataChange: (configCallData: ConfigCallData) => unknown;
    onRunChange: (run: RunRoutine | null) => unknown;
    onSchemaInputChange: (schema: FormSchema) => unknown;
    onSchemaOutputChange: (schema: FormSchema) => unknown;
    routineId: string;
    routineName: string;
    /**
     * If in a single-step routine, the current run object
     */
    run: RunRoutine | null | undefined;
    schemaInput: FormSchema;
    schemaOutput: FormSchema;
}
type RoutineFormTypeApi = RoutineFormPropsBase;
type RoutineFormTypeCode = RoutineFormPropsBase;
type RoutineFormTypeData = RoutineFormPropsBase;
type RoutineFormTypeGenerate = RoutineFormPropsBase;
type RoutineFormTypeInformational = Omit<RoutineFormPropsBase, "onSchemaOutputChange" | "schemaOutput">;
type RoutineFormTypeSmartContract = RoutineFormPropsBase;

const routineTypeTitleSxs = { stack: { paddingLeft: 0 } } as const;

export const RoutineApiForm = memo(function RoutineApiFormMemo({
    disabled,
    display,
    onSchemaInputChange,
    onSchemaOutputChange,
    schemaInput,
    schemaOutput,
}: RoutineFormTypeApi) {
    const { t } = useTranslation();
    const isEditing = display === "edit";

    return (
        <>
            <Title
                Icon={ApiIcon}
                title={isEditing ? "Connect API" : "Connected API"}
                help={"Connect API that will receive the defined inputs and is expected to return the defined outputs.\n\nIf the API fails or does not return the expected data, the routine will fail."}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <ContentCollapse
                helpText={"Define the inputs that will be passed to the API. Any input without a default value will be entered by the user at runtime."}
                title={t("Input", { count: schemaInput.elements.length })}
                isOpen={!disabled}
                titleVariant="h4"
            >
                <FormView
                    disabled={disabled}
                    fieldNamePrefix={PREFIX_INPUT}
                    isEditing={isEditing}
                    onSchemaChange={onSchemaInputChange}
                    schema={schemaInput}
                />
            </ContentCollapse>
            <ContentCollapse
                helpText={"Define the outputs that the API is expected to return. If the API fails or does not return the expected data, the routine will fail."}
                title={t("Output", { count: schemaOutput.elements.length })}
                isOpen={!disabled}
                titleVariant="h4"
            >
                <FormView
                    disabled={disabled}
                    fieldNamePrefix={PREFIX_OUTPUT}
                    isEditing={isEditing}
                    onSchemaChange={onSchemaOutputChange}
                    schema={schemaOutput}
                />
            </ContentCollapse>
        </>
    );
});

type CodeObjectInfo = Pick<CodeVersionShape, "__typename" | "id" | "codeLanguage" | "content"> & {
    translations?: Pick<CodeVersionTranslationShape, "id" | "name" | "description" | "jsonVariable" | "language">[];
};

const findCodeLimitTo = ["DataConverter"] as const;

export const RoutineDataConverterForm = memo(function RoutineDataConverterFormMemo({
    disabled,
    display,
    onSchemaInputChange,
    onSchemaOutputChange,
    schemaInput,
    schemaOutput,
}: RoutineFormTypeCode) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const session = useContext(SessionContext);
    const [codeObject, setCodeObject] = useState<CodeObjectInfo | null>(null);
    const [isCodeSearchOpen, setIsCodeSearchOpen] = useState(false);
    const isEditing = display === "edit";

    const closeCodeSearch = useCallback((selected?: CodeObjectInfo) => {
        setIsCodeSearchOpen(false);
        if (selected) {
            setCodeObject(selected);
        }
    }, []);

    const handleCodeButtonClick = useCallback(() => {
        if (codeObject) setCodeObject(null);
        else setIsCodeSearchOpen(true);
    }, [codeObject]);

    return (
        <>
            <Title
                Icon={TerminalIcon}
                title={isEditing ? "Connect code" : "Connected code"}
                help={"Connect or create a data converter function to this routine.\n\nThe code will be passed all non-file inputs, and is expected to return all non-file outputs.\n\nIf the code fails or does not return the expected data, the routine will fail."}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            {isEditing && (
                <Button
                    fullWidth
                    color="secondary"
                    variant="contained"
                    onClick={handleCodeButtonClick}
                    startIcon={codeObject ? <MinusIcon /> : <AddIcon />}
                >
                    {codeObject ? "Remove code" : "Choose code"}
                </Button>
            )}
            {isCodeSearchOpen && (
                <FindObjectDialog
                    find="Full"
                    isOpen={isCodeSearchOpen}
                    handleCancel={closeCodeSearch}
                    handleComplete={closeCodeSearch as (item: object) => unknown}
                    limitTo={findCodeLimitTo}
                />
            )}
            {codeObject && (
                <Box display="flex" flexDirection="column" gap={1}>
                    <Typography variant="h6">{getTranslation(codeObject, getCurrentUser(session).languages).name}</Typography>
                    <Typography variant="body2" color={palette.background.textSecondary}>
                        {getTranslation(codeObject, getCurrentUser(session).languages).description}
                    </Typography>
                    <CodeInputBase
                        codeLanguage={codeObject.codeLanguage as CodeLanguage}
                        content={codeObject.content}
                        disabled={true}
                        handleCodeLanguageChange={noop}
                        handleContentChange={noop}
                        name="content"
                    />
                </Box>
            )}
            <ContentCollapse
                helpText={"Define the inputs that will be passed to the code. Any input without a default value will be entered by the user at runtime."}
                title={t("Input", { count: schemaInput.elements.length })}
                isOpen={!disabled}
                titleVariant="h4"
            >
                <FormView
                    disabled={disabled}
                    fieldNamePrefix={PREFIX_INPUT}
                    isEditing={isEditing}
                    onSchemaChange={onSchemaInputChange}
                    schema={schemaInput}
                />
            </ContentCollapse>
            <ContentCollapse
                helpText={"Define the outputs that the code is expected to return. If the code fails or does not return the expected data, the routine will fail."}
                title={t("Output", { count: schemaOutput.elements.length })}
                isOpen={!disabled}
                titleVariant="h4"
            >
                <FormView
                    disabled={disabled}
                    fieldNamePrefix={PREFIX_OUTPUT}
                    isEditing={isEditing}
                    onSchemaChange={onSchemaOutputChange}
                    schema={schemaOutput}
                />
            </ContentCollapse>
        </>
    );
});

export const RoutineDataForm = memo(function RoutineDataFormMemo({
    disabled,
    display,
    onSchemaOutputChange,
    schemaOutput,
}: RoutineFormTypeData) {
    const isEditing = display === "edit";

    return (
        <FormView
            disabled={disabled}
            fieldNamePrefix={PREFIX_OUTPUT}
            isEditing={isEditing}
            onSchemaChange={onSchemaOutputChange}
            schema={schemaOutput}
        />
    );
});

type BotInfo = Pick<BotShape, "__typename" | "id" | "handle" | "model" | "name" | "profileImage">;

const botStyleOptions: BotStyleOption[] = [{
    description: "If a bot runs this routine, their style will be used to generate the response. Otherwise, no style will be used.",
    label: "Default",
    value: BotStyle.Default,
}, {
    description: "Select a specific bot to use their style for generating the response.",
    label: "Specific bot",
    value: BotStyle.Specific,
}, {
    description: "Do not use any style to generate the response.",
    label: "None",
    value: BotStyle.None,
}];
function getBotStyleDescription(option: BotStyleOption) { return option.description; }
function getBotStyleLabel(option: BotStyleOption) { return option.label; }

const findBotLimitTo = ["User"] as const;
const findBotWhere = { isBot: true };
const BOT_AVATAR_IMG_SRC_TARGET_SIZE = 100;

const BotCardOuter = styled(Card)(({ theme }) => ({
    cursor: "pointer",
    display: "flex",
    alignItems: "center",
    padding: 0,
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
    borderRadius: "8px",
}));

export const BotAvatar = styled(ProfileAvatar)(() => ({
    width: 72,
    height: 72,
    borderRadius: "0px",
}));

const BotCardAvatar = memo(function BotCardAvatarMemo({ bot }: { bot: BotInfo }) {
    const alt = `${bot.name} profile image`;
    const src = useMemo(() => extractImageUrl(bot.profileImage, undefined, BOT_AVATAR_IMG_SRC_TARGET_SIZE), [bot.profileImage]);
    const profileColors = useMemo(() => placeholderColor(), []);

    return (
        <BotAvatar
            alt={alt}
            isBot={true}
            profileColors={profileColors}
            src={src}
        >
            <BotIcon width="75%" height="75%" />
        </BotAvatar>
    );
});

const BotHandle = styled(Typography)(({ theme }) => ({
    color: theme.palette.secondary.dark,
    fontFamily: "monospace",
}));

const botCardOpenIconBoxStyle = {
    display: "grid",
    marginLeft: "auto",
    paddingRight: 1,
} as const;

const BotCard = memo(function BotCardMemo({ bot }: { bot: BotInfo }) {
    const [, setLocation] = useLocation();

    const handleBotCardClick = useCallback(function handleBotCardClickCallback() {
        openObject(bot, setLocation);
    }, [bot, setLocation]);

    return (
        <BotCardOuter onClick={handleBotCardClick}>
            <BotCardAvatar bot={bot} />
            <Box p={1}>
                <Typography variant="h6">{bot.name}</Typography>
                {bot.handle && <BotHandle variant="body2">
                    @{bot.handle}
                </BotHandle>}
            </Box>
            <Box sx={botCardOpenIconBoxStyle}>
                <OpenInNewIcon />
            </Box>
        </BotCardOuter>
    );
});

const inputsDividerStyle = { marginTop: 2, marginBottom: 2 } as const;
const outputsDividerStyle = { marginTop: 4, marginBottom: 2 } as const;

export const RoutineGenerateForm = memo(function RoutineGenerateFormMemo({
    configCallData,
    disabled,
    display,
    handleRunStep,
    isRunStepDisabled,
    isRunningStep,
    onConfigCallDataChange,
    onSchemaInputChange,
    schemaInput,
    schemaOutput,
}: RoutineFormTypeGenerate) {
    const { t } = useTranslation();
    const isEditing = display === "edit";

    const [botStyle, setBotStyle] = useState<BotStyleOption>(function initBotStyleState() {
        const style = configCallData.botStyle || BotStyle.Default;
        return botStyleOptions.find(option => option.value === style) || botStyleOptions[0];
    });

    const [bot, setBot] = useState<BotInfo | null>(function initBotState() {
        if (!configCallData.respondingBot || !uuidValidate(configCallData.respondingBot)) return null;
        const storedData = getCookiePartialData<PartialWithType<User>>({ __typename: "User", id: configCallData.respondingBot });
        if (!storedData) return null;
        return storedData as BotInfo;
    });

    const handleConfigCallDataChange = useCallback(function handleConfigCallDataChangeCallback(updatedConfigData: Partial<ConfigCallDataGenerate>) {
        onConfigCallDataChange({ ...(configCallData as ConfigCallDataGenerate), ...updatedConfigData });
    }, [configCallData, onConfigCallDataChange]);
    const handleModelChange = useCallback(function handleModelChangeCallback(newModel: LlmModel | null) {
        handleConfigCallDataChange({ model: newModel });
    }, [handleConfigCallDataChange]);
    const handleRespondingBotChange = useCallback(function handleRespondingBotChangeCallback(respondingBot: BotInfo | null) {
        handleConfigCallDataChange({ respondingBot: respondingBot?.id ?? null });
        setBot(respondingBot);
    }, [handleConfigCallDataChange]);
    const handleBotStyleChange = useCallback(function handleBotStyleChangeCallback(newStyle: BotStyleOption) {
        handleConfigCallDataChange({ botStyle: newStyle.value });
        setBotStyle(newStyle);
        if (newStyle.value !== BotStyle.Specific) handleRespondingBotChange(null);
        if (newStyle.value === BotStyle.Specific) setIsBotSearchOpen(true);
    }, [handleConfigCallDataChange, handleRespondingBotChange]);

    const [isBotSearchOpen, setIsBotSearchOpen] = useState(false);
    const closeBotSearch = useCallback(function closeBotSearchCallback(selected?: BotInfo) {
        setIsBotSearchOpen(false);
        if (selected) {
            handleRespondingBotChange(selected as unknown as BotInfo);
        }
    }, [handleRespondingBotChange]);
    const handleBotButtonClick = useCallback(function handleBotButtonClickCallback() {
        setIsBotSearchOpen(true);
    }, []);

    const modelTipElement = useMemo(function modelTipElementMemo() {
        return {
            type: FormStructureType.Tip,
            icon: "Info",
            id: uuid(),
            label: `Generating with ${configCallData?.model?.name ?? `${DEFAULT_MODEL} (default)`}`,
        } as const;
    }, [configCallData.model]);

    return (
        <>
            <Box id={ELEMENT_IDS.RoutineGenerateSettings}>
                {isEditing && <Title
                    title={"Choose AI model"}
                    help={"Connect API that will receive the defined inputs and is expected to return the defined outputs.\n\nIf the API fails or does not return the expected data, the routine will fail."}
                    variant="subsection"
                    sxs={routineTypeTitleSxs}
                />}
                {display !== "edit" && <FormTip
                    element={modelTipElement}
                    isEditing={false}
                    onDelete={noop}
                    onUpdate={noop}
                />}
                {isEditing && <SelectorBase
                    name="model"
                    options={getAvailableModels(getExistingAIConfig()?.service?.config)}
                    getOptionLabel={getModelName}
                    getOptionDescription={getModelDescription}
                    fullWidth={true}
                    inputAriaLabel="Model"
                    label={t("Model", { count: 1 })}
                    noneOption={true}
                    noneText="default"
                    onChange={handleModelChange}
                    value={configCallData.model ?? null}
                />}
                {isEditing && <Title
                    title={!bot ? "Choose style" : `Using style of ${bot.name}`}
                    help={isEditing ? "Connecting a bot allows you to generate data using the bot's personality and style.\n\nYou can choose whichever bot runs the routine, a specific bot, or *none* if you don't want to add personality/style to the response." : undefined}
                    variant="subsection"
                    sxs={routineTypeTitleSxs}
                />}
                {isEditing && <SelectorBase
                    name="style"
                    options={botStyleOptions}
                    getOptionDescription={getBotStyleDescription}
                    getOptionLabel={getBotStyleLabel}
                    fullWidth={true}
                    inputAriaLabel="Bot style"
                    label="Bot style"
                    onChange={handleBotStyleChange}
                    value={botStyle}

                />}
                {isEditing && botStyle.value === BotStyle.Specific && <FindObjectDialog
                    find="List"
                    isOpen={isBotSearchOpen}
                    limitTo={findBotLimitTo}
                    handleCancel={closeBotSearch}
                    handleComplete={closeBotSearch as (item: object) => unknown}
                    where={findBotWhere}
                />}
                {bot && <BotCard bot={bot} />}
                {isEditing && botStyle.value === BotStyle.Specific && <Button
                    fullWidth
                    color="secondary"
                    onClick={handleBotButtonClick}
                    startIcon={!bot ? <AddIcon /> : null}
                    variant={bot ? "outlined" : "contained"}
                >{bot ? "Change bot" : "Choose bot"}</Button>}
            </Box>
            <Divider sx={inputsDividerStyle} />
            <ContentCollapse
                helpText={"Inputs are passed in sequential order to the AI model, within the same request message.\n\nYou may define help text, limits, etc. for each input, though the AI model may not always decide to follow them.\n\nAny input without a default value will be entered by the user at runtime, or the specified bot."}
                title={t("Input", { count: schemaInput.elements.length })}
                isOpen={!disabled}
                titleVariant="h4"
            >
                <FormView
                    disabled={disabled}
                    fieldNamePrefix={PREFIX_INPUT}
                    isEditing={isEditing}
                    onSchemaChange={onSchemaInputChange}
                    schema={schemaInput}
                />
                {display !== "edit" && (
                    <Box onClick={handleRunStep}>
                        <LoadableButton
                            disabled={isRunStepDisabled}
                            fullWidth
                            isLoading={isRunningStep}
                            startIcon={<MagicIcon />}
                            variant="contained"
                        >Generate</LoadableButton>
                    </Box>
                )}
            </ContentCollapse>
            <Divider sx={outputsDividerStyle} />
            <ContentCollapse
                title={t("Output", { count: schemaOutput.elements.length })}
                isOpen={!disabled}
                titleVariant="h4"
            >
                <FormView
                    disabled={true} // Can't currently edit
                    fieldNamePrefix={PREFIX_OUTPUT}
                    isEditing={false}
                    onSchemaChange={noop}
                    schema={schemaOutput}
                />
            </ContentCollapse>
        </>
    );
});

export const RoutineInformationalForm = memo(function RoutineInformationalFormMemo({
    disabled,
    display,
    handleClearRun,
    handleCompleteStep,
    hasErrors,
    isCompleteStepDisabled,
    isPartOfMultiStepRoutine,
    isRunningStep,
    onRunChange,
    onSchemaInputChange,
    routineId,
    routineName,
    run,
    schemaInput,
}: RoutineFormTypeInformational) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const isLoggedIn = uuidValidate(getCurrentUser(session).id);
    console.log("ghgh isLoggedIn", isLoggedIn, session);
    const isEditing = display === "edit";

    const showRunButtons = !isPartOfMultiStepRoutine && display !== "edit";

    const showComplete = showRunButtons && (!run || run.status !== RunStatus.Completed) && !hasErrors;
    const showUpdate = showRunButtons && !showComplete;
    const showFirstButton = showComplete || showUpdate;

    const showClearRun = showRunButtons && run;
    const showPickRun = showRunButtons && !run;
    const showSecondButton = showClearRun || showPickRun;

    // eslint-disable-next-line no-magic-numbers
    const gridItemSm = showFirstButton && showSecondButton ? 6 : 12;

    const [selectRunAnchor, openSelectRunDialog, closeSelectRunDialog] = usePopover();
    const handleOpenSelectRunDialog = useCallback((event: React.MouseEvent<HTMLElement, MouseEvent>) => {
        if (!isLoggedIn) {
            PubSub.get().publish("proDialog");
            return;
        }
        openSelectRunDialog(event);
    }, [isLoggedIn, openSelectRunDialog]);
    const handleRunSelect = useCallback((run: RunProject | RunRoutine | null) => {
        if (!showRunButtons) return;
        onRunChange(run as RunRoutine | null);
    }, [onRunChange, showRunButtons]);

    return (
        <>
            <FormView
                disabled={disabled}
                fieldNamePrefix={PREFIX_INPUT}
                isEditing={isEditing}
                onSchemaChange={onSchemaInputChange}
                schema={schemaInput}
            />
            {
                showRunButtons && (
                    <RunPickerMenu
                        anchorEl={selectRunAnchor}
                        handleClose={closeSelectRunDialog}
                        objectId={routineId}
                        objectName={routineName}
                        objectType={"RoutineVersion"}
                        onSelect={handleRunSelect}
                    />
                )
            }
            {
                (showFirstButton || showSecondButton) && (
                    <Grid container spacing={2} marginTop={4}>
                        {showFirstButton && (
                            <Grid item xs={12} sm={gridItemSm}>
                                <Box onClick={handleCompleteStep}>
                                    <LoadableButton
                                        aria-label={showComplete ? t("Complete") : t("Update")}
                                        disabled={showComplete && isCompleteStepDisabled}
                                        fullWidth
                                        isLoading={isRunningStep}
                                        startIcon={!isLoggedIn ? <LockIcon /> : showComplete ? <SuccessIcon /> : <SaveIcon />}
                                        variant="contained"
                                    >{showComplete ? t("Complete") : t("Update")}</LoadableButton>
                                </Box>
                            </Grid>
                        )}
                        {showSecondButton && (
                            <Grid item xs={12} sm={gridItemSm}>
                                <Box>
                                    <Button
                                        fullWidth
                                        color="secondary"
                                        onClick={showClearRun ? handleClearRun : handleOpenSelectRunDialog}
                                        startIcon={!isLoggedIn ? <LockIcon /> : showClearRun ? <CancelIcon /> : <PlayIcon />}
                                        variant="outlined"
                                    >{showClearRun ? t("Clear") : t("SelectRun")}</Button>
                                </Box>
                            </Grid>
                        )}
                    </Grid>
                )
            }
        </>
    );
});

export const RoutineSmartContractForm = memo(function RoutineSmartContractFormMemo({
    disabled,
    display,
    onSchemaInputChange,
    onSchemaOutputChange,
    schemaInput,
    schemaOutput,
}: RoutineFormTypeSmartContract) {
    const { t } = useTranslation();
    const isEditing = display === "edit";

    return (
        <>
            <Title
                Icon={SmartContractIcon}
                title={isEditing ? "Connect smart contract" : "Connected smart contract"}
                help={"Connect or create a smart contract to this routine.\n\nThe contract will be passed all non-file inputs, and is expected to return all non-file outputs.\n\nIf the contract fails or does not return the expected data, the routine will fail."}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <ContentCollapse
                helpText={"Define the inputs that will be passed to the contract. Any input without a default value will be entered by the user at runtime."}
                title={t("Input", { count: schemaInput.elements.length })}
                isOpen={!disabled}
                titleVariant="h4"
            >
                <FormView
                    disabled={disabled}
                    fieldNamePrefix={PREFIX_INPUT}
                    isEditing={isEditing}
                    onSchemaChange={onSchemaInputChange}
                    schema={schemaInput}
                />
            </ContentCollapse>
            <ContentCollapse
                helpText={"Define the outputs that the contract is expected to return. If the contract fails or does not return the expected data, the routine will fail."}
                title={t("Output", { count: schemaOutput.elements.length })}
                isOpen={!disabled}
                titleVariant="h4"
            >
                <FormView
                    disabled={disabled}
                    fieldNamePrefix={PREFIX_OUTPUT}
                    isEditing={isEditing}
                    onSchemaChange={onSchemaOutputChange}
                    schema={schemaOutput}
                />
            </ContentCollapse>
        </>
    );
});
