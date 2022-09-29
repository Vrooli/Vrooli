import { Box, IconButton, LinearProgress, Link, Stack, Tab, Tabs, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, ResourceListUsedFor, StarFor } from "@shared/consts";
import { adaHandleRegex } from "@shared/validation";
import { useLazyQuery } from "@apollo/client";
import { project, projectVariables } from "graphql/generated/project";
import { projectQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu, DateDisplay, ResourceListVertical, SearchList, SelectLanguageMenu, StarButton, SelectRoutineTypeMenu } from "components";
import { containerShadow } from "styles";
import { ProjectViewProps } from "../types";
import { Project, ResourceList } from "types";
import { SearchListGenerator } from "components/lists/types";
import { getLanguageSubtag, getLastUrlPart, getPreferredLanguage, getTranslation, getUserLanguages, ObjectType, SearchType } from "utils";
import { validate as uuidValidate } from 'uuid';
import { DonateIcon, EditIcon, EllipsisIcon } from "@shared/icons";
import { ObjectAction, ObjectActionComplete } from "components/dialogs/types";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { VisibilityType } from "graphql/generated/globalTypes";

enum TabOptions {
    Resources = "Resources",
    Routines = "Routines",
    Standards = "Standards",
}

export const ProjectView = ({
    partialData,
    session,
    zIndex,
}: ProjectViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    // Fetch data
    const id = useMemo(() => getLastUrlPart(), []);
    const [getData, { data, loading }] = useLazyQuery<project, projectVariables>(projectQuery, { errorPolicy: 'all'});
    const [project, setProject] = useState<Project | null | undefined>(null);
    useEffect(() => {
        if (uuidValidate(id)) getData({ variables: { input: { id } } })
        else if (adaHandleRegex.test(id)) getData({ variables: { input: { handle: id } } })
    }, [getData, id]);
    useEffect(() => {
        setProject(data?.project);
    }, [data]);
    const canEdit = useMemo<boolean>(() => project?.permissionsProject?.canEdit === true, [project?.permissionsProject?.canEdit]);

    const availableLanguages = useMemo<string[]>(() => (project?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [project?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { canStar, name, description, handle, resourceList } = useMemo(() => {
        const permissions = project?.permissionsProject;
        const resourceList: ResourceList | undefined = Array.isArray(project?.resourceLists) ? project?.resourceLists?.find(r => r.usedFor === ResourceListUsedFor.Display) : undefined;
        return {
            canStar: permissions?.canStar === true,
            name: getTranslation(project, 'name', [language]) ?? getTranslation(partialData, 'name', [language]),
            description: getTranslation(project, 'description', [language]) ?? getTranslation(partialData, 'description', [language]),
            handle: project?.handle ?? partialData?.handle,
            resourceList,
        };
    }, [language, project, partialData]);

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
                if (!project) return;
                setProject({
                    ...project,
                    resourceLists: [updatedList]
                })
            }}
            loading={loading}
            mutate={true}
            zIndex={zIndex}
        />
    ) : null, [canEdit, loading, project, resourceList, session, zIndex]);

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
        tabs.push(TabOptions.Routines);
        tabs.push(TabOptions.Standards);
        return tabs;
    }, [resources]);

    const currTabType = useMemo(() => tabIndex >= 0 && tabIndex < availableTabs.length ? availableTabs[tabIndex] : null, [availableTabs, tabIndex]);

    const onEdit = useCallback(() => {
        setLocation(`${APP_LINKS.Project}/edit/${id}`);
    }, [setLocation, id]);

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
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp:
                if (data.vote.success) {
                    setProject({
                        ...project,
                        isUpvoted: action === ObjectActionComplete.VoteUp,
                    } as any)
                }
                break;
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.star.success) {
                    setProject({
                        ...project,
                        isStarred: action === ObjectActionComplete.Star,
                    } as any)
                }
                break;
            case ObjectActionComplete.Fork:
                setLocation(`${APP_LINKS.Project}/${data.fork.project.id}`);
                window.location.reload();
                break;
            case ObjectActionComplete.Copy:
                setLocation(`${APP_LINKS.Project}/${data.copy.project.id}`);
                window.location.reload();
                break;
        }
    }, [project, setLocation]);

    // Menu for picking which routine type to add
    const [addRoutineAnchor, setAddRoutineAnchor] = useState<any>(null);
    const openAddRoutine = useCallback((ev: React.MouseEvent<HTMLElement>) => {
        setAddRoutineAnchor(ev.currentTarget)
    }, []);
    const closeAddRoutine = useCallback(() => setAddRoutineAnchor(null), []);

    // Create search data
    const { searchType, itemKeyPrefix, placeholder, where, noResultsText, onSearchSelect } = useMemo<SearchListGenerator>(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`);
        // The first tab doesn't have search results, as it is the project's set resources
        switch (currTabType) {
            case TabOptions.Routines:
                return {
                    searchType: SearchType.Routine,
                    itemKeyPrefix: 'routine-list-item',
                    placeholder: "Search project's routines...",
                    noResultsText: "No routines found",
                    where: { projectId: id, isComplete: !canEdit ? true : undefined, isInternal: false, visibility: VisibilityType.All },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Routine, newValue.id),
                };
            case TabOptions.Standards:
                return {
                    searchType: SearchType.Standard,
                    itemKeyPrefix: 'standard-list-item',
                    placeholder: "Search project's standards...",
                    noResultsText: "No standards found",
                    where: { projectId: id, visibility: VisibilityType.All },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Standard, newValue.id),
                }
            default:
                return {
                    searchType: SearchType.Routine,
                    itemKeyPrefix: '',
                    placeholder: '',
                    noResultsText: '',
                    searchQuery: null,
                    where: {},
                    onSearchSelect: (o: any) => { },
                    searchItemFactory: (a: any, b: any) => null
                }
        }
    }, [canEdit, currTabType, id, setLocation]);

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
                            <Tooltip title="Edit project">
                                <IconButton
                                    aria-label="Edit project"
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
                {/* Created date */}
                <DateDisplay
                    loading={loading}
                    showIcon={true}
                    textBeforeDate="Created"
                    timestamp={project?.created_at}
                    width={"33%"}
                />
                {/* Description */}
                {
                    loading && (
                        <Stack sx={{ width: '85%', color: 'grey.500' }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    )
                }
                {
                    !loading && Boolean(description) && <Typography variant="body1" sx={{ color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary }}>{description}</Typography>
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip>
                    <ShareButton objectType={ObjectType.Project} zIndex={zIndex} />
                    {canStar && <StarButton
                        session={session}
                        objectId={project?.id ?? ''}
                        starFor={StarFor.Project}
                        isStar={project?.isStarred ?? false}
                        stars={project?.stars ?? 0}
                        onChange={(isStar: boolean) => { }}
                        tooltipPlacement="bottom"
                    />}
                </Stack>
            </Stack>
        </Box>
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, palette.secondary.dark, openMoreMenu, loading, canEdit, name, onEdit, handle, project?.created_at, project?.id, project?.isStarred, project?.stars, description, zIndex, canStar, session]);

    /**
    * Opens add new page
    */
    const toAddNew = useCallback((event: any) => {
        switch (currTabType) {
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
                isUpvoted={project?.isUpvoted}
                isStarred={project?.isStarred}
                objectId={id}
                objectName={name ?? ''}
                objectType={ObjectType.Project}
                anchorEl={moreMenuAnchor}
                title='Project Options'
                onActionStart={onMoreActionStart}
                onActionComplete={onMoreActionComplete}
                onClose={closeMoreMenu}
                permissions={project?.permissionsProject}
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
            {/* View routines and standards associated with this project */}
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
                                id="project-view-list"
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