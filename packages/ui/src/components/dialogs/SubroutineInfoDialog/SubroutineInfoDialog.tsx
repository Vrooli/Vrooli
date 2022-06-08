/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    AccountTree as GraphIcon,
    Close as CloseIcon,
    Restore as RevertIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import {
    Box,
    Button,
    Checkbox,
    FormControlLabel,
    Grid,
    IconButton,
    Stack,
    SwipeableDrawer,
    TextField,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { useLocation } from 'wouter';
import { SubroutineInfoDialogProps } from '../types';
import { getOwnedByString, getTranslation, toOwnedBy, updateArray } from 'utils';
import Markdown from 'markdown-to-jsx';
import { ResourceListUsedFor, routineUpdateForm as validationSchema } from '@local/shared';
import { InputOutputContainer, LanguageInput, LinkButton, MarkdownInput, QuantityBox, ResourceListHorizontal, TagList, TagSelector, UserOrganizationSwitch } from 'components';
import { useFormik } from 'formik';
import { NewObject, NodeDataRoutineListItem, Organization, ResourceList, Routine, RoutineInputList, RoutineOutputList } from 'types';
import { owns } from 'utils/authentication';
import { v4 as uuidv4 } from 'uuid';
import { TagSelectorTag } from 'components/inputs/types';

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
    const [, setLocation] = useLocation();

    const subroutine = useMemo<NodeDataRoutineListItem | undefined>(() => {
        if (!data?.node || !data?.routineItemId) return undefined;
        return data.node.routines.find(r => r.id === data.routineItemId);
    }, [data]);

    // Handle user/organization switch
    const [organizationFor, setOrganizationFor] = useState<Organization | null>(null);
    const onSwitchChange = useCallback((organization: Organization | null) => { setOrganizationFor(organization) }, [setOrganizationFor]);

    // Handle inputs
    const [inputsList, setInputsList] = useState<RoutineInputList>([]);
    const handleInputsUpdate = useCallback((updatedList: RoutineInputList) => {
        setInputsList(updatedList);
    }, [setInputsList]);

    // Handle outputs
    const [outputsList, setOutputsList] = useState<RoutineOutputList>([]);
    const handleOutputsUpdate = useCallback((updatedList: RoutineOutputList) => {
        setOutputsList(updatedList);
    }, [setOutputsList]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuidv4(), usedFor: ResourceListUsedFor.Display } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagSelectorTag[]>([]);
    const addTag = useCallback((tag: TagSelectorTag) => {
        setTags(t => [...t, tag]);
    }, [setTags]);
    const removeTag = useCallback((tag: TagSelectorTag) => {
        setTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setTags]);
    const clearTags = useCallback(() => {
        setTags([]);
    }, [setTags]);

    // Handle translations
    type Translation = NewObject<Routine['translations'][0]>;
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
        if (subroutine?.routine?.owner?.__typename === 'Organization') setOrganizationFor(subroutine?.routine.owner as Organization);
        else setOrganizationFor(null);
        setInputsList(subroutine?.routine?.inputs ?? []);
        setOutputsList(subroutine?.routine?.outputs ?? []);
        setResourceList(subroutine?.routine?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuidv4(), usedFor: ResourceListUsedFor.Display } as any);
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
            index: subroutine?.index ?? 1,
            description: getTranslation(subroutine?.routine, 'description', [defaultLanguage]) ?? '',
            instructions: getTranslation(subroutine?.routine, 'instructions', [defaultLanguage]) ?? '',
            isComplete: subroutine?.routine?.isComplete ?? true,
            isInternal: subroutine?.routine?.isInternal ?? false,
            title: getTranslation(subroutine?.routine, 'title', [defaultLanguage]) ?? '',
            version: subroutine?.routine?.version ?? '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
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
            const ownedBy: { organizationId: string; } | { userId: string; } = organizationFor ? { organizationId: organizationFor.id } : { userId: session?.id ?? '' };
            const allTranslations = getTranslationsUpdate(language, {
                language,
                description: values.description,
                instructions: values.instructions,
                title: values.title,
            })
            handleUpdate({
                ...subroutine,
                index: Math.max(values.index - 1, 1), // Formik index starts at 1, for user convenience
                routine: {
                    ...subroutine.routine,
                    ...ownedBy,
                    isInternal: values.isInternal,
                    isComplete: values.isComplete,
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

    const ownedBy = useMemo<string | null>(() => getOwnedByString(subroutine?.routine, [language]), [subroutine, language]);
    const toOwner = useCallback(() => { toOwnedBy(subroutine?.routine, setLocation) }, [subroutine, setLocation]);
    const canEdit = useMemo<boolean>(() => isEditing && (subroutine?.routine?.isInternal || owns(subroutine?.routine?.role)), [isEditing, subroutine?.routine?.isInternal, subroutine?.routine?.role]);

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
                {/* Owned by and version */}
                <Stack direction="row" sx={{ marginLeft: 'auto' }}>
                    {ownedBy ? (
                        <LinkButton
                            onClick={toOwner}
                            text={`${ownedBy} - `}
                        />
                    ) : null}
                    <Typography variant="body1">{subroutine?.routine?.version}</Typography>
                </Stack>
                {/* Close button */}
                <IconButton onClick={handleClose} sx={{
                    color: palette.primary.contrastText,
                    borderRadius: 0,
                    borderBottom: `1px solid ${palette.primary.dark}`,
                    justifyContent: 'end',
                }}>
                    <CloseIcon fontSize="large" />
                </IconButton>
            </Box>
            {/* Main content */}
            <Box sx={{
                padding: 2,
                overflowY: 'auto',
            }}>
                <form onSubmit={formik.handleSubmit}>
                    {/* Position, description and instructions */}
                    <Grid container>
                        {/* Owner */}
                        <Grid item xs={12}>
                            <UserOrganizationSwitch
                                session={session}
                                selected={organizationFor}
                                onChange={onSwitchChange}
                                zIndex={zIndex}
                                disabled={!canEdit}
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
                                    padding: 1,
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
                            <Box sx={{
                                padding: 1,
                            }}>
                                <Typography variant="h6">Description</Typography>
                                {
                                    canEdit ? (
                                        <MarkdownInput
                                            id="description"
                                            placeholder="Description"
                                            value={formik.values.description}
                                            minRows={2}
                                            onChange={(newText: string) => formik.setFieldValue('description', newText)}
                                            error={formik.touched.description && Boolean(formik.errors.description)}
                                            helperText={formik.touched.description ? formik.errors.description as string : null}
                                        />
                                    ) : (
                                        <Markdown>{getTranslation(subroutine, 'description', [language]) ?? ''}</Markdown>
                                    )
                                }
                            </Box>
                        </Grid>
                        {/* Instructions */}
                        <Grid item xs={12} sm={6}>
                            <Box sx={{
                                padding: 1,
                            }}>
                                <Typography variant="h6">Instructions</Typography>
                                {
                                    canEdit ? (
                                        <MarkdownInput
                                            id="instructions"
                                            placeholder="Instructions"
                                            value={formik.values.instructions}
                                            minRows={2}
                                            onChange={(newText: string) => formik.setFieldValue('instructions', newText)}
                                            error={formik.touched.instructions && Boolean(formik.errors.instructions)}
                                            helperText={formik.touched.instructions ? formik.errors.instructions as string : null}
                                        />
                                    ) : (
                                        <Markdown>{getTranslation(subroutine, 'instructions', [language]) ?? ''}</Markdown>
                                    )
                                }
                            </Box>
                        </Grid>
                        {
                            isEditing && <Grid item xs={12}>
                                <TextField
                                    fullWidth
                                    id="version"
                                    name="version"
                                    label="version"
                                    value={formik.values.version}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.version && Boolean(formik.errors.version)}
                                    helperText={formik.touched.version && formik.errors.version}
                                />
                            </Grid>
                        }
                        {/* Inputs */}
                        {(canEdit || (inputsList?.length > 0)) && <Grid item xs={12} sm={6}>
                            <InputOutputContainer
                                isEditing={canEdit}
                                handleUpdate={handleInputsUpdate as (updatedList: RoutineInputList | RoutineOutputList) => void}
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
                                handleUpdate={handleOutputsUpdate as (updatedList: RoutineInputList | RoutineOutputList) => void}
                                isInput={false}
                                language={language}
                                list={outputsList}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Grid>}
                        {
                            (isEditing || (resourceList?.resources?.length > 0)) && <Grid item xs={12} mb={2}>
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
                        <Grid item xs={12}>
                            {
                                canEdit ? <TagSelector
                                    session={session}
                                    tags={tags}
                                    onTagAdd={addTag}
                                    onTagRemove={removeTag}
                                    onTagsClear={clearTags}
                                /> :
                                    <TagList session={session} parentId={''} tags={subroutine?.routine?.tags ?? []} />
                            }
                        </Grid>
                        <Grid item xs={12} marginBottom={4}>
                            <Tooltip placement={'top'} title='Is this routine ready for anyone to use?'>
                                <FormControlLabel
                                    label='Complete'
                                    disabled={!canEdit}
                                    control={
                                        <Checkbox
                                            id='routine-is-complete'
                                            name='isComplete'
                                            color='secondary'
                                            checked={formik.values.isComplete}
                                            onChange={formik.handleChange}
                                            onBlur={formik.handleBlur}
                                        />
                                    }
                                />
                            </Tooltip>
                        </Grid>
                    </Grid>
                    {/* Save/Cancel buttons */}
                    {
                        canEdit && <Grid container spacing={1}>
                            <Grid item xs={6}>
                                <Button
                                    fullWidth
                                    startIcon={<SaveIcon />}
                                    disabled={!formik.isValid || formik.isSubmitting}
                                    type="submit"
                                >
                                    Save
                                </Button>
                            </Grid>
                            <Grid item xs={6}>
                                <Button
                                    fullWidth
                                    startIcon={<RevertIcon />}
                                    disabled={formik.isSubmitting}
                                    onClick={handleCancel}
                                >
                                    Cancel
                                </Button>
                            </Grid>
                        </Grid>
                    }
                </form>
            </Box>
            {/* Bottom nav container */}

            {/* If subroutine has its own subroutines, display button to switch to that graph */}
            {(subroutine as any)?.nodesCount > 0 && (
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'row',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: 1,
                    background: palette.primary.dark,
                }}>
                    <Button
                        color="secondary"
                        startIcon={<GraphIcon />}
                        onClick={toGraph}
                        sx={{
                            marginLeft: 'auto'
                        }}
                    >View Graph</Button>
                </Box>
            )}
        </SwipeableDrawer>
    );
}