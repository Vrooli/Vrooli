import { Box, IconButton, LinearProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, FindByIdOrHandleInput, Organization, ResourceList, BookmarkFor, VisibilityType } from "@shared/consts";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu, DateDisplay, ReportsLink, SearchList, SelectLanguageMenu, BookmarkButton, PageTabs } from "components";
import { OrganizationViewProps } from "../types";
import { SearchListGenerator } from "components/lists/types";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, placeholderColor, toSearchListData, useObjectActions, useObjectFromUrl } from "utils";
import { ResourceListVertical } from "components/lists";
import { uuidValidate } from '@shared/uuid';
import { DonateIcon, EditIcon, EllipsisIcon, OrganizationIcon } from "@shared/icons";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { organizationFindOne } from "api/generated/endpoints/organization_findOne";
import { useTranslation } from "react-i18next";
import { PageTab } from "components/types";

enum TabOptions {
    Resource = "Resource",
    Member = "Member",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard",
}

export const OrganizationView = ({
    partialData,
    session,
    zIndex,
}: OrganizationViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const profileColors = useMemo(() => placeholderColor(), []);

    const { id, isLoading, object: organization, permissions, setObject: setOrganization } = useObjectFromUrl<Organization, FindByIdOrHandleInput>({
        query: organizationFindOne,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (organization?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [organization?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { bio, handle, name, resourceList } = useMemo(() => {
        const resourceList: ResourceList | null | undefined = organization?.resourceList;
        const { bio, name } = getTranslation(organization ?? partialData, [language]);
        return {
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            handle: organization?.handle ?? partialData?.handle,
            name,
            resourceList,
        };
    }, [language, organization, partialData]);

    useEffect(() => {
        if (handle) document.title = `${name} ($${handle}) | Vrooli`;
        else document.title = `${name} | Vrooli`;
    }, [handle, name]);

    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceListVertical
            list={resourceList as any}
            session={session}
            canUpdate={permissions.canUpdate}
            handleUpdate={(updatedList) => {
                if (!organization) return;
                setOrganization({
                    ...organization,
                    resourceList: updatedList
                })
            }}
            loading={isLoading}
            mutate={true}
            zIndex={zIndex}
        />
    ) : null, [isLoading, organization, permissions.canUpdate, resourceList, session, setOrganization, zIndex]);

    // Handle tabs
    const tabs = useMemo<PageTab<TabOptions>[]>(() => {
        let tabs: TabOptions[] = Object.values(TabOptions);
        // Remove resources if there are none, and you cannot add them
        if (!resources && !permissions.canUpdate) tabs = tabs.filter(t => t !== TabOptions.Resource);
        // Return tabs shaped for the tab component
        return tabs.map((tab, i) => ({
            color: tab === TabOptions.Resource ? '#8e6b00' : 'default', // Custom color for resources
            index: i,
            label: t(tab, { count: 2 }),
            value: tab,
        }));
    }, [permissions.canUpdate, resources, t]);
    const [currTab, setCurrTab] = useState<PageTab<TabOptions>>(tabs[0]);
    const handleTabChange = useCallback((_: unknown, value: PageTab<TabOptions>) => setCurrTab(value), []);

    // Create search data
    const { searchType, placeholder, where } = useMemo<SearchListGenerator>(() => {
        // NOTE: The first tab doesn't have search results, as it is the user's set resources
        if (currTab.value === TabOptions.Member)
            return toSearchListData('User', 'SearchMember', { organizationId: organization?.id });
        else if (currTab.value === TabOptions.Project)
            return toSearchListData('Project', 'SearchProject', { organizationId: organization?.id, isComplete: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All });
        else if (currTab.value === TabOptions.Routine)
            return toSearchListData('Routine', 'SearchRoutine', { organizationId: organization?.id, isComplete: !permissions.canUpdate ? true : undefined, isInternal: false, visibility: VisibilityType.All });
        return toSearchListData('Standard', 'SearchStandard', { organizationId: organization?.id, visibility: VisibilityType.All });
    }, [currTab, organization?.id, permissions.canUpdate]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: organization,
        objectType: 'Organization',
        session,
        setLocation,
        setObject: setOrganization,
    });

    /**
     * Displays name, avatar, bio, and quick links
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
                <OrganizationIcon fill={profileColors[1]} width='80%' height='80%' />
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
                            <Tooltip title="Edit organization">
                                <IconButton
                                    aria-label="Edit organization"
                                    size="small"
                                    onClick={() => actionData.onActionStart('Edit')}
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
                    timestamp={organization?.created_at}
                    width={"33%"}
                />
                {/* Bio */}
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
                    <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip>
                    <ShareButton object={organization} zIndex={zIndex} />
                    <ReportsLink object={organization} />
                    <BookmarkButton
                        disabled={!permissions.canBookmark}
                        session={session}
                        objectId={organization?.id ?? ''}
                        bookmarkFor={BookmarkFor.Organization}
                        isBookmarked={organization?.you?.isBookmarked ?? false}
                        bookmarks={organization?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                    />
                </Stack>
            </Stack>
        </Box >
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, palette.secondary.dark, profileColors, openMoreMenu, isLoading, permissions.canUpdate, permissions.canBookmark, name, handle, organization, bio, zIndex, session, actionData]);

    /**
     * Opens add new page
     */
    const toAddNew = useCallback((event: any) => {
        // TODO need member page
        if (currTab.value === TabOptions.Member) return;
        setLocation(`${APP_LINKS[currTab.value]}/add`);
    }, [currTab, setLocation]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={organization as any}
                onClose={closeMoreMenu}
                session={session}
                zIndex={zIndex + 1}
            />
            <Box sx={{
                background: palette.mode === 'light' ? "#b2b3b3" : "#303030",
                display: 'flex',
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
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
                        translations={organization?.translations ?? partialData?.translations ?? []}
                        zIndex={zIndex}
                    />
                </Box>
                {overviewComponent}
            </Box>
            {/* View routines, members, standards, and projects associated with this organization */}
            <Box>
                <PageTabs
                    ariaLabel="organization-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />
                <Box p={2}>
                    {
                        currTab.value === TabOptions.Resource ? resources : (
                            <SearchList
                                canSearch={uuidValidate(organization?.id)}
                                handleAdd={permissions.canUpdate ? toAddNew : undefined}
                                hideRoles={true}
                                id="organization-view-list"
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