import { DUMMY_ID, endpointGetRoutineVersion, endpointPostRoutineVersion, endpointPutRoutineVersion, LINKS, Node, NodeLink, noopSubmit, orDefault, RoutineVersion, RoutineVersionCreateInput, routineVersionTranslationValidation, RoutineVersionUpdateInput, routineVersionValidation, Session, uuid } from "@local/shared";
import { Button, Checkbox, Divider, FormControlLabel, Grid, Stack, Tooltip, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
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
import { CompleteIcon, RoutineIcon, SearchIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
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
import { BuildView } from "views/BuildView/BuildView";
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

const helpTextSubroutines = "A routine can be made from scratch (single-step), or by combining other routines (multi-step).\n\nA single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.\n\nA multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.";

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

    const [idField] = useField<string>("id");
    const [nodesField, , nodesHelpers] = useField<NodeShape[]>("nodes");
    const [nodeLinksField, , nodeLinksHelpers] = useField<NodeLinkShape[]>("nodeLinks");
    const [inputsField, , inputsHelpers] = useField<RoutineVersionInputShape[]>("inputs");
    const [outputsField, , outputsHelpers] = useField<RoutineVersionOutputShape[]>("outputs");

    const [isGraphOpen, setIsGraphOpen] = useState(false);
    const handleGraphOpen = useCallback(() => {
        // Create initial nodes/links, if not already created
        if (nodesField.value.length === 0 && nodeLinksField.value.length === 0) {
            const { nodes, nodeLinks } = initializeRoutineGraph(language, idField.value);
            nodesHelpers.setValue(nodes);
            nodeLinksHelpers.setValue(nodeLinks);
        }
        setIsGraphOpen(true);
    }, [idField.value, language, nodeLinksField.value.length, nodeLinksHelpers, nodesField.value.length, nodesHelpers]);
    const handleGraphClose = useCallback(() => { setIsGraphOpen(false); }, [setIsGraphOpen]);
    const handleGraphSubmit = useCallback(({ nodes, nodeLinks }: { nodes: RoutineVersion["nodes"], nodeLinks: RoutineVersion["nodeLinks"] }) => {
        nodesHelpers.setValue(nodes);
        nodeLinksHelpers.setValue(nodeLinks);
        setIsGraphOpen(false);
    }, [nodeLinksHelpers, nodesHelpers]);

    // You can use this component to create both single-step and multi-step routines.
    // The beginning of the form is information shared between both types of routines.
    const [isMultiStep, setIsMultiStep] = useState<boolean | null>(null);
    useEffect(() => { setIsMultiStep(nodesField.value.length > 0); }, [nodesField.value.length]);
    const handleMultiStepChange = useCallback((isMultiStep: boolean) => {
        // If setting from true to false, check if any nodes or nodeLinks have been added. 
        // If so, prompt the user to confirm (these will be lost).
        if (isMultiStep === false && (nodesField.value.length > 0 || nodeLinksField.value.length > 0)) {
            PubSub.get().publish("alertDialog", {
                messageKey: "SubroutineGraphDeleteConfirm",
                buttons: [{
                    labelKey: "Yes",
                    onClick: () => { setIsMultiStep(false); handleGraphClose(); },
                }, {
                    labelKey: "Cancel",
                }],
            });
        }
        // If settings from false to true, check if any inputs or outputs have been added.
        // If so, prompt the user to confirm (these will be lost).
        else if (isMultiStep === true && (inputsField.value.length > 0 || outputsField.value.length > 0)) {
            PubSub.get().publish("alertDialog", {
                messageKey: "RoutineInputsDeleteConfirm",
                buttons: [{
                    labelKey: "Yes",
                    onClick: () => { setIsMultiStep(true); handleGraphOpen(); },
                }, {
                    labelKey: "Cancel",
                }],
            });
        }
        // Otherwise, just set the value.
        else {
            setIsMultiStep(isMultiStep);
        }
    }, [nodesField.value.length, nodeLinksField.value.length, inputsField.value.length, outputsField.value.length, handleGraphClose, handleGraphOpen]);

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
                        {/* Selector for single-step or multi-step routine */}
                        <Grid item xs={12} mb={isMultiStep === null ? 8 : 2}>
                            {/* Title with help text */}
                            <Title
                                title="Use subroutines?"
                                help={helpTextSubroutines}
                                variant="subheader"
                            />
                            {/* Yes/No buttons */}
                            <Stack direction="row" display="flex" alignItems="center" justifyContent="center" spacing={1} >
                                <Button
                                    disabled={isMultiStep === true}
                                    fullWidth
                                    color="secondary"
                                    onClick={() => handleMultiStepChange(true)}
                                    variant="outlined"
                                    startIcon={isMultiStep === true ? <CompleteIcon /> : undefined}
                                >{t("Yes")}</Button>
                                <Button
                                    disabled={isMultiStep === false}
                                    fullWidth
                                    color="secondary"
                                    onClick={() => handleMultiStepChange(false)}
                                    variant="outlined"
                                    startIcon={isMultiStep === false ? <CompleteIcon /> : undefined}
                                >{t("No")}</Button>
                            </Stack >
                        </Grid >
                        {/* Data displayed only by multi-step routines */}
                        {
                            isMultiStep === true && (
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
                            )
                        }
                        {/* Data displayed only by single-step routines */}
                        {
                            isMultiStep === false && (
                                <>
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
                            )
                        }
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
