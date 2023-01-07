import { Box, Button, CircularProgress, Dialog, Grid, Stack, TextField, Typography } from "@mui/material"
import { useMutation, useLazyQuery } from "graphql/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { RoutineUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils';
import { routineUpdate as validationSchema, routineVersionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, base36ToUuid, getLastUrlPart, getUserLanguages, handleTranslationBlur, handleTranslationChange, initializeRoutineGraph, InputShape, OutputShape, PubSub, removeTranslation, shapeRoutineVersion, TagShape, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { BuildView, GridSubmitButtons, HelpButton, LanguageInput, MarkdownInput, PageTitle, RelationshipButtons, ResourceListHorizontal, SnackSeverity, TagSelector, UpTransition, userFromSession, VersionInput } from "components";
import { DUMMY_ID, uuid, uuidValidate } from '@shared/uuid';
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipItemRoutineVersion, RelationshipsObject } from "components/inputs/types";
import { RoutineIcon } from "@shared/icons";
import { FindByIdInput, ResourceList, Routine, RoutineUpdateInput, RoutineVersion } from "@shared/consts";
import { routineEndpoint, routineVersionEndpoint } from "graphql/endpoints";

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
    const [getData, { data, loading }] = useLazyQuery<RoutineVersion, FindByIdInput, 'routineVersion'>(...routineVersionEndpoint.findOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (uuidValidate(id) || uuidValidate(versionGroupId)) getData({ variables: { id, versionGroupId } });
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: SnackSeverity.Error });
    }, [getData, id, versionGroupId])
    const routineVersion = useMemo(() => data?.routineVersion, [data]);

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
            isComplete: routineVersion?.isComplete ?? false,
            isPrivate: routineVersion?.isPrivate ?? false,
            owner: routineVersion?.owner ?? null,
            parent: null,
            // parent: routineVersion?.parent ?? null, TODO
            project: null, // TODO
        });
        setInputsList(routineVersion?.inputs ?? []);
        setOutputsList(routineVersion?.outputs ?? []);
        setResourceList(routineVersion?.resourceList ?? { id: uuid() } as any);
        setTags(routineVersion?.tags ?? []);
    }, [routineVersion]);

    // Handle update
    const [mutation] = useMutation<Routine, RoutineUpdateInput, 'routineUpdate'>(...routineEndpoint.update);
    const formik = useFormik({
        initialValues: {
            id: id ?? DUMMY_ID,
            nodeLinks: routineVersion?.nodeLinks ?? [] as RoutineVersion['nodeLinks'],
            nodes: routineVersion?.nodes ?? [] as RoutineVersion['nodes'],
            translationsUpdate: routineVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                instructions: '',
                name: '',
            }],
            version: routineVersion?.version ?? '1.0.0',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: validationSchema({ minVersion: routineVersion?.version ?? '0.0.1' }),
        onSubmit: (values) => {
            if (!routineVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadRoutine', severity: SnackSeverity.Error });
                return;
            }
            mutationWrapper<Routine, RoutineUpdateInput>({
                mutation,
                input: shapeRoutineVersion.update(routineVersion, {
                    id: routineVersion.id,
                    version: values.version,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    owner: relationships.owner,
                    parent: relationships.parent as RelationshipItemRoutineVersion | null,
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
    const translations = useTranslatedFields({
        fields: ['description', 'instructions', 'name'],
        formik, 
        formikField: 'translationsUpdate', 
        language, 
        validationSchema: routineVersionTranslationValidation.update(),
    });
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
    const handleGraphSubmit = useCallback(({ nodes, nodeLinks }: { nodes: RoutineVersion['nodes'], nodeLinks: RoutineVersion['nodeLinks'] }) => {
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
                    value={translations.name}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={translations.touchedName && Boolean(translations.errorName)}
                    helperText={translations.touchedName && translations.errorName}
                />
            </Grid>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="description"
                    name="description"
                    label="Description"
                    value={translations.description}
                    multiline
                    maxRows={3}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={translations.touchedDescription && Boolean(translations.errorDescription)}
                    helperText={translations.touchedDescription && translations.errorDescription}
                />
            </Grid>
            <Grid item xs={12} mb={4}>
                <MarkdownInput
                    id="instructions"
                    placeholder="Instructions"
                    value={translations.instructions}
                    minRows={4}
                    onChange={(newText: string) => onTranslationChange({ target: { name: 'instructions', value: newText } })}
                    error={translations.touchedInstructions && Boolean(translations.errorInstructions)}
                    helperText={translations.touchedInstructions ? translations.errorInstructions : null}
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
                errors={translations.errorsWithTranslations}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid >
    ), [formik, onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, onTranslationBlur, onTranslationChange, translations, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, isMultiStep, isGraphOpen, handleGraphClose, handleGraphSubmit, handleDeleteLanguage, handleGraphOpen, handleInputsUpdate, inputsList, handleOutputsUpdate, outputsList, onCancel, handleMultiStepChange]);

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