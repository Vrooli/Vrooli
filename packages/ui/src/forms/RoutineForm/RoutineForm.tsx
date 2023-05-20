import { DUMMY_ID, Node, NodeLink, orDefault, RoutineIcon, RoutineVersion, routineVersionTranslationValidation, routineVersionValidation, Session, uuid } from "@local/shared";
import { Button, Checkbox, FormControlLabel, Grid, Stack, Tooltip, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { Subheader } from "components/text/Subheader/Subheader";
import { useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { RoutineFormProps } from "forms/types";
import { forwardRef, useCallback, useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { PubSub } from "utils/pubsub";
import { initializeRoutineGraph } from "utils/runUtils";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { RoutineVersionShape, shapeRoutineVersion } from "utils/shape/models/routineVersion";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { BuildView } from "views/BuildView/BuildView";

export const routineInitialValues = (
    session: Session | undefined,
    existing?: RoutineVersion | null | undefined,
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
    resourceList: {
        __typename: "ResourceList" as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: "Routine" as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session)!.id! },
        parent: null,
        permissions: JSON.stringify({}),
        tags: [],
    },
    versionLabel: "1.0.0",
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "RoutineVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        instructions: "",
        name: "",
    }]),
});

export const transformRoutineValues = (values: RoutineVersionShape, existing?: RoutineVersionShape) => {
    return existing === undefined
        ? shapeRoutineVersion.create(values)
        : shapeRoutineVersion.update(existing, values);
};

export const validateRoutineValues = async (values: RoutineVersionShape, existing?: RoutineVersionShape) => {
    const transformedValues = transformRoutineValues(values, existing);
    const validationSchema = routineVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};


const helpTextSubroutines = "A routine can be made from scratch (single-step), or by combining other routines (multi-step).\n\nA single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.\n\nA multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.";

export const RoutineForm = forwardRef<any, RoutineFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    isSubroutine,
    onCancel,
    values,
    versions,
    zIndex,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
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
        validationSchema: routineVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    console.log("valuesssssss", values, transformRoutineValues(values), validateAndGetYupErrors(routineVersionValidation.create({}), transformRoutineValues(values)));

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
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(700px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <RelationshipList
                        isEditing={true}
                        objectType={"Routine"}
                        zIndex={zIndex}
                    />
                    <ResourceListHorizontalInput
                        isCreate={true}
                        zIndex={zIndex}
                    />
                    <Stack direction="column" spacing={2} sx={{
                        borderRadius: 2,
                        background: palette.mode === "dark" ? palette.background.paper : palette.background.default,
                        padding: 2,
                    }}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                            zIndex={zIndex + 1}
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Name")}
                            language={language}
                            name="name"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Description")}
                            language={language}
                            multiline
                            minRows={2}
                            maxRows={2}
                            name="description"
                        />
                        <TranslatedMarkdownInput
                            language={language}
                            name="instructions"
                            minRows={4}
                        />
                        <br />
                        <TagSelector
                            name="root.tags"
                            zIndex={zIndex}
                        />
                        <VersionInput
                            fullWidth
                            versions={versions}
                        />
                    </Stack>
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
                    <Stack direction="column" spacing={2} sx={{
                        borderRadius: 2,
                        background: palette.mode === "dark" ? palette.background.paper : palette.background.default,
                        padding: 2,
                        paddingTop: 0,
                    }}>
                        {/* Selector for single-step or multi-step routine */}
                        <Grid item xs={12} mb={isMultiStep === null ? 8 : 2}>
                            {/* Title with help text */}
                            <Subheader
                                title="Use subroutines?"
                                help={helpTextSubroutines}
                            />
                            {/* Yes/No buttons */}
                            <Stack direction="row" display="flex" alignItems="center" justifyContent="center" spacing={1} >
                                <Button fullWidth color="secondary" onClick={() => handleMultiStepChange(true)} variant={isMultiStep === true ? "outlined" : "contained"}>Yes</Button>
                                <Button fullWidth color="secondary" onClick={() => handleMultiStepChange(false)} variant={isMultiStep === false ? "outlined" : "contained"}>No</Button>
                            </Stack >
                        </Grid >
                        {/* Data displayed only by multi-step routines */}
                        {
                            isMultiStep === true && (
                                <>
                                    {/* Dialog for building routine graph */}
                                    <LargeDialog
                                        id="build-routine-graph-dialog"
                                        onClose={handleGraphClose}
                                        isOpen={isGraphOpen}
                                        titleId=""
                                        zIndex={zIndex + 1300}
                                        sxs={{ paper: { display: "contents" } }}
                                    >
                                        <BuildView
                                            handleCancel={handleGraphClose}
                                            handleClose={handleGraphClose}
                                            handleSubmit={handleGraphSubmit}
                                            isEditing={true}
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
                                            zIndex={zIndex + 1300}
                                        />
                                    </LargeDialog>
                                    {/* Button to display graph */}
                                    <Grid item xs={12} mb={4}>
                                        <Button startIcon={<RoutineIcon />} fullWidth color="secondary" onClick={handleGraphOpen} variant="contained">View Graph</Button>
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
                                            zIndex={zIndex}
                                        />
                                    </Grid>
                                    <Grid item xs={12} mb={4}>
                                        <InputOutputContainer
                                            isEditing={true}
                                            handleUpdate={outputsHelpers.setValue as any}
                                            isInput={false}
                                            language={language}
                                            list={outputsField.value}
                                            zIndex={zIndex}
                                        />
                                    </Grid>
                                </>
                            )
                        }
                    </Stack>
                </Stack>
            </BaseForm>
            <GridSubmitButtons
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
