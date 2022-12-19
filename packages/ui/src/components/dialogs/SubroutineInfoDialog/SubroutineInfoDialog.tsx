/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Box,
    Grid,
    IconButton,
    SwipeableDrawer,
    TextField,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { SubroutineInfoDialogProps } from '../types';
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, InputShape, OutputShape, removeTranslation, TagShape, updateArray, usePromptBeforeUnload } from 'utils';
import { routineTranslationUpdate, routineUpdate as validationSchema } from '@shared/validation';
import { EditableTextCollapse, GridSubmitButtons, InputOutputContainer, LanguageInput, QuantityBox, RelationshipButtons, ResourceListHorizontal, SelectLanguageMenu, TagList, TagSelector, userFromSession, VersionDisplay, VersionInput } from 'components';
import { useFormik } from 'formik';
import { NodeDataRoutineListItem, ResourceList } from 'types';
import { DUMMY_ID, uuid } from '@shared/uuid';
import { CloseIcon, OpenInNewIcon } from '@shared/icons';
import { RelationshipItemRoutine, RelationshipsObject } from 'components/inputs/types';
import { getCurrentUser } from 'utils/authentication';

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
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

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
            isComplete: subroutine?.routineVersion?.isComplete ?? false,
            isPrivate: subroutine?.routineVersion?.isPrivate ?? false,
            owner: subroutine?.routineVersion?.owner ?? null,
            parent: null,
            // parent: subroutine?.routineVersion?.parent ?? null, TODO
            project: null //TODO
        });
        setInputsList(subroutine?.routineVersion?.inputs ?? []);
        setOutputsList(subroutine?.routineVersion?.outputs ?? []);
        setResourceList(subroutine?.routineVersion?.resourceList ?? { id: uuid() } as any);
        setTags(subroutine?.routineVersion?.tags ?? []);
    }, [subroutine?.routineVersion]);

    // Handle languages
    const [language, setLanguage] = useState<string>('');
    const [languages, setLanguages] = useState<string[]>([]);

    // Handle update
    const formik = useFormik({
        initialValues: {
            index: (subroutine?.index ?? 0) + 1,
            isComplete: subroutine?.routineVersion?.isComplete ?? true,
            isInternal: subroutine?.routineVersion?.isInternal ?? false,
            translationsUpdate: subroutine?.routineVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                instructions: '',
                name: '',
            }],
            version: subroutine?.routineVersion?.version ?? '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: validationSchema({ minVersion: subroutine?.routineVersion?.version ?? '0.0.1' }),
        onSubmit: (values) => {
            if (!subroutine) return;
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
                    resourceList: resourceList,
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

    // Current description, instructions, and name info, as well as errors
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

    const canEdit = useMemo<boolean>(() => isEditing && (subroutine?.routineVersion?.isInternal || subroutine?.routineVersion?.owner?.id === userId || subroutine?.routineVersion?.permissionsRoutine?.canEdit === true), [isEditing, subroutine?.routineVersion?.isInternal, subroutine?.routineVersion?.owner?.id, subroutine?.routineVersion?.permissionsRoutine?.canEdit, userId]);

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
                {/* Subroutine name and position */}
                <Typography variant="h5">{name}</Typography>
                <Typography variant="h6" ml={1} mr={1}>{`(${(subroutine?.index ?? 0) + 1} of ${(data?.node?.routines?.length ?? 1)})`}</Typography>
                {/* Version */}
                <VersionDisplay
                    currentVersion={subroutine?.routineVersion?.version}
                    prefix={" - "}
                    versions={subroutine?.routineVersion?.version ? [subroutine?.routineVersion?.version] : []} //TODO need to query versions
                />
                {/* Button to open in full page */}
                {!subroutine?.routineVersion?.isInternal && (
                    <Tooltip title="Open in full page">
                        <IconButton onClick={toGraph}>
                            <OpenInNewIcon fill={palette.primary.contrastText} />
                        </IconButton>
                    </Tooltip>
                )}
                {/* Close button */}
                <IconButton onClick={handleClose} sx={{
                    color: palette.primary.contrastText,
                    borderBottom: `1px solid ${palette.primary.dark}`,
                    justifyContent: 'end',
                    marginLeft: 'auto',
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
                                isEditing={isEditing}
                                isFormDirty={formik.dirty}
                                objectType={'Routine'}
                                onRelationshipsChange={onRelationshipsChange}
                                relationships={relationships}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            {canEdit ? <LanguageInput
                                currentLanguage={language}
                                handleAdd={handleAddLanguage}
                                handleDelete={handleLanguageDelete}
                                handleCurrent={setLanguage}
                                session={session}
                                translations={formik.values.translationsUpdate}
                                zIndex={zIndex}
                            /> : <SelectLanguageMenu
                                currentLanguage={language}
                                handleCurrent={setLanguage}
                                session={session}
                                translations={formik.values.translationsUpdate}
                                zIndex={zIndex}
                            />}
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
                        {/* Name */}
                        {
                            canEdit && <Grid item xs={12}>
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
                                    onChange: (newText: string) => onTranslationChange({ target: { name: 'instructions', value: newText } }),
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
                                    <TagList session={session} parentId={''} tags={subroutine?.routineVersion?.tags ?? []} />
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