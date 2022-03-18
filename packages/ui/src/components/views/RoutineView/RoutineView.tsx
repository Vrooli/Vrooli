import { Box, Button, CircularProgress, Dialog, Grid, IconButton, Link, Stack, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, MemberRole } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { routine } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    AccountTree as GraphIcon,
    Delete as DeleteIcon,
    Edit as EditIcon,
    ForkRight as ForkIcon,
    MoreHoriz as EllipsisIcon,
    PlayCircle as StartIcon,
    StarOutline as StartLaterIcon,
    QueryStats as StatsIcon,
} from "@mui/icons-material";
import { BaseObjectActionDialog, DeleteRoutineDialog, ResourceListHorizontal, RunRoutineView, StarButton, UpTransition } from "components";
import { RoutineViewProps } from "../types";
import { getTranslation, Pubs } from "utils";
import { ResourceList, Routine, User } from "types";
import Markdown from "markdown-to-jsx";
import { routineDeleteOneMutation, starMutation } from "graphql/mutation";
import { mutationWrapper } from "graphql/utils/wrappers";
import { star } from "graphql/generated/star";
import { NodeType, StarFor } from "graphql/generated/globalTypes";
import { BaseObjectAction } from "components/dialogs/types";
import { containerShadow } from "styles";

const TERTIARY_COLOR = '#95f3cd';

export const RoutineView = ({
    session,
    partialData,
}: RoutineViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Run}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchRoutines}/view/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch data
    const { data, loading } = useQuery<routine>(routineQuery, { variables: { input: { id } } });
    const routine = useMemo(() => data?.routine, [data]);
    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null); // Routine may change if it is starred/upvoted/etc.
    const canEdit: boolean = useMemo(() => [MemberRole.Admin, MemberRole.Owner].includes(routine?.role ?? ''), [routine]);
    const languages = useMemo(() => session?.languages ?? navigator.languages, [session]);
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
    }, [routine, routineDelete])

    const { title, description, instructions } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            title: getTranslation(routine, 'title', languages) ?? getTranslation(partialData, 'title', languages),
            description: getTranslation(routine, 'description', languages) ?? getTranslation(partialData, 'description', languages),
            instructions: getTranslation(routine, 'instructions', languages) ?? getTranslation(partialData, 'instructions', languages),
        };
    }, [routine, partialData, session]);

    /**
     * Name of user or organization that owns this routine
     */
    const ownedBy = useMemo<string | null>(() => {
        if (!routine?.owner) return null;
        return getTranslation(routine.owner, 'username', languages) ?? getTranslation(routine.owner, 'name', languages);
    }, [routine?.owner, languages]);

    /**
     * Navigate to owner's profile
     */
    const toOwner = useCallback(() => {
        if (!routine?.owner) {
            PubSub.publish(Pubs.Snack, { message: 'Could not find owner.', severity: 'Error' });
            return;
        }
        // Check if user or organization
        if (routine.owner.hasOwnProperty('username')) {
            setLocation(`${APP_LINKS.User}/${(routine.owner as User).username}`);
        } else {
            setLocation(`${APP_LINKS.Organization}/${routine.owner.id}`);
        }
    }, [routine?.owner, setLocation]);

    const viewGraph = () => {
        setLocation(`${APP_LINKS.Build}/${routine?.id}`);
    }

    const [isRunOpen, setIsRunOpen] = useState(false)
    const runRoutine = () => {
        setIsRunOpen(true)
        // Find first node
        const firstNode = routine?.nodes?.find(node => node.type === NodeType.Start);
        if (!firstNode) {
            PubSub.publish(Pubs.Snack, { message: 'Routine invalid - cannot run.', severity: 'Error' });
            return;
        }
        setLocation(`${APP_LINKS.Run}/${id}?step=1.1`);
    };
    const stopRoutine = () => { setIsRunOpen(false) };

    const onEdit = useCallback(() => {
        // Depends on if we're in a search popup or a normal organization page
        setLocation(Boolean(params?.id) ? `${APP_LINKS.Run}/${id}/edit` : `${APP_LINKS.SearchRoutines}/edit/${id}`);
    }, [setLocation, id]);

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
     * If routine has nodes (i.e. is not just this page), display "View Graph" and "Start" buttons
     */
    const actions = useMemo(() => {
        if (!routine?.nodes?.length) return null;
        return (
            <Grid container spacing={1}>
                <Grid item xs={12} sm={6}>
                    <Button startIcon={<GraphIcon />} fullWidth onClick={viewGraph} color="secondary">View Graph</Button>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <Button startIcon={<StartIcon />} fullWidth onClick={runRoutine} color="secondary">Start Now</Button>
                </Grid>
            </Grid>
        )
    }, [routine, viewGraph, runRoutine]);

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
                    {/* Metadata */}
                    {Array.isArray(routine?.nodes) && (routine as any).nodes.length > 0 && <Box sx={{
                        padding: 1,
                        borderRadius: 1,
                    }}>
                        {/* TODO click to view */}
                        <Typography variant="h6">Metadata</Typography>
                        <Typography variant="body1">Status: TODO</Typography>
                        <Typography variant="body1">Complexity: {routine?.complexity}</Typography>

                    </Box>}
                    {/* Resources */}
                    {Array.isArray(routine?.resourceLists) && (routine?.resourceLists as ResourceList[]).length > 0 ? <ResourceListHorizontal
                        title={'Resources'}
                        list={(routine as any).resourceLists[0]}
                        canEdit={false}
                        handleUpdate={() => { }} // Intentionally blank
                        session={session}
                    /> : null}
                    {/* Description */}
                    <Box sx={{
                        padding: 1,
                        borderRadius: 1,
                        color: Boolean(instructions) ? 'text.primary' : 'text.secondary',
                    }}>
                        <Typography variant="h6">Description</Typography>
                        <Typography variant="body1" sx={{ color: description ? 'black' : 'gray' }}>{description ?? 'No description set'}</Typography>
                    </Box>
                    {/* Instructions */}
                    <Box sx={{
                        padding: 1,
                        borderRadius: 1,
                        color: Boolean(instructions) ? 'text.primary' : 'text.secondary',
                    }}>
                        <Typography variant="h6">Instructions</Typography>
                        <Markdown>{instructions ?? 'No instructions'}</Markdown>
                    </Box>
                </Stack>
                {/* Actions */}
                {actions}
            </>
        )
    }, [loading, actions, description, instructions]);

    return (
        <Box sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            margin: 'auto',
            minHeight: '88vh',
        }}>
            {/* Delete routine confirmation dialog */}
            <DeleteRoutineDialog
                isOpen={deleteOpen}
                routineName={getTranslation(routine, 'title', ['en']) ?? ''}
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
                <RunRoutineView
                    handleClose={stopRoutine}
                    session={session}
                />
            </Dialog>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                handleActionComplete={() => { }} //TODO
                handleDelete={() => { }} //TODO
                handleEdit={onEdit}
                objectId={id}
                objectType={'Routine'}
                anchorEl={moreMenuAnchor}
                title='Routine Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
            />
            {/* Main container */}
            <Box sx={{
                background: (t) => t.palette.background.paper,
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
                    background: (t) => t.palette.primary.main,
                    color: (t) => t.palette.primary.contrastText,
                }}>
                    {/* Show star button and ellipsis next to title */}
                    <Stack direction="row" spacing={1} alignItems="center">
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
                        <Typography variant="h5">{title}</Typography>
                        {canEdit && <Tooltip title="Edit routine">
                            <IconButton
                                aria-label="Edit routine"
                                size="small"
                                onClick={onEdit}
                            >
                                <EditIcon sx={{ fill: TERTIARY_COLOR }} />
                            </IconButton>
                        </Tooltip>}
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
                                <EllipsisIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    <Stack direction="row" spacing={1}>
                        {ownedBy && (
                            <Link onClick={toOwner}>
                                <Typography variant="body1" sx={{ color: (t) => t.palette.primary.contrastText, cursor: 'pointer' }}>{ownedBy} - </Typography>
                            </Link>
                        )}
                        <Typography variant="body1">{routine?.version}</Typography>
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