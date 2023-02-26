import { Box, Button, CircularProgress, Dialog, Grid, Stack, TextField, Typography } from "@mui/material"
import { useCustomMutation, useCustomLazyQuery } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { RoutineUpdateProps } from "../types";
import { mutationWrapper } from 'api/utils';
import { routineVersionTranslationValidation, routineVersionValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, defaultResourceList, getMinimumVersion, getUserLanguages, handleTranslationBlur, handleTranslationChange, initializeRoutineGraph, NodeLinkShape, NodeShape, parseSingleItemUrl, PubSub, removeTranslation, RoutineVersionInputShape, RoutineVersionOutputShape, shapeRoutineVersion, TagShape, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { BuildView, GridSubmitButtons, HelpButton, LanguageInput, MarkdownInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, UpTransition, VersionInput } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipItemRoutineVersion, RelationshipsObject } from "components/inputs/types";
import { RoutineIcon } from "@shared/icons";
import { FindVersionInput, Node, NodeLink, ResourceList, RoutineVersion, RoutineVersionUpdateInput } from "@shared/consts";
import { routineVersionFindOne, routineVersionUpdate } from "api/generated/endpoints/routineVersion";

const helpTextSubroutines = `A routine can be made from scratch (single-step), or by combining other routines (multi-step).\n\nA single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.\n\nA multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.`

export const RoutineUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: RoutineUpdateProps) => {
    // Fetch existing data
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: routineVersion, loading }] = useCustomLazyQuery<RoutineVersion, FindVersionInput>(routineVersionFindOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (urlData.id || urlData.idRoot) getData({ variables: urlData });
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: 'Error' });
    }, [getData, urlData])

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle inputs
    const [inputsList, setInputsList] = useState<RoutineVersionInputShape[]>([]);
    const handleInputsUpdate = useCallback((updatedList: RoutineVersionInputShape[]) => {
        setInputsList(updatedList);
    }, [setInputsList]);

    // Handle outputs
    const [outputsList, setOutputsList] = useState<RoutineVersionOutputShape[]>([]);
    const handleOutputsUpdate = useCallback((updatedList: RoutineVersionOutputShape[]) => {
        setOutputsList(updatedList);
    }, [setOutputsList]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: routineVersion?.isComplete ?? false,
            isPrivate: routineVersion?.isPrivate ?? false,
            owner: routineVersion?.root?.owner ?? null,
            parent: null,
            // parent: routineVersion?.parent ?? null, TODO
            project: null, // TODO
        });
        setInputsList(routineVersion?.inputs ?? []);
        setOutputsList(routineVersion?.outputs ?? []);
        setResourceList(routineVersion?.resourceList ?? { id: uuid() } as any);
        setTags(routineVersion?.root?.tags ?? []);
    }, [routineVersion]);

    // Handle update
    const [mutation] = useCustomMutation<RoutineVersion, RoutineVersionUpdateInput>(routineVersionUpdate);
    const formik = useFormik({
        initialValues: {
            id: routineVersion?.id ?? DUMMY_ID,
            nodeLinks: routineVersion?.nodeLinks ?? [] as NodeLinkShape[],
            nodes: routineVersion?.nodes ?? [] as NodeShape[],
            translationsUpdate: routineVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                instructions: '',
                name: '',
            }],
            versionInfo: {
                versionIndex: routineVersion?.root?.versions?.length ?? 0,
                versionLabel: routineVersion?.versionLabel ?? '1.0.0',
                versionNotes: '',
            }
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: routineVersionValidation.update({ minVersion: getMinimumVersion(routineVersion?.root?.versions ?? []) }),
        onSubmit: (values) => {
            if (!routineVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadRoutine', severity: 'Error' });
                return;
            }
            mutationWrapper<RoutineVersion, RoutineVersionUpdateInput>({
                mutation,
                input: shapeRoutineVersion.update(routineVersion, {
                    id: routineVersion.id,
                    isComplete: relationships.isComplete,
                    isLatest: true,
                    isPrivate: relationships.isPrivate,
                    // project: relationships.project,
                    inputs: inputsList,
                    outputs: outputsList,
                    resourceList: resourceList,
                    root: {
                        id: routineVersion.root.id,
                        isPrivate: relationships.isPrivate,
                        owner: relationships.owner,
                        parent: relationships.parent as RelationshipItemRoutineVersion | null,
                        permissions: JSON.stringify({}),
                        tags: tags,
                    },
                    translations: values.translationsUpdate,
                    ...values.versionInfo,
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
        validationSchema: routineVersionTranslationValidation.update({}),
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
            const { nodes, nodeLinks } = initializeRoutineGraph(language, formik.values.id);
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
                    canUpdate={true}
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
                    versionInfo={formik.values.versionInfo}
                    versions={routineVersion?.root?.versions ?? []}
                    onBlur={formik.handleBlur}
                    onChange={(newVersionInfo) => {
                        formik.setFieldValue('versionInfo', newVersionInfo);
                        setRelationships({
                            ...relationships,
                            isComplete: false,
                        })
                    }}
                    error={formik.touched.versionInfo?.versionLabel && Boolean(formik.errors.versionInfo?.versionLabel)}
                    helperText={formik.touched.versionInfo?.versionLabel ? formik.errors.versionInfo?.versionLabel : null}
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
                                routineVersion={{
                                    id: formik.values.id,
                                    nodeLinks: formik.values.nodeLinks as NodeLink[],
                                    nodes: formik.values.nodes as Node[],
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
    ), [session, formik, onRelationshipsChange, relationships, zIndex, language, handleAddLanguage, handleDeleteLanguage, translations.name, translations.touchedName, translations.errorName, translations.description, translations.touchedDescription, translations.errorDescription, translations.instructions, translations.touchedInstructions, translations.errorInstructions, translations.errorsWithTranslations, onTranslationBlur, onTranslationChange, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, routineVersion?.root?.versions, isMultiStep, isGraphOpen, handleGraphClose, handleGraphSubmit, handleGraphOpen, handleInputsUpdate, inputsList, handleOutputsUpdate, outputsList, onCancel, handleMultiStepChange]);

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