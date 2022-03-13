import { Box, Button, CircularProgress, Grid, IconButton, Link, Stack, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, MemberRole } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { routine } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { MouseEvent, useCallback, useMemo, useState } from "react";
import {
    Close as CloseIcon,
    MoreHoriz as EllipsisIcon,
    Share as ShareIcon,
    StarOutline as StarOutlineIcon,
} from "@mui/icons-material";
import { BaseObjectActionDialog, ResourceListHorizontal, StarButton } from "components";
import { RoutineViewProps } from "../types";
import { getTranslation, Pubs } from "utils";
import { Resource, ResourceList, User } from "types";
import Markdown from "markdown-to-jsx";
import { starMutation } from "graphql/mutation";
import { mutationWrapper } from "graphql/utils/wrappers";
import { star } from "graphql/generated/star";
import { NodeType, StarFor } from "graphql/generated/globalTypes";
import { BaseObjectAction } from "components/dialogs/types";
import { containerShadow } from "styles";

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
    const canEdit: boolean = useMemo(() => [MemberRole.Admin, MemberRole.Owner].includes(routine?.role ?? ''), [routine]);
    const languages = useMemo(() => session?.languages ?? navigator.languages, [session]);

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

    const [star] = useMutation<star>(starMutation);
    /**
     * Start a routine later (i.e. star the routine)
     */
    const startLater = useCallback(() => {
        // Must be logged in
        if (!session?.id) {
            PubSub.publish(Pubs.Snack, { message: 'Must be logged in to perform this action.', severity: 'Error' });
            setLocation(`${APP_LINKS.Start}?redirect=${encodeURIComponent(window.location.pathname)}`);
            return;
        }
        if (routine?.isStarred) {
            PubSub.publish(Pubs.Snack, { message: 'Routine is already starred.', severity: 'Warning' });
        }
        else {
            mutationWrapper({
                mutation: star,
                input: { isStar: true, starFor: StarFor.Routine, forId: id },
                onSuccess: () => { PubSub.publish(Pubs.Snack, { message: 'Routine starred.' }); },
            })
        }
    }, [id, routine, session]);

    /**
     * Start a routine now (i.e. navigate to start page)
     */
    const startNow = useCallback(() => {
        // Find first node
        const firstNode = routine?.nodes?.find(node => node.type === NodeType.Start);
        if (!firstNode) {
            PubSub.publish(Pubs.Snack, { message: 'Routine invalid.', severity: 'Error' });
            return;
        }
        setLocation(`${APP_LINKS.Run}/${id}/${firstNode.id}`);
    }, [id, routine?.nodes, setLocation]);

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
     * If routine has nodes (i.e. is not just this page), display "Start Later" and "Start Now" buttons
     */
    const actions = useMemo(() => {
        if (!routine?.nodes?.length) return null;
        return (
            <Grid container spacing={1}>
                <Grid item xs={12} sm={6}>
                    <Button fullWidth onClick={startLater} color="secondary">Start Later</Button>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <Button fullWidth onClick={startNow} color="secondary">Start Now</Button>
                </Grid>
            </Grid>
        )
    }, [routine, startLater, startNow]);

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
                    {Array.isArray(routine?.resourceLists) && (routine?.resourceLists as ResourceList[]).length > 0 ? <ResourceListHorizontal
                        title={'Resources'}
                        list={(routine as any).resourceLists[0]}
                        canEdit={false}
                        handleUpdate={() => { }}
                        session={session}
                    /> : null}
                    {/* Description */}
                    <Box sx={{
                        padding: 1,
                        border: `1px solid ${(t) => t.palette.primary.dark}`,
                        borderRadius: 1,
                        color: Boolean(instructions) ? 'text.primary' : 'text.secondary',
                    }}>
                        <Typography variant="h6">Description</Typography>
                        <Typography variant="body1" sx={{ color: description ? 'black' : 'gray' }}>{description ?? 'No description set'}</Typography>
                    </Box>
                    {/* Instructions */}
                    <Box sx={{
                        padding: 1,
                        border: `1px solid ${(t) => t.palette.background.paper}`,
                        borderRadius: 1,
                        color: Boolean(instructions) ? 'text.primary' : 'text.secondary',
                    }}>
                        <Typography variant="h6">Instructions</Typography>
                        <Markdown>{instructions ?? 'No instructions'}</Markdown>
                    </Box>
                </Stack>
                {/* Action buttons */}
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
            overflowY: 'scroll',
            minHeight: '88vh',
        }}>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
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
                            isStar={routine?.isStarred ?? false}
                            stars={routine?.stars ?? 0}
                            onChange={(isStar: boolean) => { }}
                            tooltipPlacement="bottom"
                        />
                        <Typography variant="h5">{title}</Typography>
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
                        {ownedBy ? (
                            <Link onClick={toOwner}>
                                <Typography variant="body1" sx={{ color: (t) => t.palette.primary.contrastText, cursor: 'pointer' }}>{ownedBy} - </Typography>
                            </Link>
                        ) : null}
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