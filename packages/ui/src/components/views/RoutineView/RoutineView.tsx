import { Box, Button, CircularProgress, Dialog, Grid, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useLazyQuery } from "@apollo/client";
import { routine, routineVariables } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    AccountTree as GraphIcon,
    DoneAll as MarkAsCompleteIcon,
    Edit as EditIcon,
    MoreHoriz as EllipsisIcon,
    PlayCircle as StartIcon,
} from "@mui/icons-material";
import { BaseObjectActionDialog, DeleteRoutineDialog, LinkButton, ResourceListHorizontal, RunPickerDialog, RunView, SelectLanguageDialog, StarButton, UpTransition } from "components";
import { RoutineViewProps } from "../types";
import { getLanguageSubtag, getOwnedByString, getPreferredLanguage, getTranslation, getUserLanguages, Pubs, stringifySearchParams, toOwnedBy, useReactSearch } from "utils";
import { Routine, Run } from "types";
import Markdown from "markdown-to-jsx";
import { runCompleteMutation, routineDeleteOneMutation } from "graphql/mutation";
import { mutationWrapper } from "graphql/utils/wrappers";
import { NodeType, StarFor } from "graphql/generated/globalTypes";
import { BaseObjectAction } from "components/dialogs/types";
import { containerShadow } from "styles";
import { validate as uuidValidate } from 'uuid';
import { runComplete } from "graphql/generated/runComplete";
import { owns } from "utils/authentication";

const TERTIARY_COLOR = '#95f3cd';

export const RoutineView = ({
    partialData,
    session,
}: RoutineViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Routine}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchRoutines}/view/:id`);
    const id = params?.id ?? params2?.id;
    // Fetch data
    const [getData, { data, loading }] = useLazyQuery<routine, routineVariables>(routineQuery);
    useEffect(() => {
        if (id) getData({ variables: { input: { id } } });
    }, [getData, id])
    const routine = useMemo(() => data?.routine, [data]);
    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null); // Routine may change if it is starred/upvoted/etc.
    const canEdit = useMemo<boolean>(() => owns(routine?.role), [routine]);
    // Open boolean for delete routine confirmation
    const [deleteOpen, setDeleteOpen] = useState(false);
    const openDelete = () => setDeleteOpen(true);
    const closeDelete = () => setDeleteOpen(false);

    useEffect(() => {
        if (routine) { setChangedRoutine(routine) }
    }, [routine]);

    const [routineDelete, { loading: loadingDelete }] = useMutation<any>(routineDeleteOneMutation);
    /**
     * Deletes the entire routine. Assumes confirmation was already given.
     */
    const deleteRoutine = useCallback(() => {
        if (!routine) return;
        mutationWrapper({
            mutation: routineDelete,
            input: { id: routine.id },
            successMessage: () => 'Routine deleted.',
            onSuccess: () => { setLocation(APP_LINKS.Home) },
        })
    }, [routine, routineDelete, setLocation])

    const search = useReactSearch(null);
    const { runId } = useMemo(() => ({
        runId: typeof search.run === 'string' && uuidValidate(search.run) ? search.run : null,
    }), [search]);

    const [language, setLanguage] = useState<string>('');
    const [availableLanguages, setAvailableLanguages] = useState<string[]>([]);
    useEffect(() => {
        const availableLanguages = routine?.translations?.map(t => getLanguageSubtag(t.language)) ?? [];
        const userLanguages = getUserLanguages(session);
        setAvailableLanguages(availableLanguages);
        setLanguage(getPreferredLanguage(availableLanguages, userLanguages));
    }, [routine, session]);
    const { title, description, instructions } = useMemo(() => {
        return {
            title: getTranslation(routine, 'title', [language]) ?? getTranslation(partialData, 'title', [language]),
            description: getTranslation(routine, 'description', [language]) ?? getTranslation(partialData, 'description', [language]),
            instructions: getTranslation(routine, 'instructions', [language]) ?? getTranslation(partialData, 'instructions', [language]),
        };
    }, [routine, language, partialData]);

    useEffect(() => {
        document.title = `${title} | Vrooli`;
    }, [title]);

    const ownedBy = useMemo<string | null>(() => getOwnedByString(routine, [language]), [routine, language]);
    const toOwner = useCallback(() => { toOwnedBy(routine, setLocation) }, [routine, setLocation]);

    const viewGraph = useCallback(() => {
        setLocation(`${APP_LINKS.Build}/${routine?.id}`);
    }, [routine?.id, setLocation]);

    const [runComplete] = useMutation<runComplete>(runCompleteMutation);
    const markAsComplete = useCallback(() => {
        if (!routine) return;
        mutationWrapper({
            mutation: runComplete,
            input: { id: routine.id, exists: false },
            successMessage: () => 'Routine completed!🎉',
            onSuccess: () => {
                PubSub.publish(Pubs.Celebration);
                setLocation(APP_LINKS.Home)
            },
        })
    }, [routine, runComplete, setLocation]);

    const [isRunOpen, setIsRunOpen] = useState(false)
    const [selectRunAnchor, setSelectRunAnchor] = useState<any>(null);
    const handleRunSelect = useCallback((run: Run) => {
        console.log('handle run selectttt', run)
        setLocation(stringifySearchParams({
            run: run.id,
            step: run.steps.length > 0 ? run.steps[run.steps.length - 1].step : undefined,
        }), { replace: true });
        setIsRunOpen(true);
    }, [setLocation]);
    const handleSelectRunClose = useCallback(() => setSelectRunAnchor(null), []);

    const runRoutine = useCallback((e: any) => {
        // Validate routine before trying to run
        if (!routine || !uuidValidate(routine.id)) {
            PubSub.publish(Pubs.Snack, { message: 'Error loading routine.', severity: 'error' });
            return;
        }
        // Find first node
        const firstNode = routine?.nodes?.find(node => node.type === NodeType.Start);
        if (!firstNode) {
            PubSub.publish(Pubs.Snack, { message: 'Routine invalid - cannot run.', severity: 'Error' });
            return;
        }
        // If run specified use that
        if (runId) {
            handleRunSelect({ id: runId } as Run);
        }
        // Otherwise, open dialog to select runs
        else {
            setSelectRunAnchor(e.currentTarget);
        }
    }, [handleRunSelect, routine, runId]);
    const stopRoutine = () => { setIsRunOpen(false) };

    const onEdit = useCallback(() => {
        // Depends on if we're in a search popup or a normal organization page
        setLocation(Boolean(params?.id) ? `${APP_LINKS.Routine}/edit/${id}` : `${APP_LINKS.SearchRoutines}/edit/${id}`);
    }, [setLocation, params?.id, id]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    // Determine options available to object, in order
    const moreOptions: BaseObjectAction[] = useMemo(() => {
        // Initialize
        let options: BaseObjectAction[] = [];
        if (canEdit) {
            options.push(BaseObjectAction.Edit);
        }
        options.push(BaseObjectAction.Stats);
        if (session && !canEdit) {
            options.push(routine?.isUpvoted ? BaseObjectAction.Downvote : BaseObjectAction.Upvote);
            options.push(routine?.isStarred ? BaseObjectAction.Unstar : BaseObjectAction.Star);
            options.push(BaseObjectAction.Fork);
        }
        options.push(BaseObjectAction.Donate, BaseObjectAction.Share)
        if (session?.id) {
            options.push(BaseObjectAction.Report);
        }
        if (canEdit) {
            options.push(BaseObjectAction.Delete);
        }
        return options;
    }, [routine, canEdit, session]);

    /**
     * If routine has nodes (i.e. is not just this page), display "View Graph" and "Start" (or "Continue") buttons. 
     * Otherwise, display "Mark as Complete" button.
     */
    const actions = useMemo(() => {
        // If routine has no nodes
        if (!routine?.nodes?.length) return (
            <Grid container spacing={1}>
                <Grid item xs={12}>
                    <Button startIcon={<MarkAsCompleteIcon />} fullWidth onClick={markAsComplete} color="secondary">Mark as Complete</Button>
                </Grid>
            </Grid>
        )
        // If routine has nodes
        return (
            <Grid container spacing={1}>
                <Grid item xs={12} sm={6}>
                    <Button startIcon={<GraphIcon />} fullWidth onClick={viewGraph} color="secondary">View Graph</Button>
                </Grid>
                {/* Show continue if routine already has progress TODO */}
                <Grid item xs={12} sm={6}>
                    {routine && routine.runs.length > 0 ?
                        <Button startIcon={<StartIcon />} fullWidth onClick={runRoutine} color="secondary">Continue</Button> :
                        <Button startIcon={<StartIcon />} fullWidth onClick={runRoutine} color="secondary">Start Now</Button>
                    }
                </Grid>
            </Grid>
        )
    }, [routine, markAsComplete, viewGraph, runRoutine]);

    const resourceList = useMemo(() => {
        if (!routine ||
            !Array.isArray(routine.resourceLists) ||
            routine.resourceLists.length < 1 ||
            routine.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={(routine as any).resourceLists[0]}
            canEdit={false}
            handleUpdate={() => { }} // Intentionally blank
            loading={loading}
            session={session}
        />
    }, [loading, routine, session]);

    /**
     * Display body or loading indicator
     */
    const body = useMemo(() => {
        if (loading) return (
            <Box sx={{
                minHeight: 'min(300px, 25vh)',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
            }}>
                <CircularProgress color="secondary" />
            </Box>
        )
        return (
            <>
                {/* Stack that shows routine info, such as resources, description, inputs/outputs */}
                <Stack direction="column" spacing={2} padding={1}>
                    {/* Resources */}
                    {resourceList}
                    {/* Description */}
                    <Box sx={{
                        padding: 1,
                        borderRadius: 1,
                        color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary,
                    }}>
                        <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>Description</Typography>
                        <Typography variant="body1">{description ?? 'No description set'}</Typography>
                    </Box>
                    {/* Instructions */}
                    <Box sx={{
                        padding: 1,
                        borderRadius: 1,
                        color: Boolean(instructions) ? palette.background.textPrimary : palette.background.textSecondary,
                    }}>
                        <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>Instructions</Typography>
                        <Markdown>{instructions ?? 'No instructions'}</Markdown>
                    </Box>
                    {/* Stats */}
                    {Array.isArray(routine?.nodes) && (routine as any).nodes.length > 0 && <Box sx={{
                        padding: 1,
                        borderRadius: 1,
                    }}>
                        {/* TODO click to view */}
                        <Typography variant="h6">Stats</Typography>
                        <Typography variant="body1">Complexity: {routine?.complexity}</Typography>
                        <Typography variant="body1">Simplicity: {routine?.simplicity}</Typography>
                        <Typography variant="body1">Score: {routine?.score}</Typography>
                        <Typography variant="body1">Stars: {routine?.stars}</Typography>
                    </Box>}
                </Stack>
                {/* Actions */}
                {actions}
            </>
        )
    }, [loading, routine, resourceList, description, palette.background.textPrimary, palette.background.textSecondary, instructions, actions]);

    return (
        <Box sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            margin: 'auto',
            minHeight: '88vh',
        }}>
            {/* Chooses which run to use */}
            <RunPickerDialog
                anchorEl={selectRunAnchor}
                handleClose={handleSelectRunClose}
                onSelect={handleRunSelect}
                routine={routine}
                session={session}
            />
            {/* Delete routine confirmation dialog */}
            <DeleteRoutineDialog
                isOpen={deleteOpen}
                routineName={getTranslation(routine, 'title', getUserLanguages(session)) ?? ''}
                handleClose={closeDelete}
                handleDelete={deleteRoutine}
            />
            {/* Dialog for running routine */}
            <Dialog
                id="run-routine-view-dialog"
                fullScreen
                open={isRunOpen}
                onClose={stopRoutine}
                TransitionComponent={UpTransition}
            >
                {routine && <RunView
                    handleClose={stopRoutine}
                    routine={routine}
                    session={session}
                />}
            </Dialog>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                handleActionComplete={() => { }} //TODO
                handleDelete={() => { }} //TODO
                handleEdit={onEdit}
                objectId={id ?? ''}
                objectType={'Routine'}
                anchorEl={moreMenuAnchor}
                title='Routine Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
                session={session}
            />
            {/* Main container */}
            <Box sx={{
                background: palette.background.paper,
                overflowY: 'auto',
                width: 'min(96vw, 600px)',
                borderRadius: '8px',
                overflow: 'overlay',
                ...containerShadow
            }}>
                {/* Heading container */}
                <Stack direction="column" spacing={1} sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    padding: 2,
                    marginBottom: 1,
                    background: palette.primary.main,
                    color: palette.primary.contrastText,
                }}>
                    {/* Show star button and ellipsis next to title */}
                    <Stack direction="row" spacing={1} alignItems="center">
                        {loading ?
                            <LinearProgress color="inherit" sx={{
                                borderRadius: 1,
                                width: '50vw',
                                height: 8,
                                marginTop: '12px !important',
                                marginBottom: '12px !important',
                                maxWidth: '300px',
                            }} /> :
                            <Typography variant="h5" sx={{ textAlign: 'center' }}>{title}</Typography>}

                        <Tooltip title="More options">
                            <IconButton
                                aria-label="More"
                                size="small"
                                onClick={openMoreMenu}
                                sx={{
                                    display: 'block',
                                    marginLeft: 'auto',
                                    marginRight: 1,
                                }}
                            >
                                <EllipsisIcon sx={{ fill: palette.primary.contrastText }} />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    <Stack direction="row" spacing={1} sx={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                    }}>
                        <StarButton
                            session={session}
                            objectId={routine?.id ?? ''}
                            showStars={false}
                            starFor={StarFor.Routine}
                            isStar={changedRoutine?.isStarred ?? false}
                            stars={changedRoutine?.stars ?? 0}
                            onChange={(isStar: boolean) => { changedRoutine && setChangedRoutine({ ...changedRoutine, isStarred: isStar }) }}
                            tooltipPlacement="bottom"
                        />
                        {ownedBy && (
                            <LinkButton
                                onClick={toOwner}
                                text={ownedBy}
                            />
                        )}
                        <Typography variant="body1"> - {routine?.version}</Typography>
                        <SelectLanguageDialog
                            availableLanguages={availableLanguages}
                            canDropdownOpen={availableLanguages.length > 1}
                            handleSelect={setLanguage}
                            language={language}
                            session={session}
                        />
                        {canEdit && <Tooltip title="Edit routine">
                            <IconButton
                                aria-label="Edit routine"
                                size="small"
                                onClick={onEdit}
                            >
                                <EditIcon sx={{ fill: TERTIARY_COLOR }} />
                            </IconButton>
                        </Tooltip>}
                    </Stack>
                </Stack>
                {/* Body container */}
                <Box sx={{
                    padding: 2,
                }}>
                    {body}
                </Box>
            </Box>
        </Box >
    )
}