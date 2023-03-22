import { Button, Checkbox, Dialog, FormControlLabel, Grid, Stack, TextField, Tooltip, Typography } from "@mui/material";
import { Node, NodeLink, ResourceList, RoutineVersion, RoutineVersionCreateInput } from "@shared/consts";
import { RoutineIcon } from "@shared/icons";
import { parseSearchParams } from "@shared/route";
import { uuid } from '@shared/uuid';
import { routineVersionTranslationValidation, routineVersionValidation } from '@shared/validation';
import { routineVersionCreate } from "api/generated/endpoints/routineVersion_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { UpTransition } from "components/dialogs/transitions";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { RelationshipButtons } from "components/inputs/RelationshipButtons/RelationshipButtons";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { RelationshipItemRoutineVersion } from "components/inputs/types";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "components/lists/inputOutput";
import { ResourceListHorizontal } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { checkIfLoggedIn } from "utils/authentication/session";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getUserLanguages } from "utils/display/translationTools";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { PubSub } from "utils/pubsub";
import { initializeRoutineGraph } from "utils/runUtils";
import { SessionContext } from "utils/SessionContext";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { shapeRoutineVersion } from "utils/shape/models/routineVersion";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { TagShape } from "utils/shape/models/tag";
import { BuildView } from "views/BuildView/BuildView";
import { RoutineCreateProps } from "../types";

const helpTextSubroutines = `A routine can be made from scratch (single-step), or by combining other routines (multi-step).\n\nA single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.\n\nA multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.`

export const RoutineCreate = ({
    display = 'page',
    isSubroutine = false,
    zIndex = 200,
}: RoutineCreateProps) => {
    const session = useContext(SessionContext);

    const { onCancel, onCreated } = useCreateActions<RoutineVersion>();

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
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useCustomMutation<RoutineVersion, RoutineVersionCreateInput>(routineVersionCreate);
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
                versionLabel: '1.0.0',
                versionNotes: '',
            }
        },
        validationSchema: routineVersionValidation.create({}),
        onSubmit: (values) => {
            mutationWrapper<RoutineVersion, RoutineVersionCreateInput>({
                mutation,
                input: shapeRoutineVersion.create({
                    id: values.id,
                    isComplete: relationships.isComplete,
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
        formikField: 'translationsCreate',
        validationSchema: routineVersionTranslationValidation.create({}),
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

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateRoutine',
                }}
            />
            <BaseForm onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipButtons
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
                            canUpdate={true}
                            handleUpdate={handleResourcesUpdate}
                            loading={false}
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
                    )}
                    <GridSubmitButtons
                        disabledSubmit={!isLoggedIn}
                        display={display}
                        errors={translations.errorsWithTranslations}
                        isCreate={true}
                        loading={formik.isSubmitting}
                        onCancel={onCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </BaseForm>
        </>
    )
}