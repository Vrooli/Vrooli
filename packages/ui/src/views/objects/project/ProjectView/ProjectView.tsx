import { BookmarkFor, EditIcon, EllipsisIcon, FindVersionInput, ProjectVersion, projectVersionFindOne, useLocation } from "@local/shared";
import { Box, IconButton, LinearProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { ObjectTitle } from "components/text/ObjectTitle/ObjectTitle";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ObjectAction } from "utils/actions/objectActions";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SessionContext } from "utils/SessionContext";
import { ProjectViewProps } from "../types";

export const ProjectView = ({
    display = "page",
    onClose,
    partialData,
    zIndex = 200,
}: ProjectViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const { isLoading, object: existing, permissions, setObject: setProjectVersion } = useObjectFromUrl<ProjectVersion, FindVersionInput>({
        query: projectVersionFindOne,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name, description, handle } = useMemo(() => {
        const { description, name } = getTranslation(existing ?? partialData, [language]);
        return {
            name,
            description,
            handle: existing?.root?.handle ?? partialData?.root?.handle,
        };
    }, [language, existing, partialData]);

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
        object: existing,
        objectType: "ProjectVersion",
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
            bgcolor={palette.background.paper}
            sx={{
                borderRadius: { xs: "0", sm: 2 },
                boxShadow: { xs: "none", sm: 2 },
                width: { xs: "100%", sm: "min(500px, 100vw)" },
            }}
        >
            <Tooltip title="See all options">
                <IconButton
                    aria-label="More"
                    size="small"
                    onClick={openMoreMenu}
                    sx={{
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                        paddingRight: "1em",
                    }}
                >
                    <EllipsisIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
            <Stack direction="column" spacing={1} p={1} alignItems="center" justifyContent="center">
                {/* Title */}
                <ObjectTitle
                    language={language}
                    languages={availableLanguages}
                    loading={isLoading}
                    title={name}
                    setLanguage={setLanguage}
                    translations={existing?.translations ?? []}
                    zIndex={zIndex}
                />
                {/* Handle */}
                {
                    handle && <Link href={`https://handle.me/${handle}`} underline="hover">
                        <Typography
                            variant="h6"
                            textAlign="center"
                            sx={{
                                color: palette.secondary.dark,
                                cursor: "pointer",
                            }}
                        >${handle}</Typography>
                    </Link>
                }
                {/* Created date */}
                <DateDisplay
                    loading={isLoading}
                    showIcon={true}
                    textBeforeDate="Created"
                    timestamp={existing?.created_at}
                    width={"33%"}
                />
                {/* Description */}
                {
                    isLoading && (
                        <Stack sx={{ width: "85%", color: "grey.500" }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    )
                }
                {
                    !isLoading && Boolean(description) && <Typography variant="body1" sx={{ color: description ? palette.background.textPrimary : palette.background.textSecondary }}>{description}</Typography>
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    {/* <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip> */}
                    <ShareButton object={existing} zIndex={zIndex} />
                    <BookmarkButton
                        disabled={!permissions.canBookmark}
                        objectId={existing?.id ?? ""}
                        bookmarkFor={BookmarkFor.Project}
                        isBookmarked={existing?.root?.you?.isBookmarked ?? false}
                        bookmarks={existing?.root?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                        zIndex={zIndex}
                    />
                </Stack>
            </Stack>
        </Box>
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.dark, openMoreMenu, language, availableLanguages, isLoading, name, existing, zIndex, handle, description, permissions.canBookmark]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Project",
                }}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={existing as any}
                onClose={closeMoreMenu}
                zIndex={zIndex + 1}
            />
            {overviewComponent}
            {/* View routines and standards associated with this project */}
            <Box>
                {/* Breadcrumbs to show directory hierarchy */}
                {/* TODO */}
                {/* List of items in current directory */}
                {/* <SearchList
                    canSearch={() => Boolean(projectVersion?.id)}
                    dummyLength={display === "page" ? 5 : 3}
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
            {/* Edit button (if canUpdate) */}
            <SideActionButtons
                // Treat as a dialog when build view is open
                display={display}
                zIndex={zIndex + 2}
                sx={{ position: "fixed" }}
            >
                {permissions.canUpdate ? (
                    <ColorIconButton aria-label="edit-routine" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
            </SideActionButtons>
        </>
    );
};
