import { BookmarkFor, LINKS, ListObject, ResourceList as ResourceListType, Team, endpointGetTeam, uuidValidate } from "@local/shared";
import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { ResourceList } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { useFindMany } from "hooks/useFindMany";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTabs } from "hooks/useTabs";
import { EditIcon, EllipsisIcon, ExportIcon, SearchIcon, TeamIcon } from "icons";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { BannerImageContainer, OverviewContainer, OverviewProfileAvatar, OverviewProfileStack, SideActionsButton } from "styles";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { TeamPageTabOption, teamTabParams } from "utils/search/objectToSearch";
import { TeamViewProps } from "../types";

export function TeamView({
    display,
    onClose,
}: TeamViewProps) {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const profileColors = useMemo(() => placeholderColor(), []);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const { isLoading, object: team, permissions, setObject: setTeam } = useObjectFromUrl<Team>({
        ...endpointGetTeam,
        objectType: "Team",
    });

    const availableLanguages = useMemo<string[]>(() => (team?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [team?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { bannerImageUrl, bio, handle, name, resourceList } = useMemo(() => {
        const resourceList: ResourceListType | null | undefined = team?.resourceList;
        const { bio, name } = getTranslation(team, [language]);
        return {
            bannerImageUrl: extractImageUrl(team?.bannerImage, team?.updated_at, 1000),
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            handle: team?.handle,
            name,
            resourceList,
        };
    }, [language, team]);

    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceList
            horizontal={false}
            list={resourceList}
            canUpdate={permissions.canUpdate}
            handleUpdate={(updatedList) => {
                if (!team) return;
                setTeam({
                    ...team,
                    resourceList: updatedList,
                });
            }}
            loading={isLoading}
            mutate={true}
            parent={{ __typename: "Team", id: team?.id ?? "" }}
        />
    ) : null, [isLoading, team, permissions.canUpdate, resourceList, setTeam]);

    const {
        currTab,
        handleTabChange,
        searchPlaceholderKey,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "team-tabs", tabParams: teamTabParams, display });

    const findManyData = useFindMany<ListObject>({
        canSearch: () => uuidValidate(team?.id),
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where({ teamId: team?.id ?? "", permissions }),
    });

    const [showSearchFilters, setShowSearchFilters] = useState<boolean>(false);
    const toggleSearchFilters = useCallback(() => setShowSearchFilters(!showSearchFilters), [showSearchFilters]);
    // If showing search filter, focus the search input
    useEffect(() => {
        if (!showSearchFilters) return;
        const searchInput = document.getElementById("search-bar-team-view-list");
        searchInput?.focus();
    }, [showSearchFilters]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: team,
        objectType: "Team",
        setLocation,
        setObject: setTeam,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                tabTitle={handle ? `${name} (@${handle})` : name}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={team as any}
                onClose={closeMoreMenu}
            />
            <BannerImageContainer sx={{
                backgroundImage: bannerImageUrl ? `url(${bannerImageUrl})` : undefined,
            }}>
                {/* Language display/select */}
                <Box sx={{
                    position: "absolute",
                    top: 8,
                    right: 8,
                    paddingRight: "1em",
                }}>
                    {availableLanguages.length > 1 && <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        languages={availableLanguages}
                    />}
                </Box>
            </BannerImageContainer>
            <OverviewContainer>
                <OverviewProfileStack>
                    <OverviewProfileAvatar
                        isBot={false}
                        profileColors={profileColors}
                        src={extractImageUrl(team?.profileImage, team?.updated_at, 100)}
                    >
                        <TeamIcon width="75%" height="75%" />
                    </OverviewProfileAvatar>
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
                    <BookmarkButton
                        objectId={team?.id ?? ""}
                        bookmarkFor={BookmarkFor.Team}
                        isBookmarked={team?.you?.isBookmarked ?? false}
                        bookmarks={team?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                    />
                </OverviewProfileStack>
                <Stack direction="column" spacing={1} p={2} justifyContent="center" sx={{
                    alignItems: "flex-start",
                }}>
                    {/* Title */}
                    {
                        (isLoading && !name) ? (
                            <TextLoading size="header" sx={{ width: "50%" }} />
                        ) : <Title
                            title={name}
                            variant="header"
                            sxs={{ stack: { padding: 0, paddingBottom: handle ? 0 : 2 } }}
                        />
                    }
                    {/* Handle */}
                    {
                        handle && <Typography
                            variant="h6"
                            textAlign="center"
                            fontFamily="monospace"
                            onClick={() => {
                                navigator.clipboard.writeText(`${window.location.origin}${LINKS.Team}/${handle}`);
                                PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
                            }}
                            sx={{
                                color: palette.secondary.dark,
                                cursor: "pointer",
                                paddingBottom: 2,
                            }}
                        >@{handle}</Typography>
                    }
                    {/* Bio */}
                    {
                        (isLoading && !bio) ? (
                            <TextLoading lines={2} size="body1" sx={{ width: "85%" }} />
                        ) : (
                            <MarkdownDisplay
                                variant="body1"
                                sx={{ color: bio ? palette.background.textPrimary : palette.background.textSecondary }}
                                content={firstString(bio, "No bio set")}
                            />
                        )
                    }
                    <Stack direction="row" spacing={2} sx={{
                        alignItems: "center",
                    }}>
                        {/* Created date */}
                        <DateDisplay
                            loading={isLoading}
                            showIcon={true}
                            textBeforeDate="Created"
                            timestamp={team?.created_at}
                        />
                        <ReportsLink object={team} />
                    </Stack>
                </Stack>
            </OverviewContainer>
            {/* View routines, members, standards, and projects associated with this team */}
            <Box sx={{ margin: "auto", maxWidth: `min(${breakpoints.values.sm}px, 100%)` }}>
                <PageTabs
                    ariaLabel="team-tabs"
                    fullWidth
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                    sx={{
                        background: palette.background.paper,
                        borderBottom: `1px solid ${palette.divider}`,
                    }}
                />
                <Box>
                    {
                        currTab.key === TeamPageTabOption.Resource ? resources : (
                            <SearchList
                                {...findManyData}
                                display={display}
                                hideUpdateButton={true}
                                id="team-view-list"
                                searchPlaceholder={searchPlaceholderKey}
                                sxs={showSearchFilters ? {
                                    search: { marginTop: 2 },
                                    listContainer: { borderRadius: 0 },
                                } : {
                                    search: { display: "none" },
                                    buttons: { display: "none" },
                                    listContainer: { borderRadius: 0 },
                                }}
                            />
                        )
                    }
                </Box>
            </Box>
            <SideActionsButtons display={display}>
                {/* Toggle search filters */}
                {currTab.key !== TeamPageTabOption.Resource ? <SideActionsButton aria-label={t("FilterList")} onClick={toggleSearchFilters}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton> : null}
                {permissions.canUpdate ? <SideActionsButton aria-label={t("Edit")} onClick={() => { actionData.onActionStart("Edit"); }}>
                    <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton> : null}
                <SideActionsButton aria-label={t("Share")} onClick={() => { actionData.onActionStart("Share"); }}>
                    <ExportIcon fill={palette.secondary.contrastText} width='32px' height='32px' />
                </SideActionsButton>
            </SideActionsButtons>
        </>
    );
}
