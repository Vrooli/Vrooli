import { Box, IconButton, LinearProgress, Link, Stack, Tab, Tabs, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, StarFor } from "@local/shared";
import { useLazyQuery } from "@apollo/client";
import { user, userVariables } from "graphql/generated/user";
import { organizationsQuery, projectsQuery, routinesQuery, standardsQuery, userQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    CardGiftcard as DonateIcon,
    Edit as EditIcon,
    MoreHoriz as EllipsisIcon,
    Person as PersonIcon,
    Share as ShareIcon,
    Today as CalendarIcon,
} from "@mui/icons-material";
import { BaseObjectActionDialog, organizationDefaultSortOption, OrganizationSortOptions, projectDefaultSortOption, ProjectSortOptions, ResourceListVertical, routineDefaultSortOption, RoutineSortOptions, SearchList, SelectLanguageDialog, standardDefaultSortOption, StandardSortOptions, StarButton } from "components";
import { containerShadow } from "styles";
import { UserViewProps } from "../types";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, Pubs } from "utils";
import { ResourceList, Standard, User } from "types";
import { BaseObjectAction } from "components/dialogs/types";
import { SearchListGenerator } from "components/lists/types";
import { validate as uuidValidate } from 'uuid';
import { ResourceListUsedFor } from "graphql/generated/globalTypes";

enum TabOptions {
    Resources = "Resources",
    Organizations = "Organizations",
    Projects = "Projects",
    Routines = "Routines",
    Standards = "Standards",
}

export const UserView = ({
    session,
    partialData,
}: UserViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [isProfile] = useRoute(`${APP_LINKS.Profile}`);
    const [, params1] = useRoute(`${APP_LINKS.Profile}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchUsers}/view/:id`);
    const id: string = useMemo(() => {
        return isProfile ? (session.id ?? '') : (params1?.id ?? params2?.id ?? '');
    }, [isProfile, params1, params2, session]);
    const isOwn: boolean = useMemo(() => Boolean(session?.id && session.id === id), [id, session]);
    // Fetch data
    const [getData, { data, loading }] = useLazyQuery<user, userVariables>(userQuery);
    const [user, setUser] = useState<User | null | undefined>(null);
    useEffect(() => {
        //TODO handle handles
        if (uuidValidate(id)) getData({ variables: { input: { id } } })
    }, [getData, id]);
    useEffect(() => {
        setUser((data?.user as User) ?? partialData);
    }, [data, isProfile, partialData]);

    const [language, setLanguage] = useState<string>('');
    const [availableLanguages, setAvailableLanguages] = useState<string[]>([]);
    useEffect(() => {
        const availableLanguages = user?.translations?.map(t => getLanguageSubtag(t.language)) ?? [];
        const userLanguages = getUserLanguages(session);
        setAvailableLanguages(availableLanguages);
        setLanguage(getPreferredLanguage(availableLanguages, userLanguages));
    }, [session, user]);

    const { bio, name, handle, resourceList } = useMemo(() => {
        const resourceList: ResourceList | undefined = Array.isArray(user?.resourceLists) ? user?.resourceLists?.find(r => r.usedFor === ResourceListUsedFor.Display) : undefined;
        return {
            bio: getTranslation(user, 'bio', [language]) ?? getTranslation(partialData, 'bio', [language]),
            name: user?.name ?? partialData?.name,
            handle: user?.handle ?? partialData?.handle,
            resourceList,
        };
    }, [language, partialData, user]);

    useEffect(() => {
        if (handle) document.title = `${name} ($${handle}) | Vrooli`;
        else document.title = `${name} | Vrooli`;
    }, [handle, name]);

    const resources = useMemo(() => (resourceList || isOwn) ? (
        <ResourceListVertical
            list={resourceList}
            session={session}
            canEdit={isOwn}
            handleUpdate={(updatedList) => {
                if (!user) return;
                setUser({
                    ...user,
                    resourceLists: [updatedList]
                })
            }}
            mutate={true}
        />
    ) : null, [isOwn, resourceList, session, user]);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(0);
    const handleTabChange = (event, newValue) => { setTabIndex(newValue) };

    /**
     * Calculate which tabs to display
     */
    const availableTabs = useMemo(() => {
        const tabs: TabOptions[] = [];
        // Only display resources if there are any
        if (resources) tabs.push(TabOptions.Resources);
        // Always display others (for now)
        tabs.push(TabOptions.Organizations);
        tabs.push(TabOptions.Projects);
        tabs.push(TabOptions.Routines);
        tabs.push(TabOptions.Standards);
        return tabs;
    }, [resources]);

    const currTabType = useMemo(() => tabIndex >= 0 && tabIndex < availableTabs.length ? availableTabs[tabIndex] : null, [availableTabs, tabIndex]);

    const shareLink = () => {
        navigator.clipboard.writeText(`https://vrooli.com${APP_LINKS.Profile}/${id}`);
        PubSub.publish(Pubs.Snack, { message: 'Copied🎉' })
    }

    const onEdit = useCallback(() => {
        // Depends on if we're in a search popup or a normal organization page
        setLocation(isProfile ? `${APP_LINKS.Settings}?page=profile` : `${APP_LINKS.SearchUsers}/edit/${id}`);
    }, [id, isProfile, setLocation]);

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

    // Create search data
    const { itemKeyPrefix, placeholder, sortOptions, defaultSortOption, searchQuery, where, noResultsText, onSearchSelect } = useMemo<SearchListGenerator>(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`);
        // The first tab doesn't have search results, as it is the user's set resources
        switch (currTabType) {
            case TabOptions.Organizations:
                return {
                    itemKeyPrefix: 'organization-list-item',
                    placeholder: "Search user's organizations...",
                    noResultsText: "No organizations found",
                    sortOptions: OrganizationSortOptions,
                    defaultSortOption: organizationDefaultSortOption,
                    searchQuery: organizationsQuery,
                    where: { userId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Organization, newValue.id),
                }
            case TabOptions.Projects:
                return {
                    itemKeyPrefix: 'project-list-item',
                    placeholder: "Search user's projects...",
                    noResultsText: "No projects found",
                    sortOptions: ProjectSortOptions,
                    defaultSortOption: projectDefaultSortOption,
                    searchQuery: projectsQuery,
                    where: { userId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Project, newValue.id),
                }
            case TabOptions.Routines:
                return {
                    itemKeyPrefix: 'routine-list-item',
                    placeholder: "Search user's routines...",
                    noResultsText: "No routines found",
                    sortOptions: RoutineSortOptions,
                    defaultSortOption: routineDefaultSortOption,
                    searchQuery: routinesQuery,
                    where: { userId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Run, newValue.id),
                }
            case TabOptions.Standards:
                return {
                    itemKeyPrefix: 'standard-list-item',
                    placeholder: "Search user's standards...",
                    noResultsText: "No standards found",
                    sortOptions: StandardSortOptions,
                    defaultSortOption: standardDefaultSortOption,
                    searchQuery: standardsQuery,
                    where: { userId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Standard, newValue.id),
                }
            default:
                return {
                    itemKeyPrefix: '',
                    placeholder: '',
                    noResultsText: '',
                    sortOptions: [],
                    defaultSortOption: { label: '', value: null },
                    searchQuery: null,
                    where: {},
                    onSearchSelect: (o: any) => { },
                }
        }
    }, [currTabType, id, session]);

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
     * Displays name, handle, avatar, bio, and quick links
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
                {/* Title */}
                {
                    loading ? (
                        <Stack sx={{ width: '50%', color: 'grey.500', paddingTop: 2, paddingBottom: 2 }} spacing={2}>
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : isOwn ? (
                        <Stack direction="row" alignItems="center" justifyContent="center">
                            <Typography variant="h4" textAlign="center">{name}</Typography>
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
                        <Typography variant="h4" textAlign="center">{name}</Typography>
                    )
                }
                {/* Handle */}
                {
                    handle && <Link href={`https://handle.me/${handle}`} underline="hover">
                        <Typography
                            variant="h6"
                            textAlign="center"
                            sx={{
                                color: (t) => t.palette.secondary.dark,
                                cursor: 'pointer',
                            }}
                        >${handle}</Typography>
                    </Link>
                }
                {/* Joined date */}
                {
                    loading ? (
                        <Box sx={{ width: '33%', color: "#00831e" }}>
                            <LinearProgress color="inherit" />
                        </Box>
                    ) :
                        user?.created_at && (<Box sx={{ display: 'flex' }} >
                            <CalendarIcon />
                            {`Joined ${new Date(user.created_at).toLocaleDateString(navigator.language, { year: 'numeric', month: 'long' })}`}
                        </Box>)
                }
                {/* Description */}
                {
                    loading ? (
                        <Stack sx={{ width: '85%', color: 'grey.500' }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : (
                        <Typography variant="body1" sx={{ color: bio ? 'black' : 'gray' }}>{bio ?? 'No bio set'}</Typography>
                    )
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Share">
                        <IconButton aria-label="Share" size="small" onClick={shareLink}>
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
    ), [bio, handle, isOwn, loading, name, onEdit, openMoreMenu, partialData, session, shareLink, user]);

    /**
     * Opens add new page
     */
    const toAddNew = useCallback(() => {
        switch (currTabType) {
            case TabOptions.Organizations:
                setLocation(`${APP_LINKS.Organization}/add`);
                break;
            case TabOptions.Projects:
                setLocation(`${APP_LINKS.Project}/add`);
                break;
            case TabOptions.Routines:
                // setLocation(`${APP_LINKS.Routine}/add`);TODO
                break;
            case TabOptions.Standards:
                setLocation(`${APP_LINKS.Standard}/add`);
                break;
        }
    }, [currTabType]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                handleActionComplete={() => { }} //TODO
                handleDelete={() => { }} //TODO
                handleEdit={onEdit}
                objectId={id}
                objectType={'User'}
                anchorEl={moreMenuAnchor}
                title='User Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
                session={session}
            />
            <Box sx={{
                display: 'flex',
                paddingTop: 5,
                paddingBottom: 5,
                background: "#b2b3b3",
                position: "relative",
            }}>
                {/* Language display/select */}
                <Box sx={{
                    position: 'absolute',
                    top: 8,
                    right: 8,
                }}>
                    <SelectLanguageDialog
                        availableLanguages={availableLanguages}
                        canDropdownOpen={availableLanguages.length > 1}
                        handleSelect={setLanguage}
                        language={language}
                        session={session}
                    />
                </Box>
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
                            justifyContent: 'space-around',
                        },
                    }}
                >
                    {availableTabs.map((tabType, index) => (
                        <Tab
                            key={index}
                            id={`profile-tab-${index}`}
                            {...{ 'aria-controls': `profile-tabpanel-${index}` }}
                            label={<span style={{ color: tabType === TabOptions.Resources ? '#8e6b00' : 'default' }}>{tabType}</span>}
                        />
                    ))}
                </Tabs>
                <Box p={2}>
                    {
                        currTabType === TabOptions.Resources ? resources : (
                            <SearchList
                                defaultSortOption={defaultSortOption}
                                handleAdd={toAddNew}
                                itemKeyPrefix={itemKeyPrefix}
                                noResultsText={noResultsText}
                                onObjectSelect={onSearchSelect}
                                query={searchQuery}
                                searchPlaceholder={placeholder}
                                searchString={searchString}
                                session={session}
                                setSearchString={setSearchString}
                                setSortBy={setSortBy}
                                setTimeFrame={setTimeFrame}
                                sortBy={sortBy}
                                sortOptions={sortOptions}
                                take={20}
                                timeFrame={timeFrame}
                                where={where}
                            />
                        )
                    }
                </Box>
            </Box>
        </>
    )
}