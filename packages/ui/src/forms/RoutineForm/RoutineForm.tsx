import { Button, Dialog, Grid, Stack, Typography } from "@mui/material";
import { Node, NodeLink, RoutineVersion } from "@shared/consts";
import { RoutineIcon } from "@shared/icons";
import { routineVersionTranslationValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { UpTransition } from "components/dialogs/transitions";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { RoutineFormProps } from "forms/types";
import { forwardRef, useCallback, useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { PubSub } from "utils/pubsub";
import { initializeRoutineGraph } from "utils/runUtils";
import { SessionContext } from "utils/SessionContext";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { BuildView } from "views/BuildView/BuildView";

const helpTextSubroutines = `A routine can be made from scratch (single-step), or by combining other routines (multi-step).\n\nA single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.\n\nA multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.`

export const RoutineForm = forwardRef<any, RoutineFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    versions,
    zIndex,
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
        fields: ['description', 'instructions', 'name'],
        validationSchema: routineVersionTranslationValidation.update({}),
    });

    const [idField] = useField<string>('id');
    const [nodesField, , nodesHelpers] = useField<NodeShape[]>('nodes');
    const [nodeLinksField, , nodeLinksHelpers] = useField<NodeLinkShape[]>('nodeLinks');
    const [inputsField, , inputsHelpers] = useField<RoutineVersionInputShape[]>('inputs');
    const [outputsField, , outputsHelpers] = useField<RoutineVersionOutputShape[]>('outputs');

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
    const handleGraphSubmit = useCallback(({ nodes, nodeLinks }: { nodes: RoutineVersion['nodes'], nodeLinks: RoutineVersion['nodeLinks'] }) => {
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
                messageKey: 'SubroutineGraphDeleteConfirm',
                buttons: [{
                    labelKey: 'Yes',
                    onClick: () => { setIsMultiStep(false); handleGraphClose(); }
                }, {
                    labelKey: 'Cancel',
                }]
            })
        }
        // If settings from false to true, check if any inputs or outputs have been added.
        // If so, prompt the user to confirm (these will be lost).
        else if (isMultiStep === true && (inputsField.value.length > 0 || outputsField.value.length > 0)) {
            PubSub.get().publishAlertDialog({
                messageKey: 'RoutineInputsDeleteConfirm',
                buttons: [{
                    labelKey: 'Yes',
                    onClick: () => { setIsMultiStep(true); handleGraphOpen(); }
                }, {
                    labelKey: 'Cancel',
                }]
            })
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
                    display: 'block',
                    maxWidth: '700px',
                    margin: 'auto',
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <RelationshipList
                        isEditing={true}
                        objectType={'Routine'}
                        zIndex={zIndex}
                    />
                    <Stack direction="column" spacing={2}>
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
                            label={t('Name')}
                            language={language}
                            name="name"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t('Description')}
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
                    </Stack>
                    <ResourceListHorizontalInput
                        isCreate={true}
                        zIndex={zIndex}
                    />
                    <TagSelector />
                    <VersionInput
                        fullWidth
                        versions={versions}
                    />
                    {/* Selector for single-step or multi-step routine */}
                    <Grid item xs={12} mb={isMultiStep === null ? 8 : 2}>
                        {/* Title with help text */}
                        <Stack direction="row" alignItems="center" justifyContent="center" spacing={1} mb={2} >
                            <Typography variant="h4" component="h4">Use Subroutines?</Typography>
                            <HelpButton markdown={helpTextSubroutines} />
                        </Stack >
                        {/* Yes/No buttons */}
                        <Stack direction="row" display="flex" alignItems="center" justifyContent="center" spacing={1} >
                            <Button fullWidth color="secondary" onClick={() => handleMultiStepChange(true)} variant={isMultiStep === true ? 'outlined' : 'contained'}>Yes</Button>
                            <Button fullWidth color="secondary" onClick={() => handleMultiStepChange(false)} variant={isMultiStep === false ? 'outlined' : 'contained'}>No</Button>
                        </Stack >
                    </Grid >
                    {/* Data displayed only by multi-step routines */}
                    {
                        isMultiStep === true && (
                            <>
                                {/* Dialog for building routine graph */}
                                <Dialog
                                    id="run-routine-view-dialog"
                                    fullScreen
                                    open={isGraphOpen}
                                    onClose={handleGraphClose}
                                    TransitionComponent={UpTransition}
                                    sx={{ zIndex: zIndex + 1 }}
                                >
                                    <BuildView
                                        handleCancel={handleGraphClose}
                                        handleClose={handleGraphClose}
                                        handleSubmit={handleGraphSubmit}
                                        isEditing={true}
                                        loading={false}
                                        owner={relationships.owner}
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
                                        zIndex={zIndex + 1}
                                    />
                                </Dialog>
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
                <GridSubmitButtons
                    display={display}
                    errors={{
                        ...props.errors,
                        ...translationErrors,
                    }}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    )
})