import { ApiVersion, BookmarkFor, endpointGetApiVersion, ResourceList } from "@local/shared";
import { Avatar, Box, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ResourceListVertical } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { Title } from "components/text/Title/Title";
import { ApiIcon, EditIcon, EllipsisIcon } from "icons";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { OverviewContainer } from "styles";
import { placeholderColor } from "utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SessionContext } from "utils/SessionContext";
import { ApiViewProps } from "../types";

export const ApiView = ({
    display = "page",
    onClose,
    partialData,
    zIndex,
}: ApiViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);

    const { id, isLoading, object: apiVersion, permissions, setObject: setApiVersion } = useObjectFromUrl<ApiVersion>({
        ...endpointGetApiVersion,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (apiVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [apiVersion?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { canBookmark, details, name, resourceList, summary } = useMemo(() => {
        const { canBookmark } = apiVersion?.root?.you ?? {};
        const resourceList: ResourceList | null | undefined = apiVersion?.resourceList;
        const { details, name, summary } = getTranslation(apiVersion ?? partialData, [language]);
        return {
            details,
            canBookmark,
            name,
            resourceList,
            summary,
        };
    }, [language, apiVersion, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceListVertical
            list={resourceList as any}
            canUpdate={permissions.canUpdate}
            handleUpdate={(updatedList) => {
                if (!apiVersion) return;
                setApiVersion({
                    ...apiVersion,
                    resourceList: updatedList,
                });
            }}
            loading={isLoading}
            mutate={true}
            zIndex={zIndex}
        />
    ) : null, [resourceList, permissions.canUpdate, isLoading, zIndex, apiVersion, setApiVersion]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: apiVersion,
        objectType: "ApiVersion",
        setLocation,
        setObject: setApiVersion,
    });

    /**
     * Displays name, avatar, summary, and quick links
     */
    const overviewComponent = useMemo(() => (
        <OverviewContainer>
            <Avatar
                src="/broken-image.jpg" //TODO
                sx={{
                    backgroundColor: profileColors[0],
                    color: profileColors[1],
                    boxShadow: 2,
                    transform: "translateX(-50%)",
                    width: "min(100px, 25vw)",
                    height: "min(100px, 25vw)",
                    left: "50%",
                    top: "-55px",
                    position: "absolute",
                    fontSize: "min(50px, 10vw)",
                }}
            >
                <ApiIcon fill="white" width='75%' height='75%' />
            </Avatar>
            <Tooltip title={t("MoreOptions")}>
                <IconButton
                    aria-label={t("MoreOptions")}
                    size="small"
                    onClick={openMoreMenu}
                    sx={{
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                    }}
                >
                    <EllipsisIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
            <Stack direction="column" spacing={1} p={1} alignItems="center" justifyContent="center">
                {/* Title */}
                {
                    isLoading ? (
                        <Stack sx={{ width: "50%", color: "grey.500", paddingTop: 2, paddingBottom: 2 }} spacing={2}>
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : <Title
                        title={name}
                        variant="header"
                        options={permissions.canUpdate ? [{
                            label: t("Edit"),
                            Icon: EditIcon,
                            onClick: () => { actionData.onActionStart("Edit"); },
                        }] : []}
                        zIndex={zIndex}
                    />
                }
                {/* Joined date */}
                <DateDisplay
                    loading={isLoading}
                    showIcon={true}
                    textBeforeDate="Joined"
                    timestamp={apiVersion?.created_at}
                    width={"33%"}
                    zIndex={zIndex}
                />
                {/* Bio */}
                {
                    isLoading ? (
                        <Stack sx={{ width: "85%", color: "grey.500" }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : (
                        <Typography variant="body1" sx={{ color: summary ? palette.background.textPrimary : palette.background.textSecondary }}>{summary ?? "No summary set"}</Typography>
                    )
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    <ShareButton object={apiVersion} zIndex={zIndex} />
                    <ReportsLink object={apiVersion} />
                    <BookmarkButton
                        disabled={!canBookmark}
                        objectId={apiVersion?.id ?? ""}
                        bookmarkFor={BookmarkFor.Api}
                        isBookmarked={apiVersion?.root?.you?.isBookmarked ?? false}
                        bookmarks={apiVersion?.root?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                        zIndex={zIndex}
                    />
                </Stack>
            </Stack>
        </OverviewContainer>
    ), [palette.background.textSecondary, palette.background.textPrimary, profileColors, openMoreMenu, isLoading, name, permissions.canUpdate, t, apiVersion, summary, zIndex, canBookmark, actionData]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                zIndex={zIndex}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={apiVersion as any}
                onClose={closeMoreMenu}
                zIndex={zIndex + 1}
            />
            <Box sx={{
                background: palette.mode === "light" ? "#b2b3b3" : "#303030",
                display: "flex",
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                position: "relative",
            }}>
                {/* Language display/select */}
                <Box sx={{
                    position: "absolute",
                    top: 8,
                    right: 8,
                }}>
                    {availableLanguages.length > 1 && <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        languages={availableLanguages}
                        zIndex={zIndex}
                    />}
                </Box>
                {overviewComponent}
            </Box>
        </>
    );
};
