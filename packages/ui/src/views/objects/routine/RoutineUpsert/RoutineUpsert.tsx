import { DUMMY_ID, endpointGetRoutineVersion, endpointPostRoutineVersion, endpointPutRoutineVersion, LINKS, Node, NodeLink, noopSubmit, orDefault, RoutineType, RoutineVersion, RoutineVersionCreateInput, routineVersionTranslationValidation, RoutineVersionUpdateInput, routineVersionValidation, Session, uuid } from "@local/shared";
import { Button, Checkbox, Divider, FormControlLabel, Grid, Tooltip, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListInput } from "components/lists/resource/ResourceList/ResourceList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { ActionIcon, ApiIcon, CaseSensitiveIcon, HelpIcon, MagicIcon, RoutineIcon, SearchIcon, SmartContractIcon, TerminalIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { AVAILABLE_MODELS, LlmModel } from "utils/botUtils";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { initializeRoutineGraph } from "utils/runUtils";
import { SearchPageTabOption } from "utils/search/objectToSearch";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { RoutineShape } from "utils/shape/models/routine";
import { RoutineVersionShape, shapeRoutineVersion } from "utils/shape/models/routineVersion";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { validateFormValues } from "utils/validateFormValues";
import { BuildView } from "views/objects/routine/BuildView/BuildView";
import { RoutineFormProps, RoutineUpsertProps } from "../types";

export const routineInitialValues = (
    session: Session | undefined,
    existing?: Partial<RoutineVersion> | null | undefined,
): RoutineVersionShape => ({
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
});

const transformRoutineVersionValues = (values: RoutineVersionShape, existing: RoutineVersionShape, isCreate: boolean) =>
    isCreate ? shapeRoutineVersion.create(values) : shapeRoutineVersion.update(existing, values);

const routineTypes = [
    {
        type: RoutineType.Informational,
        label: "Informational/Placeholder",
        description: "Contains no additional data. Used to provide instructions, a way to prompt users for information, or as a placeholder.",
        Icon: HelpIcon,
    }, {
        type: RoutineType.MultiStep,
        label: "Multi-step",
        description: "A combination of other routines, using a graph to define the order of execution.",
        Icon: RoutineIcon,
    }, {
        type: RoutineType.Generate,
        label: "Generate",
        description: "Sends inputs to an AI (e.g. GPT-4o) and returns its output.",
        Icon: MagicIcon,
    }, {
        type: RoutineType.Data,
        label: "Data",
        description: "Contains a single output and nothing else. Useful for providing hard-coded data to other routines, such as a prompt for a \"Generate\" routine.",
        Icon: CaseSensitiveIcon,
    }, {
        type: RoutineType.Action,
        label: "Action",
        description: "Performs specific actions within Vrooli, such as creating, updating, or deleting objects.",
        Icon: ActionIcon,
    }, {
        type: RoutineType.Code,
        label: "Code",
        description: "Runs sandboxed JavaScript code to convert inputs to outputs. Useful for converting plaintext to structured data. Does not have access to the internet.",
        Icon: TerminalIcon,
    }, {
        type: RoutineType.Api,
        label: "API",
        description: "Sends inputs to an API and returns its output. Useful for connecting to external services.",
        Icon: ApiIcon,
    }, {
        type: RoutineType.SmartContract,
        label: "Smart Contract",
        description: "Connects to a smart contract on the blockchain, sending inputs and returning outputs.",
        Icon: SmartContractIcon,
    },
];

const RoutineForm = ({
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
}: RoutineFormProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { palette } = useTheme();

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

    // Formik fields we need to access and/or set values for
    const [idField] = useField<string>("id");
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
    const handleGraphClose = useCallback(() => { console.log("yeet"); setIsGraphOpen(false); }, [setIsGraphOpen]);
    const handleGraphSubmit = useCallback(({ nodes, nodeLinks }: { nodes: RoutineVersion["nodes"], nodeLinks: RoutineVersion["nodeLinks"] }) => {
        nodesHelpers.setValue(nodes);
        nodeLinksHelpers.setValue(nodeLinks);
        setIsGraphOpen(false);
    }, [nodeLinksHelpers, nodesHelpers]);

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
        const performSwitch = () => {
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
        };
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
    const routineTypeComponents = useMemo(() => {
        switch (routineType) {
            case RoutineType.Api:
                return (
                    <>
                        <Title
                            Icon={ApiIcon}
                            title={"Connect API"}
                            help={"Connect API that will receive the defined inputs and is expected to return the defined outputs.\n\nIf the API fails or does not return the expected data, the routine will fail."}
                            variant="subsection"
                            sxs={{ stack: { paddingLeft: 0 } }}
                        />
                        {/* TODO */}
                        <Grid item xs={12}>
                            <InputOutputContainer
                                isEditing={true}
                                handleUpdate={inputsHelpers.setValue as any}
                                isInput={true}
                                language={language}
                                list={inputsField.value}
                            />
                        </Grid>
                        <Grid item xs={12} mb={4}>
                            <InputOutputContainer
                                isEditing={true}
                                handleUpdate={outputsHelpers.setValue as any}
                                isInput={false}
                                language={language}
                                list={outputsField.value}
                            />
                        </Grid>
                    </>
                );
            case RoutineType.Code:
                return (
                    <>
                        <Title
                            Icon={TerminalIcon}
                            title={"Connect code"}
                            help={"Connect or create a data converter function to this routine.\n\nThe code will be passed all non-file inputs, and is expected to return all non-file outputs.\n\nIf the code fails or does not return the expected data, the routine will fail."}
                            variant="subsection"
                            sxs={{ stack: { paddingLeft: 0 } }}
                        />
                        {/* TODO */}
                        <Grid item xs={12}>
                            <InputOutputContainer
                                isEditing={true}
                                handleUpdate={inputsHelpers.setValue as any}
                                isInput={true}
                                language={language}
                                list={inputsField.value}
                            />
                        </Grid>
                        <Grid item xs={12} mb={4}>
                            <InputOutputContainer
                                isEditing={true}
                                handleUpdate={outputsHelpers.setValue as any}
                                isInput={false}
                                language={language}
                                list={outputsField.value}
                            />
                        </Grid>
                    </>
                );
            case RoutineType.Data:
                return (
                    <>
                        <InputOutputContainer
                            isEditing={true}
                            handleUpdate={outputsHelpers.setValue as any}
                            isInput={false}
                            language={language}
                            list={outputsField.value}
                        />
                    </>
                );
            case RoutineType.Generate:
                return (
                    <>
                        <SelectorBase
                            name="model"
                            options={AVAILABLE_MODELS}
                            getOptionLabel={(r) => r.name}
                            getOptionDescription={(r) => r.description}
                            fullWidth={true}
                            inputAriaLabel="Mode"
                            label={t("Model", { count: 1 })}
                            onChange={(newModel) => {
                                setModel(newModel);
                            }}
                            value={model}
                        />
                        {/* TODO */}
                        <Grid item xs={12}>
                            <InputOutputContainer
                                isEditing={true}
                                handleUpdate={inputsHelpers.setValue as any}
                                isInput={true}
                                language={language}
                                list={inputsField.value}
                            />
                        </Grid>
                        <Grid item xs={12} mb={4}>
                            <InputOutputContainer
                                isEditing={true}
                                handleUpdate={outputsHelpers.setValue as any}
                                isInput={false}
                                language={language}
                                list={outputsField.value}
                            />
                        </Grid>
                    </>
                );
            case RoutineType.Informational:
                // Allow inputs to be entered. Since nothing else is connected to the routine, these inputs 
                // will have to be filled out manually by a user or bot
                return (
                    <>
                        <InputOutputContainer
                            isEditing={true}
                            handleUpdate={inputsHelpers.setValue as any}
                            isInput={true}
                            language={language}
                            list={inputsField.value}
                        />
                    </>
                );
            case RoutineType.MultiStep:
                // Display graph editor
                return (
                    <>
                        <BuildView
                            display="dialog"
                            handleCancel={handleGraphClose}
                            onClose={handleGraphClose}
                            handleSubmit={handleGraphSubmit}
                            isEditing={true}
                            isOpen={isGraphOpen}
                            loading={false}
                            routineVersion={{
                                id: idField.value,
                                nodeLinks: nodeLinksField.value as NodeLink[],
                                nodes: nodesField.value as Node[],
                            }}
                            translationData={{
                                language,
                                setLanguage,
                                handleAddLanguage,
                                handleDeleteLanguage,
                                languages,
                            }}
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
            case RoutineType.SmartContract:
                return (
                    <>
                        <Title
                            Icon={SmartContractIcon}
                            title={"Connect smart contract"}
                            help={"Connect or create a smart contract to this routine.\n\nThe contract will be passed all non-file inputs, and is expected to return all non-file outputs.\n\nIf the contract fails or does not return the expected data, the routine will fail."}
                            variant="subsection"
                            sxs={{ stack: { paddingLeft: 0 } }}
                        />
                        {/* TODO */}
                        <Grid item xs={12}>
                            <InputOutputContainer
                                isEditing={true}
                                handleUpdate={inputsHelpers.setValue as any}
                                isInput={true}
                                language={language}
                                list={inputsField.value}
                            />
                        </Grid>
                        <Grid item xs={12} mb={4}>
                            <InputOutputContainer
                                isEditing={true}
                                handleUpdate={outputsHelpers.setValue as any}
                                isInput={false}
                                language={language}
                                list={outputsField.value}
                            />
                        </Grid>
                    </>
                );
        }
    }, [handleAddLanguage, handleDeleteLanguage, handleGraphClose, handleGraphOpen, handleGraphSubmit, idField.value, inputsField.value, inputsHelpers.setValue, isGraphOpen, language, languages, nodeLinksField.value, nodesField.value, outputsField.value, outputsHelpers.setValue, routineType, setLanguage]);

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
            <Button
                href={`${LINKS.Search}?type=${SearchPageTabOption.Routine}`}
                sx={{
                    color: palette.background.textSecondary,
                    display: "flex",
                    marginTop: 2,
                    textAlign: "center",
                    textTransform: "none",
                }}
                variant="text"
                endIcon={<SearchIcon />}
            >
                Search existing routines
            </Button>
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
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
                            <TranslatedTextInput
                                autoFocus
                                fullWidth
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
                            getOptionLabel={(r) => r.label}
                            getOptionDescription={(r) => r.description}
                            getOptionIcon={(r) => r.Icon}
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
};

export const RoutineUpsert = ({
    isCreate,
    isOpen,
    isSubroutine = false,
    overrideObject,
    ...props
}: RoutineUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<RoutineVersion, RoutineVersionShape>({
        ...endpointGetRoutineVersion,
        isCreate,
        objectType: "RoutineVersion",
        overrideObject,
        transform: (existing) => routineInitialValues(session, existing),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformRoutineVersionValues, routineVersionValidation)}
        >
            {(formik) => <RoutineForm
                disabled={!(isCreate || permissions.canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                isSubroutine={isSubroutine}
                versions={(existing?.root as RoutineShape)?.versions?.map(v => v.versionLabel) ?? []}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
