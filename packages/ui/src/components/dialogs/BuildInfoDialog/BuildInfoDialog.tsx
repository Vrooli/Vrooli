/**
 * Drawer to display overall routine info on the build page. 
 * Swipes left from right of screen
 */
import { useCallback, useMemo, useState } from 'react';
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
import { fork, forkVariables } from 'graphql/generated/fork';
import { star, starVariables } from 'graphql/generated/star';
import { vote, voteVariables } from 'graphql/generated/vote';
import { ObjectAction, BuildInfoDialogProps } from '../types';
import { DeleteDialog, EditableLabel, EditableTextCollapse, LanguageInput, OwnerLabel, RelationshipButtons, ReportDialog, ResourceListHorizontal, ShareObjectDialog, SnackSeverity, TagList, TagSelector, VersionDisplay, VersionInput } from 'components';
import { addEmptyTranslation, getTranslation, getTranslationData, handleTranslationBlur, handleTranslationChange, ObjectType, PubSub, removeTranslation } from 'utils';
import { useLocation } from '@shared/route';
import { APP_LINKS, DeleteOneType, ForkType, ReportFor, StarFor, VoteFor } from '@shared/consts';
import { SelectLanguageMenu } from '../SelectLanguageMenu/SelectLanguageMenu';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils';
import { forkMutation, starMutation, voteMutation } from 'graphql/mutation';
import { BranchIcon, CloseIcon, DeleteIcon, DownvoteWideIcon, InfoIcon, ReportIcon, ShareIcon, StarFilledIcon, StarOutlineIcon, StatsIcon, SvgComponent, UpvoteWideIcon } from '@shared/icons';
import { requiredErrorMessage, title as titleValidation } from '@shared/validation';

export const BuildInfoDialog = ({
    formik,
    handleAction,
    handleLanguageChange,
    handleRelationshipsChange,
    handleResourcesUpdate,
    handleTagsUpdate,
    isEditing,
    language,
    loading,
    relationships,
    routine,
    session,
    sxs,
    tags,
    zIndex,
}: BuildInfoDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    console.log('buildinfodialog renderrr')

    // Handle translations
    const { description, instructions, title, errorDescription, errorInstructions, touchedDescription, touchedInstructions } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            description: value?.description ?? '',
            instructions: value?.instructions ?? '',
            title: value?.title ?? '',
            errorDescription: error?.description ?? '',
            errorInstructions: error?.instructions ?? '',
            touchedDescription: touched?.description ?? false,
            touchedInstructions: touched?.instructions ?? false,
        }
    }, [formik, language]);
    const languages = useMemo(() => formik.values.translationsUpdate.map(t => t.language), [formik.values.translationsUpdate]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        handleLanguageChange(newLanguage);
        addEmptyTranslation(formik, 'translationsUpdate', newLanguage);
    }, [formik, handleLanguageChange]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        handleLanguageChange(newLanguages[0]);
        removeTranslation(formik, 'translationsUpdate', language);
    }, [formik, handleLanguageChange, languages]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsUpdate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsUpdate', e, language)
    }, [formik, language]);

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
        const results: [ObjectAction, string, SvgComponent, string | null][] = [];
        // If signed in and not editing, show vote/star options
        if (session?.isLoggedIn === true && !isEditing) {
            results.push(routine?.isUpvoted ?
                [ObjectAction.VoteDown, 'Downvote', DownvoteWideIcon, null] :
                [ObjectAction.VoteUp, 'Upvote', UpvoteWideIcon, null]
            );
            results.push(routine?.isStarred ?
                [ObjectAction.StarUndo, 'Unstar', StarFilledIcon, null] :
                [ObjectAction.Star, 'Star', StarOutlineIcon, null]
            );
        }
        // If not editing, show "Stats" and "Fork" buttons
        if (!isEditing) {
            results.push(
                [ObjectAction.Stats, 'Stats', StatsIcon, 'Coming Soon'],
                [ObjectAction.Share, 'Share', ShareIcon, null],
            )
            if (routine?.permissionsRoutine?.canFork) {
                results.push([ObjectAction.Fork, 'Fork', BranchIcon, null]);
            }
            if (routine?.permissionsRoutine?.canReport) {
                results.push([ObjectAction.Report, 'Report', ReportIcon, null]);
            }
        }
        // Only show "Delete" when editing an existing routine
        if (isEditing && Boolean(routine?.id)) {
            results.push(
                [ObjectAction.Delete, 'Delete', DeleteIcon, null],
            )
        }
        return results;
    }, [isEditing, routine?.id, routine?.isStarred, routine?.isUpvoted, routine?.permissionsRoutine?.canFork, routine?.permissionsRoutine?.canReport, session?.isLoggedIn]);

    // Handle delete
    const [deleteOpen, setDeleteOpen] = useState(false);
    const openDelete = useCallback(() => setDeleteOpen(true), []);
    const handleDeleteClose = useCallback((wasDeleted: boolean) => {
        if (wasDeleted) setLocation(APP_LINKS.Home);
        else setDeleteOpen(false);
    }, [setLocation])

    const [shareOpen, setShareOpen] = useState<boolean>(false);
    const [reportOpen, setReportOpen] = useState<boolean>(false);

    const openShare = useCallback(() => setShareOpen(true), [setShareOpen]);
    const closeShare = useCallback(() => setShareOpen(false), [setShareOpen]);

    const openReport = useCallback(() => setReportOpen(true), [setReportOpen]);
    const closeReport = useCallback(() => setReportOpen(false), [setReportOpen]);

    // Mutations
    const [fork] = useMutation<fork, forkVariables>(forkMutation);
    const [star] = useMutation<star, starVariables>(starMutation);
    const [vote] = useMutation<vote, voteVariables>(voteMutation);

    const handleFork = useCallback(() => {
        if (!routine?.id) return;
        mutationWrapper({
            mutation: fork,
            input: { id: routine.id, objectType: ForkType.Routine },
            onSuccess: (data) => {
                PubSub.get().publishSnack({ message: `${getTranslation(routine, 'title', [language], true)} forked.`, severity: SnackSeverity.Success });
                handleAction(ObjectAction.Fork, data);
            }
        })
    }, [fork, handleAction, language, routine]);

    const handleStar = useCallback((isStar: boolean) => {
        if (!routine?.id) return;
        mutationWrapper({
            mutation: star,
            input: { isStar, forId: routine.id, starFor: StarFor.Routine },
            onSuccess: (data) => {
                handleAction(isStar ? ObjectAction.Star : ObjectAction.StarUndo, data);
            }
        })
    }, [handleAction, routine?.id, star]);

    const handleVote = useCallback((isUpvote: boolean | null) => {
        if (!routine?.id) return;
        mutationWrapper({
            mutation: vote,
            input: { isUpvote, forId: routine.id, voteFor: VoteFor.Routine },
            onSuccess: (data) => {
                handleAction(isUpvote ? ObjectAction.VoteUp : ObjectAction.VoteDown, data);
            }
        })
    }, [handleAction, routine?.id, vote]);

    const onSelect = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Delete:
                openDelete();
                break;
            case ObjectAction.Fork:
                handleFork();
                break;
            case ObjectAction.Report:
                openReport();
                break;
            case ObjectAction.Share:
                openShare();
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
    }, [handleFork, handleStar, handleVote, openDelete, openReport, openShare]);

    const languageComponent = useMemo(() => {
        if (isEditing) return (
            <LanguageInput
                currentLanguage={language}
                handleAdd={handleAddLanguage}
                handleDelete={handleLanguageDelete}
                handleCurrent={handleLanguageChange}
                session={session}
                translations={formik.values.translationsUpdate}
                zIndex={zIndex}
            />
        )
        return (
            <SelectLanguageMenu
                currentLanguage={language}
                handleCurrent={handleLanguageChange}
                session={session}
                translations={formik.values.translationsUpdate}
                zIndex={zIndex}
            />
        )
    }, [formik.values.translationsUpdate, handleAddLanguage, handleLanguageChange, handleLanguageDelete, isEditing, language, session, zIndex]);

    return (
        <>
            {/* Report dialog */}
            {routine?.id && <ReportDialog
                forId={routine.id}
                onClose={closeReport}
                open={reportOpen}
                reportFor={ReportFor.Routine}
                session={session}
                zIndex={zIndex + 1}
            />}
            {/* Share dialog */}
            <ShareObjectDialog
                objectType={ObjectType.Routine}
                open={shareOpen}
                onClose={closeShare}
                zIndex={zIndex + 1}
            />
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
                            handleUpdate={(newText: string) => onTranslationChange({ target: { name: 'title', value: newText }})}
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
                            text={title}
                            validationSchema={titleValidation.required(requiredErrorMessage)}
                        />
                        {!isEditing && <Stack direction="row" spacing={1}>
                            <OwnerLabel objectType={ObjectType.Routine} owner={routine?.owner} session={session} />
                            <VersionDisplay
                                currentVersion={routine?.version}
                                prefix={" - "}
                            />
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
                    {languageComponent}
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
                                value: description,
                                multiline: true,
                                maxRows: 3,
                                onBlur: onTranslationBlur,
                                onChange: onTranslationChange,
                                error: touchedDescription && Boolean(errorDescription),
                                helperText: touchedDescription ? errorDescription as string : null,
                            }}
                            text={description}
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
                                value: instructions,
                                minRows: 3,
                                onChange: (newText: string) => onTranslationChange({ target: { name: 'instructions', value: newText }}),
                                error: touchedInstructions && Boolean(errorInstructions),
                                helperText: touchedInstructions ? errorInstructions as string : null,
                            }}
                            text={instructions}
                            title="Instructions"
                        />
                    </Box>
                    {isEditing && <VersionInput
                        fullWidth
                        id="version"
                        name="version"
                        value={formik.values.version}
                        onBlur={formik.handleBlur}
                        onChange={(newVersion: string) => {
                            formik.setFieldValue('version', newVersion);
                            handleRelationshipsChange({
                                ...relationships,
                                isComplete: false,
                            })
                        }}
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