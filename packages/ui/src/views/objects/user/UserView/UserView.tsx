import { BookmarkFor, ChatShape, FindByIdOrHandleInput, LINKS, ListObject, User, UserPageTabOption, endpointGetProfile, endpointGetUser, findBotData, getAvailableModels, getObjectUrl, getTranslation, noop, uuid, uuidValidate } from "@local/shared";
import { Box, IconButton, InputAdornment, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { getExistingAIConfig } from "api/ai";
import BannerDefault from "assets/img/BannerDefault.webp";
import BannerDefaultBot from "assets/img/BannerDefaultBot.webp";
import { PageTabs } from "components/PageTabs/PageTabs";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts";
import { useObjectActions } from "hooks/objectActions";
import { useFindMany } from "hooks/useFindMany";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useTabs } from "hooks/useTabs";
import { AddIcon, BotIcon, CommentIcon, EditIcon, EllipsisIcon, ExportIcon, HeartFilledIcon, KeyPhrasesIcon, LearnIcon, PersonaIcon, RoutineValidIcon, SearchIcon, TeamIcon, UserIcon } from "icons";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { BannerImageContainer, FormSection, OverviewContainer, OverviewProfileAvatar, OverviewProfileStack, ScrollBox, SideActionsButton } from "styles";
import { PartialWithType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { extractImageUrl } from "utils/display/imageTools";
import { defaultYou, getDisplay, getYou, placeholderColor } from "utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { getCookieMatchingChat, getCookiePartialData, setCookiePartialData } from "utils/localStorage";
import { UrlInfo, parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { userTabParams } from "utils/search/objectToSearch";
import { FeatureSlider } from "views/objects/bot";
import { UserViewProps } from "../types";

const scrollContainerId = "user-search-scroll";

const occupationInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <TeamIcon />
        </InputAdornment>
    ),
} as const;
const personaInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <PersonaIcon />
        </InputAdornment>
    ),
} as const;
const startMessageInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <CommentIcon />
        </InputAdornment>
    ),
} as const;
const toneInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <RoutineValidIcon />
        </InputAdornment>
    ),
} as const;
const keyPhrasesInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <KeyPhrasesIcon />
        </InputAdornment>
    ),
} as const;
const domainKnowledgeInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <LearnIcon />
        </InputAdornment>
    ),
} as const;
const biasInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <HeartFilledIcon />
        </InputAdornment>
    ),
} as const;

export function UserView({
    display,
    onClose,
}: UserViewProps) {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [{ pathname }, setLocation] = useLocation();
    const { t } = useTranslation();
    const profileColors = useMemo(() => placeholderColor(), []);

    // Parse information from URL
    const urlInfo = useMemo<UrlInfo & { isOwnProfile: boolean }>(() => {
        // Use common function to parse URL
        let urlInfo: UrlInfo & { isOwnProfile?: boolean } = parseSingleItemUrl({ pathname });
        // If it returns the handle "profile", or the path it returns nothing, then it's the current user's profile
        if (session && (urlInfo.handle === "profile" || Object.keys(urlInfo).length === 0)) {
            urlInfo.isOwnProfile = true;
            const currentUser = getCurrentUser(session);
            urlInfo = { ...urlInfo, handle: currentUser?.handle ?? undefined, id: currentUser?.id };
        } else {
            urlInfo.isOwnProfile = false;
        }
        return urlInfo as UrlInfo & { isOwnProfile: boolean };
    }, [pathname, session]);
    // Logic to find user is a bit different from other objects, as "profile" is mapped to the current user
    const [getUserData, { data: userData, loading: isUserLoading }] = useLazyFetch<FindByIdOrHandleInput, User>(endpointGetUser);
    const [getProfileData, { data: profileData, loading: isProfileLoading }] = useLazyFetch<undefined, User>(endpointGetProfile);
    const [user, setUser] = useState<PartialWithType<User> | null | undefined>(() => getCookiePartialData<PartialWithType<User>>({ __typename: "User", id: urlInfo.id, handle: urlInfo.handle }));
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

    const { adornments, bannerImageUrl, bio, botData, name, handle } = useMemo(() => {
        const availableModels = getAvailableModels(getExistingAIConfig()?.service?.config);
        const { model, creativity, verbosity, translations } = findBotData(language, availableModels, user);
        const { bio, ...botTranslations } = getTranslation({ translations }, [language], true);
        const { adornments } = getDisplay(user, [language], palette);
        let bannerImageUrl = extractImageUrl(user?.bannerImage, user?.updated_at, 1000);
        if (!bannerImageUrl) bannerImageUrl = user?.isBot ? BannerDefaultBot : BannerDefault;
        return {
            adornments,
            bannerImageUrl,
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            botData: { ...botTranslations, model, creativity, verbosity },
            name: user?.name,
            handle: user?.handle,
        };
    }, [language, palette, user]);

    const availableTabs = useMemo(() => {
        // Details tab is only for bots
        if (user?.isBot) return userTabParams;
        return userTabParams.filter(tab => tab.key !== UserPageTabOption.Details);
    }, [user]);
    const {
        currTab,
        handleTabChange,
        searchPlaceholderKey,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "user-tabs", tabParams: availableTabs, display });

    const findManyData = useFindMany<ListObject>({
        canSearch: () => uuidValidate(user?.id ?? ""),
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where({ userId: user?.id ?? "", permissions }),
    });

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

    /** Opens dialog to add or invite user to a team/meeting/chat */
    const handleAddOrInvite = useCallback(() => {
        if (!user) return;
        // Users are invited, and bots are added (since you don't need permission to use a public bot)
        const needsInvite = !user.isBot;
        // TODO open dialog
    }, [user]);

    /** Starts a new chat */
    const handleStartChat = useCallback(() => {
        if (!user || !user.id) return;
        const yourId = getCurrentUser(session).id;
        if (!yourId) return;
        // Check for last chat you opened with this user
        const existingChatId = getCookieMatchingChat([yourId, user.id]);
        console.log("got existing chat id", existingChatId);
        // Use that to determine URL
        const url = existingChatId ? getObjectUrl({ __typename: "Chat" as const, id: existingChatId }) : `${LINKS.Chat}/add`;
        // Create search params to initialize new chat. 
        // If the chat isn't new, this will initialize the chat if the one 
        // we're looking for doesn't exist (i.e. it was deleted)
        const initialChatData = {
            invites: [{
                __typename: "ChatInvite" as const,
                id: uuid(),
                user: { __typename: "User", id: user.id } as Partial<User>,
            }] as Partial<ChatShape["invites"]>,
            translations: [{
                __typename: "ChatTranslation" as const,
                language: getUserLanguages(session)[0],
                name: user.name,
                description: "",
            }] as Partial<ChatShape["translations"]>,
        };
        // For bots, add a start message
        if (user.isBot) {
            const bestLanguage = getUserLanguages(session)[0];
            const availableModels = getAvailableModels(getExistingAIConfig()?.service?.config);
            const { translations } = findBotData(bestLanguage, availableModels, user);
            const bestTranslation = getTranslation({ translations }, [bestLanguage]);
            const startingMessage = bestTranslation.startingMessage ?? "";
            if (startingMessage.length > 0) {
                (initialChatData as unknown as { messages: Partial<ChatShape["messages"]> }).messages = [{
                    id: uuid(),
                    status: "unsent",
                    translations: [{
                        __typename: "ChatMessageTranslation" as const,
                        id: uuid(),
                        language,
                        text: startingMessage,
                    }],
                    user: {
                        __typename: "User" as const,
                        id: user.id,
                    },
                }] as Partial<ChatShape["messages"]>;
            }
        }
        openLink(setLocation, url, initialChatData);
    }, [language, session, setLocation, user]);

    return (
        <ScrollBox id={scrollContainerId}>
            <TopBar
                display={display}
                onClose={onClose}
                tabTitle={handle ? `${name} (@${handle})` : name}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={user}
                onClose={closeMoreMenu}
            />
            {/* Popup menu for adding/inviting to a team/meeting/chat */}
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
                    />}
                </Box>
            </BannerImageContainer>
            <OverviewContainer>
                <OverviewProfileStack>
                    <OverviewProfileAvatar
                        isBot={user?.isBot ?? false}
                        profileColors={profileColors}
                        src={extractImageUrl(user?.profileImage, user?.updated_at, 100)}
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
                            adornments={adornments}
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
                                content={bio ?? "No bio set"}
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
                        />
                        <ReportsLink object={user ? { ...user, reportsCount: user.reportsReceivedCount } : undefined} />
                    </Stack>
                </Stack>
            </OverviewContainer>
            {/* View routines, teams, standards, and projects associated with this user */}
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
                {currTab.key === UserPageTabOption.Details && (
                    <FormSection sx={{
                        overflowX: "hidden",
                        marginTop: 0,
                        borderRadius: "0px",
                    }}>
                        {botData.model && <TextInput
                            disabled
                            fullWidth
                            label={t("Model", { count: 1 })}
                            value={botData.model}
                        />}
                        {botData.occupation && <TextInput
                            disabled
                            fullWidth
                            label={t("Occupation")}
                            value={botData.occupation}
                            InputProps={occupationInputProps}
                        />}
                        {botData.persona && <TextInput
                            disabled
                            fullWidth
                            label={t("Persona")}
                            value={botData.persona}
                            InputProps={personaInputProps}
                        />}
                        {botData.startingMessage && <TextInput
                            disabled
                            fullWidth
                            label={t("StartMessage")}
                            value={botData.startingMessage}
                            InputProps={startMessageInputProps}
                        />}
                        {botData.tone && <TextInput
                            disabled
                            fullWidth
                            label={t("Tone")}
                            value={botData.tone}
                            InputProps={toneInputProps}
                        />}
                        {botData.keyPhrases && <TextInput
                            disabled
                            fullWidth
                            label={t("KeyPhrases")}
                            value={botData.keyPhrases}
                            InputProps={keyPhrasesInputProps}
                        />}
                        {botData.domainKnowledge && <TextInput
                            disabled
                            fullWidth
                            label={t("DomainKnowledge")}
                            value={botData.domainKnowledge}
                            InputProps={domainKnowledgeInputProps}
                        />}
                        {botData.bias && <TextInput
                            disabled
                            fullWidth
                            label={t("Bias")}
                            value={botData.bias}
                            InputProps={biasInputProps}
                        />}
                        <FeatureSlider
                            id="creativity-slider"
                            disabled
                            labelLeft={t("Low")}
                            labelRight={t("High")}
                            setValue={noop}
                            title={t("CreativityPlaceholder")}
                            value={botData.creativity ?? 0.5}
                        />
                        <FeatureSlider
                            id="verbosity-slider"
                            disabled
                            labelLeft={t("Low")}
                            labelRight={t("High")}
                            setValue={noop}
                            title={t("VerbosityPlaceholder")}
                            value={botData.verbosity ?? 0.5}
                        />
                    </FormSection>
                )}
                {currTab.key !== UserPageTabOption.Details && <Box>
                    <SearchList
                        {...findManyData}
                        borderRadius={0}
                        display={display}
                        hideUpdateButton={true}
                        searchPlaceholder={searchPlaceholderKey}
                        scrollContainerId={scrollContainerId}
                        variant={showSearchFilters ? "normal" : "minimal"}
                    />
                </Box>}
            </Box>
            <SideActionsButtons display={display}>
                {currTab.key !== UserPageTabOption.Details ? <SideActionsButton aria-label={t("FilterList")} onClick={toggleSearchFilters}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton> : null}
                {permissions.canUpdate ? <SideActionsButton aria-label={t("Edit")} onClick={() => { actionData.onActionStart("Edit"); }}>
                    <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton> : null}
                <SideActionsButton aria-label={t("Share")} onClick={() => { actionData.onActionStart("Share"); }}>
                    <ExportIcon fill={palette.secondary.contrastText} width='32px' height='32px' />
                </SideActionsButton>
                <SideActionsButton aria-label={t("AddToTeam")} onClick={handleAddOrInvite}>
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton>
                <SideActionsButton aria-label={t("MessageSend")} onClick={handleStartChat}>
                    <CommentIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton>
            </SideActionsButtons>
        </ScrollBox>
    );
}
