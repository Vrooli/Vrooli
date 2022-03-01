import { Box, IconButton, Stack, Tab, Tabs, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, StarFor } from "@local/shared";
import { useLazyQuery, useMutation } from "@apollo/client";
import { user, userVariables } from "graphql/generated/user";
import { organizationsQuery, projectsQuery, routinesQuery, standardsQuery, userQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    CardGiftcard as DonateIcon,
    Edit as EditIcon,
    MoreHoriz as EllipsisIcon,
    Person as PersonIcon,
    Share as ShareIcon,
} from "@mui/icons-material";
import { BaseObjectActionDialog, organizationDefaultSortOption, OrganizationListItem, organizationOptionLabel, OrganizationSortOptions, projectDefaultSortOption, ProjectListItem, projectOptionLabel, ProjectSortOptions, routineDefaultSortOption, RoutineListItem, routineOptionLabel, RoutineSortOptions, SearchList, standardDefaultSortOption, StandardListItem, standardOptionLabel, StandardSortOptions, StarButton } from "components";
import { containerShadow } from "styles";
import { UserViewProps } from "../types";
import { LabelledSortOption } from "utils";
import { Organization, Project, Routine, Standard } from "types";
import { BaseObjectAction } from "components/dialogs/types";
import { SearchListGenerator } from "components/lists/types";

const tabLabels = ['Resources', 'Organizations', 'Projects', 'Routines', 'Standards'];

export const UserView = ({
    session,
    partialData,
}: UserViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [isProfile] = useRoute(`${APP_LINKS.Settings}/:params*`);
    const [, params2] = useRoute(`${APP_LINKS.SearchUsers}/view/:id`);
    const id: string = useMemo(() => {
        if (isProfile) return session?.id ?? '';
        return params2?.id ?? ''
    }, [isProfile, params2, session]);
    const isOwn: boolean = useMemo(() => Boolean(session?.id && session.id === id), [id, session]);
    // Fetch data
    const [getData, { data, loading }] = useLazyQuery<user, userVariables>(userQuery, { variables: { input: { id } } });
    const user = useMemo(() => isProfile ? partialData : data?.user, [data]);
    useEffect(() => {
        if (id && !isProfile) getData();
    }, [getData, id, isProfile]);

    console.log('isProfile', isProfile, id);

    const onEdit = useCallback(() => {
        // Depends on if we're in a search popup or a normal organization page
        setLocation(isProfile ? `${APP_LINKS.Settings}?page=profile&editing=true` : `${APP_LINKS.SearchUsers}/edit/${id}`);
    }, [setLocation, id]);

    // Determine options available to object, in order
    const moreOptions: BaseObjectAction[] = useMemo(() => {
        // Initialize
        let options: BaseObjectAction[] = [];
        if (user && session && !isOwn) {
            options.push(user.isStarred ? BaseObjectAction.Unstar : BaseObjectAction.Star);
        }
        options.push(BaseObjectAction.Donate, BaseObjectAction.Share)
        if (session?.id) {
            options.push(BaseObjectAction.Report);
        }
        if (isOwn) {
            options.push(BaseObjectAction.Delete);
        }
        return options;
    }, [user, isOwn, session]);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(0);
    const handleTabChange = (event, newValue) => { setTabIndex(newValue) };

    // Create search data
    const { placeholder, sortOptions, defaultSortOption, sortOptionLabel, searchQuery, where, noResultsText, onSearchSelect, searchItemFactory } = useMemo<SearchListGenerator>(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`);
        // The first tab doesn't have search results, as it is the user's set resources
        switch (tabIndex) {
            case 1:
                return {
                    placeholder: "Search user's organizations...",
                    noResultsText: "No organizations found",
                    sortOptions: OrganizationSortOptions,
                    defaultSortOption: organizationDefaultSortOption,
                    sortOptionLabel: organizationOptionLabel,
                    searchQuery: organizationsQuery,
                    where: { userId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Organization, newValue.id),
                    searchItemFactory: (node: Organization, index: number) => (
                        <OrganizationListItem
                            key={`organization-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            onClick={(selected: Organization) => openLink(APP_LINKS.Organization, selected.id)}
                        />)
                }
            case 2:
                return {
                    placeholder: "Search user's projects...",
                    noResultsText: "No projects found",
                    sortOptions: ProjectSortOptions,
                    defaultSortOption: projectDefaultSortOption,
                    sortOptionLabel: projectOptionLabel,
                    searchQuery: projectsQuery,
                    where: { userId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Project, newValue.id),
                    searchItemFactory: (node: Project, index: number) => (
                        <ProjectListItem
                            key={`project-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            onClick={(selected: Project) => openLink(APP_LINKS.Project, selected.id)}
                        />)
                }
            case 3:
                return {
                    placeholder: "Search user's routines...",
                    noResultsText: "No routines found",
                    sortOptions: RoutineSortOptions,
                    defaultSortOption: routineDefaultSortOption,
                    sortOptionLabel: routineOptionLabel,
                    searchQuery: routinesQuery,
                    where: { userId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Routine, newValue.id),
                    searchItemFactory: (node: Routine, index: number) => (
                        <RoutineListItem
                            key={`routine-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            onClick={(selected: Routine) => openLink(APP_LINKS.Routine, selected.id)}
                        />)
                }
            case 4:
                return {
                    placeholder: "Search user's standards...",
                    noResultsText: "No standards found",
                    sortOptions: StandardSortOptions,
                    defaultSortOption: standardDefaultSortOption,
                    sortOptionLabel: standardOptionLabel,
                    searchQuery: standardsQuery,
                    where: { userId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Standard, newValue.id),
                    searchItemFactory: (node: Standard, index: number) => (
                        <StandardListItem
                            key={`standard-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            onClick={(selected: Standard) => openLink(APP_LINKS.Standard, selected.id)}
                        />)
                }
            default:
                return {
                    placeholder: '',
                    noResultsText: '',
                    sortOptions: [],
                    defaultSortOption: { label: '', value: null },
                    sortOptionLabel: (o: any) => '',
                    searchQuery: null,
                    where: {},
                    onSearchSelect: (o: any) => { },
                    searchItemFactory: (a: any, b: any) => null
                }
        }
    }, [tabIndex]);

    // Handle url search
    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<string | undefined>(undefined);
    const [timeFrame, setTimeFrame] = useState<string | undefined>(undefined);
    useEffect(() => {
        setSortBy(defaultSortOption.value ?? sortOptions.length > 0 ? sortOptions[0].value : undefined);
    }, [defaultSortOption, sortOptions]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    /**
     * Displays username, avatar, bio, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            width={'min(500px, 100vw)'}
            position="relative"
            borderRadius={2}
            ml='auto'
            mr='auto'
            mt={3}
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
                top="-55px"
                sx={{ transform: 'translateX(-50%)' }}
            >
                <PersonIcon sx={{
                    fill: '#cfd0d1',
                    width: '80%',
                    height: '80%',
                }} />
            </Box>
            <Tooltip title="See all options">
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
                    <EllipsisIcon />
                </IconButton>
            </Tooltip>
            <Stack direction="column" spacing={1} p={1} alignItems="center" justifyContent="center">
                {
                    isOwn ? (
                        <Stack direction="row" alignItems="center" justifyContent="center">
                            <Typography variant="h4" textAlign="center">{user?.username ?? partialData?.username}</Typography>
                            <Tooltip title="Edit profile">
                                <IconButton
                                    aria-label="Edit profile"
                                    size="small"
                                    onClick={onEdit}
                                >
                                    <EditIcon color="primary" />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                    ) : (
                        <Typography variant="h4" textAlign="center">{user?.username ?? partialData?.username}</Typography>

                    )
                }
                <Typography variant="body1" sx={{ color: "#00831e" }}>{user?.created_at ? `🕔 Joined ${new Date(user.created_at).toDateString()}` : ''}</Typography>
                <Typography variant="body1" sx={{ color: user?.bio ? 'black' : 'gray' }}>{user?.bio ?? partialData?.bio ?? 'No bio set'}</Typography>
                <Stack direction="row" spacing={2} alignItems="center">
                    <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Share">
                        <IconButton aria-label="Share" size="small" onClick={() => { }}>
                            <ShareIcon />
                        </IconButton>
                    </Tooltip>
                    {
                        !isOwn ? <StarButton
                            session={session}
                            objectId={user?.id ?? ''}
                            starFor={StarFor.User}
                            isStar={user?.isStarred ?? false}
                            stars={user?.stars ?? 0}
                            onChange={(isStar: boolean) => { }}
                            tooltipPlacement="bottom"
                        /> : null
                    }
                </Stack>
            </Stack>
        </Box>
    ), [user, partialData, isOwn, session]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                objectId={id}
                objectType={'User'}
                anchorEl={moreMenuAnchor}
                title='User Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
            />
            <Box sx={{ display: 'flex', paddingTop: 5, paddingBottom: 5, background: "#b2b3b3" }}>
                {overviewComponent}
            </Box>
            {/* View routines, organizations, standards, and projects associated with this user */}
            <Box>
                <Tabs
                    value={tabIndex}
                    onChange={handleTabChange}
                    indicatorColor="secondary"
                    textColor="inherit"
                    variant="scrollable"
                    scrollButtons="auto"
                    allowScrollButtonsMobile
                    aria-label="site-statistics-tabs"
                    sx={{
                        marginBottom: 1,
                        '& .MuiTabs-flexContainer': {
                            justifyContent: 'space-between',
                        },
                    }}
                >
                    {tabLabels.map((label, index) => (
                        <Tab
                            key={index}
                            id={`profile-tab-${index}`}
                            {...{ 'aria-controls': `profile-tabpanel-${index}` }}
                            label={<span style={{ color: index === 0 ? '#8e6b00' : 'default' }}>{label}</span>}
                        />
                    ))}
                </Tabs>
                <Box p={2}>
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
                                searchString={searchString}
                                sortBy={sortBy}
                                timeFrame={timeFrame}
                                where={where}
                                noResultsText={noResultsText}
                                setSearchString={setSearchString}
                                setSortBy={setSortBy}
                                setTimeFrame={setTimeFrame}
                                listItemFactory={searchItemFactory}
                                getOptionLabel={sortOptionLabel}
                                onObjectSelect={onSearchSelect}
                            />
                        )
                    }
                </Box>
            </Box>
        </>
    )
}