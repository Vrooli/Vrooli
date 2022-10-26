import { Box, IconButton, LinearProgress, Link, Stack, Tab, Tabs, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, StarFor } from "@shared/consts";
import { adaHandleRegex } from '@shared/validation';
import { useLazyQuery } from "@apollo/client";
import { organization, organizationVariables } from "graphql/generated/organization";
import { organizationQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu, DateDisplay, ReportsLink, SearchList, SelectLanguageMenu, StarButton } from "components";
import { OrganizationViewProps } from "../types";
import { Organization, ResourceList } from "types";
import { SearchListGenerator } from "components/lists/types";
import { base36ToUuid, getLanguageSubtag, getLastUrlPart, getPreferredLanguage, getTranslation, getUserLanguages, ObjectAction, ObjectActionComplete, openObject, placeholderColor, SearchType, uuidToBase36 } from "utils";
import { ResourceListVertical } from "components/lists";
import { uuidValidate } from '@shared/uuid';
import { ResourceListUsedFor, VisibilityType } from "graphql/generated/globalTypes";
import { DonateIcon, EditIcon, EllipsisIcon, OrganizationIcon } from "@shared/icons";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";

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
    zIndex,
}: OrganizationViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    // Fetch data
    const id = useMemo(() => base36ToUuid(getLastUrlPart()), []);
    const [getData, { data, loading }] = useLazyQuery<organization, organizationVariables>(organizationQuery, { errorPolicy: 'all' });
    const [organization, setOrganization] = useState<Organization | null | undefined>(null);
    useEffect(() => {
        if (uuidValidate(id)) getData({ variables: { input: { id } } })
        else if (adaHandleRegex.test(id)) getData({ variables: { input: { handle: id } } })
    }, [getData, id]);
    useEffect(() => {
        setOrganization(data?.organization);
    }, [data]);
    const canEdit = useMemo<boolean>(() => organization?.permissionsOrganization?.canEdit === true, [organization?.permissionsOrganization?.canEdit]);

    const availableLanguages = useMemo<string[]>(() => (organization?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [organization?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { bio, canStar, handle, name, resourceList } = useMemo(() => {
        const { canStar } = organization?.permissionsOrganization ?? {};
        const resourceList: ResourceList | undefined = Array.isArray(organization?.resourceLists) ? organization?.resourceLists?.find(r => r.usedFor === ResourceListUsedFor.Display) : undefined;
        const { bio, name } = getTranslation(organization ?? partialData, [language]);
        return {
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            canStar,
            handle: organization?.handle ?? partialData?.handle,
            name,
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
            zIndex={zIndex}
        />
    ) : null, [canEdit, loading, organization, resourceList, session, zIndex]);

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

    const onEdit = useCallback(() => {
        setLocation(`${APP_LINKS.Organization}/edit/${uuidToBase36(id)}`);
    }, [setLocation, id]);

    // Create search data
    const { searchType, itemKeyPrefix, placeholder, where, noResultsText } = useMemo<SearchListGenerator>(() => {
        switch (currTabType) {
            case TabOptions.Members:
                return {
                    searchType: SearchType.User,
                    itemKeyPrefix: 'member-list-item',
                    placeholder: "Search orgnization's members...",
                    noResultsText: "No members found",
                    where: { organizationId: id },
                };
            case TabOptions.Projects:
                return {
                    searchType: SearchType.Project,
                    itemKeyPrefix: 'project-list-item',
                    placeholder: "Search organization's projects...",
                    noResultsText: "No projects found",
                    where: { organizationId: id, isComplete: !canEdit ? true : undefined, visibility: VisibilityType.All },
                };
            case TabOptions.Routines:
                return {
                    searchType: SearchType.Routine,
                    itemKeyPrefix: 'routine-list-item',
                    placeholder: "Search organization's routines...",
                    noResultsText: "No routines found",
                    where: { organizationId: id, isComplete: !canEdit ? true : undefined, isInternal: false, visibility: VisibilityType.All },
                };
            case TabOptions.Standards:
                return {
                    searchType: SearchType.Standard,
                    itemKeyPrefix: 'standard-list-item',
                    placeholder: "Search organization's standards...",
                    noResultsText: "No standards found",
                    where: { organizationId: id, visibility: VisibilityType.All },
                }
            default:
                return {
                    searchType: SearchType.User,
                    itemKeyPrefix: '',
                    placeholder: '',
                    noResultsText: '',
                    where: {},
                }
        }
    }, [currTabType, id, canEdit]);

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
                if (data.success) {
                    setOrganization({
                        ...organization,
                        isStarred: action === ObjectActionComplete.Star,
                    } as any)
                }
                break;
            case ObjectActionComplete.Fork:
                openObject(data.organization, setLocation);
                window.location.reload();
                break;
            case ObjectActionComplete.Copy:
                openObject(data.organization, setLocation);
                window.location.reload();
                break;
        }
    }, [organization, setLocation]);

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
                    ) : canEdit ? (
                        <Stack direction="row" alignItems="center" justifyContent="center">
                            <Typography variant="h4" textAlign="center">{name}</Typography>
                            <Tooltip title="Edit organization">
                                <IconButton
                                    aria-label="Edit organization"
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
                    timestamp={organization?.created_at}
                    width={"33%"}
                />
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
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip>
                    <ShareButton object={organization} zIndex={zIndex} />
                    <ReportsLink object={organization} />
                    <StarButton
                        disabled={!canStar}
                        session={session}
                        objectId={organization?.id ?? ''}
                        starFor={StarFor.Organization}
                        isStar={organization?.isStarred ?? false}
                        stars={organization?.stars ?? 0}
                        onChange={(isStar: boolean) => { }}
                        tooltipPlacement="bottom"
                    />
                </Stack>
            </Stack>
        </Box >
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, palette.secondary.dark, profileColors, openMoreMenu, loading, canEdit, name, onEdit, handle, organization, bio, zIndex, canStar, session]);

    /**
     * Opens add new page
     */
    const toAddNew = useCallback((event: any) => {
        switch (currTabType) {
            case TabOptions.Members:
                //TODO
                break;
            case TabOptions.Projects:
                setLocation(`${APP_LINKS.Project}/add`);
                break;
            case TabOptions.Routines:
                setLocation(`${APP_LINKS.Routine}/add`);
                break;
            case TabOptions.Standards:
                setLocation(`${APP_LINKS.Standard}/add`);
                break;
        }
    }, [currTabType, setLocation]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                anchorEl={moreMenuAnchor}
                object={organization as any}
                onActionStart={onMoreActionStart}
                onActionComplete={onMoreActionComplete}
                onClose={closeMoreMenu}
                session={session}
                title='Organization Options'
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
                                handleAdd={canEdit ? toAddNew : undefined}
                                hideRoles={true}
                                id="organization-view-list"
                                itemKeyPrefix={itemKeyPrefix}
                                noResultsText={noResultsText}
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