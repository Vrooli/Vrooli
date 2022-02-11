import { Box, IconButton, Stack, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, MemberRole } from "@local/shared";
import { useQuery } from "@apollo/client";
import { standard, standard_standard_creator_Organization, standard_standard_creator_User } from "graphql/generated/standard";
import { standardQuery } from "graphql/query";
import { MouseEvent, useCallback, useMemo, useState } from "react";
import {
    CardGiftcard as DonateIcon,
    MoreHoriz as EllipsisIcon,
    Person as PersonIcon,
    Share as ShareIcon,
    StarOutline as StarOutlineIcon,
} from "@mui/icons-material";
import { containerShadow } from "styles";
import { StandardViewProps } from "../types";
import { BaseObjectActionDialog } from "components";
import { BaseObjectAction } from "components/dialogs/types";

export const StandardView = ({
    session,
    partialData,
}: StandardViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Standard}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchStandards}/view/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch data
    const { data, loading } = useQuery<standard>(standardQuery, { variables: { input: { id } } });
    const standard = useMemo(() => data?.standard, [data]);
    const canEdit: boolean = useMemo(() => [MemberRole.Admin, MemberRole.Owner].includes(standard?.role ?? ''), [standard]);

    const contributedBy = useMemo(() => {
        if (!standard || !standard.creator) return null;
        const creator = standard.creator;
        if (creator.__typename === 'User') {
            return (creator as standard_standard_creator_User).username;
        }
        return (creator as standard_standard_creator_Organization).name;
    }, [standard])

    /**
     * Displays name, avatar, bio, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            width={'min(500px, 100vw)'}
            borderRadius={2}
            ml='auto'
            mr='auto'
            mt={8}
            bgcolor={(t) => t.palette.background.paper}
            sx={{ ...containerShadow }}
        >
            <Box
                width={'min(100px, 25vw)'}
                height={'min(100px, 25vw)'}
                borderRadius='100%'
                border={(t) => `4px solid ${t.palette.primary.dark}`}
                bgcolor='#939eb9'
                position='absolute'
                display='flex'
                justifyContent='center'
                alignItems='center'
                left='50%'
                top='max(-50px, -12vw)'
                sx={{ transform: 'translateX(-50%)' }}
            >
                <PersonIcon sx={{
                    fill: '#cfd0d1',
                    width: '80%',
                    height: '80%',
                }} />
            </Box>
            <Stack direction="row" padding={1}>
                <Tooltip title="Favorite organization">
                    <IconButton aria-label="Favorite" size="small">
                        <StarOutlineIcon onClick={() => { }} />
                    </IconButton>
                </Tooltip>
                <Tooltip title="See all options">
                    <IconButton aria-label="More" size="small" edge="end">
                        <EllipsisIcon onClick={openMoreMenu} />
                    </IconButton>
                </Tooltip>
            </Stack>
            <Stack direction="column" spacing={1} mt={5}>
                <Typography variant="h4" textAlign="center">{standard?.name ?? partialData?.name}</Typography>
                <Typography variant="h4" textAlign="center">Submitted by: {contributedBy}</Typography>
                <Typography variant="body1" sx={{ color: (standard?.description || partialData?.description) ? 'black' : 'gray' }}>{standard?.description ?? partialData?.description ?? 'No description set'}</Typography>
                <Stack direction="row" spacing={4} alignItems="center">
                    <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small">
                            <DonateIcon onClick={() => { }} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Share">
                        <IconButton aria-label="Share" size="small">
                            <ShareIcon onClick={() => { }} />
                        </IconButton>
                    </Tooltip>
                </Stack>
            </Stack>
        </Box>
    ), [standard])

    // Determine options available to object, in order
    const moreOptions: BaseObjectAction[] = useMemo(() => {
        // Initialize
        let options: BaseObjectAction[] = [];
        if (session && !canEdit) {
            options.push(standard?.isUpvoted ? BaseObjectAction.Downvote : BaseObjectAction.Upvote);
            options.push(standard?.isStarred ? BaseObjectAction.Unstar : BaseObjectAction.Star);
            options.push(BaseObjectAction.Copy);
        }
        options.push(BaseObjectAction.Donate, BaseObjectAction.Share)
        if (session?.id) {
            options.push(BaseObjectAction.Report);
        }
        if (canEdit) {
            options.push(BaseObjectAction.Delete);
        }
        return options;
    }, [standard, session, canEdit]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                objectId={id}
                objectType={'Standard'}
                anchorEl={moreMenuAnchor}
                title='Standard Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
            />
            <Box sx={{ display: 'flex', paddingTop: 5, paddingBottom: 5, background: "#b2b3b3" }}>
                {overviewComponent}
            </Box>
        </>
    )
}