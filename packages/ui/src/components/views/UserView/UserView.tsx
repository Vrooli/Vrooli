import { Box, IconButton, LinearProgress, Link, Stack, Tab, Tabs, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, StarFor } from "@shared/consts";
import { adaHandleRegex } from '@shared/validation';
import { useLazyQuery } from "@apollo/client";
import { user, userVariables } from "graphql/generated/user";
import { userQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu, DateDisplay, ReportsLink, ResourceListVertical, SearchList, SelectLanguageMenu, StarButton, SelectRoutineTypeMenu } from "components";
import { containerShadow } from "styles";
import { UserViewProps } from "../types";
import { base36ToUuid, getLanguageSubtag, getLastUrlPart, getPreferredLanguage, getTranslation, getUserLanguages, ObjectType, placeholderColor, SearchType } from "utils";
import { ResourceList, User } from "types";
import { SearchListGenerator } from "components/lists/types";
import { uuidValidate } from '@shared/uuid';
import { ResourceListUsedFor, VisibilityType } from "graphql/generated/globalTypes";
import { DonateIcon, EditIcon, EllipsisIcon, UserIcon } from "@shared/icons";
import { ObjectAction, ObjectActionComplete } from "components/dialogs/types";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";

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
    zIndex,
}: UserViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    // Get URL params
    const id: string = useMemo(() => {
        const pathnameEnd = base36ToUuid(getLastUrlPart());
        // If no id is provided, use the current user's id
        if (!uuidValidate(pathnameEnd)) return session.id ?? '';
        // Otherwise, use the id provided in the URL
        return pathnameEnd;
    }, [session]);
    const isOwn: boolean = useMemo(() => Boolean(session?.id && session.id === id), [id, session]);
    // Fetch data
    const [getData, { data, loading }] = useLazyQuery<user, userVariables>(userQuery, { errorPolicy: 'all' });
    const [user, setUser] = useState<User | null | undefined>(null);
    useEffect(() => {
        if (uuidValidate(id)) getData({ variables: { input: { id } } })
        else if (adaHandleRegex.test(id)) getData({ variables: { input: { handle: id } } })
    }, [getData, id]);
    useEffect(() => {
        setUser((data?.user as User) ?? partialData);
    }, [data, partialData]);

    const availableLanguages = useMemo<string[]>(() => (user?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [user?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { bio, name, handle, resourceList } = useMemo(() => {
        const resourceList: ResourceList | undefined = Array.isArray(user?.resourceLists) ? user?.resourceLists?.find(r => r.usedFor === ResourceListUsedFor.Display) : undefined;
        const bioText = getTranslation(user, 'bio', [language]) ?? getTranslation(partialData, 'bio', [language])
        return {
            bio: bioText && bioText.trim().length > 0 ? bioText : undefined,
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
            loading={loading}
            mutate={true}
            zIndex={zIndex}
        />
    ) : null, [isOwn, loading, resourceList, session, user, zIndex]);

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

    const onEdit = useCallback(() => {
        setLocation(`${APP_LINKS.Settings}?page="profile"`);
    }, [setLocation]);

    // Create search data
    const { searchType, itemKeyPrefix, placeholder, where, noResultsText, onSearchSelect } = useMemo<SearchListGenerator>(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`);
        // The first tab doesn't have search results, as it is the user's set resources
        switch (currTabType) {
            case TabOptions.Organizations:
                return {
                    searchType: SearchType.Organization,
                    itemKeyPrefix: 'organization-list-item',
                    placeholder: "Search user's organizations...",
                    noResultsText: "No organizations found",
                    where: { userId: id, visibilityType: VisibilityType.All },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Organization, newValue.id),
                }
            case TabOptions.Projects:
                return {
                    searchType: SearchType.Project,
                    itemKeyPrefix: 'project-list-item',
                    placeholder: "Search user's projects...",
                    noResultsText: "No projects found",
                    where: { userId: id, isComplete: !isOwn ? true : undefined, visibility: VisibilityType.All },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Project, newValue.id),
                }
            case TabOptions.Routines:
                return {
                    searchType: SearchType.Routine,
                    itemKeyPrefix: 'routine-list-item',
                    placeholder: "Search user's routines...",
                    noResultsText: "No routines found",
                    where: { userId: id, isComplete: !isOwn ? true : undefined, isInternal: false, visibility: VisibilityType.All },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Routine, newValue.id),
                }
            case TabOptions.Standards:
                return {
                    searchType: SearchType.Standard,
                    itemKeyPrefix: 'standard-list-item',
                    placeholder: "Search user's standards...",
                    noResultsText: "No standards found",
                    where: { userId: id, visibilityType: VisibilityType.All },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Standard, newValue.id),
                }
            default:
                return {
                    searchType: SearchType.Organization,
                    itemKeyPrefix: '',
                    placeholder: '',
                    noResultsText: '',
                    where: {},
                    onSearchSelect: (o: any) => { },
                }
        }
    }, [currTabType, id, isOwn, setLocation]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const onMoreActionStart = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Edit:
                onEdit();
                break;
            case ObjectAction.Stats:
                //TODO
                break;
        }
    }, [onEdit]);

    const onMoreActionComplete = useCallback((action: ObjectActionComplete, data: any) => {
        switch (action) {
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.star.success) {
                    setUser({
                        ...user,
                        isStarred: action === ObjectActionComplete.Star,
                    } as any)
                }
                break;
            case ObjectActionComplete.Fork:
                setLocation(`${APP_LINKS.User}/${data.fork.user.id}`);
                window.location.reload();
                break;
            case ObjectActionComplete.Copy:
                setLocation(`${APP_LINKS.User}/${data.copy.user.id}`);
                window.location.reload();
                break;
        }
    }, [user, setLocation]);

    // Menu for picking which routine type to add
    const [addRoutineAnchor, setAddRoutineAnchor] = useState<any>(null);
    const openAddRoutine = useCallback((ev: React.MouseEvent<HTMLElement>) => {
        setAddRoutineAnchor(ev.currentTarget)
    }, []);
    const closeAddRoutine = useCallback(() => setAddRoutineAnchor(null), []);

    /**
     * Displays name, handle, avatar, bio, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            position="relative"
            ml='auto'
            mr='auto'
            mt={3}
            bgcolor={palette.background.paper}
            sx={{
                borderRadius: { xs: '0', sm: 2 },
                boxShadow: { xs: 'none', sm: (containerShadow as any).boxShadow },
                width: { xs: '100%', sm: 'min(500px, 100vw)' }
            }}
        >
            <Box
                width={'min(100px, 25vw)'}
                height={'min(100px, 25vw)'}
                borderRadius='100%'
                border={`4px solid ${palette.primary.dark}`}
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
                <UserIcon fill={profileColors[1]} width='80%' height='80%' />
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
                    <EllipsisIcon fill={palette.background.textSecondary} />
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
                                    <EditIcon fill={palette.secondary.main} />
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
                <DateDisplay
                    loading={loading}
                    showIcon={true}
                    textBeforeDate="Joined"
                    timestamp={user?.created_at}
                    width={"33%"}
                />
                {/* Description */}
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
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip>
                    <ShareButton objectType={ObjectType.User} zIndex={zIndex} />
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
                    <ReportsLink
                        href={`${APP_LINKS.User}/reports/${user?.id}`}
                        reports={user?.reportsCount}
                    />
                </Stack>
            </Stack>
        </Box>
    ), [bio, handle, isOwn, loading, name, onEdit, openMoreMenu, palette.background.paper, palette.background.textPrimary, palette.background.textSecondary, palette.primary.dark, palette.secondary.dark, palette.secondary.main, profileColors, session, user?.created_at, user?.id, user?.isStarred, user?.reportsCount, user?.stars, zIndex]);

    /**
     * Opens add new page
     */
    const toAddNew = useCallback((event: any) => {
        switch (currTabType) {
            case TabOptions.Organizations:
                setLocation(`${APP_LINKS.Organization}/add`);
                break;
            case TabOptions.Projects:
                setLocation(`${APP_LINKS.Project}/add`);
                break;
            case TabOptions.Routines:
                openAddRoutine(event);
                break;
            case TabOptions.Standards:
                setLocation(`${APP_LINKS.Standard}/add`);
                break;
        }
    }, [currTabType, openAddRoutine, setLocation]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                isUpvoted={null}
                isStarred={user?.isStarred}
                objectId={id}
                objectName={name ?? ''}
                objectType={ObjectType.User}
                anchorEl={moreMenuAnchor}
                title='User Options'
                onActionStart={onMoreActionStart}
                onActionComplete={onMoreActionComplete}
                onClose={closeMoreMenu}
                permissions={{
                    canEdit: isOwn,
                    canReport: !isOwn,
                    canStar: !isOwn,
                }}
                session={session}
                zIndex={zIndex + 1}
            />
            {/* Add menu for selecting between single-step and multi-step routines */}
            <SelectRoutineTypeMenu
                anchorEl={addRoutineAnchor}
                handleClose={closeAddRoutine}
                session={session}
                zIndex={zIndex + 1}
            />
            <Box sx={{
                display: 'flex',
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                background: palette.mode === 'light' ? "#b2b3b3" : "#303030",
                position: "relative",
            }}>
                {/* Language display/select */}
                <Box sx={{
                    position: 'absolute',
                    top: 8,
                    right: 8,
                }}>
                    <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        session={session}
                        translations={user?.translations ?? partialData?.translations ?? []}
                        zIndex={zIndex}
                    />
                </Box>
                {overviewComponent}
            </Box>
            {/* View routines, organizations, standards, and projects associated with this user */}
            <Box>
                <Box display="flex" justifyContent="center" width="100%">
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
                </Box>
                <Box p={2}>
                    {
                        currTabType === TabOptions.Resources ? resources : (
                            <SearchList
                                canSearch={uuidValidate(id)}
                                handleAdd={toAddNew}
                                hideRoles={true}
                                id="user-view-list"
                                itemKeyPrefix={itemKeyPrefix}
                                noResultsText={noResultsText}
                                searchType={searchType}
                                onObjectSelect={onSearchSelect}
                                searchPlaceholder={placeholder}
                                session={session}
                                take={20}
                                where={where}
                                zIndex={zIndex}
                            />
                        )
                    }
                </Box>
            </Box>
        </>
    )
}