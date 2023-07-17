import { BookmarkFor, BotIcon, CommentIcon, EditIcon, EllipsisIcon, endpointGetProfile, endpointGetUser, FindByIdOrHandleInput, LINKS, OrganizationIcon, ProjectIcon, SvgComponent, User, UserIcon, uuidValidate, VisibilityType } from "@local/shared";
import { Avatar, Box, IconButton, LinearProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { SearchList } from "components/lists/SearchList/SearchList";
import { SearchListGenerator } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Title } from "components/text/Title/Title";
import { PageTab } from "components/types";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getLastUrlPart, useLocation } from "route";
import { OverviewContainer } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { defaultYou, getYou, placeholderColor, toSearchListData } from "utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { base36ToUuid } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SearchType } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { UserViewProps } from "../types";

enum TabOptions {
    Project = "Project",
    Organization = "Organization",
}

type TabParams = {
    Icon: SvgComponent;
    searchType: SearchType;
    tabType: TabOptions;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: TabParams[] = [{
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
    display = "page",
    onClose,
    partialData,
    zIndex,
}: UserViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const profileColors = useMemo(() => placeholderColor(), []);

    // Logic to find user is a bit different from other objects, as "profile" is mapped to the current user
    const [getUserData, { data: userData, errors: userErrors, loading: isUserLoading }] = useLazyFetch<FindByIdOrHandleInput, User>(endpointGetUser);
    const [getProfileData, { data: profileData, errors: profileErrors, loading: isProfileLoading }] = useLazyFetch<any, User>(endpointGetProfile);
    const [user, setUser] = useState<User | null | undefined>(null);
    useDisplayServerError(userErrors ?? profileErrors);
    useEffect(() => {
        const urlEnding = getLastUrlPart({});
        if (urlEnding && uuidValidate(base36ToUuid(urlEnding))) getUserData({ id: base36ToUuid(urlEnding) });
        else if (typeof urlEnding === "string" && urlEnding.toLowerCase() === "profile") getProfileData();
        else PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error" });
    }, [getUserData, getProfileData]);
    useEffect(() => {
        setUser(userData ?? profileData ?? partialData as any);
    }, [userData, profileData, partialData]);
    const permissions = useMemo(() => user ? getYou(user) : defaultYou, [user]);
    console.log("permissions", permissions, user);
    const isLoading = useMemo(() => isUserLoading || isProfileLoading, [isUserLoading, isProfileLoading]);

    const availableLanguages = useMemo<string[]>(() => (user?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [user?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { bio, name, handle } = useMemo(() => {
        const { bio } = getTranslation(user ?? partialData, [language]);
        return {
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            name: user?.name ?? partialData?.name,
            handle: user?.handle ?? partialData?.handle,
        };
    }, [language, partialData, user]);

    useEffect(() => {
        if (handle) document.title = `${name} ($${handle}) | Vrooli`;
        else document.title = `${name} | Vrooli`;
    }, [handle, name]);

    // Handle tabs
    const tabs = useMemo<PageTab<TabOptions>[]>(() => {
        const tabs = tabParams;
        // Return tabs shaped for the tab component
        return tabs.map((tab, i) => ({
            color: palette.secondary.dark,
            index: i,
            Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            value: tab.tabType,
        }));
    }, [palette.secondary.dark, t]);
    const [currTab, setCurrTab] = useState<PageTab<TabOptions>>(tabs[0]);
    const handleTabChange = useCallback((_: unknown, value: PageTab<TabOptions>) => setCurrTab(value), []);

    // Create search data
    const { searchType, placeholder, where } = useMemo<SearchListGenerator>(() => {
        if (!user?.id) return { searchType: SearchType.Project, placeholder: "SearchProject", where: {} };
        if (currTab.value === TabOptions.Organization)
            return toSearchListData("Organization", "SearchOrganization", { memberUserIds: [user.id], visibility: VisibilityType.All });
        return toSearchListData("Project", "SearchProject", { ownedByUserId: user.id, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All });
    }, [currTab.value, user?.id, permissions.canUpdate]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: user,
        objectType: "User",
        setLocation,
        setObject: setUser,
    });

    /**
     * Displays name, handle, avatar, bio, and quick links
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
                {user?.isBot ? <BotIcon
                    width="75%"
                    height="75%"
                /> : <UserIcon
                    width="75%"
                    height="75%"
                />}
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
                {/* Joined date */}
                <DateDisplay
                    loading={isLoading}
                    showIcon={true}
                    textBeforeDate="Joined"
                    timestamp={user?.created_at}
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
                        <MarkdownDisplay
                            variant="body1"
                            sx={{ color: bio ? palette.background.textPrimary : palette.background.textSecondary }}
                            content={bio ?? "No bio set"}
                            zIndex={zIndex}
                        />
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
                        disabled={user?.id === getCurrentUser(session).id}
                        objectId={user?.id ?? ""}
                        bookmarkFor={BookmarkFor.User}
                        isBookmarked={user?.you?.isBookmarked ?? false}
                        bookmarks={user?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                        zIndex={zIndex}
                    />
                    <ReportsLink object={user ? { ...user, reportsCount: user.reportsReceivedCount } : undefined} />
                </Stack>
            </Stack>
        </OverviewContainer>
    ), [profileColors, user, openMoreMenu, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.dark, isLoading, name, permissions.canUpdate, t, zIndex, handle, bio, session, actionData]);

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
                onClose={onClose}
                zIndex={zIndex}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={user}
                onClose={closeMoreMenu}
                zIndex={zIndex + 1}
            />
            <Box sx={{
                display: "flex",
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                background: palette.mode === "light" ? "#b2b3b3" : "#303030",
                position: "relative",
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
                        zIndex={zIndex}
                    />}
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
                    <SearchList
                        canSearch={() => Boolean(user?.id) && uuidValidate(user?.id)}
                        dummyLength={display === "page" ? 5 : 3}
                        handleAdd={permissions.canUpdate ? toAddNew : undefined}
                        hideUpdateButton={true}
                        id="user-view-list"
                        searchType={searchType}
                        searchPlaceholder={placeholder}
                        take={20}
                        where={where}
                        zIndex={zIndex}
                    />
                </Box>
            </Box>
            <SideActionButtons
                display={display}
                zIndex={zIndex + 2}
                sx={{ position: "fixed" }}
            >
                {/* Message button */}
                {user?.isBot ? (
                    <ColorIconButton aria-label="message" background={palette.secondary.main} onClick={() => { }} >
                        <CommentIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
            </SideActionButtons>
        </>
    );
};
