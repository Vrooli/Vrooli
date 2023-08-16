import { BookmarkFor, CommonKey, endpointGetProfile, endpointGetUser, FindByIdOrHandleInput, LINKS, User, VisibilityType } from "@local/shared";
import { Box, IconButton, Slider, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Title } from "components/text/Title/Title";
import { AddIcon, BotIcon, CommentIcon, EditIcon, EllipsisIcon, SearchIcon, UserIcon } from "icons";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { BannerImageContainer, FormSection, OverviewContainer, OverviewProfileAvatar, OverviewProfileStack } from "styles";
import { PartialWithType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { findBotData } from "utils/botUtils";
import { getCookiePartialData, setCookiePartialData } from "utils/cookies";
import { extractImageUrl } from "utils/display/imageTools";
import { defaultYou, getYou, placeholderColor, YouInflated } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useTabs } from "utils/hooks/useTabs";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SearchType, UserPageTabOption } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { UserViewProps } from "../types";

type TabWhereParams = {
    userId: string;
    permissions: YouInflated;
}

const tabColor = (palette: Palette) => ({ active: palette.secondary.main, inactive: palette.background.textSecondary });
const tabParams = [{
    color: tabColor,
    titleKey: "Details" as CommonKey,
    searchType: SearchType.User, // Ignored
    tabType: UserPageTabOption.Details,
    where: () => ({}),
}, {
    color: tabColor,
    titleKey: "Project" as CommonKey,
    searchPlaceholderKey: "SearchProject" as CommonKey,
    searchType: SearchType.Project,
    tabType: UserPageTabOption.Project,
    where: ({ userId, permissions }: TabWhereParams) => ({ ownedByUserId: userId, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All }),
}, {
    color: tabColor,
    titleKey: "Organization" as CommonKey,
    searchPlaceholderKey: "SearchOrganization" as CommonKey,
    searchType: SearchType.Organization,
    tabType: UserPageTabOption.Organization,
    where: ({ userId }: TabWhereParams) => ({ memberUserIds: [userId], visibility: VisibilityType.All }),
}];

export const UserView = ({
    isOpen,
    onClose,
    zIndex,
}: UserViewProps) => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);
    const profileColors = useMemo(() => placeholderColor(), []);

    // Parse information from URL
    const urlInfo = useMemo(() => {
        // Use common function to parse URL
        let urlInfo = { ...parseSingleItemUrl({}), isOwnProfile: false };
        // If it returns a handle of "profile", it's not actually a handle - it's the current user
        if (urlInfo.handle === "profile" && session) {
            urlInfo.isOwnProfile = true;
            const currentUser = getCurrentUser(session);
            urlInfo = { ...urlInfo, handle: currentUser?.handle ?? undefined, id: currentUser?.id };
        }
        return urlInfo;
    }, [session]);
    // Logic to find user is a bit different from other objects, as "profile" is mapped to the current user
    const [getUserData, { data: userData, errors: userErrors, loading: isUserLoading }] = useLazyFetch<FindByIdOrHandleInput, User>(endpointGetUser);
    const [getProfileData, { data: profileData, errors: profileErrors, loading: isProfileLoading }] = useLazyFetch<undefined, User>(endpointGetProfile);
    const [user, setUser] = useState<PartialWithType<User> | null | undefined>(() => getCookiePartialData<PartialWithType<User>>({ __typename: "User", id: urlInfo.id, handle: urlInfo.handle }));
    console.log("got user data", user);
    useDisplayServerError(userErrors ?? profileErrors);
    // Get user or profile data
    useEffect(() => {
        if (urlInfo.isOwnProfile) getProfileData();
        else if (urlInfo.id) getUserData({ id: urlInfo.id });
        else if (urlInfo.handle) getUserData({ handle: urlInfo.handle });
    }, [getUserData, getProfileData, urlInfo]);
    // Set user data
    useEffect(() => {
        const knownData = userData ?? profileData;
        // If there is knownData, update local storage
        if (knownData) setCookiePartialData(knownData, "full");
        setUser(knownData ?? getCookiePartialData<PartialWithType<User>>({ __typename: "User", id: urlInfo.id, handle: urlInfo.handle }));
    }, [userData, profileData, urlInfo]);
    const permissions = useMemo(() => user ? getYou(user) : defaultYou, [user]);
    const isLoading = useMemo(() => isUserLoading || isProfileLoading, [isUserLoading, isProfileLoading]);

    const availableLanguages = useMemo<string[]>(() => (user?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [user?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { bannerImageUrl, bio, botData, name, handle } = useMemo(() => {
        const { creativity, verbosity, translations } = findBotData(language, user);
        const { bio, ...botTranslations } = getTranslation({ translations }, [language]);
        return {
            bannerImageUrl: extractImageUrl(user?.bannerImage, user?.updated_at, 1000),
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            botData: { ...botTranslations, creativity, verbosity },
            name: user?.name,
            handle: user?.handle,
        };
    }, [language, user]);

    const availableTabs = useMemo(() => {
        // Details tab is only for bots
        if (user?.isBot) return tabParams;
        return tabParams.filter(tab => tab.tabType !== UserPageTabOption.Details);
    }, [user]);
    const {
        currTab,
        handleTabChange,
        searchPlaceholderKey,
        searchType,
        tabs,
        where,
    } = useTabs<UserPageTabOption>({ tabParams: availableTabs, display });

    const [showSearchFilters, setShowSearchFilters] = useState<boolean>(false);
    const toggleSearchFilters = useCallback(() => setShowSearchFilters(!showSearchFilters), [showSearchFilters]);
    // If showing search filter, focus the search input
    useEffect(() => {
        if (!showSearchFilters) return;
        const searchInput = document.getElementById("search-bar-user-view-list");
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
        object: user,
        objectType: "User",
        setLocation,
        setObject: setUser,
    });

    /** Opens add new page */
    const toAddNew = useCallback(() => {
        setLocation(`${LINKS[currTab.tabType]}/add`);
    }, [currTab.tabType, setLocation]);

    /** Opens dialog to add or invite user to an organization/meeting/chat */
    const handleAddOrInvite = useCallback(() => {
        if (!user) return;
        // Users are invited, and bots are added (since you don't need permission to use a public bot)
        const needsInvite = !user.isBot;
        // TODO open dialog
    }, [user]);

    /** Starts a new chat */
    const handleStartChat = useCallback(() => {
        if (!user) return;
        // TODO
    }, [user]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                tabTitle={handle ? `${name} (@${handle})` : name}
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
            {/* Popup menu for adding/inviting to an organization/meeting/chat */}
            {/* TODO */}
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
                        zIndex={zIndex}
                    />}
                </Box>
            </BannerImageContainer>
            <OverviewContainer>
                <OverviewProfileStack>
                    <OverviewProfileAvatar
                        src={extractImageUrl(user?.profileImage, user?.updated_at, 100)}
                        sx={{
                            backgroundColor: profileColors[0],
                            color: profileColors[1],
                            // Bots show up as squares, to distinguish them from users
                            ...(user?.isBot ? { borderRadius: "8px" } : {}),
                        }}
                    >
                        {user?.isBot ?
                            <BotIcon width="75%" height="75%" /> :
                            <UserIcon width="75%" height="75%" />}
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
                        disabled={user?.id === getCurrentUser(session).id}
                        objectId={user?.id ?? ""}
                        bookmarkFor={BookmarkFor.User}
                        isBookmarked={user?.you?.isBookmarked ?? false}
                        bookmarks={user?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                        zIndex={zIndex}
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
                            options={permissions.canUpdate ? [{
                                label: t("Edit"),
                                Icon: EditIcon,
                                onClick: () => { actionData.onActionStart("Edit"); },
                            }] : []}
                            zIndex={zIndex}
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
                                navigator.clipboard.writeText(`${window.location.origin}${LINKS.User}/${handle}`);
                                PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
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
                                content={bio ?? "No bio set"}
                                zIndex={zIndex}
                            />
                        )
                    }
                    <Stack direction="row" spacing={2} sx={{
                        alignItems: "center",
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
            {/* View routines, organizations, standards, and projects associated with this user */}
            <Box sx={{ margin: "auto", maxWidth: `min(${breakpoints.values.sm}px, 100%)` }}>
                <PageTabs
                    ariaLabel="user-tabs"
                    fullWidth
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                    sx={{
                        background: palette.background.paper,
                        borderBottom: `1px solid ${palette.divider}`,
                    }}
                />
                {currTab.tabType === UserPageTabOption.Details && (
                    <FormSection sx={{
                        overflowX: "hidden",
                        marginTop: 0,
                        borderRadius: "0px",
                    }}>
                        {botData.occupation && <TextField
                            disabled
                            fullWidth
                            label={t("Occupation")}
                            value={botData.occupation}
                        />}
                        {botData.persona && <TextField
                            disabled
                            fullWidth
                            label={t("Persona")}
                            value={botData.persona}
                        />}
                        {botData.startMessage && <TextField
                            disabled
                            fullWidth
                            label={t("StartMessage")}
                            value={botData.startMessage}
                        />}
                        {botData.tone && <TextField
                            disabled
                            fullWidth
                            label={t("Tone")}
                            value={botData.tone}
                        />}
                        {botData.keyPhrases && <TextField
                            disabled
                            fullWidth
                            label={t("KeyPhrases")}
                            value={botData.keyPhrases}
                        />}
                        {botData.domainKnowledge && <TextField
                            disabled
                            fullWidth
                            label={t("DomainKnowledge")}
                            value={botData.domainKnowledge}
                        />}
                        {botData.bias && <TextField
                            disabled
                            fullWidth
                            label={t("Bias")}
                            value={botData.bias}
                        />}
                        <Stack>
                            <Typography id="creativity-slider" gutterBottom>
                                {t("Creativity")}
                            </Typography>
                            <Slider
                                aria-labelledby="creativity-slider"
                                disabled
                                value={botData.creativity as number}
                                valueLabelDisplay="auto"
                                min={0.1}
                                max={1}
                                step={0.1}
                                marks={[
                                    {
                                        value: 0.1,
                                        label: t("Low"),
                                    },
                                    {
                                        value: 1,
                                        label: t("High"),
                                    },
                                ]}
                                sx={{
                                    "& .MuiSlider-markLabel": {
                                        "&[data-index=\"0\"]": {
                                            marginLeft: 2,
                                        },
                                        "&[data-index=\"1\"]": {
                                            marginLeft: -2,
                                        },
                                    },
                                }}
                            />
                        </Stack>
                        <Stack>
                            <Typography id="verbosity-slider" gutterBottom>
                                {t("Verbosity")}
                            </Typography>
                            <Slider
                                aria-labelledby="verbosity-slider"
                                disabled
                                value={botData.verbosity as number}
                                valueLabelDisplay="auto"
                                min={0.1}
                                max={1}
                                step={0.1}
                                marks={[
                                    {
                                        value: 0.1,
                                        label: t("Low"),
                                    },
                                    {
                                        value: 1,
                                        label: t("High"),
                                    },
                                ]}
                                sx={{
                                    "& .MuiSlider-markLabel": {
                                        "&[data-index=\"0\"]": {
                                            marginLeft: 2,
                                        },
                                        "&[data-index=\"1\"]": {
                                            marginLeft: -2,
                                        },
                                    },
                                }}
                            />
                        </Stack>
                    </FormSection>
                )}
                {currTab.tabType !== UserPageTabOption.Details && <Box>
                    <SearchList
                        dummyLength={display === "page" ? 5 : 3}
                        handleAdd={permissions.canUpdate ? toAddNew : undefined}
                        hideUpdateButton={true}
                        id="user-view-list"
                        searchType={searchType}
                        searchPlaceholder={searchPlaceholderKey}
                        sxs={showSearchFilters ? {
                            search: { marginTop: 2 },
                            listContainer: { borderRadius: 0 },
                        } : {
                            search: { display: "none" },
                            buttons: { display: "none" },
                            listContainer: { borderRadius: 0 },
                        }}
                        take={20}
                        where={where({ userId: user?.id ?? "", permissions })}
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
                {currTab.tabType !== UserPageTabOption.Details ? <ColorIconButton aria-label="filter-list" background={palette.secondary.main} onClick={toggleSearchFilters} >
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton> : null}
                {/* Add/invite to meeting/chat button */}
                <ColorIconButton aria-label="message" background={palette.secondary.main} onClick={handleAddOrInvite} >
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton>
                {/* Message button */}
                <ColorIconButton aria-label="message" background={palette.secondary.main} onClick={handleStartChat} >
                    <CommentIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton>
            </SideActionButtons>
        </>
    );
};
