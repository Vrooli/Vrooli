import { Box, IconButton, LinearProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkFor, FindVersionInput, ProjectVersion } from "@shared/consts";
import { EditIcon, EllipsisIcon } from "@shared/icons";
import { useLocation } from '@shared/route';
import { projectVersionFindOne } from "api/generated/endpoints/projectVersion_findOne";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SessionContext } from "utils/SessionContext";
import { ProjectViewProps } from "../types";

export const ProjectView = ({
    display = 'page',
    partialData,
    zIndex = 200,
}: ProjectViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const { id, isLoading, object: projectVersion, permissions, setObject: setProjectVersion } = useObjectFromUrl<ProjectVersion, FindVersionInput>({
        query: projectVersionFindOne,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (projectVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [projectVersion?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name, description, handle } = useMemo(() => {
        const { description, name } = getTranslation(projectVersion ?? partialData, [language]);
        return {
            name,
            description,
            handle: projectVersion?.root?.handle ?? partialData?.root?.handle,
        };
    }, [language, projectVersion, partialData]);

    useEffect(() => {
        if (handle) document.title = `${name} ($${handle}) | Vrooli`;
        else document.title = `${name} | Vrooli`;
    }, [handle, name]);

    // // Handle tabs
    // const tabs = useMemo<PageTab<TabOptions>[]>(() => {
    //     let tabs: TabOptions[] = Object.values(TabOptions);
    //     // Return tabs shaped for the tab component
    //     return tabs.map((tab, i) => ({
    //         index: i,
    //         label: t(tab, { count: 2 }),
    //         value: tab,
    //     }));
    // }, [t]);
    // const [currTab, setCurrTab] = useState<PageTab<TabOptions>>(tabs[0]);
    // const handleTabChange = useCallback((_: unknown, value: PageTab<TabOptions>) => setCurrTab(value), []);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: projectVersion,
        objectType: 'ProjectVersion',
        setLocation,
        setObject: setProjectVersion,
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
                            <Tooltip title="Edit project">
                                <IconButton
                                    aria-label="Edit project"
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
                {/* Created date */}
                <DateDisplay
                    loading={isLoading}
                    showIcon={true}
                    textBeforeDate="Created"
                    timestamp={projectVersion?.created_at}
                    width={"33%"}
                />
                {/* Description */}
                {
                    isLoading && (
                        <Stack sx={{ width: '85%', color: 'grey.500' }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    )
                }
                {
                    !isLoading && Boolean(description) && <Typography variant="body1" sx={{ color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary }}>{description}</Typography>
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    {/* <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip> */}
                    <ShareButton object={projectVersion} zIndex={zIndex} />
                    <BookmarkButton
                        disabled={!permissions.canBookmark}
                        objectId={projectVersion?.id ?? ''}
                        bookmarkFor={BookmarkFor.Project}
                        isBookmarked={projectVersion?.root?.you?.isBookmarked ?? false}
                        bookmarks={projectVersion?.root?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                    />
                </Stack>
            </Stack>
        </Box>
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, palette.secondary.dark, openMoreMenu, isLoading, permissions.canUpdate, permissions.canBookmark, name, handle, projectVersion, description, zIndex, actionData]);

    // /**
    // * Opens add new page
    // */
    // const toAddNew = useCallback((event: any) => {
    //     setLocation(`${LINKS[currTab.value]}/add`);
    // }, [currTab.value, setLocation]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Project',
                }}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={projectVersion as any}
                onClose={closeMoreMenu}
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
                        translations={projectVersion?.translations ?? partialData?.translations ?? []}
                        zIndex={zIndex}
                    />
                </Box>
                {overviewComponent}
            </Box>
            {/* View routines and standards associated with this project */}
            <Box>
                {/* Breadcrumbs to show directory hierarchy */}
                {/* TODO */}
                {/* List of items in current directory */}
                {/* <SearchList
                    canSearch={Boolean(projectVersion?.id)}
                    handleAdd={permissions.canUpdate ? toAddNew : undefined}
                    hideUpdateButton={true}
                    id="directory-view-list"
                    searchType={searchType}
                    searchPlaceholder={placeholder}
                    take={20}
                    where={where}
                    zIndex={zIndex}
                /> */}
            </Box>
        </>
    )
}