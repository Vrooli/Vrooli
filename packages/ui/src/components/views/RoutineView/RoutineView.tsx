import { Box, Grid, IconButton, Stack, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useQuery, useLazyQuery } from "@apollo/client";
import { routine } from "graphql/generated/routine";
import { usersQuery, organizationsQuery, routineQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    CardGiftcard as DonateIcon,
    MoreHoriz as EllipsisIcon,
    Person as PersonIcon,
    ReportProblem as ReportIcon,
    Share as ShareIcon,
    Star as StarIcon,
    StarOutline as StarOutlineIcon,
    SvgIconComponent
} from "@mui/icons-material";
import { ListMenu, SearchBar } from "components";
import { ListMenuItemData } from "components/dialogs/types";
import { containerShadow } from "styles";
import { RoutineViewProps } from "../types";
import { users, usersVariables } from "graphql/generated/users";
import { organizations, organizationsVariables } from "graphql/generated/organizations";

enum Actions {
    Report = "Report",
    Share = "Share",
    Star = "Star",
}
const moreOptionsMap: { [x: string]: [string, SvgIconComponent] } = ({
    [Actions.Star]: ['Favorite', StarOutlineIcon],
    [Actions.Share]: ['Share', ShareIcon],
    [Actions.Report]: ['Delete', ReportIcon],
})
const moreOptions: ListMenuItemData[] = Object.keys(moreOptionsMap).map(o => ({
    label: moreOptionsMap[o][0],
    value: o,
    Icon: moreOptionsMap[o][1]
}));

export const RoutineView = ({
    partialData,
}: RoutineViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Routine}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchRoutines}/view/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch data
    const { data, loading } = useQuery<routine>(routineQuery, { variables: { input: { id } } });
    const routine = useMemo(() => data?.routine, [data]);

    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((newValue: any) => setSearchString(newValue), []);
    const [getUsers, { data: usersData }] = useLazyQuery<users, usersVariables>(usersQuery, { variables: { input: { searchString, routineId: params?.id ?? '' } } });
    const [getOrganizations, { data: organizationsData }] = useLazyQuery<organizations, organizationsVariables>(organizationsQuery, { variables: { input: { searchString, routineId: params?.id ?? '' } } });
    useEffect(() => {
        console.log('refetching...');
        getUsers();
        getOrganizations();
    }, [searchString]);
    const usersSearchData = useMemo(() => usersData?.users ?? [], [usersData]);
    const organizationsSearchData = useMemo(() => organizationsData?.organizations ?? [], [organizationsData]);
    const onUserSelect = (_e, newValue) => setLocation(`${APP_LINKS.Profile}/${newValue.id}`);
    const onOrganizationSelect = (_e, newValue) => setLocation(`${APP_LINKS.Organization}/${newValue.id}`);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const moreMenuId = useMemo(() => `routine-options-menu-${routine?.id}`, [routine]);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);
    const onMoreMenuSelect = useCallback((value: string) => {
        console.log('onMoreMenuSelect', value);
        switch (value) {
            case Actions.Star:
                break;
            case Actions.Share:
                break;
            case Actions.Report:
                break;
        }
        closeMoreMenu();
    }, [closeMoreMenu]);

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
                <Typography variant="h4" textAlign="center">{routine?.title ?? partialData?.title}</Typography>
                <Typography variant="body1" sx={{ color: (routine?.description || partialData?.description) ? 'black' : 'gray' }}>{routine?.description ?? partialData?.description ?? 'No description set'}</Typography>
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
    ), [routine])

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ListMenu
                id={moreMenuId}
                anchorEl={moreMenuAnchor}
                title='Routine options'
                data={moreOptions}
                onSelect={onMoreMenuSelect}
                onClose={closeMoreMenu}
            />
            {overviewComponent}
            {/* View routine contributors */}
            <Typography variant="h4" textAlign="center">Contributors</Typography>
            <Grid container spacing={2}>
                <Grid item xs={12}>
                    <SearchBar
                        id="routine-search-bar"
                        placeholder="Search routine's contributors..."
                        value={searchString}
                        onChange={updateSearch}
                        sx={{ width: 'min(100%, 600px)' }}
                    />
                </Grid>
            </Grid>
        </>
    )
}