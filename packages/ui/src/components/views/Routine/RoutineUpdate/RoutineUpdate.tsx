import { Box, Button, CircularProgress, Dialog, Grid, Stack, TextField, Typography } from "@mui/material"
import { useMutation, useLazyQuery } from "graphql/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { RoutineUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils';
import { routineTranslationUpdate, routineUpdate as validationSchema, routineVersionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, base36ToUuid, getFormikErrorsWithTranslations, getLastUrlPart, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, initializeRoutineGraph, InputShape, OutputShape, PubSub, removeTranslation, shapeRoutineUpdate, TagShape, usePromptBeforeUnload } from "utils";
import { BuildView, GridSubmitButtons, HelpButton, LanguageInput, MarkdownInput, PageTitle, RelationshipButtons, ResourceListHorizontal, SnackSeverity, TagSelector, UpTransition, userFromSession, VersionInput } from "components";
import { DUMMY_ID, uuid, uuidValidate } from '@shared/uuid';
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipItemRoutine, RelationshipsObject } from "components/inputs/types";
import { RoutineIcon } from "@shared/icons";
import { FindByIdInput, ResourceList, Routine, RoutineUpdateInput } from "@shared/consts";
import { routineEndpoint } from "graphql/endpoints";

const helpTextSubroutines = `A routine can be made from scratch (single-step), or by combining other routines (multi-step).

A single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.

A multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.`

export const RoutineUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: RoutineUpdateProps) => {
    // Fetch existing data
    const { id, versionGroupId } = useMemo(() => {
        // URL is /object/:versionGroupId/?:id
        const last = base36ToUuid(getLastUrlPart(0), false);
        const secondLast = base36ToUuid(getLastUrlPart(1), false);
        return {
            id: uuidValidate(secondLast) ? last : undefined,
            versionGroupId: uuidValidate(secondLast) ? secondLast : last,
        }
    }, []);
    const [getData, { data, loading }] = useLazyQuery<Routine, FindByIdInput, 'routine'>(...routineEndpoint.findOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (uuidValidate(id) || uuidValidate(versionGroupId)) getData({ variables: { id, versionGroupId } });
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: SnackSeverity.Error });
    }, [getData, id, versionGroupId])
    const routine = useMemo(() => data?.routine, [data]);

    const [relationships, setRelationships] = useState<RelationshipsObject>({
        isComplete: false,
        isPrivate: false,
        owner: userFromSession(session),
        parent: null,
        project: null,
    });
    const onRelationshipsChange = useCallback((newRelationshipsObject: Partial<RelationshipsObject>) => {
        setRelationships({
            ...relationships,
            ...newRelationshipsObject,
        });
    }, [relationships]);

    // Handle inputs
    const [inputsList, setInputsList] = useState<InputShape[]>([]);
    const handleInputsUpdate = useCallback((updatedList: InputShape[]) => {
        setInputsList(updatedList);
    }, [setInputsList]);

    // Handle outputs
    const [outputsList, setOutputsList] = useState<OutputShape[]>([]);
    const handleOutputsUpdate = useCallback((updatedList: OutputShape[]) => {
        setOutputsList(updatedList);
    }, [setOutputsList]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid() } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: routine?.isComplete ?? false,
            isPrivate: routine?.isPrivate ?? false,
            owner: routine?.owner ?? null,
            parent: null,
            // parent: routine?.parent ?? null, TODO
            project: null, // TODO
        });
        setInputsList(routine?.inputs ?? []);
        setOutputsList(routine?.outputs ?? []);
        setResourceList(routine?.resourceList ?? { id: uuid() } as any);
        setTags(routine?.tags ?? []);
    }, [routine]);

    // Handle update
    const [mutation] = useMutation<Routine, RoutineUpdateInput, 'routineUpdate'>(...routineEndpoint.update);
    const formik = useFormik({
        initialValues: {
            id: id ?? DUMMY_ID,
            nodeLinks: routine?.nodeLinks ?? [] as Routine['nodeLinks'],
            nodes: routine?.nodes ?? [] as Routine['nodes'],
            translationsUpdate: routine?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                instructions: '',
                name: '',
            }],
            version: routine?.version ?? '1.0.0',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: validationSchema({ minVersion: routine?.version ?? '0.0.1' }),
        onSubmit: (values) => {
            if (!routine) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadRoutine', severity: SnackSeverity.Error });
                return;
            }
            mutationWrapper<Routine, RoutineUpdateInput>({
                mutation,
                input: shapeRoutineUpdate(routine, {
                    id: routine.id,
                    version: values.version,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    owner: relationships.owner,
                    parent: relationships.parent as RelationshipItemRoutine | null,
                    project: relationships.project,
                    inputs: inputsList,
                    outputs: outputsList,
                    resourceLists: [resourceList],
                    tags: tags,
                    translations: values.translationsUpdate.map(t => ({
                        ...t,
                        id: t.id === DUMMY_ID ? uuid() : t.id,
                    })),
                }),
                onSuccess: (data) => { onUpdated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const { description, instructions, name, errorDescription, errorInstructions, errorName, touchedDescription, touchedInstructions, touchedName, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            description: value?.description ?? '',
            instructions: value?.instructions ?? '',
            name: value?.name ?? '',
            errorDescription: error?.description ?? '',
            errorInstructions: error?.instructions ?? '',
            errorName: error?.name ?? '',
            touchedDescription: touched?.description ?? false,
            touchedInstructions: touched?.instructions ?? false,
            touchedName: touched?.name ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', routineVersionTranslationValidation.update!()),
        }
    }, [formik, language]);
    const languages = useMemo(() => formik.values.translationsUpdate.map(t => t.language), [formik.values.translationsUpdate]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(formik, 'translationsUpdate', newLanguage);
    }, [formik]);
    const handleDeleteLanguage = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        removeTranslation(formik, 'translationsUpdate', language);
    }, [formik, languages]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsUpdate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsUpdate', e, language)
    }, [formik, language]);

    const [isGraphOpen, setIsGraphOpen] = useState(false);
    const handleGraphOpen = useCallback(() => {
        // Create initial nodes/links, if not already created
        if (formik.values.nodes.length === 0 && formik.values.nodeLinks.length === 0) {
            const { nodes, nodeLinks } = initializeRoutineGraph(language);
            formik.setValues({
                ...formik.values,
                nodes,
                nodeLinks,
            });
        }
        setIsGraphOpen(true);
    }, [formik, language]);
    const handleGraphClose = useCallback(() => { setIsGraphOpen(false); }, [setIsGraphOpen]);
    const handleGraphSubmit = useCallback(({ nodes, nodeLinks }: { nodes: Routine['nodes'], nodeLinks: Routine['nodeLinks'] }) => {
        formik.setFieldValue('nodes', nodes);
        formik.setFieldValue('nodeLinks', nodeLinks);
        setIsGraphOpen(false);
    }, [formik]);

    // You can use this component to create both single-step and multi-step routines.
    // The beginning of the form is information shared between both types of routines.
    const [isMultiStep, setIsMultiStep] = useState<boolean | null>(null);
    useEffect(() => { setIsMultiStep(formik.values.nodes.length > 0); }, [formik.values.nodes]);
    const handleMultiStepChange = useCallback((isMultiStep: boolean) => {
        // If setting from true to false, check if any nodes or nodeLinks have been added. 
        // If so, prompt the user to confirm (these will be lost).
        if (isMultiStep === false && (formik.values.nodes.length > 0 || formik.values.nodeLinks.length > 0)) {
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
        else if (isMultiStep === true && (inputsList.length > 0 || outputsList.length > 0)) {
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
    }, [formik.values.nodes.length, formik.values.nodeLinks.length, inputsList.length, outputsList.length, handleGraphClose, handleGraphOpen]);

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <PageTitle titleKey='UpdateRoutine' session={session} />
            </Grid>
            <Grid item xs={12} mb={4}>
                <RelationshipButtons
                    isEditing={true}
                    isFormDirty={formik.dirty}
                    objectType={'Routine'}
                    onRelationshipsChange={onRelationshipsChange}
                    relationships={relationships}
                    session={session}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12}>
                <LanguageInput
                    currentLanguage={language}
                    handleAdd={handleAddLanguage}
                    handleDelete={handleDeleteLanguage}
                    handleCurrent={setLanguage}
                    session={session}
                    translations={formik.values.translationsUpdate}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="name"
                    name="name"
                    label="Name"
                    value={name}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={touchedName && Boolean(errorName)}
                    helperText={touchedName && errorName}
                />
            </Grid>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="description"
                    name="description"
                    label="Description"
                    value={description}
                    multiline
                    maxRows={3}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={touchedDescription && Boolean(errorDescription)}
                    helperText={touchedDescription && errorDescription}
                />
            </Grid>
            <Grid item xs={12} mb={4}>
                <MarkdownInput
                    id="instructions"
                    placeholder="Instructions"
                    value={instructions}
                    minRows={4}
                    onChange={(newText: string) => onTranslationChange({ target: { name: 'instructions', value: newText } })}
                    error={touchedInstructions && Boolean(errorInstructions)}
                    helperText={touchedInstructions ? errorInstructions : null}
                />
            </Grid>
            <Grid item xs={12}>
                <ResourceListHorizontal
                    title={'Resources'}
                    list={resourceList}
                    canEdit={true}
                    handleUpdate={handleResourcesUpdate}
                    loading={loading}
                    session={session}
                    mutate={false}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12}>
                <TagSelector
                    handleTagsUpdate={handleTagsUpdate}
                    session={session}
                    tags={tags}
                />
            </Grid>
            <Grid item xs={12} mb={4}>
                <VersionInput
                    fullWidth
                    id="version"
                    name="version"
                    value={formik.values.version}
                    onBlur={formik.handleBlur}
                    onChange={(newVersion: string) => {
                        formik.setFieldValue('version', newVersion);
                        setRelationships({
                            ...relationships,
                            isComplete: false,
                        })
                    }}
                    error={formik.touched.version && Boolean(formik.errors.version)}
                    helperText={formik.touched.version ? formik.errors.version : null}
                />
            </Grid>
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
                                routine={{
                                    id: formik.values.id,
                                    nodeLinks: formik.values.nodeLinks,
                                    nodes: formik.values.nodes,
                                }}
                                translationData={{
                                    language,
                                    setLanguage,
                                    handleAddLanguage,
                                    handleDeleteLanguage,
                                    translations: formik.values.translationsUpdate as any[],
                                }}
                                session={session}
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
                                handleUpdate={handleInputsUpdate}
                                isInput={true}
                                language={language}
                                list={inputsList}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Grid>
                        <Grid item xs={12} mb={4}>
                            <InputOutputContainer
                                isEditing={true}
                                handleUpdate={handleOutputsUpdate}
                                isInput={false}
                                language={language}
                                list={outputsList}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Grid>
                    </>
                )
            }
            <GridSubmitButtons
                errors={errors}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid >
    ), [formik, onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, name, onTranslationBlur, onTranslationChange, touchedName, errorName, description, touchedDescription, errorDescription, instructions, touchedInstructions, errorInstructions, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, isMultiStep, isGraphOpen, handleGraphClose, handleGraphSubmit, handleDeleteLanguage, handleGraphOpen, handleInputsUpdate, inputsList, handleOutputsUpdate, outputsList, errors, onCancel, handleMultiStepChange]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex,
        }}
        >
            {loading ? (
                <Box sx={{
                    position: 'absolute',
                    top: '-5vh', // Half of toolbar height
                    width: '100%',
                    height: '100%',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                    <CircularProgress size={100} color="secondary" />
                </Box>
            ) : formInput}
        </form>
    )
}