import { BookmarkFor, CommonKey, endpointGetProfile, endpointGetUser, FindByIdOrHandleInput, LINKS, User, uuidValidate, VisibilityType } from "@local/shared";
import { Avatar, Box, IconButton, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { SearchListGenerator } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Title } from "components/text/Title/Title";
import { PageTab } from "components/types";
import { BotIcon, CommentIcon, EditIcon, EllipsisIcon, InfoIcon, OrganizationIcon, ProjectIcon, SearchIcon, UserIcon } from "icons";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getLastUrlPart, useLocation } from "route";
import { OverviewContainer } from "styles";
import { SvgComponent } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { findBotData } from "utils/botUtils";
import { extractImageUrl } from "utils/display/imageTools";
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
    Details = "Details",
    Project = "Project",
    Organization = "Organization",
}

type TabParams = {
    Icon: SvgComponent;
    searchType: SearchType;
    tabType: Omit<TabOptions, TabOptions.Details>;
    where: { [x: string]: any };
} | {
    Icon: SvgComponent;
    tabType: TabOptions.Details;
}

// Data for each tab
const tabParams: TabParams[] = [
    // Only available for bots
    {
        Icon: InfoIcon,
        tabType: TabOptions.Details,
    }, {
        Icon: ProjectIcon,
        searchType: SearchType.Project,
        tabType: TabOptions.Project,
        where: {},
    }, {
        Icon: OrganizationIcon,
        searchType: SearchType.Organization,
        tabType: TabOptions.Organization,
        where: {},
    },
];

export const UserView = ({
    display = "page",
    onClose,
    partialData,
    zIndex,
}: UserViewProps) => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
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

    const { bannerImageUrl, bio, botData, name, handle } = useMemo(() => {
        const { creativity, verbosity, translations } = findBotData(language, user ?? partialData as User | null | undefined);
        const { bio, ...botTranslations } = getTranslation({ translations }, [language]);
        return {
            bannerImageUrl: extractImageUrl(user?.bannerImage, user?.updated_at, 1000),
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            botData: { ...botTranslations, creativity, verbosity },
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
        // Remove details tab if not a bot
        const tabs = user?.isBot ? tabParams : tabParams.filter(t => t.tabType !== TabOptions.Details);
        // Return tabs shaped for the tab component
        return tabs.map((tab, i) => ({
            color: palette.secondary.dark,
            index: i,
            Icon: tab.Icon,
            label: t(tab.tabType as CommonKey, { count: 2, defaultValue: tab.tabType }),
            value: tab.tabType,
        })) as PageTab<TabOptions>[];
    }, [palette.secondary.dark, t, user?.isBot]);
    const [currTab, setCurrTab] = useState<PageTab<TabOptions>>(tabs[0]);
    const handleTabChange = useCallback((_: unknown, value: PageTab<TabOptions>) => setCurrTab(value), []);
    useEffect(() => {
        setCurrTab(tabs[0]);
    }, [tabs]);

    // Create search data
    const searchData = useMemo<SearchListGenerator | null>(() => {
        if (!user || !user.id || !uuidValidate(user.id) || currTab.value === TabOptions.Details) return null;
        if (currTab.value === TabOptions.Organization)
            return toSearchListData("Organization", "SearchOrganization", { memberUserIds: [user.id], visibility: VisibilityType.All });
        return toSearchListData("Project", "SearchProject", { ownedByUserId: user.id, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All });
    }, [user, currTab.value, permissions.canUpdate]);

    const [showSearchFilters, setShowSearchFilters] = useState<boolean>(false);
    const toggleSearchFilters = useCallback(() => setShowSearchFilters(!showSearchFilters), [showSearchFilters]);

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
            <Stack direction="row" spacing={1} sx={{
                height: "48px",
                marginLeft: 2,
                marginRight: 2,
                marginTop: 1,
                alignItems: "flex-start",
                // Apply auto margin to the second element to push the first one to the left
                "& > :nth-child(2)": {
                    marginLeft: "auto",
                },
            }}>
                <Avatar
                    src={extractImageUrl(user?.profileImage, user?.updated_at, 100)}
                    sx={{
                        backgroundColor: profileColors[0],
                        color: profileColors[1],
                        boxShadow: 2,
                        width: "max(min(100px, 40vw), 75px)",
                        height: "max(min(100px, 40vw), 75px)",
                        top: "-100%",
                        fontSize: "min(50px, 10vw)",
                        marginRight: "auto",
                        // Bots show up as squares, to distinguish them from users
                        ...(user?.isBot ? { borderRadius: "8px" } : {}),
                        // Show in center on large screens
                        [breakpoints.up("sm")]: {
                            position: "absolute",
                            left: "50%",
                            top: "-25%",
                            transform: "translateX(-50%)",
                        },
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
                <BookmarkButton
                    disabled={user?.id === getCurrentUser(session).id}
                    objectId={user?.id ?? ""}
                    bookmarkFor={BookmarkFor.User}
                    isBookmarked={user?.you?.isBookmarked ?? false}
                    bookmarks={user?.bookmarks ?? 0}
                    onChange={(isBookmarked: boolean) => { }}
                    zIndex={zIndex}
                />
            </Stack>
            <Stack direction="column" p={2} justifyContent="center" sx={{
                alignItems: "flex-start",
                [breakpoints.up("sm")]: {
                    alignItems: "center",
                },
            }}>
                {/* Title */}
                {
                    isLoading ? (
                        <TextLoading size="header" sx={{ width: "50%" }} />
                    ) : <Title
                        title={name}
                        variant="header"
                        options={permissions.canUpdate ? [{
                            label: t("Edit"),
                            Icon: EditIcon,
                            onClick: () => { actionData.onActionStart("Edit"); },
                        }] : []}
                        zIndex={zIndex}
                        sxs={{ stack: { padding: 0, paddingBottom: 2 } }}
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
                {/* Bio */}
                {
                    isLoading ? (
                        <TextLoading lines={2} size="body1" sx={{ width: "85%" }} />
                    ) : (
                        <MarkdownDisplay
                            variant="body1"
                            sx={{ color: bio ? palette.background.textPrimary : palette.background.textSecondary }}
                            content={bio ?? "No bio set"}
                            zIndex={zIndex}
                        />
                    )
                }
                <Stack direction="row" spacing={2} sx={{
                    alignItems: "center",
                    [breakpoints.up("sm")]: {
                        alignItems: "flex-start",
                    },
                }}>
                    {/* Joined date */}
                    <DateDisplay
                        loading={isLoading}
                        showIcon={true}
                        textBeforeDate="Joined"
                        timestamp={user?.created_at}
                        zIndex={zIndex}
                    />
                    <ReportsLink object={user ? { ...user, reportsCount: user.reportsReceivedCount } : undefined} />
                </Stack>
            </Stack>
        </OverviewContainer>
    ), [user, profileColors, breakpoints, t, openMoreMenu, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.dark, session, zIndex, isLoading, name, permissions.canUpdate, handle, bio, actionData]);

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
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                backgroundColor: palette.mode === "light" ? "#b2b3b3" : "#303030",
                backgroundImage: bannerImageUrl ? `url(${bannerImageUrl})` : undefined,
                backgroundSize: "cover",
                backgroundPosition: "center",
                position: "relative",
                paddingTop: "40px",
                [breakpoints.down("sm")]: {
                    paddingTop: "120px",
                },
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
            <Box sx={{ margin: "auto", maxWidth: "800px" }}>
                <PageTabs
                    ariaLabel="user-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                    sx={{
                        [breakpoints.down("sm")]: {
                            background: palette.background.paper,
                            borderBottom: `1px solid ${palette.divider}`,
                        },
                    }}
                />
                {currTab.value === TabOptions.Details && (
                    <Stack direction="column" spacing={2} sx={{ padding: 2 }}>
                        {botData.occupation && <Typography variant="h6">Occupation: {botData.occupation}</Typography>}
                        {botData.persona && <Typography variant="h6">Persona: {botData.persona}</Typography>}
                        {botData.startMessage && <Typography variant="h6">Starting Message: {botData.startMessage}</Typography>}
                        {botData.tone && <Typography variant="h6">Tone: {botData.tone}</Typography>}
                        {botData.keyPhrases && <Typography variant="h6">Key Phrases: {botData.keyPhrases}</Typography>}
                        {botData.domainKnowledge && <Typography variant="h6">Domain Knowledge: {botData.domainKnowledge}</Typography>}
                        {botData.bias && <Typography variant="h6">Bias: {botData.bias}</Typography>}
                        {botData.creativity && <Typography variant="h6">Creativity: {botData.creativity * 100}%</Typography>}
                        {botData.verbosity && <Typography variant="h6">Verbosity: {botData.verbosity * 100}%</Typography>}
                    </Stack>
                )}
                {searchData !== null && currTab.value !== TabOptions.Details && <Box>
                    <SearchList
                        canSearch={() => true}
                        dummyLength={display === "page" ? 5 : 3}
                        handleAdd={permissions.canUpdate ? toAddNew : undefined}
                        hideUpdateButton={true}
                        id="user-view-list"
                        searchType={searchData.searchType}
                        searchPlaceholder={searchData.placeholder}
                        sxs={showSearchFilters ? {
                            search: { marginTop: 2 },
                        } : {
                            search: { display: "none" },
                            buttons: { display: "none" },
                        }}
                        take={20}
                        where={searchData.where}
                        zIndex={zIndex}
                    />
                </Box>}
            </Box>
            <SideActionButtons
                display={display}
                zIndex={zIndex + 2}
                sx={{ position: "fixed" }}
            >
                {/* Toggle search filters */}
                {currTab.value !== TabOptions.Details ? <ColorIconButton aria-label="filter-list" background={palette.secondary.main} onClick={toggleSearchFilters} >
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton> : null}
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
