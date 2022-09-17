/**
 * Drawer to display overall routine info on the build page. 
 * Swipes left from right of screen
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    FileCopy as CopyIcon,
    Delete as DeleteIcon,
    ForkRight as ForkIcon,
    QueryStats as StatsIcon,
    StarOutline as StarIcon,
    Star as UnstarIcon,
} from '@mui/icons-material';
import {
    Box,
    Checkbox,
    FormControlLabel,
    IconButton,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    Stack,
    SwipeableDrawer,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { copy, copyVariables } from 'graphql/generated/copy';
import { fork, forkVariables } from 'graphql/generated/fork';
import { star, starVariables } from 'graphql/generated/star';
import { vote, voteVariables } from 'graphql/generated/vote';
import { ObjectAction, BuildInfoDialogProps } from '../types';
import { DeleteDialog, EditableLabel, EditableTextCollapse, LanguageInput, OwnerLabel, RelationshipButtons, ResourceListHorizontal, TagList, TagSelector, userFromSession, VersionInput } from 'components';
import { AllLanguages, getLanguageSubtag, getTranslation, ObjectType, PubSub } from 'utils';
import { useLocation } from '@shared/route';
import { APP_LINKS, CopyType, DeleteOneType, ForkType, StarFor, VoteFor } from '@shared/consts';
import { SelectLanguageMenu } from '../SelectLanguageMenu/SelectLanguageMenu';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils';
import { copyMutation, forkMutation, starMutation, voteMutation } from 'graphql/mutation';
import { v4 as uuid } from 'uuid';
import { CloseIcon, DownvoteWideIcon, InfoIcon, UpvoteWideIcon } from '@shared/icons';
import { requiredErrorMessage, title as titleValidation } from '@shared/validation';

export const BuildInfoDialog = ({
    formik,
    handleAction,
    handleLanguageChange,
    handleRelationshipsChange,
    handleResourcesUpdate,
    handleTagsUpdate,
    handleTranslationDelete,
    handleTranslationUpdate,
    handleUpdate,
    isEditing,
    language,
    loading,
    relationships,
    routine,
    session,
    sxs,
    tags,
    translations,
    zIndex,
}: BuildInfoDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    console.log('buildinfodialog renderrr')

    // Handle languages
    const availableLanguages = useMemo<string[]>(() => {
        if (isEditing) return Object.keys(AllLanguages);
        return routine?.translations?.map(t => getLanguageSubtag(t.language)) ?? [];
    }, [isEditing, routine?.translations]);
    const [languages, setLanguages] = useState<string[]>([]);

    // useEffect(() => {
    //     if (languages.length === 0 && translations.length > 0) {
    //         // setLanguage(translations[0].language);
    //         setLanguages(translations.map(t => t.language));
    //         console.log('buildinfodialog updating formik translation 1')
    //         formik.setValues({
    //             ...formik.values,
    //             description: translations[0].description ?? '',
    //             instructions: translations[0].instructions ?? '',
    //             title: translations[0].title ?? '',
    //         })
    //     }
    // }, [formik, languages, setLanguages, translations])

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
        handleTranslationUpdate(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            instructions: formik.values.instructions,
            title: formik.values.title,
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        handleLanguageChange(newLanguage);
    }, [formik.values.description, formik.values.instructions, formik.values.title, handleLanguageChange, handleTranslationUpdate, language, updateFormikTranslation]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
    }, [handleLanguageSelect, languages, setLanguages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        handleTranslationDelete(language);
        updateFormikTranslation(newLanguages[0]);
        handleLanguageChange(newLanguages[0]);
        setLanguages(newLanguages);
    }, [handleTranslationDelete, handleLanguageChange, languages, updateFormikTranslation]);

    const updateRoutineTitle = useCallback((title: string) => {
        if (!routine) return;
        handleTranslationUpdate(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            instructions: formik.values.instructions,
            title: title,
        })
        formik.setFieldValue('title', title);
    }, [formik, language, routine, handleTranslationUpdate]);

    // Open boolean for drawer
    const [open, setOpen] = useState(false);
    const toggleOpen = () => { setOpen(o => !o) };
    const closeMenu = () => { setOpen(false) };

    // TODO doesn't really work. resource list query returns empty resources inside, 
    // and this won't display editor if there are no resources yet. 
    // Ideally this should create a new resource list if none exists
    const resourceListObject = useMemo(() => {
        if (!routine ||
            !Array.isArray(routine.resourceLists) ||
            routine.resourceLists.length < 1 ||
            !Array.isArray(routine.resourceLists[0].resources) ||
            routine.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={routine.resourceLists[0]}
            canEdit={false}
            handleUpdate={handleResourcesUpdate}
            loading={loading}
            session={session}
            zIndex={zIndex}
        />
    }, [handleResourcesUpdate, loading, routine, session, zIndex]);

    /**
     * Determines which action buttons to display
     */
    const actions = useMemo(() => {
        // [value, label, icon, secondaryLabel]
        const results: [ObjectAction, string, any, string | null][] = [];
        // If signed in and not editing, show vote/star options
        if (session?.isLoggedIn === true && !isEditing) {
            results.push(routine?.isUpvoted ?
                [ObjectAction.VoteDown, 'Downvote', DownvoteWideIcon, null] :
                [ObjectAction.VoteUp, 'Upvote', UpvoteWideIcon, null]
            );
            results.push(routine?.isStarred ?
                [ObjectAction.StarUndo, 'Unstar', UnstarIcon, null] :
                [ObjectAction.Star, 'Star', StarIcon, null]
            );
        }
        // If not editing, show "Stats" and "Fork" buttons
        if (!isEditing) {
            results.push(
                [ObjectAction.Stats, 'Stats', StatsIcon, 'Coming Soon'],
                [ObjectAction.Copy, 'Copy', CopyIcon, null],
                [ObjectAction.Fork, 'Fork', ForkIcon, null],
            )
        }
        // Only show "Delete" when editing an existing routine
        if (isEditing && Boolean(routine?.id)) {
            results.push(
                [ObjectAction.Delete, 'Delete', DeleteIcon, null],
            )
        }
        return results;
    }, [isEditing, routine?.id, routine?.isStarred, routine?.isUpvoted, session]);

    // Handle delete
    const [deleteOpen, setDeleteOpen] = useState(false);
    const openDelete = useCallback(() => setDeleteOpen(true), []);
    const handleDeleteClose = useCallback((wasDeleted: boolean) => {
        if (wasDeleted) setLocation(APP_LINKS.Home);
        else setDeleteOpen(false);
    }, [setLocation])

    // Mutations
    const [copy] = useMutation<copy, copyVariables>(copyMutation);
    const [fork] = useMutation<fork, forkVariables>(forkMutation);
    const [star] = useMutation<star, starVariables>(starMutation);
    const [vote] = useMutation<vote, voteVariables>(voteMutation);

    const handleCopy = useCallback(() => {
        if (!routine?.id) return;
        mutationWrapper({
            mutation: copy,
            input: { id: routine.id, objectType: CopyType.Routine },
            onSuccess: ({ data }) => {
                PubSub.get().publishSnack({ message: `${getTranslation(routine, 'title', [language], true)} copied.`, severity: 'success' });
                handleAction(ObjectAction.Copy, data);
            },
        })
    }, [copy, handleAction, language, routine]);

    const handleFork = useCallback(() => {
        if (!routine?.id) return;
        mutationWrapper({
            mutation: fork,
            input: { id: routine.id, objectType: ForkType.Routine },
            onSuccess: ({ data }) => {
                PubSub.get().publishSnack({ message: `${getTranslation(routine, 'title', [language], true)} forked.`, severity: 'success' });
                handleAction(ObjectAction.Fork, data);
            }
        })
    }, [fork, handleAction, language, routine]);

    const handleStar = useCallback((isStar: boolean) => {
        if (!routine?.id) return;
        mutationWrapper({
            mutation: star,
            input: { isStar, forId: routine.id, starFor: StarFor.Routine },
            onSuccess: ({ data }) => {
                handleAction(isStar ? ObjectAction.Star : ObjectAction.StarUndo, data);
            }
        })
    }, [handleAction, routine?.id, star]);

    const handleVote = useCallback((isUpvote: boolean | null) => {
        if (!routine?.id) return;
        mutationWrapper({
            mutation: vote,
            input: { isUpvote, forId: routine.id, voteFor: VoteFor.Routine },
            onSuccess: ({ data }) => {
                handleAction(isUpvote ? ObjectAction.VoteUp : ObjectAction.VoteDown, data);
            }
        })
    }, [handleAction, routine?.id, vote]);

    const onSelect = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Copy:
                handleCopy();
                break;
            case ObjectAction.Delete:
                openDelete();
                break;
            case ObjectAction.Fork:
                handleFork();
                break;
            case ObjectAction.Star:
            case ObjectAction.StarUndo:
                handleStar(action === ObjectAction.Star);
                break;
            case ObjectAction.VoteDown:
            case ObjectAction.VoteUp:
                handleVote(action === ObjectAction.VoteUp);
                break;
        }
    }, [handleCopy, handleFork, handleStar, handleVote, openDelete]);

    // const languageComponent = useMemo(() => {
    //     if (isEditing) return (
    //         <LanguageInput
    //             currentLanguage={language}
    //             handleAdd={handleAddLanguage}
    //             handleDelete={handleLanguageDelete}
    //             handleCurrent={handleLanguageSelect}
    //             selectedLanguages={languages}
    //             session={session}
    //             zIndex={zIndex}
    //         />
    //     )
    //     return (
    //         <SelectLanguageMenu
    //             availableLanguages={availableLanguages}
    //             canDropdownOpen={availableLanguages.length > 1}
    //             currentLanguage={language}
    //             handleCurrent={handleLanguageSelect}
    //             session={session}
    //             zIndex={zIndex}
    //         />
    //     )
    // }, [availableLanguages, handleAddLanguage, handleLanguageDelete, handleLanguageSelect, isEditing, language, languages, session, zIndex]);

    return (
        <>
            {/* Delete routine confirmation dialog */}
            <DeleteDialog
                isOpen={deleteOpen}
                objectId={routine?.id ?? ''}
                objectType={DeleteOneType.Routine}
                objectName={getTranslation(routine, 'title', [language], false) ?? ''}
                handleClose={handleDeleteClose}
                zIndex={zIndex + 3}
            />
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen} sx={sxs?.iconButton}>
                <InfoIcon {...sxs?.icon ?? {}} />
            </IconButton>
            <SwipeableDrawer
                anchor="right"
                open={open}
                onOpen={() => { }} // Intentionally empty
                onClose={closeMenu}
                sx={{
                    zIndex,
                    '& .MuiDrawer-paper': {
                        background: palette.background.default,
                        maxWidth: { xs: '100%', sm: '75%', md: '50%', lg: '40%', xl: '30%' },
                    }
                }}
            >
                {/* Title bar */}
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'row',
                    alignItems: 'center',
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    padding: 1,
                }}>
                    {/* Title, created by, and version  */}
                    <Stack direction="column" spacing={1} alignItems="center" sx={{ marginLeft: 'auto' }}>
                        <EditableLabel
                            canEdit={isEditing}
                            handleUpdate={updateRoutineTitle}
                            placeholder={loading ? 'Loading...' : 'Enter title...'}
                            renderLabel={(t) => (
                                <Typography
                                    component="h2"
                                    variant="h5"
                                    textAlign="center"
                                    sx={{
                                        fontSize: { xs: '1em', sm: '1.25em', md: '1.5em' },
                                    }}
                                >{t ?? (loading ? 'Loading...' : 'Enter title')}</Typography>
                            )}
                            text={getTranslation(routine, 'title', [language], false) ?? ''}
                            validationSchema={titleValidation.required(requiredErrorMessage)}
                        />
                        {!isEditing && <Stack direction="row" spacing={1}>
                            <OwnerLabel objectType={ObjectType.Routine} owner={routine?.owner} session={session} />
                            <Typography variant="body1"> - {routine?.version}</Typography>
                        </Stack>}
                    </Stack>
                    <IconButton onClick={closeMenu} sx={{
                        color: palette.primary.contrastText,
                        borderRadius: 0,
                        borderBottom: `1px solid ${palette.primary.dark}`,
                        justifyContent: 'end',
                        flexDirection: 'row-reverse',
                        marginLeft: 'auto',
                    }}>
                        <CloseIcon />
                    </IconButton>
                </Box>
                {/* Main content */}
                {/* Stack that shows routine info, such as resources, description, inputs/outputs */}
                <Stack direction="column" spacing={2} padding={2}>
                    {/* Relationships */}
                    <RelationshipButtons
                        disabled={!isEditing}
                        objectType={ObjectType.Routine}
                        onRelationshipsChange={handleRelationshipsChange}
                        relationships={relationships}
                        session={session}
                        zIndex={zIndex}
                    />
                    {/* Language */}
                    {/* {languageComponent} */}
                    {/* Resources */}
                    {resourceListObject}
                    {/* Description */}
                    <Box>
                        <EditableTextCollapse
                            isEditing={isEditing}
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
                            text={getTranslation(routine, 'description', [language]) ?? ''}
                            title="Description"
                        />
                    </Box>
                    {/* Instructions */}
                    <Box>
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
                            text={getTranslation(routine, 'instructions', [language]) ?? ''}
                            title="Instructions"
                        />
                    </Box>
                    {isEditing && <VersionInput
                        fullWidth
                        id="version"
                        name="version"
                        value={formik.values.version}
                        onBlur={formik.handleBlur}
                        onChange={(newVersion: string) => { formik.setFieldValue('version', newVersion) }}
                        error={formik.touched.version && Boolean(formik.errors.version)}
                        helperText={formik.touched.version ? formik.errors.version : null}
                    />}
                    {/* Inputs/Outputs TODO*/}
                    {/* Tags */}
                    {
                        isEditing ? (
                            <TagSelector
                                handleTagsUpdate={handleTagsUpdate}
                                session={session}
                                tags={tags}
                            />
                        ) : tags.length > 0 ? (
                            <TagList session={session} parentId={routine?.id ?? ''} tags={tags as any[]} />
                        ) : null
                    }
                    {/* Is internal checkbox */}
                    <Tooltip placement={'top'} title='Indicates if this routine is meant to be a subroutine for only one other routine. If so, it will not appear in search resutls.'>
                        <FormControlLabel
                            disabled={!isEditing}
                            label='Internal'
                            control={
                                <Checkbox
                                    id='routine-info-dialog-is-internal'
                                    size="small"
                                    name='isInternal'
                                    color='secondary'
                                    checked={formik.values.isInternal}
                                    onChange={formik.handleChange}
                                />
                            }
                        />
                    </Tooltip>
                </Stack>
                {/* List of actions that can be taken, such as viewing stats, forking, and deleting */}
                <List sx={{ marginTop: 'auto' }}>
                    {actions.map(([value, label, Icon, secondaryLabel]) => (
                        <ListItem
                            key={value}
                            button
                            onClick={() => onSelect(value)}
                        >
                            <ListItemIcon>
                                <Icon fill={palette.background.textSecondary} />
                            </ListItemIcon>
                            <ListItemText primary={label} secondary={secondaryLabel} sx={{
                                '& .MuiListItemText-secondary': {
                                    color: 'red',
                                },
                            }} />
                        </ListItem>
                    ))}
                </List>
            </SwipeableDrawer>
        </>
    );
}