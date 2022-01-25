import { Box, IconButton, Stack, Tab, Tabs, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useQuery } from "@apollo/client";
import { user } from "graphql/generated/user";
import { organizationsQuery, projectsQuery, routinesQuery, standardsQuery, userQuery } from "graphql/query";
import { MouseEvent, useCallback, useMemo, useState } from "react";
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
import { ListMenu, organizationDefaultSortOption, OrganizationListItem, organizationOptionLabel, OrganizationSortOptions, projectDefaultSortOption, ProjectListItem, projectOptionLabel, ProjectSortOptions, routineDefaultSortOption, RoutineListItem, routineOptionLabel, RoutineSortOptions, SearchList, standardDefaultSortOption, StandardListItem, standardOptionLabel, StandardSortOptions } from "components";
import { ListMenuItemData } from "components/dialogs/types";
import { containerShadow } from "styles";
import { ActorViewProps } from "../types";
import { LabelledSortOption } from "utils";
import { Organization, Project, RoutineDeep, Standard } from "types";

const tabLabels = ['Resources', 'Organizations', 'Projects', 'Routines', 'Standards'];

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

export const ActorView = ({
    session,
    partialData,
}: ActorViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Profile}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchUsers}/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch data
    const { data, loading } = useQuery<user>(userQuery, { variables: { input: { id } } });
    const user = useMemo(() => data?.user, [data]);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(0);
    const handleTabChange = (event, newValue) => { setTabIndex(newValue) };

    // Create search data
    const [
        placeholder, 
        sortOptions, 
        defaultSortOption, 
        sortOptionLabel, 
        searchQuery, 
        onSearchSelect,
        searchItemFactory,
    ] = useMemo<[string, LabelledSortOption<any>[], {label: string, value: any}, (o: any) => string, any, (objectData: any) => void, (node: any, index: number) => JSX.Element]>(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`);
        // The first tab doesn't have search results, as it is the user's set resources
        switch (tabIndex) {
            case 1:
                return [
                    "Search user's organizations...",
                    OrganizationSortOptions,
                    organizationDefaultSortOption,
                    organizationOptionLabel,
                    organizationsQuery,
                    (newValue) => openLink(APP_LINKS.Organization, newValue.id),
                    (node: Organization, index: number) => (
                        <OrganizationListItem
                            key={`organization-list-item-${index}`}
                            session={session}
                            data={node}
                            isOwn={false}
                            onClick={(selected: Organization) => openLink(APP_LINKS.Organization, selected.id)}
                        />)
                ];
            case 2:
                return [
                    "Search user's projects...",
                    ProjectSortOptions,
                    projectDefaultSortOption,
                    projectOptionLabel,
                    projectsQuery,
                    (newValue) => openLink(APP_LINKS.Project, newValue.id),
                    (node: Project, index: number) => (
                        <ProjectListItem
                            key={`project-list-item-${index}`}
                            session={session}
                            data={node}
                            isOwn={false}
                            onClick={(selected: Project) => openLink(APP_LINKS.Project, selected.id)}
                        />)
                ];
            case 3:
                return [
                    "Search user's routines...",
                    RoutineSortOptions,
                    routineDefaultSortOption,
                    routineOptionLabel,
                    routinesQuery,
                    (newValue) => openLink(APP_LINKS.Routine, newValue.id),
                    (node: RoutineDeep, index: number) => (
                        <RoutineListItem
                            key={`routine-list-item-${index}`}
                            session={session}
                            data={node}
                            isOwn={false}
                            onClick={(selected: RoutineDeep) => openLink(APP_LINKS.Routine, selected.id)}
                        />)
                ];
            case 4:
                return [
                    "Search user's standards...",
                    StandardSortOptions,
                    standardDefaultSortOption,
                    standardOptionLabel,
                    standardsQuery,
                    (newValue) => openLink(APP_LINKS.Standard, newValue.id),
                    (node: Standard, index: number) => (
                        <StandardListItem
                            key={`standard-list-item-${index}`}
                            session={session}
                            data={node}
                            isOwn={false}
                            onClick={(selected: Standard) => openLink(APP_LINKS.Standard, selected.id)}
                        />)
                ];
            default:
                return [
                    '', 
                    [], 
                    {label: '', value: null}, 
                    () => '', 
                    null,
                    () => { },
                    () => (<></>)
                ];
        }
    }, [tabIndex]);

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
            {/* View routines, organizations, standards, and projects associated with this user */}
            <Tabs
                value={tabIndex}
                onChange={handleTabChange}
                indicatorColor="secondary"
                textColor="inherit"
                variant="scrollable"
                scrollButtons="auto"
                allowScrollButtonsMobile
                aria-label="site-statistics-tabs"
                sx={{ marginBottom: 2 }}
            >
                {tabLabels.map((label, index) => (
                    <Tab
                        key={index}
                        id={`profile-tab-${index}`}
                        {...{ 'aria-controls': `profile-tabpanel-${index}` }}
                        label={label}
                        color={index === 0 ? '#ce6c12' : 'default'}
                    />
                ))}
            </Tabs>
            {
                tabIndex === 0 ? (
                    <></>
                ) : (
                    <SearchList
                        searchPlaceholder={placeholder}
                        sortOptions={sortOptions}
                        defaultSortOption={defaultSortOption}
                        query={searchQuery}
                        take={20}
                        listItemFactory={searchItemFactory}
                        getOptionLabel={sortOptionLabel}
                        onObjectSelect={onSearchSelect}
                    />
                )
            }
        </>
    )
}