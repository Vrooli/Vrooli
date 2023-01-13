import { Button, Checkbox, Dialog, FormControlLabel, Grid, Stack, TextField, Tooltip, Typography } from "@mui/material";
import { useMutation } from "graphql/hooks";
import { routineVersionTranslationValidation, routineVersionValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, getUserLanguages, handleTranslationBlur, handleTranslationChange, initializeRoutineGraph, NodeLinkShape, NodeShape, parseSearchParams, PubSub, removeTranslation, RoutineVersionInputShape, RoutineVersionOutputShape, shapeRoutineVersion, TagShape, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { RoutineCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BuildView, GridSubmitButtons, HelpButton, LanguageInput, MarkdownInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector, UpTransition, userFromSession, VersionInput } from "components";
import { uuid } from '@shared/uuid';
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipItemRoutineVersion, RelationshipsObject } from "components/inputs/types";
import { getCurrentUser } from "utils/authentication";
import { RoutineIcon } from "@shared/icons";
import { Node, NodeLink, ResourceList, RoutineVersion, RoutineVersionCreateInput } from "@shared/consts";
import { routineVersionEndpoint } from "graphql/endpoints";
import { mutationWrapper } from "graphql/utils";

const helpTextSubroutines = `A routine can be made from scratch (single-step), or by combining other routines (multi-step).

A single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.

A multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.`

export const RoutineCreate = ({
    isSubroutine = false,
    onCreated,
    onCancel,
    session,
    zIndex,
}: RoutineCreateProps) => {
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
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid() } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useMutation<RoutineVersion, RoutineVersionCreateInput, 'routineVersionCreate'>(...routineVersionEndpoint.create);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            nodeLinks: [] as NodeLinkShape[],
            nodes: [] as NodeShape[],
            root: {
                id: uuid(),
                isInternal: false,
            },
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                description: '',
                instructions: '',
                name: '',
            }],
            versionInfo: {
                versionIndex: 0,
                versionLabel: '1.0.0',
                versionNotes: '',
            }
        },
        validationSchema: routineVersionValidation.create(),
        onSubmit: (values) => {
            mutationWrapper<RoutineVersion, RoutineVersionCreateInput>({
                mutation,
                input: shapeRoutineVersion.create({
                    id: values.id,
                    isComplete: relationships.isComplete,
                    isLatest: true,
                    isPrivate: relationships.isPrivate,
                    nodeLinks: values.nodeLinks,
                    nodes: values.nodes,
                    // project: relationships.project,
                    inputs: inputsList,
                    outputs: outputsList,
                    resourceList: resourceList,
                    root: {
                        id: values.root.id,
                        isInternal: values.root.isInternal,
                        isPrivate: relationships.isPrivate,
                        owner: relationships.owner,
                        parent: relationships.parent as RelationshipItemRoutineVersion | null,
                        permissions: JSON.stringify({}),
                        tags: tags,
                    },
                    translations: values.translationsCreate,
                    ...values.versionInfo,
                }),
                onSuccess: (data) => { onCreated(data) },
                onError: () => { formik.setSubmitting(false) }
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });


    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const translations = useTranslatedFields({
        fields: ['description', 'instructions', 'name'],
        formik,
        formikField: 'translationsCreate',
        language,
        validationSchema: routineVersionTranslationValidation.create(),
    });
    const languages = useMemo(() => formik.values.translationsCreate.map(t => t.language), [formik.values.translationsCreate]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(formik, 'translationsCreate', newLanguage);
    }, [formik]);
    const handleDeleteLanguage = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        removeTranslation(formik, 'translationsCreate', language);
    }, [formik, languages]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsCreate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsCreate', e, language)
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

    const isLoggedIn = useMemo(() => Boolean(getCurrentUser(session).id), [session]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex,
        }}
        >
            <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                <Grid item xs={12}>
                    <PageTitle titleKey='CreateRoutine' session={session} />
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
                        translations={formik.values.translationsCreate}
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
                        label="description"
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
                        loading={false}
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
                        versions={[]}
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
                {/* Is internal checkbox */}
                {isSubroutine && (
                    <Grid item xs={12}>
                        <Tooltip placement={'top'} title='Indicates if this routine is meant to be a subroutine for only one other routine. If so, it will not appear in search resutls.'>
                            <FormControlLabel
                                label='Internal'
                                control={
                                    <Checkbox
                                        id='routine-info-dialog-is-internal'
                                        size="small"
                                        name='isInternal'
                                        color='secondary'
                                        checked={formik.values.root.isInternal}
                                        onChange={formik.handleChange}
                                    />
                                }
                            />
                        </Tooltip>
                    </Grid>
                )}
                {/* Selector for single-step or multi-step routine */}
                <Grid item xs={12} mb={isMultiStep === null ? 8 : 2}>
                    {/* Title with help text */}
                    <Stack direction="row" alignItems="center" justifyContent="center" spacing={1} mb={2}>
                        <Typography variant="h4" component="h4">Use Subroutines?</Typography>
                        <HelpButton markdown={helpTextSubroutines} />
                    </Stack>
                    {/* Yes/No buttons */}
                    <Stack direction="row" display="flex" alignItems="center" justifyContent="center" spacing={1}>
                        <Button fullWidth color="secondary" onClick={() => handleMultiStepChange(true)} variant={isMultiStep === true ? 'outlined' : 'contained'}>Yes</Button>
                        <Button fullWidth color="secondary" onClick={() => handleMultiStepChange(false)} variant={isMultiStep === false ? 'outlined' : 'contained'}>No</Button>
                    </Stack>
                </Grid>
                {/* Data displayed only by multi-step routines */}
                {isMultiStep === true && (
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
                                    translations: formik.values.translationsCreate as any[],
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
                )}
                {/* Data displayed only by single-step routines */}
                {isMultiStep === false && (
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
                )}
                <GridSubmitButtons
                    disabledSubmit={!isLoggedIn}
                    errors={translations.errorsWithTranslations}
                    isCreate={true}
                    loading={formik.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </Grid>
        </form>
    )
}