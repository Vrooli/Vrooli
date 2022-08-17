import { Box, IconButton, LinearProgress, Link, Stack, Tab, Tabs, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation, useRoute } from '@shared/route';
import { APP_LINKS, StarFor } from "@shared/consts";
import { adaHandleRegex } from '@shared/validation';
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
import { BaseObjectActionDialog, SearchList, SelectLanguageDialog, StarButton } from "components";
import { containerShadow } from "styles";
import { OrganizationViewProps } from "../types";
import { Organization, ResourceList } from "types";
import { SearchListGenerator } from "components/lists/types";
import { displayDate, getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, ObjectType, placeholderColor, PubSub } from "utils";
import { ResourceListVertical } from "components/lists";
import { validate as uuidValidate } from 'uuid';
import { ResourceListUsedFor } from "graphql/generated/globalTypes";

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
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Organization}/:id`);
    const id: string = useMemo(() => params?.id ?? '', [params]);
    // Fetch data
    const [getData, { data, loading }] = useLazyQuery<organization, organizationVariables>(organizationQuery, { errorPolicy: 'all'});
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
        const permissions = organization?.permissionsOrganization;
        const resourceList: ResourceList | undefined = Array.isArray(organization?.resourceLists) ? organization?.resourceLists?.find(r => r.usedFor === ResourceListUsedFor.Display) : undefined;
        const bioText = getTranslation(organization, 'bio', [language]) ?? getTranslation(partialData, 'bio', [language]);
        return {
            bio: bioText && bioText.trim().length > 0 ? bioText : undefined,
            canStar: permissions?.canStar === true,
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

    const shareLink = useCallback(() => {
        navigator.clipboard.writeText(`https://vrooli.com${APP_LINKS.Organization}/${id}`);
        PubSub.get().publishSnack({ message: 'Copied🎉' })
    }, [id]);

    const onEdit = useCallback(() => {
        setLocation(`${APP_LINKS.Organization}/edit/${id}`);
    }, [setLocation, id]);

    // Create search data
    const { objectType, itemKeyPrefix, placeholder, searchQuery, where, noResultsText, onSearchSelect } = useMemo<SearchListGenerator>(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`);
        switch (currTabType) {
            case TabOptions.Members:
                return {
                    objectType: ObjectType.User,
                    itemKeyPrefix: 'member-list-item',
                    placeholder: "Search orgnization's members...",
                    noResultsText: "No members found",
                    searchQuery: usersQuery,
                    where: { organizationId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Profile, newValue.id),
                };
            case TabOptions.Projects:
                return {
                    objectType: ObjectType.Project,
                    itemKeyPrefix: 'project-list-item',
                    placeholder: "Search organization's projects...",
                    noResultsText: "No projects found",
                    searchQuery: projectsQuery,
                    where: { organizationId: id, isComplete: !canEdit ? true : undefined },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Project, newValue.id),
                };
            case TabOptions.Routines:
                return {
                    objectType: ObjectType.Routine,
                    itemKeyPrefix: 'routine-list-item',
                    placeholder: "Search organization's routines...",
                    noResultsText: "No routines found",
                    searchQuery: routinesQuery,
                    where: { organizationId: id, isComplete: !canEdit ? true : undefined, isInternal: false },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Routine, newValue.id),
                };
            case TabOptions.Standards:
                return {
                    objectType: ObjectType.Standard,
                    itemKeyPrefix: 'standard-list-item',
                    placeholder: "Search organization's standards...",
                    noResultsText: "No standards found",
                    searchQuery: standardsQuery,
                    where: { organizationId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Standard, newValue.id),
                }
            default:
                return {
                    objectType: ObjectType.User,
                    itemKeyPrefix: '',
                    placeholder: '',
                    noResultsText: '',
                    searchQuery: null,
                    where: {},
                    onSearchSelect: (o: any) => { },
                }
        }
    }, [currTabType, setLocation, id, canEdit]);

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
                    {canStar && <StarButton
                        session={session}
                        objectId={organization?.id ?? ''}
                        starFor={StarFor.Organization}
                        isStar={organization?.isStarred ?? false}
                        stars={organization?.stars ?? 0}
                        onChange={(isStar: boolean) => { }}
                        tooltipPlacement="bottom"
                    />}
                </Stack>
            </Stack>
        </Box >
    ), [palette.background.paper, palette.background.textPrimary, palette.background.textSecondary, palette.mode, palette.primary.main, palette.secondary.light, palette.secondary.dark, profileColors, openMoreMenu, loading, canEdit, name, onEdit, handle, organization?.created_at, organization?.id, organization?.isStarred, organization?.stars, bio, shareLink, canStar, session]);

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
                handleEdit={onEdit}
                isUpvoted={null}
                isStarred={organization?.isStarred}
                objectId={id}
                objectName={name ?? ''}
                objectType={ObjectType.Organization}
                anchorEl={moreMenuAnchor}
                title='Organization Options'
                onClose={closeMoreMenu}
                permissions={organization?.permissionsOrganization}
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
                }}>
                    <SelectLanguageDialog
                        availableLanguages={availableLanguages}
                        canDropdownOpen={availableLanguages.length > 1}
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        session={session}
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
                                handleAdd={toAddNew}
                                hideRoles={true}
                                itemKeyPrefix={itemKeyPrefix}
                                noResultsText={noResultsText}
                                objectType={objectType}
                                onObjectSelect={onSearchSelect}
                                query={searchQuery}
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