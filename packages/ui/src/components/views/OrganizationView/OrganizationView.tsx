import { Box, IconButton, LinearProgress, Link, Stack, Tab, Tabs, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { adaHandleRegex, APP_LINKS, StarFor } from "@local/shared";
import { useLazyQuery } from "@apollo/client";
import { organization, organizationVariables } from "graphql/generated/organization";
import { usersQuery, projectsQuery, routinesQuery, standardsQuery, organizationQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    Apartment as ProfileIcon,
    CardGiftcard as DonateIcon,
    Edit as EditIcon,
    MoreHoriz as EllipsisIcon,
    Share as ShareIcon,
    Today as CalendarIcon,
} from "@mui/icons-material";
import { userDefaultSortOption, UserSortOptions, BaseObjectActionDialog, projectDefaultSortOption, ProjectSortOptions, routineDefaultSortOption, RoutineSortOptions, SearchList, SelectLanguageDialog, standardDefaultSortOption, StandardSortOptions, StarButton } from "components";
import { containerShadow } from "styles";
import { OrganizationViewProps } from "../types";
import { Organization, ResourceList } from "types";
import { BaseObjectAction } from "components/dialogs/types";
import { SearchListGenerator } from "components/lists/types";
import { displayDate, getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, placeholderColor, Pubs } from "utils";
import { ResourceListVertical } from "components/lists";
import { validate as uuidValidate } from 'uuid';
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { owns } from "utils/authentication";

enum TabOptions {
    Resources = "Resources",
    Members = "Members",
    Projects = "Projects",
    Routines = "Routines",
    Standards = "Standards",
}

export const OrganizationView = ({
    partialData,
    session,
}: OrganizationViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Organization}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchOrganizations}/view/:id`);
    const id: string = useMemo(() => params?.id ?? params2?.id ?? '', [params, params2]);
    // Fetch data
    const [getData, { data, loading }] = useLazyQuery<organization, organizationVariables>(organizationQuery);
    const [organization, setOrganization] = useState<Organization | null | undefined>(null);
    useEffect(() => {
        if (uuidValidate(id)) getData({ variables: { input: { id } } })
        else if (adaHandleRegex.test(id)) getData({ variables: { input: { handle: id } } })
    }, [getData, id]);
    useEffect(() => {
        setOrganization(data?.organization);
    }, [data]);
    const canEdit = useMemo<boolean>(() => owns(organization?.role), [organization]);

    const [language, setLanguage] = useState<string>('');
    const [availableLanguages, setAvailableLanguages] = useState<string[]>([]);
    useEffect(() => {
        const availableLanguages = organization?.translations?.map(t => getLanguageSubtag(t.language)) ?? [];
        const userLanguages = getUserLanguages(session);
        setAvailableLanguages(availableLanguages);
        setLanguage(getPreferredLanguage(availableLanguages, userLanguages));
    }, [organization, session]);

    const { bio, handle, name, resourceList } = useMemo(() => {
        const resourceList: ResourceList | undefined = Array.isArray(organization?.resourceLists) ? organization?.resourceLists?.find(r => r.usedFor === ResourceListUsedFor.Display) : undefined;
        return {
            bio: getTranslation(organization, 'bio', [language]) ?? getTranslation(partialData, 'bio', [language]),
            handle: organization?.handle ?? partialData?.handle,
            name: getTranslation(organization, 'name', [language]) ?? getTranslation(partialData, 'name', [language]),
            resourceList,
        };
    }, [language, organization, partialData]);

    useEffect(() => {
        if (handle) document.title = `${name} ($${handle}) | Vrooli`;
        else document.title = `${name} | Vrooli`;
    }, [handle, name]);

    const resources = useMemo(() => (resourceList || canEdit) ? (
        <ResourceListVertical
            list={resourceList as any}
            session={session}
            canEdit={canEdit}
            handleUpdate={(updatedList) => {
                if (!organization) return;
                setOrganization({
                    ...organization,
                    resourceLists: [updatedList]
                })
            }}
            loading={loading}
            mutate={true}
        />
    ) : null, [canEdit, loading, organization, resourceList, session]);

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
        tabs.push(TabOptions.Members);
        tabs.push(TabOptions.Projects);
        tabs.push(TabOptions.Routines);
        tabs.push(TabOptions.Standards);
        return tabs;
    }, [resources]);

    const currTabType = useMemo(() => tabIndex >= 0 && tabIndex < availableTabs.length ? availableTabs[tabIndex] : null, [availableTabs, tabIndex]);

    const shareLink = useCallback(() => {
        navigator.clipboard.writeText(`https://vrooli.com${APP_LINKS.Organization}/${id}`);
        PubSub.publish(Pubs.Snack, { message: 'Copied🎉' })
    }, [id]);

    const onEdit = useCallback(() => {
        // Depends on if we're in a search popup or a normal organization page
        setLocation(Boolean(params?.id) ? `${APP_LINKS.Organization}/edit/${id}` : `${APP_LINKS.SearchOrganizations}/edit/${id}`);
    }, [setLocation, params?.id, id]);

    // Create search data
    const { itemKeyPrefix, placeholder, sortOptions, defaultSortOption, searchQuery, where, noResultsText, onSearchSelect } = useMemo<SearchListGenerator>(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`);
        switch (currTabType) {
            case TabOptions.Members:
                return {
                    itemKeyPrefix: 'member-list-item',
                    placeholder: "Search orgnization's members...",
                    noResultsText: "No members found",
                    sortOptions: UserSortOptions,
                    defaultSortOption: userDefaultSortOption,
                    searchQuery: usersQuery,
                    where: { organizationId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Profile, newValue.id),
                };
            case TabOptions.Projects:
                return {
                    itemKeyPrefix: 'project-list-item',
                    placeholder: "Search organization's projects...",
                    noResultsText: "No projects found",
                    sortOptions: ProjectSortOptions,
                    defaultSortOption: projectDefaultSortOption,
                    searchQuery: projectsQuery,
                    where: { organizationId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Project, newValue.id),
                };
            case TabOptions.Routines:
                return {
                    itemKeyPrefix: 'routine-list-item',
                    placeholder: "Search organization's routines...",
                    noResultsText: "No routines found",
                    sortOptions: RoutineSortOptions,
                    defaultSortOption: routineDefaultSortOption,
                    searchQuery: routinesQuery,
                    where: { organizationId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Routine, newValue.id),
                };
            case TabOptions.Standards:
                return {
                    itemKeyPrefix: 'standard-list-item',
                    placeholder: "Search organization's standards...",
                    noResultsText: "No standards found",
                    sortOptions: StandardSortOptions,
                    defaultSortOption: standardDefaultSortOption,
                    searchQuery: standardsQuery,
                    where: { organizationId: id },
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
    }, [currTabType, setLocation, id]);

    // Handle url search
    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<string | undefined>(undefined);
    const [timeFrame, setTimeFrame] = useState<string | undefined>(undefined);
    useEffect(() => {
        setSortBy(defaultSortOption.value ?? sortOptions.length > 0 ? sortOptions[0].value : undefined);
    }, [defaultSortOption, sortOptions]);

    // Determine options available to object, in order
    const moreOptions: BaseObjectAction[] = useMemo(() => {
        // Initialize
        let options: BaseObjectAction[] = [];
        if (session && !canEdit) {
            options.push(organization?.isStarred ? BaseObjectAction.Unstar : BaseObjectAction.Star);
        }
        options.push(BaseObjectAction.Donate, BaseObjectAction.Share)
        if (session?.id) {
            options.push(BaseObjectAction.Report);
        }
        if (canEdit) {
            options.push(BaseObjectAction.Delete);
        }
        return options;
    }, [session, canEdit, organization?.isStarred]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    /**
     * Displays name, avatar, bio, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            width={'min(500px, 100vw)'}
            position="relative"
            borderRadius={2}
            ml='auto'
            mr='auto'
            mt={3}
            bgcolor={palette.background.paper}
            sx={{ ...containerShadow }}
        >
            <Box
                width={'min(100px, 25vw)'}
                height={'min(100px, 25vw)'}
                borderRadius='100%'
                position='absolute'
                display='flex'
                justifyContent='center'
                alignItems='center'
                left='50%'
                top="-55px"
                sx={{ 
                    border: `1px solid black`,
                    backgroundColor: profileColors[0],
                    transform: 'translateX(-50%)',
                }}
            >
                <ProfileIcon sx={{
                    fill: profileColors[1],
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
                    ) : canEdit ? (
                        <Stack direction="row" alignItems="center" justifyContent="center">
                            <Typography variant="h4" textAlign="center">{name}</Typography>
                            <Tooltip title="Edit organization">
                                <IconButton
                                    aria-label="Edit organization"
                                    size="small"
                                    onClick={onEdit}
                                >
                                    <EditIcon sx={{
                                        fill: palette.mode === 'light' ? 
                                            palette.primary.main : palette.secondary.light,
                                    }} />
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
                                color: palette.secondary.dark,
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
                    ) : (
                        organization?.created_at && (<Box sx={{ display: 'flex' }} >
                            <CalendarIcon />
                            {`Joined ${displayDate(organization.created_at, false)}`}
                        </Box>)
                    )
                }
                {/* Bio */}
                {
                    loading ? (
                        <Stack sx={{ width: '85%', color: 'grey.500' }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : (
                        <Typography variant="body1" sx={{ color: Boolean(bio) ? palette.background.textPrimary : palette.background.textSecondary }}>{bio ?? 'No bio set'}</Typography>
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
                        !canEdit ? <StarButton
                            session={session}
                            objectId={organization?.id ?? ''}
                            starFor={StarFor.Organization}
                            isStar={organization?.isStarred ?? false}
                            stars={organization?.stars ?? 0}
                            onChange={(isStar: boolean) => { }}
                            tooltipPlacement="bottom"
                        /> : null
                    }
                </Stack>
            </Stack>
        </Box >
    ), [palette, profileColors, openMoreMenu, loading, canEdit, name, onEdit, handle, organization?.created_at, organization?.id, organization?.isStarred, organization?.stars, bio, shareLink, session]);

    /**
     * Opens add new page
     */
    const toAddNew = useCallback(() => {
        switch (currTabType) {
            case TabOptions.Members:
                //TODO
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
    }, [currTabType, setLocation]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                handleActionComplete={() => { }} //TODO
                handleDelete={() => { }} //TODO
                handleEdit={onEdit}
                objectId={id}
                objectType={'Organization'}
                anchorEl={moreMenuAnchor}
                title='Organization Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
                session={session}
            />
            <Box sx={{
                background: "#b2b3b3",
                display: 'flex',
                paddingTop: 5,
                paddingBottom: 5,
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
            {/* View routines, members, standards, and projects associated with this organization */}
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