/**
 * Drawer to display overall routine info on the build page. 
 * Swipes left from right of screen
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    FileCopy as CopyIcon,
    Delete as DeleteIcon,
    ForkRight as ForkIcon,
    Info as InfoIcon,
    QueryStats as StatsIcon,
    StarOutline as StarIcon,
    Star as UnstarIcon,
    ThumbDown as DownvoteIcon,
    ThumbUp as UpvoteIcon,
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
    TextField,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { copy, copyVariables } from 'graphql/generated/copy';
import { fork, forkVariables } from 'graphql/generated/fork';
import { star, starVariables } from 'graphql/generated/star';
import { vote, voteVariables } from 'graphql/generated/vote';
import { BaseObjectAction, BuildInfoDialogProps } from '../types';
import Markdown from 'markdown-to-jsx';
import { DeleteDialog, EditableLabel, LanguageInput, LinkButton, MarkdownInput, ResourceListHorizontal } from 'components';
import { AllLanguages, getLanguageSubtag, getOwnedByString, getTranslation, PubSub, RoutineTranslationShape, toOwnedBy, updateArray } from 'utils';
import { useLocation } from '@shared/route';
import { useFormik } from 'formik';
import { APP_LINKS, CopyType, DeleteOneType, ForkType, StarFor, VoteFor } from '@shared/consts';
import { routineUpdateForm as validationSchema } from '@shared/validation';
import { SelectLanguageDialog } from '../SelectLanguageDialog/SelectLanguageDialog';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils';
import { copyMutation, forkMutation, starMutation, voteMutation } from 'graphql/mutation';
import { v4 as uuid } from 'uuid';
import { CloseIcon } from 'assets/img';

export const BuildInfoDialog = ({
    handleAction,
    handleLanguageChange,
    handleUpdate,
    isEditing,
    language,
    loading,
    routine,
    session,
    sxs,
    zIndex,
}: BuildInfoDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const ownedBy = useMemo<string | null>(() => getOwnedByString(routine, [language]), [routine, language]);
    const toOwner = useCallback(() => { toOwnedBy(routine, setLocation) }, [routine, setLocation]);

    // Handle translations
    type Translation = RoutineTranslationShape;
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        setTranslations([...translations.filter(t => t.language !== language)]);
    }, [translations]);
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(translations, index, translation) : [...translations, translation];
    }, [translations]);
    const updateTranslation = useCallback((language: string, translation: Translation) => {
        setTranslations(getTranslationsUpdate(language, translation));
    }, [getTranslationsUpdate]);

    // Handle update
    const formik = useFormik({
        initialValues: {
            description: getTranslation(routine, 'description', [language]) ?? '',
            instructions: getTranslation(routine, 'instructions', [language]) ?? '',
            isInternal: routine?.isInternal ?? false,
            isComplete: routine?.isComplete ?? true,
            title: getTranslation(routine, 'title', [language]) ?? '',
            version: routine?.version ?? '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                instructions: values.instructions,
                title: values.title,
            })
            handleUpdate({
                ...routine,
                isInternal: values.isInternal,
                isComplete: values.isComplete,
                version: values.version,
                translations: allTranslations,
            } as any);
        },
    });

    // Handle languages
    const availableLanguages = useMemo<string[]>(() => {
        if (isEditing) return Object.keys(AllLanguages);
        return routine?.translations?.map(t => getLanguageSubtag(t.language)) ?? [];
    }, [isEditing, routine?.translations]);
    const [languages, setLanguages] = useState<string[]>([]);

    useEffect(() => {
        if (!language || !translations) return;
        const translation = translations.find(t => language === t.language);
        if (translation) {
            formik.setValues({
                ...formik.values,
                description: translations[0].description ?? '',
                instructions: translations[0].instructions,
                title: translations[0].title,
            })
        }
    }, [formik, language, translations])
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
        handleLanguageChange(newLanguage);
    }, [formik.values.description, formik.values.instructions, formik.values.title, handleLanguageChange, language, updateFormikTranslation, updateTranslation]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
    }, [handleLanguageSelect, languages, setLanguages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        deleteTranslation(language);
        updateFormikTranslation(newLanguages[0]);
        handleLanguageChange(newLanguages[0]);
        setLanguages(newLanguages);
    }, [deleteTranslation, handleLanguageChange, languages, updateFormikTranslation]);

    const updateRoutineTitle = useCallback((title: string) => {
        if (!routine) return;
        updateTranslation(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            instructions: formik.values.instructions,
            title: title,
        })
        formik.setFieldValue('title', title);
    }, [formik, language, routine, updateTranslation]);

    // Open boolean for drawer
    const [open, setOpen] = useState(false);
    const toggleOpen = () => setOpen(o => !o);
    const closeMenu = () => {
        // If not editing, just close 
        if (!isEditing) {
            setOpen(false);
            return;
        }
        // If editing, try to save changes
        if (formik.isValid) {
            formik.handleSubmit();
            setOpen(false);
        } else {
            PubSub.get().publishSnack({ message: 'Please fix errors before closing.', severity: 'Error' });
        }
    };

    const resourceList = useMemo(() => {
        if (!routine ||
            !Array.isArray(routine.resourceLists) ||
            routine.resourceLists.length < 1 ||
            routine.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={routine.resourceLists[0]}
            canEdit={false}
            handleUpdate={() => { }} // Intentionally blank
            loading={loading}
            session={session}
            zIndex={zIndex}
        />
    }, [loading, routine, session, zIndex]);

    /**
     * Determines which action buttons to display
     */
    const actions = useMemo(() => {
        // [value, label, icon, secondaryLabel]
        const results: [BaseObjectAction, string, any, string | null][] = [];
        // If signed in and not editing, show vote/star options
        if (session?.isLoggedIn === true && !isEditing) {
            results.push(routine?.isUpvoted ?
                [BaseObjectAction.Downvote, 'Downvote', DownvoteIcon, null] :
                [BaseObjectAction.Upvote, 'Upvote', UpvoteIcon, null]
            );
            results.push(routine?.isStarred ?
                [BaseObjectAction.Unstar, 'Unstar', UnstarIcon, null] :
                [BaseObjectAction.Star, 'Star', StarIcon, null]
            );
        }
        // If not editing, show "Stats" and "Fork" buttons
        if (!isEditing) {
            results.push(
                [BaseObjectAction.Stats, 'Stats', StatsIcon, 'Coming Soon'],
                [BaseObjectAction.Copy, 'Copy', CopyIcon, null],
                [BaseObjectAction.Fork, 'Fork', ForkIcon, null],
            )
        }
        // Only show "Delete" when editing an existing routine
        if (isEditing && Boolean(routine?.id)) {
            results.push(
                [BaseObjectAction.Delete, 'Delete', DeleteIcon, null],
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
                handleAction(BaseObjectAction.Copy, data);
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
                handleAction(BaseObjectAction.Fork, data);
            }
        })
    }, [fork, handleAction, language, routine]);

    const handleStar = useCallback((isStar: boolean) => {
        if (!routine?.id) return;
        mutationWrapper({
            mutation: star,
            input: { isStar, forId: routine.id, starFor: StarFor.Routine },
            onSuccess: ({ data }) => {
                handleAction(isStar ? BaseObjectAction.Star : BaseObjectAction.Unstar, data);
            }
        })
    }, [handleAction, routine?.id, star]);

    const handleVote = useCallback((isUpvote: boolean | null) => {
        if (!routine?.id) return;
        mutationWrapper({
            mutation: vote,
            input: { isUpvote, forId: routine.id, voteFor: VoteFor.Routine },
            onSuccess: ({ data }) => {
                handleAction(isUpvote ? BaseObjectAction.Upvote : BaseObjectAction.Downvote, data);
            }
        })
    }, [handleAction, routine?.id, vote]);

    const onSelect = useCallback((action: BaseObjectAction) => {
        switch (action) {
            case BaseObjectAction.Copy:
                handleCopy();
                break;
            case BaseObjectAction.Delete:
                openDelete();
                break;
            case BaseObjectAction.Downvote:
                handleVote(false);
                break;
            case BaseObjectAction.Fork:
                handleFork();
                break;
            case BaseObjectAction.Star:
                handleStar(true);
                break;
            case BaseObjectAction.Unstar:
                handleStar(false);
                break;
            case BaseObjectAction.Upvote:
                handleVote(true);
                break;
        }
    }, [handleCopy, handleFork, handleStar, handleVote, openDelete]);

    const languageComponent = useMemo(() => {
        if (isEditing) return (
            <LanguageInput
                currentLanguage={language}
                handleAdd={handleAddLanguage}
                handleDelete={handleLanguageDelete}
                handleCurrent={handleLanguageSelect}
                selectedLanguages={languages}
                session={session}
                zIndex={zIndex}
            />
        )
        return (
            <SelectLanguageDialog
                availableLanguages={availableLanguages}
                canDropdownOpen={availableLanguages.length > 1}
                currentLanguage={language}
                handleCurrent={handleLanguageSelect}
                session={session}
                zIndex={zIndex}
            />
        )
    }, [availableLanguages, handleAddLanguage, handleLanguageDelete, handleLanguageSelect, isEditing, language, languages, session, zIndex]);

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
                <InfoIcon sx={sxs?.icon} />
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
                        />
                        <Stack direction="row" spacing={1}>
                            {ownedBy ? (
                                <LinkButton
                                    onClick={toOwner}
                                    text={ownedBy}
                                />
                            ) : null}
                            <Typography variant="body1"> - {routine?.version}</Typography>
                        </Stack>
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
                <Stack direction="column" spacing={2} padding={1}>
                    {/* Language */}
                    {languageComponent}
                    {/* Resources */}
                    {resourceList}
                    {/* Description */}
                    <Box sx={{
                        padding: 1,
                    }}>
                        <Typography variant="h6">Description</Typography>
                        {
                            isEditing ? (
                                <TextField
                                    fullWidth
                                    id="description"
                                    name="description"
                                    label="description"
                                    value={formik.values.description}
                                    multiline
                                    maxRows={3}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.description && Boolean(formik.errors.description)}
                                    helperText={formik.touched.description && formik.errors.description}
                                />
                            ) : (
                                <Markdown>{getTranslation(routine, 'description', [language]) ?? ''}</Markdown>
                            )
                        }
                    </Box>
                    {/* Instructions */}
                    <Box sx={{
                        padding: 1,
                    }}>
                        <Typography variant="h6">Instructions</Typography>
                        {
                            isEditing ? (
                                <MarkdownInput
                                    id="instructions"
                                    placeholder="Instructions"
                                    value={formik.values.instructions}
                                    minRows={3}
                                    onChange={(newText: string) => formik.setFieldValue('instructions', newText)}
                                    error={formik.touched.instructions && Boolean(formik.errors.instructions)}
                                    helperText={formik.touched.instructions ? formik.errors.instructions as string : null}
                                />
                            ) : (
                                <Markdown>{getTranslation(routine, 'instructions', [language]) ?? ''}</Markdown>
                            )
                        }
                    </Box>
                    {/* Inputs/Outputs TODO*/}
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
                    {/* Is complete checkbox */}
                    <Tooltip placement={'top'} title='Indicates if this routine is ready for others to run. If not, it will not show in default search results'>
                        <FormControlLabel
                            disabled={!isEditing}
                            label='Complete'
                            control={
                                <Checkbox
                                    id='routine-info-dialog-is-complete'
                                    size="small"
                                    name='isComplete'
                                    color='secondary'
                                    checked={formik.values.isComplete}
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
                                <Icon />
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