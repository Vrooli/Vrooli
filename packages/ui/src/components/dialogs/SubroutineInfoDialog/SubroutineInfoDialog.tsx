/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Box,
    Grid,
    IconButton,
    Stack,
    SwipeableDrawer,
    TextField,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { SubroutineInfoDialogProps } from '../types';
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, InputShape, ObjectType, OutputShape, removeTranslation, TagShape, updateArray, usePromptBeforeUnload } from 'utils';
import { routineTranslationUpdate, routineUpdate as validationSchema } from '@shared/validation';
import { ResourceListUsedFor } from '@shared/consts';
import { EditableTextCollapse, GridSubmitButtons, InputOutputContainer, LanguageInput, OwnerLabel, QuantityBox, RelationshipButtons, ResourceListHorizontal, TagList, TagSelector, userFromSession, VersionInput } from 'components';
import { useFormik } from 'formik';
import { NodeDataRoutineListItem, ResourceList } from 'types';
import { DUMMY_ID, uuid } from '@shared/uuid';
import { CloseIcon, OpenInNewIcon } from '@shared/icons';
import { RelationshipItemRoutine, RelationshipsObject } from 'components/inputs/types';

export const SubroutineInfoDialog = ({
    data,
    defaultLanguage,
    handleUpdate,
    handleReorder,
    handleViewFull,
    isEditing,
    open,
    session,
    onClose,
    zIndex,
}: SubroutineInfoDialogProps) => {
    const { palette } = useTheme();

    const subroutine = useMemo<NodeDataRoutineListItem | undefined>(() => {
        if (!data?.node || !data?.routineItemId) return undefined;
        return data.node.routines.find(r => r.id === data.routineItemId);
    }, [data]);

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
    const [resourceList, setResourceList] = useState<ResourceList>({
        __typename: 'ResourceList',
        id: uuid(),
        usedFor: ResourceListUsedFor.Display,
        resources: [],
    } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: subroutine?.routine?.isComplete ?? false,
            isPrivate: subroutine?.routine?.isPrivate ?? false,
            owner: subroutine?.routine?.owner ?? null,
            parent: null,
            // parent: subroutine?.routine?.parent ?? null, TODO
            project: null //TODO
        });
        setInputsList(subroutine?.routine?.inputs ?? []);
        setOutputsList(subroutine?.routine?.outputs ?? []);
        setResourceList(subroutine?.routine?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
        setTags(subroutine?.routine?.tags ?? []);
    }, [subroutine?.routine]);

    // Handle languages
    const [language, setLanguage] = useState<string>('');
    const [languages, setLanguages] = useState<string[]>([]);

    // Handle update
    const formik = useFormik({
        initialValues: {
            index: (subroutine?.index ?? 0) + 1,
            isComplete: subroutine?.routine?.isComplete ?? true,
            isInternal: subroutine?.routine?.isInternal ?? false,
            translationsUpdate: subroutine?.routine?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                instructions: '',
                title: '',
            }],
            version: subroutine?.routine?.version ?? '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: validationSchema({ minVersion: subroutine?.routine?.version ?? '0.0.1' }),
        onSubmit: (values) => {
            if (!subroutine) return;
            const resourceIndex = Array.isArray(subroutine?.routine?.resourceLists) ? (subroutine?.routine?.resourceLists?.findIndex(list => list.usedFor === ResourceListUsedFor.Display) ?? -1) : -1;
            const resourceListUpdate = resourceIndex > -1 ? {
                resourceLists: updateArray(
                    subroutine?.routine?.resourceLists ?? [],
                    resourceIndex,
                    resourceList,
                )
            } : {};
            handleUpdate({
                ...subroutine,
                index: Math.max(values.index - 1, 0), // Formik index starts at 1, for user convenience
                routine: {
                    ...subroutine.routine,
                    isInternal: values.isInternal,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    owner: relationships.owner,
                    parent: relationships.parent as RelationshipItemRoutine | null,
                    project: relationships.project,
                    version: values.version,
                    inputs: inputsList,
                    outputs: outputsList,
                    ...resourceListUpdate,
                    tags,
                    translations: values.translationsUpdate.map(t => ({
                        ...t,
                        id: t.id === DUMMY_ID ? uuid() : t.id,
                    })),
                }
            } as any);
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Current description, instructions, and title info, as well as errors
    const { description, instructions, title, errorDescription, errorInstructions, errorTitle, touchedDescription, touchedInstructions, touchedTitle, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            description: value?.description ?? '',
            instructions: value?.instructions ?? '',
            title: value?.title ?? '',
            errorDescription: error?.description ?? '',
            errorInstructions: error?.instructions ?? '',
            errorTitle: error?.title ?? '',
            touchedDescription: touched?.description ?? false,
            touchedInstructions: touched?.instructions ?? false,
            touchedTitle: touched?.title ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', routineTranslationUpdate),
        }
    }, [formik, language]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsUpdate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsUpdate', e, language)
    }, [formik, language]);

    // Handle languages
    useEffect(() => {
        if (languages.length === 0 && formik.values.translationsUpdate.length > 0) {
            setLanguage(formik.values.translationsUpdate[0].language);
            setLanguages(formik.values.translationsUpdate.map(t => t.language));
        }
    }, [formik, languages, setLanguage, setLanguages])
    const handleLanguageSelect = useCallback((newLanguage: string) => { setLanguage(newLanguage) }, []);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
        addEmptyTranslation(formik, 'translationsUpdate', newLanguage);
    }, [formik, handleLanguageSelect, languages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        setLanguages(newLanguages);
        removeTranslation(formik, 'translationsUpdate', language);
    }, [formik, languages]);

    const canEdit = useMemo<boolean>(() => isEditing && (subroutine?.routine?.isInternal || subroutine?.routine?.owner?.id === session.id || subroutine?.routine?.permissionsRoutine?.canEdit === true), [isEditing, session.id, subroutine?.routine?.isInternal, subroutine?.routine?.owner?.id, subroutine?.routine?.permissionsRoutine?.canEdit]);

    /**
     * Navigate to the subroutine's build page
     */
    const toGraph = useCallback(() => {
        handleViewFull();
    }, [handleViewFull]);

    /**
     * Before closing, update subroutine if changes were made
     */
    const handleClose = useCallback(() => {
        if (canEdit && formik.dirty) {
            formik.submitForm();
        } else {
            onClose();
        }
    }, [formik, canEdit, onClose]);

    /**
     * Close without saving
     */
    const handleCancel = useCallback(() => {
        formik.resetForm();
        onClose();
    }, [formik, onClose]);

    return (
        <SwipeableDrawer
            anchor="bottom"
            variant='persistent'
            open={open}
            onOpen={() => { }}
            onClose={handleClose}
            sx={{
                zIndex,
                '& .MuiDrawer-paper': {
                    background: palette.background.default,
                    borderTop: 'none',
                }
            }}
        >
            {/* Title bar with close icon */}
            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                alignItems: 'center',
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                padding: 1,
            }}>
                {/* Subroutine title and position */}
                <Typography variant="h5">{title}</Typography>
                <Typography variant="h6" ml={1}>{`(${(subroutine?.index ?? 0) + 1} of ${(data?.node?.routines?.length ?? 1)})`}</Typography>
                {/* Button to open in full page */}
                {!subroutine?.routine?.isInternal && (
                    <Tooltip title="Open in full page">
                        <IconButton onClick={toGraph}>
                            <OpenInNewIcon fill={palette.primary.contrastText} />
                        </IconButton>
                    </Tooltip>
                )}{/* Owned by and version */}
                <Stack direction="row" sx={{ marginLeft: 'auto' }}>
                    <OwnerLabel objectType={ObjectType.Routine} owner={subroutine?.routine?.owner} session={session} />
                    <Typography variant="body1">{subroutine?.routine?.version}</Typography>
                </Stack>
                {/* Close button */}
                <IconButton onClick={handleClose} sx={{
                    color: palette.primary.contrastText,
                    borderRadius: 0,
                    borderBottom: `1px solid ${palette.primary.dark}`,
                    justifyContent: 'end',
                }}>
                    <CloseIcon />
                </IconButton>
            </Box>
            {/* Main content */}
            <Box sx={{
                padding: 2,
                overflowY: 'auto',
            }}>
                <form onSubmit={formik.handleSubmit}>
                    {/* Position, description and instructions */}
                    <Grid container spacing={2}>
                        {/* owner, project, isPrivate, etc. */}
                        <Grid item xs={12}>
                            <RelationshipButtons
                                disabled={!isEditing}
                                isFormDirty={formik.dirty}
                                objectType={ObjectType.Routine}
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
                                handleDelete={handleLanguageDelete}
                                handleCurrent={handleLanguageSelect}
                                selectedLanguages={languages}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Grid>
                        {/* Position */}
                        <Grid item xs={12}>
                            {
                                canEdit && <Box sx={{
                                    marginBottom: 2,
                                }}>
                                    <Typography variant="h6">Order</Typography>
                                    <QuantityBox
                                        id="subroutine-position"
                                        disabled={!canEdit}
                                        label="Order"
                                        min={1}
                                        max={data?.node?.routines?.length ?? 1}
                                        tooltip="The order of this subroutine in its parent routine"
                                        value={formik.values.index}
                                        handleChange={(value: number) => {
                                            formik.setFieldValue('index', value);
                                            handleReorder(data?.node?.id ?? '', subroutine?.index ?? 0, value - 1);
                                        }}
                                    />
                                </Box>
                            }
                        </Grid>
                        {/* Title */}
                        {
                            canEdit && <Grid item xs={12}>
                                <TextField
                                    fullWidth
                                    id="title"
                                    name="title"
                                    label="title"
                                    value={title}
                                    onBlur={onTranslationBlur}
                                    onChange={onTranslationChange}
                                    error={touchedTitle && Boolean(errorTitle)}
                                    helperText={touchedTitle && errorTitle}
                                />
                            </Grid>
                        }
                        {/* Description */}
                        <Grid item xs={12} sm={6}>
                            <EditableTextCollapse
                                isEditing={canEdit}
                                propsTextField={{
                                    fullWidth: true,
                                    id: "description",
                                    name: "description",
                                    InputLabelProps: { shrink: true },
                                    value: description,
                                    multiline: true,
                                    maxRows: 3,
                                    onBlur: onTranslationBlur,
                                    onChange: onTranslationChange,
                                    error: touchedDescription && Boolean(errorDescription),
                                    helperText: touchedDescription && errorDescription,
                                }}
                                text={description}
                                title="Description"
                            />
                        </Grid>
                        {/* Instructions */}
                        <Grid item xs={12} sm={6}>
                            <EditableTextCollapse
                                isEditing={isEditing}
                                propsMarkdownInput={{
                                    id: "instructions",
                                    placeholder: "Instructions",
                                    value: instructions,
                                    minRows: 3,
                                    onChange: (newText: string) => onTranslationChange({ target: { name: 'instructions', value: newText }}),
                                    error: touchedInstructions && Boolean(errorInstructions),
                                    helperText: touchedInstructions ? errorInstructions as string : null,
                                }}
                                text={instructions}
                                title="Instructions"
                            />
                        </Grid>
                        {
                            canEdit && <Grid item xs={12}>
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
                        }
                        {/* Inputs */}
                        {(canEdit || (inputsList?.length > 0)) && <Grid item xs={12} sm={6}>
                            <InputOutputContainer
                                isEditing={canEdit}
                                handleUpdate={handleInputsUpdate}
                                isInput={true}
                                language={language}
                                list={inputsList}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Grid>}
                        {/* Outputs */}
                        {(canEdit || (outputsList?.length > 0)) && <Grid item xs={12} sm={6}>
                            <InputOutputContainer
                                isEditing={canEdit}
                                handleUpdate={handleOutputsUpdate}
                                isInput={false}
                                language={language}
                                list={outputsList}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Grid>}
                        {
                            (canEdit || (resourceList?.resources?.length > 0)) && <Grid item xs={12} mb={2}>
                                <ResourceListHorizontal
                                    title={'Resources'}
                                    list={resourceList}
                                    canEdit={canEdit}
                                    handleUpdate={handleResourcesUpdate}
                                    session={session}
                                    mutate={false}
                                    zIndex={zIndex}
                                />
                            </Grid>
                        }
                        <Grid item xs={12} marginBottom={4}>
                            {
                                canEdit ? <TagSelector
                                    handleTagsUpdate={handleTagsUpdate}
                                    session={session}
                                    tags={tags}
                                /> :
                                    <TagList session={session} parentId={''} tags={subroutine?.routine?.tags ?? []} />
                            }
                        </Grid>
                    </Grid>
                    {/* Save/Cancel buttons */}
                    {
                        canEdit && <Grid container spacing={1}>
                            <GridSubmitButtons
                                errors={errors}
                                isCreate={false}
                                loading={formik.isSubmitting}
                                onCancel={handleCancel}
                                onSetSubmitting={formik.setSubmitting}
                                onSubmit={formik.handleSubmit}
                            />
                        </Grid>
                    }
                </form>
            </Box>
        </SwipeableDrawer>
    );
}