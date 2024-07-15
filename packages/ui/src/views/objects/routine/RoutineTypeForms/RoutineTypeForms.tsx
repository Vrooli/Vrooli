import { DUMMY_ID, Node, NodeLink, User, noop, uuidValidate } from "@local/shared";
import { Box, Button, Card, Grid, Typography, styled, useTheme } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObject } from "components/dialogs/types";
import { CodeInputBase, CodeLanguage } from "components/inputs/CodeInput/CodeInput";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { FormView } from "forms/FormView/FormView";
import { FormSchema } from "forms/types";
import { AddIcon, ApiIcon, BotIcon, MinusIcon, OpenInNewIcon, RoutineIcon, SmartContractIcon, TerminalIcon } from "icons";
import { memo, useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ProfileAvatar } from "styles";
import { PartialWithType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { AVAILABLE_MODELS, DEFAULT_MODEL, LlmModel, getModelDescription, getModelName } from "utils/botUtils";
import { getCookiePartialData } from "utils/cookies";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor } from "utils/display/listTools";
import { getTranslation } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { BotShape } from "utils/shape/models/bot";
import { CodeVersionShape, CodeVersionTranslationShape } from "utils/shape/models/codeVersion";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { BuildView } from "views/objects/routine/BuildView/BuildView";
import { BuildRoutineVersion, BuildViewProps } from "../types";

export enum BotStyle {
    Default = "Default",
    Specific = "Specific",
    None = "None",
}
type BotStyleOption = {
    description: string;
    label: string;
    value: BotStyle;
};

export type ConfigCallDataGenerate = {
    botStyle?: BotStyle;
    model?: LlmModel | null;
    respondingBot?: string | null;
};
export type ConfigCallData = ConfigCallDataGenerate | Record<string, never>;

type RoutineFormPropsBase = {
    configCallData: ConfigCallData;
    disabled: boolean;
    isEditing: boolean;
    onConfigCallDataChange: (configCallData: ConfigCallData) => unknown;
    onSchemaInputChange: (schema: FormSchema) => unknown;
    onSchemaOutputChange: (schema: FormSchema) => unknown;
    schemaInput: FormSchema;
    schemaOutput: FormSchema;
}
type RoutineFormTypeApi = RoutineFormPropsBase;
type RoutineFormTypeCode = RoutineFormPropsBase;
type RoutineFormTypeData = RoutineFormPropsBase;
type RoutineFormTypeGenerate = RoutineFormPropsBase;
type RoutineFormTypeInformational = Omit<RoutineFormPropsBase, "onSchemaOutputChange" | "schemaOutput">;
type RoutineFormTypeMultiStep = {
    isEditing: boolean;
    isGraphOpen: boolean;
    handleGraphClose: () => unknown;
    handleGraphOpen: () => unknown;
    handleGraphSubmit: BuildViewProps["handleSubmit"];
    nodeLinks: NodeLinkShape[] | undefined;
    nodes: NodeShape[] | undefined;
    routineId: string | undefined;
    translations: BuildRoutineVersion["translations"] | undefined;
    translationData: BuildViewProps["translationData"];
}
type RoutineFormTypeSmartContract = RoutineFormPropsBase;

const routineTypeTitleSxs = { stack: { paddingLeft: 0 } } as const;

export const RoutineApiForm = memo(function RoutineApiFormMemo({
    disabled,
    isEditing,
    onSchemaInputChange,
    onSchemaOutputChange,
    schemaInput,
    schemaOutput,
}: RoutineFormTypeApi) {
    return (
        <>
            <Title
                Icon={ApiIcon}
                title={isEditing ? "Connect API" : "Connected API"}
                help={"Connect API that will receive the defined inputs and is expected to return the defined outputs.\n\nIf the API fails or does not return the expected data, the routine will fail."}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <Title
                title="Inputs"
                help="Define the inputs that will be passed to the API. Any input without a default value will be entered by the user at runtime."
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <FormView
                disabled={disabled}
                isEditing={isEditing}
                onSchemaChange={onSchemaInputChange}
                schema={schemaInput}
            />
            <Title
                title="Outputs"
                help="Define the outputs that the API is expected to return. If the API fails or does not return the expected data, the routine will fail."
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <FormView
                disabled={disabled}
                isEditing={isEditing}
                onSchemaChange={onSchemaOutputChange}
                schema={schemaOutput}
            />
        </>
    );
});

type CodeObjectInfo = Pick<CodeVersionShape, "__typename" | "id" | "codeLanguage" | "content"> & {
    translations?: Pick<CodeVersionTranslationShape, "id" | "name" | "description" | "jsonVariable" | "language">[];
};

const findCodeLimitTo = ["Code"] as const;

export const RoutineCodeForm = memo(function RoutineCodeFormMemo({
    disabled,
    isEditing,
    onSchemaInputChange,
    onSchemaOutputChange,
    schemaInput,
    schemaOutput,
}: RoutineFormTypeCode) {
    const { palette } = useTheme();
    const session = useContext(SessionContext);
    const [codeObject, setCodeObject] = useState<CodeObjectInfo | null>(null);
    const [isCodeSearchOpen, setIsCodeSearchOpen] = useState(false);

    const closeCodeSearch = useCallback((selected?: SelectOrCreateObject) => {
        setIsCodeSearchOpen(false);
        if (selected) {
            setCodeObject(selected as unknown as CodeObjectInfo);
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
                    handleComplete={closeCodeSearch}
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
            <Title
                title="Inputs"
                help="Define the inputs that will be passed to the code. Any input without a default value will be entered by the user at runtime."
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <FormView
                disabled={disabled}
                isEditing={isEditing}
                onSchemaChange={onSchemaInputChange}
                schema={schemaInput}
            />
            <Title
                title="Outputs"
                help="Define the outputs that the code is expected to return. If the code fails or does not return the expected data, the routine will fail."
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <FormView
                disabled={disabled}
                isEditing={isEditing}
                onSchemaChange={onSchemaOutputChange}
                schema={schemaOutput}
            />
        </>
    );
});

export const RoutineDataForm = memo(function RoutineDataFormMemo({
    disabled,
    isEditing,
    onSchemaOutputChange,
    schemaOutput,
}: RoutineFormTypeData) {
    return (
        <FormView
            disabled={disabled}
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

export const RoutineGenerateForm = memo(function RoutineGenerateFormMemo({
    configCallData,
    disabled,
    isEditing,
    onConfigCallDataChange,
    onSchemaInputChange,
    onSchemaOutputChange,
    schemaInput,
    schemaOutput,
}: RoutineFormTypeGenerate) {
    const { t } = useTranslation();

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

    const [isBotSearchOpen, setIsBotSearchOpen] = useState(false);
    const closeBotSearch = useCallback(function closeBotSearchCallback(selected?: SelectOrCreateObject) {
        setIsBotSearchOpen(false);
        if (selected) {
            handleRespondingBotChange(selected as unknown as BotInfo);
        }
    }, []);
    const handleBotButtonClick = useCallback(function handleBotButtonClickCallback() {
        setIsBotSearchOpen(true);
    }, []);

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

    return (
        <>
            <Title
                title={isEditing ? "Choose AI model" : `Generating with ${configCallData?.model?.name ?? `${DEFAULT_MODEL} (default)`}`}
                help={isEditing ? "Connect API that will receive the defined inputs and is expected to return the defined outputs.\n\nIf the API fails or does not return the expected data, the routine will fail." : undefined}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            {isEditing && <SelectorBase
                name="model"
                options={AVAILABLE_MODELS}
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
            {(isEditing) && <Title
                title={(isEditing || !bot) ? "Choose style" : `Using style of ${bot.name}`}
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
                handleComplete={closeBotSearch}
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
            <Title
                title={"Inputs"}
                help={"Inputs are passed in sequential order to the AI model, within the same request message.\n\nYou may define help text, limits, etc. for each input, though the AI model may not always decide to follow them.\n\nAny input without a default value will be entered by the user at runtime, or the specified bot."}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <FormView
                disabled={disabled}
                isEditing={isEditing}
                onSchemaChange={onSchemaInputChange}
                schema={schemaInput}
            />
        </>
    );
});

export const RoutineInformationalForm = memo(function RoutineInformationalFormMemo({
    disabled,
    isEditing,
    onSchemaInputChange,
    schemaInput,
}: RoutineFormTypeInformational) {
    return (
        <FormView
            disabled={disabled}
            isEditing={isEditing}
            onSchemaChange={onSchemaInputChange}
            schema={schemaInput}
        />
    );
});

export const RoutineMultiStepForm = memo(function RoutineMultiStepFormMemo({
    isEditing,
    isGraphOpen,
    handleGraphClose,
    handleGraphOpen,
    handleGraphSubmit,
    nodeLinks,
    nodes,
    routineId,
    translations,
    translationData,
}: RoutineFormTypeMultiStep) {
    const routineVersion = useMemo(() => ({
        id: routineId ?? DUMMY_ID,
        nodeLinks: (nodeLinks ?? []) as NodeLink[],
        nodes: (nodes ?? []) as Node[],
        translations: (translations ?? []) as BuildRoutineVersion["translations"],
    }), [nodeLinks, nodes, routineId, translations]);

    return (
        <>
            <BuildView
                display="dialog"
                handleCancel={handleGraphClose}
                onClose={handleGraphClose}
                handleSubmit={handleGraphSubmit}
                isEditing={isEditing}
                isOpen={isGraphOpen}
                loading={false}
                routineVersion={routineVersion}
                translationData={translationData}
            />
            {/* Button to display graph */}
            <Grid item xs={12} mb={4}>
                <Button
                    startIcon={<RoutineIcon />}
                    fullWidth color="secondary"
                    onClick={handleGraphOpen}
                    variant={isEditing ? "contained" : "outlined"}
                >View Graph</Button>
            </Grid>
            {/* # nodes, # links, Simplicity, complexity & other graph stats */}
            {/* TODO */}
        </>
    );
});

export const RoutineSmartContractForm = memo(function RoutineSmartContractFormMemo({
    disabled,
    isEditing,
    onSchemaInputChange,
    onSchemaOutputChange,
    schemaInput,
    schemaOutput,
}: RoutineFormTypeSmartContract) {
    return (
        <>
            <Title
                Icon={SmartContractIcon}
                title={isEditing ? "Connect smart contract" : "Connected smart contract"}
                help={"Connect or create a smart contract to this routine.\n\nThe contract will be passed all non-file inputs, and is expected to return all non-file outputs.\n\nIf the contract fails or does not return the expected data, the routine will fail."}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <Title
                title="Inputs"
                help="Define the inputs that will be passed to the contract. Any input without a default value will be entered by the user at runtime."
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <FormView
                disabled={disabled}
                isEditing={isEditing}
                onSchemaChange={onSchemaInputChange}
                schema={schemaInput}
            />
            <Title
                title="Outputs"
                help="Define the outputs that the contract is expected to return. If the contract fails or does not return the expected data, the routine will fail."
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <FormView
                disabled={disabled}
                isEditing={isEditing}
                onSchemaChange={onSchemaOutputChange}
                schema={schemaOutput}
            />
        </>
    );
});
