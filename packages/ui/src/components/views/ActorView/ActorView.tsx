import { Box, Grid, IconButton, Stack, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useQuery, useLazyQuery } from "@apollo/client";
import { user } from "graphql/generated/user";
import { organizationsQuery, projectsQuery, routinesQuery, standardsQuery, userQuery } from "graphql/query";
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
import { ListMenu, ResourceList, SearchBar, Selector } from "components";
import { ListMenuItemData } from "components/dialogs/types";
import { containerShadow } from "styles";
import { organizations, organizationsVariables } from "graphql/generated/organizations";
import { projects, projectsVariables } from "graphql/generated/projects";
import { routines, routinesVariables } from "graphql/generated/routines";
import { standards, standardsVariables } from "graphql/generated/standards";
import { ActorViewProps } from "../types";

enum Actions {
    Donate = "Donate",
    Report = "Report",
    Share = "Share",
    Star = "Star",
}
const moreOptionsMap: { [x: string]: [string, SvgIconComponent] } = ({
    [Actions.Star]: ['Favorite', StarOutlineIcon],
    [Actions.Share]: ['Share', ShareIcon],
    [Actions.Donate]: ['Donate', DonateIcon],
    [Actions.Report]: ['Delete', ReportIcon],
})
const moreOptions: ListMenuItemData[] = Object.keys(moreOptionsMap).map(o => ({
    label: moreOptionsMap[o][0],
    value: o,
    Icon: moreOptionsMap[o][1]
}));

enum SearchSelections {
    Organizations = "Organizations",
    Projects = "Projects",
    Routines = "Routines",
    Standards = "Standards",
}
const selectOptions = Object.keys(SearchSelections);

export const ActorView = ({
    partialData,
}: ActorViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [match, params] = useRoute(`${APP_LINKS.Profile}/:id`);
    // Fetch data
    const { data, loading } = useQuery<user>(userQuery, { variables: { input: { id: params?.id ?? '' } } });
    const user = useMemo(() => data?.user, [data]);

    const [selection, setSelection] = useState<SearchSelections>(SearchSelections.Routines);
    const makeSelection = useCallback((e) => setSelection(e.target.value), []);
    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((newValue: any) => setSearchString(newValue), []);
    // Which query is called depends on the selector
    const [getOrganizations, { data: organizationsData }] = useLazyQuery<organizations, organizationsVariables>(organizationsQuery, { variables: { input: { searchString, userId: params?.id ?? ''  } } });
    const [getProjects, { data: projectsData }] = useLazyQuery<projects, projectsVariables>(projectsQuery, { variables: { input: { searchString, userId: params?.id ?? ''  } } });
    const [getRoutines, { data: routinesData }] = useLazyQuery<routines, routinesVariables>(routinesQuery, { variables: { input: { searchString, userId: params?.id ?? ''  } } });
    const [getStandards, { data: standardsData }] = useLazyQuery<standards, standardsVariables>(standardsQuery, { variables: { input: { searchString, userId: params?.id ?? ''  } } });
    useEffect(() => { 
        console.log('refetching...'); 
        switch (selection) {
            case SearchSelections.Organizations:
                getOrganizations();
                break;
            case SearchSelections.Projects:
                getProjects();
                break;
            case SearchSelections.Routines:
                getRoutines();
                break;
            case SearchSelections.Standards:
                getStandards();
                break;
        }
    }, [searchString]);
    const [placeholder, searchData, onSearchSelect] = useMemo(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`) ;
        switch (selection) {
            case SearchSelections.Organizations:
                return [
                    "Search user's organizations...", 
                    organizationsData?.organizations, 
                    (_e, newValue) => openLink(APP_LINKS.Organization, newValue.id)
                ];
            case SearchSelections.Projects:
                return [
                    "Search user's projects...", 
                    projectsData?.projects, 
                    (_e, newValue) => openLink(APP_LINKS.Project, newValue.id)
                ];
            case SearchSelections.Routines:
                return [
                    "Search user's routines...", 
                    routinesData?.routines, 
                    (_e, newValue) => openLink(APP_LINKS.Routine, newValue.id)
                ];
            case SearchSelections.Standards:
                return [
                    "Search user's standards...", 
                    standardsData?.standards, 
                    (_e, newValue) => openLink(APP_LINKS.Standard, newValue.id)
                ];
        }
    }, [selection]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const moreMenuId = useMemo(() => `user-options-menu-${user?.id}`, [user]);
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
            case Actions.Donate:
                break;
            case Actions.Report:
                break;
        }
        closeMoreMenu();
    }, [closeMoreMenu]);

    /**
     * Displays username, avatar, bio, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            width={'min(500px, 100vw)'}
            borderRadius={2}
            ml='auto'
            mr='ato'
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
                <Tooltip title="Favorite user">
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
                <Typography variant="h4" textAlign="center">{user?.username ?? partialData?.username}</Typography>
                <Typography variant="body1" sx={{ color: user?.bio ? 'black' : 'gray' }}>{user?.bio ?? partialData?.bio ?? 'No bio set'}</Typography>
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
    ), [user])

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ListMenu
                id={moreMenuId}
                anchorEl={moreMenuAnchor}
                title='User options'
                data={moreOptions}
                onSelect={onMoreMenuSelect}
                onClose={closeMoreMenu}
            />
            {overviewComponent}
            {/* Resources pinned by the user, not you */}
            <ResourceList />
            {/* View routines, organizations, standards, and projects associated with this user */}
            <Grid container spacing={2}>
                <Grid item xs={12} sm={8}>
                    <SearchBar
                        id="user-search-bar"
                        placeholder={placeholder}
                        value={searchString}
                        onChange={updateSearch}
                        sx={{ width: 'min(100%, 600px)' }}
                    />
                </Grid>
                <Grid item xs={6} sm={2}>
                    <Selector
                        options={selectOptions}
                        getOptionLabel={(o) => o}
                        selected={selection}
                        handleChange={makeSelection}
                        fullWidth
                        inputAriaLabel="user-search-selector"
                        label=""
                    />
                </Grid>
            </Grid>
        </>
    )
}