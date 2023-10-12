import { DUMMY_ID, endpointGetRoutineVersion, endpointPostRoutineVersion, endpointPutRoutineVersion, Node, NodeLink, orDefault, RoutineVersion, RoutineVersionCreateInput, routineVersionTranslationValidation, RoutineVersionUpdateInput, routineVersionValidation, Session, uuid } from "@local/shared";
import { Button, Checkbox, FormControlLabel, Grid, Stack, Tooltip } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { RoutineFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { CompleteIcon, RoutineIcon } from "icons";
import { forwardRef, useCallback, useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { initializeRoutineGraph } from "utils/runUtils";
import { validateAndGetYupErrors } from "utils/shape/general";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { RoutineShape } from "utils/shape/models/routine";
import { RoutineVersionShape, shapeRoutineVersion } from "utils/shape/models/routineVersion";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { BuildView } from "views/BuildView/BuildView";
import { RoutineUpsertProps } from "../types";

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

const transformRoutineValues = (values: RoutineVersionShape, existing: RoutineVersionShape, isCreate: boolean) =>
    isCreate ? shapeRoutineVersion.create(values) : shapeRoutineVersion.update(existing, values);

const validateRoutineValues = async (values: RoutineVersionShape, existing: RoutineVersionShape, isCreate: boolean) => {
    const transformedValues = transformRoutineValues(values, existing, isCreate);
    const validationSchema = routineVersionValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};


const helpTextSubroutines = "A routine can be made from scratch (single-step), or by combining other routines (multi-step).\n\nA single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.\n\nA multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.";

const RoutineForm = forwardRef<BaseFormRef | undefined, RoutineFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    isSubroutine,
    onCancel,
    values,
    versions,
    ...props
}, ref) => {
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
        fields: ["description", "instructions", "name"],
        validationSchema: routineVersionTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
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
            PubSub.get().publishAlertDialog({
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
            PubSub.get().publishAlertDialog({
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
                        objectType={"Routine"}
                    />
                    <ResourceListHorizontalInput
                        isCreate={true}
                        parent={{ __typename: "RoutineVersion", id: values.id }}
                    />
                    <FormSection>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Name")}
                            language={language}
                            name="name"
                        />
                        <TranslatedRichInput
                            language={language}
                            name="description"
                            maxChars={2048}
                            maxRows={4}
                            minRows={2}
                            placeholder={t("Description")}
                        />
                        <TranslatedRichInput
                            language={language}
                            name="instructions"
                            maxChars={8192}
                            minRows={4}
                            placeholder={t("Instructions")}
                        />
                        <br />
                        <TagSelector name="root.tags" />
                        <VersionInput
                            fullWidth
                            versions={versions}
                        />
                    </FormSection>
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
                                    fullWidth
                                    color="secondary"
                                    onClick={() => handleMultiStepChange(true)}
                                    variant="outlined"
                                    startIcon={isMultiStep === true ? <CompleteIcon /> : undefined}
                                >{t("Yes")}</Button>
                                <Button
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
                                    {/* Dialog for building routine graph */}
                                    <BuildView
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
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});

export const RoutineUpsert = ({
    isCreate,
    isOpen,
    isSubroutine = false,
    onCancel,
    onCompleted,
    overrideObject,
}: RoutineUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<RoutineVersion, RoutineVersionShape>({
        ...endpointGetRoutineVersion,
        objectType: "RoutineVersion",
        overrideObject,
        transform: (existing) => routineInitialValues(session, existing),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput>({
        display,
        endpointCreate: endpointPostRoutineVersion,
        endpointUpdate: endpointPutRoutineVersion,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="routine-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateRoutine" : "UpdateRoutine")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    console.log("ROUTINE VALUES", values);
                    console.log("ROUTINE TRANSFORMED", transformRoutineValues(values, existing, isCreate));
                    fetchLazyWrapper<RoutineVersionCreateInput | RoutineVersionUpdateInput, RoutineVersion>({
                        fetch,
                        inputs: transformRoutineValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateRoutineValues(values, existing, isCreate)}
            >
                {(formik) => <RoutineForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    isSubroutine={isSubroutine}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    versions={(existing?.root as RoutineShape)?.versions?.map(v => v.versionLabel) ?? []}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
