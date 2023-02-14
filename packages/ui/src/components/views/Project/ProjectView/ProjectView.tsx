import { Box, IconButton, LinearProgress, Link, Stack, Tab, Tabs, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, FindVersionInput, ProjectVersion, BookmarkFor, VisibilityType } from "@shared/consts";
import { useLazyQuery } from "api/hooks";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu, DateDisplay, SearchList, SelectLanguageMenu, StarButton } from "components";
import { ProjectViewProps } from "../types";
import { SearchListGenerator } from "components/lists/types";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, ObjectAction, ObjectActionComplete, openObject, parseSingleItemUrl, SearchType, uuidToBase36 } from "utils";
import { DonateIcon, EditIcon, EllipsisIcon } from "@shared/icons";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { setDotNotationValue } from "@shared/utils";
import { projectVersionFindOne } from "api/generated/endpoints/projectVersion";

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
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data, loading }] = useLazyQuery<ProjectVersion, FindVersionInput, 'projectVersion'>(projectVersionFindOne, 'projectVersion', { errorPolicy: 'all' });
    const [projectVersion, setProjectVersion] = useState<ProjectVersion | null | undefined>(null);
    useEffect(() => {
        if (urlData.id || urlData.idRoot || urlData.handleRoot) getData({ variables: urlData })
    }, [getData, urlData]);
    useEffect(() => {
        setProjectVersion(data?.projectVersion);
    }, [data]);
    const canUpdate = useMemo<boolean>(() => projectVersion?.you?.canUpdate === true, [projectVersion?.you?.canUpdate]);

    const availableLanguages = useMemo<string[]>(() => (projectVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [projectVersion?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { canStar, name, description, handle } = useMemo(() => {
        const { canStar } = projectVersion?.root?.you ?? {};
        const { description, name } = getTranslation(projectVersion ?? partialData, [language]);
        return {
            canStar,
            name,
            description,
            handle: projectVersion?.root?.handle ?? partialData?.root?.handle,
        };
    }, [language, projectVersion, partialData]);

    useEffect(() => {
        if (handle) document.title = `${name} ($${handle}) | Vrooli`;
        else document.title = `${name} | Vrooli`;
    }, [handle, name]);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(0);
    const handleTabChange = (event, newValue) => { setTabIndex(newValue) };

    /**
     * Calculate which tabs to display
     */
    const availableTabs = useMemo(() => {
        const tabs: TabOptions[] = [];
        // Always display others (for now)
        tabs.push(TabOptions.Routines);
        tabs.push(TabOptions.Standards);
        return tabs;
    }, []);

    const currTabType = useMemo(() => tabIndex >= 0 && tabIndex < availableTabs.length ? availableTabs[tabIndex] : null, [availableTabs, tabIndex]);

    const onEdit = useCallback(() => {
        projectVersion?.id && setLocation(`${APP_LINKS.Project}/edit/${uuidToBase36(projectVersion.id)}`);
    }, [projectVersion?.id, setLocation]);

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
                if (data.success && projectVersion) {
                    setProjectVersion(setDotNotationValue(projectVersion, 'root.you.isUpvoted', action === ObjectActionComplete.VoteUp))
                }
                break;
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.success && projectVersion) {
                    setProjectVersion(setDotNotationValue(projectVersion, 'root.you.isStarred', action === ObjectActionComplete.Star))
                }
                break;
            case ObjectActionComplete.Fork:
                openObject(data.project, setLocation);
                window.location.reload();
                break;
        }
    }, [projectVersion, setLocation]);

    // Create search data
    const { searchType, placeholder, where } = useMemo<SearchListGenerator>(() => {
        // The first tab doesn't have search results, as it is the project's set resources
        switch (currTabType) {
            case TabOptions.Routines:
                return {
                    searchType: SearchType.Routine,
                    placeholder: 'SearchRoutine',
                    where: { projectId: projectVersion?.id, isComplete: !canUpdate ? true : undefined, isInternal: false, visibility: VisibilityType.All },
                };
            case TabOptions.Standards:
                return {
                    searchType: SearchType.Standard,
                    placeholder: 'SearchStandard',
                    where: { projectId: projectVersion?.id, visibility: VisibilityType.All },
                }
            default:
                return {
                    searchType: SearchType.Routine,
                    placeholder: 'SearchRoutine',
                    where: {},
                }
        }
    }, [canUpdate, currTabType, projectVersion?.id]);

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
                    ) : canUpdate ? (
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
                    timestamp={projectVersion?.created_at}
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
                    <ShareButton object={projectVersion} zIndex={zIndex} />
                    <StarButton
                        disabled={!canStar}
                        session={session}
                        objectId={projectVersion?.id ?? ''}
                        starFor={BookmarkFor.Project}
                        isStar={projectVersion?.root?.you?.isStarred ?? false}
                        stars={projectVersion?.root?.stars ?? 0}
                        onChange={(isStar: boolean) => { }}
                        tooltipPlacement="bottom"
                    />
                </Stack>
            </Stack>
        </Box>
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, palette.secondary.dark, openMoreMenu, loading, canUpdate, name, onEdit, handle, projectVersion, description, zIndex, canStar, session]);

    /**
    * Opens add new page
    */
    const toAddNew = useCallback((event: any) => {
        switch (currTabType) {
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
                object={projectVersion as any}
                onActionStart={onMoreActionStart}
                onActionComplete={onMoreActionComplete}
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
                }}>
                    <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        session={session}
                        translations={projectVersion?.translations ?? partialData?.translations ?? []}
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
                    <SearchList
                        canSearch={Boolean(projectVersion?.id)}
                        handleAdd={canUpdate ? toAddNew : undefined}
                        hideRoles={true}
                        id="project-view-list"
                        searchType={searchType}
                        searchPlaceholder={placeholder}
                        session={session}
                        take={20}
                        where={where}
                        zIndex={zIndex}
                    />
                </Box>
            </Box>
        </>
    )
}