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
import { getTranslation, InputShape, ObjectType, OutputShape, RoutineTranslationShape, TagShape, updateArray } from 'utils';
import { routineUpdateForm as validationSchema } from '@shared/validation';
import { ResourceListUsedFor } from '@shared/consts';
import { EditableTextCollapse, GridSubmitButtons, InputOutputContainer, LanguageInput, OwnerLabel, QuantityBox, RelationshipButtons, ResourceListHorizontal, TagList, TagSelector, userFromSession, VersionInput } from 'components';
import { useFormik } from 'formik';
import { NodeDataRoutineListItem, ResourceList } from 'types';
import { v4 as uuid } from 'uuid';
import { CloseIcon } from '@shared/icons';
import { RelationshipItemRoutine, RelationshipsObject } from 'components/inputs/types';
import { OpenInNew as OpenInNewIcon } from '@mui/icons-material';

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

    // Handle translations
    type Translation = RoutineTranslationShape;
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        setTranslations([...translations.filter(t => t.language !== language)]);
        // Also delete translations from inputs and outputs
        setInputsList(inputsList.map(i => {
            const updatedTranslationsList = i.translations.filter(t => t.language !== language);
            return { ...i, translations: updatedTranslationsList };
        }));
        setOutputsList(outputsList.map(o => {
            const updatedTranslationsList = o.translations.filter(t => t.language !== language);
            return { ...o, translations: updatedTranslationsList };
        }));
    }, [translations, inputsList, outputsList]);
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(translations, index, translation) : [...translations, translation];
    }, [translations]);
    const updateTranslation = useCallback((language: string, translation: Translation) => {
        setTranslations(getTranslationsUpdate(language, translation));
    }, [getTranslationsUpdate]);

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
        setTranslations(subroutine?.routine?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            description: t.description ?? '',
            instructions: t.instructions ?? '',
            title: t.title ?? '',
        })) ?? []);
    }, [subroutine?.routine]);

    // Handle update
    const formik = useFormik({
        initialValues: {
            index: (subroutine?.index ?? 0) + 1,
            description: getTranslation(subroutine?.routine, 'description', [defaultLanguage]) ?? '',
            instructions: getTranslation(subroutine?.routine, 'instructions', [defaultLanguage]) ?? '',
            isComplete: subroutine?.routine?.isComplete ?? true,
            isInternal: subroutine?.routine?.isInternal ?? false,
            title: getTranslation(subroutine?.routine, 'title', [defaultLanguage]) ?? '',
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
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                instructions: values.instructions,
                title: values.title,
            })
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
                    translations: allTranslations,
                }
            } as any);
        },
    });

    // Handle languages
    const [language, setLanguage] = useState<string>(defaultLanguage);
    const [languages, setLanguages] = useState<string[]>([]);
    useEffect(() => {
        if (languages.length === 0 && translations.length > 0) {
            setLanguage(translations[0].language);
            setLanguages(translations.map(t => t.language));
            formik.setValues({
                ...formik.values,
                description: translations[0].description ?? '',
                instructions: translations[0].instructions,
                title: translations[0].title,
            })
        }
    }, [formik, languages, setLanguage, setLanguages, translations])
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
            instructions: existingTranslation?.instructions ?? '',
            title: existingTranslation?.title ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            instructions: formik.values.instructions,
            title: formik.values.title,
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, formik.values.instructions, formik.values.title, updateFormikTranslation]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
    }, [handleLanguageSelect, languages, setLanguages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        deleteTranslation(language);
        updateFormikTranslation(newLanguages[0]);
        setLanguage(newLanguages[0]);
        setLanguages(newLanguages);
    }, [deleteTranslation, languages, updateFormikTranslation]);

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
                <Typography variant="h5">{formik.values.title}</Typography>
                <Typography variant="h6" ml={1}>{`(${(subroutine?.index ?? 0) + 1} of ${(data?.node?.routines?.length ?? 1)})`}</Typography>
                {/* Button to open in full page */}
                {!subroutine?.routine?.isInternal && (
                    <Tooltip title="Open in full page">
                        <IconButton onClick={toGraph}>
                            <OpenInNewIcon sx={{ fill: palette.primary.contrastText }} />
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
                                    value={formik.values.title}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.title && Boolean(formik.errors.title)}
                                    helperText={formik.touched.title && formik.errors.title}
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
                                    value: formik.values.description,
                                    multiline: true,
                                    maxRows: 3,
                                    onBlur: formik.handleBlur,
                                    onChange: formik.handleChange,
                                    error: formik.touched.description && Boolean(formik.errors.description),
                                    helperText: formik.touched.description ? formik.errors.description : null,
                                }}
                                text={formik.values.description}
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
                                    value: formik.values.instructions,
                                    minRows: 3,
                                    onChange: (newText: string) => formik.setFieldValue('instructions', newText),
                                    error: formik.touched.instructions && Boolean(formik.errors.instructions),
                                    helperText: formik.touched.instructions ? formik.errors.instructions as string : null,
                                }}
                                text={formik.values.instructions}
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
                                disabledCancel={formik.isSubmitting}
                                disabledSubmit={formik.isSubmitting || !formik.isValid}
                                errors={formik.errors}
                                isCreate={false}
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