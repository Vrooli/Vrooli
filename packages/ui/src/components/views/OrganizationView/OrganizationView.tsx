import { Box, Grid, IconButton, Stack, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useQuery, useLazyQuery } from "@apollo/client";
import { organization } from "graphql/generated/organization";
import { organizationQuery, usersQuery, projectsQuery, routinesQuery, standardsQuery } from "graphql/query";
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
import { users, usersVariables } from "graphql/generated/users";
import { projects, projectsVariables } from "graphql/generated/projects";
import { routines, routinesVariables } from "graphql/generated/routines";
import { standards, standardsVariables } from "graphql/generated/standards";
import { OrganizationViewProps } from "../types";

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
    Users = "Users",
    Projects = "Projects",
    Routines = "Routines",
    Standards = "Standards",
}
const selectOptions = Object.keys(SearchSelections);

export const OrganizationView = ({
    partialData,
}: OrganizationViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [match, params] = useRoute(`${APP_LINKS.Organization}/:id`);
    // Fetch data
    const { data, loading } = useQuery<organization>(organizationQuery, { variables: { input: { id: params?.id ?? '' } } });
    const organization = useMemo(() => data?.organization, [data]);

    const [selection, setSelection] = useState<SearchSelections>(SearchSelections.Routines);
    const makeSelection = useCallback((e) => setSelection(e.target.value), []);
    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((newValue: any) => setSearchString(newValue), []);
    // Which query is called depends on the selector
    const [getUsers, { data: usersData }] = useLazyQuery<users, usersVariables>(usersQuery, { variables: { input: { searchString, organizationId: params?.id ?? ''  } } });
    const [getProjects, { data: projectsData }] = useLazyQuery<projects, projectsVariables>(projectsQuery, { variables: { input: { searchString, organizationId: params?.id ?? ''  } } });
    const [getRoutines, { data: routinesData }] = useLazyQuery<routines, routinesVariables>(routinesQuery, { variables: { input: { searchString, organizationId: params?.id ?? ''  } } });
    const [getStandards, { data: standardsData }] = useLazyQuery<standards, standardsVariables>(standardsQuery, { variables: { input: { searchString, organizationId: params?.id ?? ''  } } });
    useEffect(() => { 
        console.log('refetching...'); 
        switch (selection) {
            case SearchSelections.Users:
                getUsers();
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
            case SearchSelections.Users:
                return [
                    "Search organization's members...", 
                    usersData?.users, 
                    (_e, newValue) => openLink(APP_LINKS.Organization, newValue.id)
                ];
            case SearchSelections.Projects:
                return [
                    "Search organization's projects...", 
                    projectsData?.projects, 
                    (_e, newValue) => openLink(APP_LINKS.Project, newValue.id)
                ];
            case SearchSelections.Routines:
                return [
                    "Search organization's routines...", 
                    routinesData?.routines, 
                    (_e, newValue) => openLink(APP_LINKS.Routine, newValue.id)
                ];
            case SearchSelections.Standards:
                return [
                    "Search organization's standards...", 
                    standardsData?.standards, 
                    (_e, newValue) => openLink(APP_LINKS.Standard, newValue.id)
                ];
        }
    }, [selection]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const moreMenuId = useMemo(() => `organization-options-menu-${organization?.id}`, [organization]);
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
     * Displays name, avatar, bio, and quick links
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
                <Typography variant="h4" textAlign="center">{organization?.name ?? partialData?.name}</Typography>
                <Typography variant="body1" sx={{ color: (organization?.bio || partialData?.bio) ? 'black' : 'gray' }}>{organization?.bio ?? partialData?.bio ?? 'No bio set'}</Typography>
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
    ), [organization])

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ListMenu
                id={moreMenuId}
                anchorEl={moreMenuAnchor}
                title='Organization options'
                data={moreOptions}
                onSelect={onMoreMenuSelect}
                onClose={closeMoreMenu}
            />
            {overviewComponent}
            {/* Resources pinned by the organization, not you */}
            <ResourceList />
            {/* View routines, members, standards, and projects associated with this organization */}
            <Grid container spacing={2}>
                <Grid item xs={12} sm={8}>
                    <SearchBar
                        id="organization-search-bar"
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
                        inputAriaLabel="organization-search-selector"
                        label=""
                    />
                </Grid>
            </Grid>
        </>
    )
}