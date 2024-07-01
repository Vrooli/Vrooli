import { DUMMY_ID, endpointGetRoutineVersion, endpointPostRoutineVersion, endpointPutRoutineVersion, LINKS, Node, NodeLink, noop, noopSubmit, orDefault, RoutineType, RoutineVersion, RoutineVersionCreateInput, routineVersionTranslationValidation, RoutineVersionUpdateInput, routineVersionValidation, Session, uuid } from "@local/shared";
import { Avatar, Box, Button, Card, Checkbox, Divider, FormControlLabel, Grid, styled, Tooltip, Typography, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { SearchExistingButton } from "components/buttons/SearchExistingButton/SearchExistingButton";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SelectOrCreateObject } from "components/dialogs/types";
import { CodeInputBase, CodeLanguage } from "components/inputs/CodeInput/CodeInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListInput } from "components/lists/resource/ResourceList/ResourceList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { FormView } from "forms/FormView/FormView";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { AddIcon, ApiIcon, BotIcon, MinusIcon, OpenInNewIcon, RoutineIcon, SmartContractIcon, TerminalIcon } from "icons";
import { memo, useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { AVAILABLE_MODELS, DEFAULT_MODEL, getModelDescription, getModelName, LlmModel } from "utils/botUtils";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor } from "utils/display/listTools";
import { combineErrorsWithTranslations, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { initializeRoutineGraph } from "utils/runUtils";
import { SearchPageTabOption } from "utils/search/objectToSearch";
import { getRoutineTypeDescription, getRoutineTypeIcon, getRoutineTypeLabel, routineTypes } from "utils/search/schemas/routine";
import { BotShape } from "utils/shape/models/bot";
import { CodeVersionShape, CodeVersionTranslationShape } from "utils/shape/models/codeVersion";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { RoutineShape } from "utils/shape/models/routine";
import { RoutineVersionShape, shapeRoutineVersion } from "utils/shape/models/routineVersion";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { validateFormValues } from "utils/validateFormValues";
import { BuildView } from "views/objects/routine/BuildView/BuildView";
import { BuildRoutineVersion, BuildViewProps, RoutineFormProps, RoutineUpsertProps } from "../types";

const routineTypeTitleSxs = { stack: { paddingLeft: 0 } } as const;

export function routineInitialValues(
    session: Session | undefined,
    existing?: Partial<RoutineVersion> | null | undefined,
): RoutineVersionShape {
    return {
        __typename: "RoutineVersion" as const,
        id: uuid(), // Cannot be a dummy ID because nodes, links, etc. reference this ID
        inputs: [],
        isComplete: false,
        isPrivate: false,
        directoryListings: [],
        nodeLinks: [],
        nodes: [],
        outputs: [],
        routineType: RoutineType.Informational,
        versionLabel: "1.0.0",
        ...existing,
        root: {
            __typename: "Routine" as const,
            id: DUMMY_ID,
            isPrivate: false,
            owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
            parent: null,
            permissions: JSON.stringify({}),
            tags: [],
            ...existing?.root,
        },
        resourceList: orDefault<RoutineVersionShape["resourceList"]>(existing?.resourceList, {
            __typename: "ResourceList" as const,
            id: DUMMY_ID,
            listFor: {
                __typename: "RoutineVersion" as const,
                id: DUMMY_ID,
            },
        }),
        translations: orDefault(existing?.translations, [{
            __typename: "RoutineVersionTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            instructions: "",
            name: "",
        }]),
    };
}

function transformRoutineVersionValues(values: RoutineVersionShape, existing: RoutineVersionShape, isCreate: boolean) {
    return isCreate ? shapeRoutineVersion.create(values) : shapeRoutineVersion.update(existing, values);
}

const RoutineApiForm = memo(function RoutineApiFormMemo({
    isEditing,
}: {
    isEditing: boolean;
}) {
    return (
        <>
            <Title
                Icon={ApiIcon}
                title={isEditing ? "Connect API" : "Connected API"}
                help={"Connect API that will receive the defined inputs and is expected to return the defined outputs.\n\nIf the API fails or does not return the expected data, the routine will fail."}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            {/* TODO inputs/outputs */}
            <FormView isEditing={isEditing} />
            <FormView isEditing={isEditing} />
        </>
    );
});

type CodeObjectInfo = Pick<CodeVersionShape, "__typename" | "id" | "codeLanguage" | "content"> & {
    translations?: Pick<CodeVersionTranslationShape, "id" | "name" | "description" | "jsonVariable" | "language">[];
};

const findCodeLimitTo = ["Code"] as const;

const RoutineCodeForm = memo(function RoutineCodeFormMemo({
    isEditing,
}: {
    isEditing: boolean;
}) {
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
            <FormView isEditing={isEditing} />
            <Title
                title="Outputs"
                help="Define the outputs that the code is expected to return. If the code fails or does not return the expected data, the routine will fail."
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <FormView isEditing={isEditing} />
        </>
    );
});

const RoutineDataForm = memo(function RoutineDataFormMemo({
    isEditing,
}: {
    isEditing: boolean;
}) {
    return (
        // Form to define outputs
        <FormView isEditing={isEditing} />
    );
});

type BotInfo = Pick<BotShape, "__typename" | "id" | "handle" | "model" | "name" | "profileImage">;

const botStyleOptions = [{
    description: "If a bot runs this routine, their style will be used to generate the response. Otherwise, no style will be used.",
    label: "Default",
    value: "default",
}, {
    description: "Select a specific bot to use their style for generating the response.",
    label: "Specific bot",
    value: "specific",
}, {
    description: "Do not use any style to generate the response.",
    label: "None",
    value: "none",
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
    padding: 2,
    marginTop: 2,
    marginBottom: 2,
    borderRadius: "8px",
}));


const BotCardAvatar = memo(function BotCardAvatarMemo({ bot }: { bot: BotInfo }) {
    const alt = `${bot.name} profile image`;
    const src = useMemo(() => extractImageUrl(bot.profileImage, undefined, BOT_AVATAR_IMG_SRC_TARGET_SIZE), [bot.profileImage]);

    const profileColors = useMemo(() => placeholderColor(), []);
    const style = useMemo(() => ({
        width: 60,
        height: 60,
        backgroundColor: profileColors[0],
        color: profileColors[1],
        borderRadius: "8px", // Bots show up as squares
        marginRight: 2,
    }), [profileColors]);

    return (
        <Avatar
            alt={alt}
            src={src}
            sx={style}
        >
            <BotIcon width="75%" height="75%" />
        </Avatar>
    );
});

const BotHandle = styled(Typography)(({ theme }) => ({
    color: theme.palette.secondary.dark,
    fontFamily: "monospace",
}));

const botCardOpenIconBoxStyle = { display: "grid", marginLeft: "auto", paddingRight: 1 } as const;

const BotCard = memo(function BotCardMemo({ bot }: { bot: BotInfo }) {
    const [, setLocation] = useLocation();

    const handleBotCardClick = useCallback(function handleBotCardClickCallback() {
        openObject(bot, setLocation);
    }, [bot, setLocation]);

    return (
        <BotCardOuter onClick={handleBotCardClick}>
            <BotCardAvatar bot={bot} />
            <Box>
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

type BotStyleOption = {
    description: string;
    label: string;
    value: string;
};

const RoutineGenerateForm = memo(function RoutineGenerateFormMemo({
    isEditing,
    model,
    setModel,
}: {
    isEditing: boolean;
    model: LlmModel | null;
    setModel: (model: LlmModel | null) => unknown;
}) {
    const { t } = useTranslation();

    const handleModelChange = useCallback(function handleModelChangeCallback(newModel: LlmModel | null) {
        setModel(newModel);
    }, [setModel]);

    const [botStyle, setBotStyle] = useState<BotStyleOption>(botStyleOptions[0]);
    const [bot, setBot] = useState<BotInfo | null>(null);
    const [isBotSearchOpen, setIsBotSearchOpen] = useState(false);
    const closeBotSearch = useCallback(function closeBotSearchCallback(selected?: SelectOrCreateObject) {
        setIsBotSearchOpen(false);
        if (selected) {
            setBot(selected as unknown as BotInfo);
        }
    }, []);
    const handleBotStyleChange = useCallback(function handleBotStyleChangeCallback(newStyle: BotStyleOption) {
        setBotStyle(newStyle);
        if (newStyle.value !== "specific") setBot(null);
        if (newStyle.value === "specific") setIsBotSearchOpen(true);
    }, []);
    const handleBotButtonClick = useCallback(function handleBotButtonClickCallback() {
        if (bot) setBot(null);
        else setIsBotSearchOpen(true);
    }, [bot]);

    return (
        <>
            <Title
                title={isEditing ? "Choose AI model" : `Generating with ${model?.name ?? `${DEFAULT_MODEL} (default)`}`}
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
                value={model}
            />}
            {(isEditing || bot) && <Title
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
            {isEditing && botStyle.value === "specific" && <FindObjectDialog
                find="List"
                isOpen={isBotSearchOpen}
                limitTo={findBotLimitTo}
                handleCancel={closeBotSearch}
                handleComplete={closeBotSearch}
                where={findBotWhere}
            />}
            {bot && <BotCard bot={bot} />}
            {isEditing && botStyle.value === "specific" && <Button
                fullWidth
                color="secondary"
                variant="contained"
                onClick={handleBotButtonClick}
                startIcon={bot ? <MinusIcon /> : <AddIcon />}
            >{bot ? "Remove bot" : "Choose bot"}</Button>}
            <Title
                title={"Inputs"}
                help={"Inputs are passed in sequential order to the AI model, within the same request message.\n\nYou may define help text, limits, etc. for each input, though the AI model may not always decide to follow them.\n\nAny input without a default value will be entered by the user at runtime, or the specified bot."}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            <FormView isEditing={isEditing} />
        </>
    );
});

const RoutineInformationalForm = memo(function RoutineInformationalFormMemo({
    isEditing,
}: {
    isEditing: boolean;
}) {
    return (
        // Form to define inputs
        <FormView isEditing={isEditing} />
    );
});

const RoutineMultiStepForm = memo(function RoutineMultiStepFormMemo({
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
}: {
    isEditing: boolean;
    isGraphOpen: boolean;
    handleGraphClose: () => unknown;
    handleGraphOpen: () => unknown;
    handleGraphSubmit: BuildViewProps["handleSubmit"];
    nodeLinks: NodeLinkShape[];
    nodes: NodeShape[];
    routineId: string;
    translations: BuildRoutineVersion["translations"];
    translationData: BuildViewProps["translationData"];
}) {
    const routineVersion = useMemo(() => ({
        id: routineId,
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
                    variant="contained"
                >View Graph</Button>
            </Grid>
            {/* # nodes, # links, Simplicity, complexity & other graph stats */}
            {/* TODO */}
        </>
    );
});

const RoutineSmartContractForm = memo(function RoutineSmartContractFormMemo({
    isEditing,
}: {
    isEditing: boolean;
}) {
    return (
        <>
            <Title
                Icon={SmartContractIcon}
                title={isEditing ? "Connect smart contract" : "Connected smart contract"}
                help={"Connect or create a smart contract to this routine.\n\nThe contract will be passed all non-file inputs, and is expected to return all non-file outputs.\n\nIf the contract fails or does not return the expected data, the routine will fail."}
                variant="subsection"
                sxs={routineTypeTitleSxs}
            />
            {/* TODO inputs/outputs */}
            <FormView isEditing={isEditing} />
            <FormView isEditing={isEditing} />
        </>
    );
});

function RoutineForm({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    isSubroutine,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    versions,
    ...props
}: RoutineFormProps) {
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
        validationSchema: routineVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });
    const translationData = useMemo(() => ({
        language,
        setLanguage,
        handleAddLanguage,
        handleDeleteLanguage,
        languages,
    }), [language, languages, setLanguage, handleAddLanguage, handleDeleteLanguage]);

    // Formik fields we need to access and/or set values for
    const [idField] = useField<string>("id");
    const [translationsField, , translationsHelpers] = useField<RoutineVersion["translations"]>("translations");
    const [nodesField, , nodesHelpers] = useField<NodeShape[]>("nodes");
    const [nodeLinksField, , nodeLinksHelpers] = useField<NodeLinkShape[]>("nodeLinks");
    const [inputsField, , inputsHelpers] = useField<RoutineVersionInputShape[]>("inputs");
    const [outputsField, , outputsHelpers] = useField<RoutineVersionOutputShape[]>("outputs");
    const [configCallDataField, , configCallDataHelpers] = useField<RoutineVersion["configCallData"]>("configCallData");
    const [apiVersionField, , apiVersionHelpers] = useField<RoutineVersion["apiVersion"]>("apiVersion");
    const [codeVersionField, , codeVersionHelpers] = useField<RoutineVersion["codeVersion"]>("codeVersion");

    // Multi-step routine data
    const [isGraphOpen, setIsGraphOpen] = useState(false);
    const handleGraphOpen = useCallback(() => {
        // Create initial nodes/links, if not already created
        if (
            (!Array.isArray(nodesField.value) || nodesField.value.length === 0) &&
            (!Array.isArray(nodeLinksField.value) || nodeLinksField.value.length === 0)
        ) {
            const { nodes, nodeLinks } = initializeRoutineGraph(language, idField.value);
            nodesHelpers.setValue(nodes);
            nodeLinksHelpers.setValue(nodeLinks);
        }
        setIsGraphOpen(true);
    }, [idField.value, language, nodeLinksField.value, nodeLinksHelpers, nodesField.value, nodesHelpers]);
    const handleGraphClose = useCallback(() => { setIsGraphOpen(false); }, [setIsGraphOpen]);
    const handleGraphSubmit = useCallback(({ nodes, nodeLinks, translations }: BuildRoutineVersion) => {
        nodesHelpers.setValue(nodes);
        nodeLinksHelpers.setValue(nodeLinks);
        translationsHelpers.setValue(translations);
        setIsGraphOpen(false);
    }, [nodeLinksHelpers, nodesHelpers, translationsHelpers]);

    // Generate AI routine data
    const [model, setModel] = useState<LlmModel | null>(null);

    // Handle routine type
    const [routineType, setRoutineType] = useState<RoutineType>(RoutineType.Informational); // Default to this because it's the most basic
    const handleRoutineTypeChange = useCallback((newType: RoutineType) => {
        // If type is the same, do nothing
        if (newType === routineType) return;
        // Map to check if the type we're switching FROM has data that will be lost
        const loseDataCheck = {
            // Has call data
            [RoutineType.Action]: typeof configCallDataField.value === "string" && configCallDataField.value.length > 0,
            // Has API information
            [RoutineType.Api]: (typeof configCallDataField.value === "string" && configCallDataField.value.length > 0) || typeof apiVersionField.value === "object",
            // Has code information
            [RoutineType.Code]: (typeof configCallDataField.value === "string" && configCallDataField.value.length > 0) || typeof codeVersionField.value === "object",
            // Has an output
            [RoutineType.Data]: Array.isArray(outputsField.value) && outputsField.value.length > 0,
            // Has an input or call data
            [RoutineType.Generate]: (Array.isArray(inputsField.value) && inputsField.value.length > 0) || (typeof configCallDataField.value === "string" && configCallDataField.value.length > 0),
            // Has no additional data, so nothing to lose
            [RoutineType.Informational]: false,
            // Has graph information
            [RoutineType.MultiStep]: (Array.isArray(nodesField.value) && nodesField.value.length > 0) || (Array.isArray(nodeLinksField.value) && nodeLinksField.value.length > 0),
            // Also uses code information
            [RoutineType.SmartContract]: (typeof configCallDataField.value === "string" && configCallDataField.value.length > 0) || typeof codeVersionField.value === "object",
        };
        // Helper function to remove all type-specific data on switch
        function performSwitch() {
            apiVersionHelpers.setValue(null);
            codeVersionHelpers.setValue(null);
            configCallDataHelpers.setValue("");
            inputsHelpers.setValue([]);
            outputsHelpers.setValue([]);
            nodesHelpers.setValue([]);
            nodeLinksHelpers.setValue([]);
            setRoutineType(newType);
            // If we switch to a multi-step routine, open the graph
            if (newType === RoutineType.MultiStep) handleGraphOpen();
        }
        // If we're losing data, confirm with user
        const losingData = loseDataCheck[routineType];
        if (losingData) {
            PubSub.get().publish("alertDialog", {
                messageKey: "RoutineTypeSwitchLoseData",
                buttons: [{
                    labelKey: "Yes",
                    onClick: performSwitch,
                }, {
                    labelKey: "Cancel",
                }],
            });
        }
        // Otherwise, just switch
        else {
            performSwitch();
        }
    }, [configCallDataField.value, configCallDataHelpers, apiVersionField.value, apiVersionHelpers, codeVersionField.value, codeVersionHelpers, handleGraphOpen, inputsField.value, inputsHelpers, nodeLinksField.value, nodeLinksHelpers, nodesField.value, nodesHelpers, outputsField.value, outputsHelpers, routineType]);

    const { handleCancel, handleCompleted } = useUpsertActions<RoutineVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "RoutineVersion",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostRoutineVersion,
        endpointUpdate: endpointPutRoutineVersion,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "RoutineVersion" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<RoutineVersionCreateInput | RoutineVersionUpdateInput, RoutineVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformRoutineVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    // Type-specific components
    const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
        const isEditing = true;
        switch (routineType) {
            case RoutineType.Api:
                return <RoutineApiForm isEditing={isEditing} />;
            case RoutineType.Code:
                return <RoutineCodeForm isEditing={isEditing} />;
            case RoutineType.Data:
                return <RoutineDataForm isEditing={isEditing} />;
            case RoutineType.Generate:
                return <RoutineGenerateForm
                    isEditing={isEditing}
                    model={model}
                    setModel={setModel}
                />;
            case RoutineType.Informational:
                return <RoutineInformationalForm isEditing={isEditing} />;
            case RoutineType.MultiStep:
                return <RoutineMultiStepForm
                    isEditing={isEditing}
                    isGraphOpen={isGraphOpen}
                    handleGraphClose={handleGraphClose}
                    handleGraphOpen={handleGraphOpen}
                    handleGraphSubmit={handleGraphSubmit}
                    nodeLinks={nodeLinksField.value}
                    nodes={nodesField.value}
                    routineId={idField.value}
                    translations={translationsField.value}
                    translationData={translationData}
                />;
            case RoutineType.SmartContract:
                return <RoutineSmartContractForm isEditing={isEditing} />;
        }
    }, [handleGraphClose, handleGraphOpen, handleGraphSubmit, idField.value, isGraphOpen, model, nodeLinksField.value, nodesField.value, routineType, translationData, translationsField.value]);

    return (
        <MaybeLargeDialog
            display={display}
            id="routine-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateRoutine" : "UpdateRoutine")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.Routine}"`}
                text="Search existing routines"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={display === "page"} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <RelationshipList
                            isEditing={true}
                            objectType={"Routine"}
                            sx={{ marginBottom: 2 }}
                        />
                        <ResourceListInput
                            horizontal
                            isCreate={true}
                            parent={{ __typename: "RoutineVersion", id: values.id }}
                            sxs={{ list: { marginBottom: 2 } }}
                        />
                        <FormSection sx={{ overflowX: "hidden", marginBottom: 2 }}>
                            {/* TODO: work on fix for autoFocus accessibility issue. Probably need to use ref and useEffect, which also requires making RichInput a forwardRef. If doing this, then we can autoFocus in the helpbutton edit mode as well */}
                            <TranslatedTextInput
                                autoFocus
                                fullWidth
                                isRequired
                                label={t("Name")}
                                language={language}
                                name="name"
                                placeholder={t("NamePlaceholder")}
                            />
                            <TranslatedRichInput
                                language={language}
                                name="description"
                                maxChars={2048}
                                maxRows={4}
                                minRows={2}
                                placeholder={t("DescriptionPlaceholder")}
                            />
                            <TranslatedRichInput
                                language={language}
                                name="instructions"
                                maxChars={8192}
                                minRows={4}
                                placeholder={t("Instructions")}
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
                        <TagSelector name="root.tags" sx={{ marginBottom: 2 }} />
                        <VersionInput
                            fullWidth
                            versions={versions}
                            sx={{ marginBottom: 2 }}
                        />
                    </ContentCollapse>
                    <Divider />
                    {/* Is internal checkbox */}
                    {isSubroutine && (
                        <Grid item xs={12}>
                            <Tooltip placement={"top"} title='Indicates if this routine is meant to be a subroutine for only one other routine. If so, it will not appear in search resutls.'>
                                <FormControlLabel
                                    label='Internal'
                                    control={
                                        <Checkbox
                                            id='routine-info-dialog-is-internal'
                                            size="small"
                                            name='isInternal'
                                            color='secondary'
                                        />
                                    }
                                />
                            </Tooltip>
                        </Grid>
                    )}
                    <FormSection>
                        {/* Selector for routine type*/}
                        <SelectorBase
                            name="routineType"
                            options={routineTypes}
                            getOptionDescription={getRoutineTypeDescription}
                            getOptionLabel={getRoutineTypeLabel}
                            getOptionIcon={getRoutineTypeIcon}
                            fullWidth={true}
                            inputAriaLabel="Routine Type"
                            label="Routine Type"
                            onChange={({ type }) => handleRoutineTypeChange(type)}
                            value={routineTypes.find(r => r.type === routineType) ?? routineTypes[0]}
                        />
                        {routineTypeComponents}
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
        </MaybeLargeDialog >
    );
}

export function RoutineUpsert({
    isCreate,
    isOpen,
    isSubroutine = false,
    overrideObject,
    ...props
}: RoutineUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<RoutineVersion, RoutineVersionShape>({
        ...endpointGetRoutineVersion,
        isCreate,
        objectType: "RoutineVersion",
        overrideObject,
        transform: (existing) => routineInitialValues(session, existing),
    });

    async function validateValues(values: unknown) {
        return await validateFormValues(values as RoutineVersionShape, existing, isCreate, transformRoutineVersionValues, routineVersionValidation);
    }

    const versions = useMemo(() => (existing?.root as RoutineShape)?.versions?.map(v => v.versionLabel) ?? [], [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <RoutineForm
                disabled={!(isCreate || permissions.canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                isSubroutine={isSubroutine}
                versions={versions}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
