import { Box, IconButton, LinearProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkFor, FindByIdOrHandleInput, LINKS, ResourceList, User, VisibilityType } from "@shared/consts";
import { EditIcon, EllipsisIcon, HelpIcon, OrganizationIcon, ProjectIcon, SvgProps, UserIcon } from "@shared/icons";
import { useLocation } from '@shared/route';
import { uuidValidate } from '@shared/uuid';
import { userFindOne } from "api/generated/endpoints/user_findOne";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ResourceListVertical } from "components/lists/resource";
import { SearchList } from "components/lists/SearchList/SearchList";
import { SearchListGenerator } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { PageTab } from "components/types";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { placeholderColor, toSearchListData } from "utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SearchType } from "utils/search/objectToSearch";
import { UserViewProps } from "../types";

enum TabOptions {
    Resource = "Resource",
    Project = "Project",
    Organization = "Organization",
}

type TabParams = {
    Icon: (props: SvgProps) => JSX.Element,
    searchType: SearchType;
    tabType: TabOptions;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: TabParams[] = [{
    Icon: HelpIcon,
    searchType: SearchType.Resource,
    tabType: TabOptions.Resource,
    where: {},
}, {
    Icon: ProjectIcon,
    searchType: SearchType.Project,
    tabType: TabOptions.Project,
    where: {},
}, {
    Icon: OrganizationIcon,
    searchType: SearchType.Organization,
    tabType: TabOptions.Organization,
    where: {},
}];

export const UserView = ({
    display = 'page',
    session,
    partialData,
    zIndex = 200,
}: UserViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const profileColors = useMemo(() => placeholderColor(), []);

    const { id, isLoading, object: user, permissions, setObject: setUser } = useObjectFromUrl<User, FindByIdOrHandleInput>({
        query: userFindOne,
        partialData,
        idFallback: getCurrentUser(session).id,
    });

    const availableLanguages = useMemo<string[]>(() => (user?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [user?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { bio, name, handle, resourceList } = useMemo(() => {
        const resourceList: ResourceList | undefined = undefined;// TODO user?.resourceList;
        const { bio } = getTranslation(user ?? partialData, [language]);
        return {
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            name: user?.name ?? partialData?.name,
            handle: user?.handle ?? partialData?.handle,
            resourceList,
        };
    }, [language, partialData, user]);

    useEffect(() => {
        if (handle) document.title = `${name} ($${handle}) | Vrooli`;
        else document.title = `${name} | Vrooli`;
    }, [handle, name]);

    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceListVertical
            list={resourceList}
            session={session}
            canUpdate={permissions.canUpdate}
            handleUpdate={(updatedList) => {
                if (!user) return;
                setUser({
                    ...user,
                    //resourceList: updatedList TODO
                })
            }}
            loading={isLoading}
            mutate={true}
            zIndex={zIndex}
        />
    ) : null, [isLoading, permissions.canUpdate, resourceList, session, setUser, user, zIndex]);


    // Handle tabs
    const tabs = useMemo<PageTab<TabOptions>[]>(() => {
        let tabs = tabParams;
        // Remove resources if there are none, and you cannot add them
        if (!resources && !permissions.canUpdate) tabs = tabs.filter(t => t.tabType !== TabOptions.Resource);
        // Return tabs shaped for the tab component
        return tabs.map((tab, i) => ({
            color: tab.tabType === TabOptions.Resource ? '#8e6b00' : 'default', // Custom color for resources
            index: i,
            // Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            value: tab.tabType,
        }));
    }, [permissions.canUpdate, resources, t]);
    const [currTab, setCurrTab] = useState<PageTab<TabOptions>>(tabs[0]);
    const handleTabChange = useCallback((_: unknown, value: PageTab<TabOptions>) => setCurrTab(value), []);

    const onEdit = useCallback(() => {
        setLocation(`${LINKS.Settings}?page="profile"`);
    }, [setLocation]);

    // Create search data
    const { searchType, placeholder, where } = useMemo<SearchListGenerator>(() => {
        // NOTE: The first tab doesn't have search results, as it is the user's set resources
        if (currTab.value === TabOptions.Organization)
            return toSearchListData('Organization', 'SearchOrganization', { memberUserIds: [id], visibility: VisibilityType.All });
        return toSearchListData('Project', 'SearchProject', { ownedByUserId: id, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All });
    }, [currTab.value, id, permissions.canUpdate]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: user,
        objectType: 'User',
        session,
        setLocation,
        setObject: setUser,
    });

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
                boxShadow: { xs: 'none', sm: 12 },
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
                        paddingRight: '1em',
                    }}
                >
                    <EllipsisIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
            <Stack direction="column" spacing={1} p={1} alignItems="center" justifyContent="center">
                {/* Title */}
                {
                    isLoading ? (
                        <Stack sx={{ width: '50%', color: 'grey.500', paddingTop: 2, paddingBottom: 2 }} spacing={2}>
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : permissions.canUpdate ? (
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
                    loading={isLoading}
                    showIcon={true}
                    textBeforeDate="Joined"
                    timestamp={user?.created_at}
                    width={"33%"}
                />
                {/* Description */}
                {
                    isLoading ? (
                        <Stack sx={{ width: '85%', color: 'grey.500' }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : (
                        <Typography variant="body1" sx={{ color: Boolean(bio) ? palette.background.textPrimary : palette.background.textSecondary }}>{bio ?? 'No bio set'}</Typography>
                    )
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    {/* <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip> */}
                    <ShareButton object={user} zIndex={zIndex} />
                    <BookmarkButton
                        disabled={permissions.canUpdate}
                        session={session}
                        objectId={user?.id ?? ''}
                        bookmarkFor={BookmarkFor.User}
                        isBookmarked={user?.you?.isBookmarked ?? false}
                        bookmarks={user?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                    />
                    <ReportsLink object={user ? { ...user, reportsCount: user.reportsReceivedCount } : undefined} />
                </Stack>
            </Stack>
        </Box>
    ), [bio, handle, permissions.canUpdate, isLoading, name, onEdit, openMoreMenu, palette.background.paper, palette.background.textPrimary, palette.background.textSecondary, palette.primary.dark, palette.secondary.dark, palette.secondary.main, profileColors, session, user, zIndex]);

    /**
     * Opens add new page
     */
    const toAddNew = useCallback((event: any) => {
        setLocation(`${LINKS[currTab.value]}/add`);
    }, [currTab.value, setLocation]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => {}}
                session={session}
                titleData={{
                    titleKey: 'User',
                }}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={user}
                onClose={closeMoreMenu}
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
                    paddingRight: '1em',
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
                <PageTabs
                    ariaLabel="user-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />
                <Box p={2}>
                    {
                        currTab.value === TabOptions.Resource ? resources : (
                            <SearchList
                                canSearch={uuidValidate(id)}
                                handleAdd={permissions.canUpdate ? toAddNew : undefined}
                                hideUpdateButton={true}
                                id="user-view-list"
                                searchType={searchType}
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