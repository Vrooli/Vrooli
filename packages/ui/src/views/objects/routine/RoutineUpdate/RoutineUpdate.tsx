import { Button, Dialog, Grid, Stack, Typography } from "@mui/material";
import { FindVersionInput, Node, NodeLink, ResourceList, RoutineVersion, RoutineVersionUpdateInput } from "@shared/consts";
import { RoutineIcon } from "@shared/icons";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { routineVersionTranslationValidation, routineVersionValidation } from '@shared/validation';
import { routineVersionFindOne } from "api/generated/endpoints/routineVersion_findOne";
import { routineVersionUpdate } from "api/generated/endpoints/routineVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { UpTransition } from "components/dialogs/transitions";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { initializeRoutineGraph } from "utils/runUtils";
import { SessionContext } from "utils/SessionContext";
import { getMinimumVersion } from "utils/shape/general";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { shapeRoutineVersion } from "utils/shape/models/routineVersion";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { TagShape } from "utils/shape/models/tag";
import { BuildView } from "views/BuildView/BuildView";
import { RoutineUpdateProps } from "../types";

const helpTextSubroutines = `A routine can be made from scratch (single-step), or by combining other routines (multi-step).\n\nA single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.\n\nA multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.`

export const RoutineUpdate = ({
    display = 'page',
    onCancel,
    onUpdated,
    zIndex = 200,
}: RoutineUpdateProps) => {
    const session = useContext(SessionContext);

    const { handleCancel, handleUpdated } = useUpdateActions<RoutineVersion>(display, onCancel, onUpdated);

    // Fetch existing data
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: routineVersion, loading }] = useCustomLazyQuery<RoutineVersion, FindVersionInput>(routineVersionFindOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (urlData.id || urlData.idRoot) getData({ variables: urlData });
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: 'Error' });
    }, [getData, urlData])

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
        setInputsList(routineVersion?.inputs ?? [] as RoutineVersionInputShape[]);
        setOutputsList(routineVersion?.outputs ?? [] as RoutineVersionOutputShape[]);
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
                versionLabel: routineVersion?.versionLabel ?? '1.0.0',
                versionNotes: '',
            }
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: routineVersionValidation.update({ minVersion: getMinimumVersion(routineVersion?.root?.versions ?? []) }),
        onSubmit: (values) => {
            if (!existing) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                return;
            }
            mutationWrapper<RoutineVersion, RoutineVersionUpdateInput>({
                mutation,
                input: shapeRoutineVersion.update(existing, values),
                onSuccess: (data) => { handleUpdated(data) },
                onError: () => { helpers.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        onTranslationBlur,
        onTranslationChange,
        setLanguage,
        translations,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['description', 'instructions', 'name'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: routineVersionTranslationValidation.update({}),
    });

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

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'UpdateRoutine',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipList
                            isEditing={true}
                            isFormDirty={formik.dirty}
                            objectType={'Routine'}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            translations={formik.values.translationsUpdate}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TranslatedTextField
                            fullWidth
                            label={t('Name')}
                            language={language}
                            name="name"
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TranslatedTextField
                            fullWidth
                            label={t('Description')}
                            language={language}
                            multiline
                            minRows={3}
                            name="description"
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
                            mutate={false}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TagSelector
                            handleTagsUpdate={handleTagsUpdate}
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
                                        handleUpdate={handleInputsUpdate as any}
                                        isInput={true}
                                        language={language}
                                        list={inputsList}
                                        zIndex={zIndex}
                                    />
                                </Grid>
                                <Grid item xs={12} mb={4}>
                                    <InputOutputContainer
                                        isEditing={true}
                                        handleUpdate={handleOutputsUpdate as any}
                                        isInput={false}
                                        language={language}
                                        list={outputsList}
                                        zIndex={zIndex}
                                    />
                                </Grid>
                            </>
                        )
                    }
                    <GridSubmitButtons
                        display={display}
                        errors={translations.errorsWithTranslations}
                        isCreate={false}
                        loading={formik.isSubmitting}
                        onCancel={handleCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </BaseForm>
        </>
    )
}