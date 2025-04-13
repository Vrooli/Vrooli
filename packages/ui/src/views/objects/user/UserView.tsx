import { BookmarkFor, ChatShape, FindByIdOrHandleInput, LINKS, ListObject, User, UserPageTabOption, endpointsUser, findBotDataForForm, getAvailableModels, getObjectUrl, getTranslation, noop, uuid, uuidValidate } from "@local/shared";
import { Box, IconButton, InputAdornment, Stack, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getExistingAIConfig } from "../../../api/ai.js";
import BannerDefault from "../../../assets/img/BannerDefault.webp";
import BannerDefaultBot from "../../../assets/img/BannerDefaultBot.webp";
import { PageTabs } from "../../../components/PageTabs/PageTabs.js";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton.js";
import { ReportsLink } from "../../../components/buttons/ReportsLink.js";
import { SideActionsButtons } from "../../../components/buttons/SideActionsButtons.js";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { TextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { SearchList } from "../../../components/lists/SearchList/SearchList.js";
import { TextLoading } from "../../../components/lists/TextLoading/TextLoading.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { DateDisplay } from "../../../components/text/DateDisplay.js";
import { MarkdownDisplay } from "../../../components/text/MarkdownDisplay.js";
import { Title } from "../../../components/text/Title.js";
import { SessionContext } from "../../../contexts/session.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useFindMany } from "../../../hooks/useFindMany.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useTabs } from "../../../hooks/useTabs.js";
import { IconCommon, IconRoutine } from "../../../icons/Icons.js";
import { openLink } from "../../../route/openLink.js";
import { useLocation } from "../../../route/router.js";
import { BannerImageContainer, FormSection, OverviewContainer, OverviewProfileAvatar, OverviewProfileStack, ScrollBox } from "../../../styles.js";
import { PartialWithType } from "../../../types.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { extractImageUrl } from "../../../utils/display/imageTools.js";
import { defaultYou, getDisplay, getYou, placeholderColor } from "../../../utils/display/listTools.js";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "../../../utils/display/translationTools.js";
import { getCookieMatchingChat, getCookiePartialData, setCookiePartialData } from "../../../utils/localStorage.js";
import { UrlInfo, parseSingleItemUrl } from "../../../utils/navigation/urlTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { userTabParams } from "../../../utils/search/objectToSearch.js";
import { FeatureSlider } from "../bot/BotUpsert.js";
import { UserViewProps } from "./types.js";

const scrollContainerId = "user-search-scroll";
const BANNER_IMAGE_TARGET_SIZE_PX = 1000;

const StyledPageTabs = styled(PageTabs<typeof userTabParams>)(({ theme }) => ({
    background: theme.palette.background.paper,
    borderBottom: `1px solid ${theme.palette.divider}`,
}));
const HandleText = styled(Typography)(({ theme }) => ({
    color: theme.palette.secondary.dark,
    cursor: "pointer",
    paddingBottom: theme.spacing(2),
}));

const occupationInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Team" />
        </InputAdornment>
    ),
} as const;
const personaInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Persona" />
        </InputAdornment>
    ),
} as const;
const startMessageInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Comment" />
        </InputAdornment>
    ),
} as const;
const toneInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconRoutine name="RoutineValid" />
        </InputAdornment>
    ),
} as const;
const keyPhrasesInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="KeyPhrases" />
        </InputAdornment>
    ),
} as const;
const domainKnowledgeInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Learn" />
        </InputAdornment>
    ),
} as const;
const biasInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="HeartFilled" />
        </InputAdornment>
    ),
} as const;

const detailsSectionStyle = {
    overflowX: "hidden",
    marginTop: 0,
    borderRadius: "0px",
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
    const [getUserData, { data: userData, loading: isUserLoading }] = useLazyFetch<FindByIdOrHandleInput, User>(endpointsUser.findOne);
    const [getProfileData, { data: profileData, loading: isProfileLoading }] = useLazyFetch<undefined, User>(endpointsUser.profile);
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
        const { model, creativity, verbosity, translations } = findBotDataForForm(language, availableModels, user);
        const { bio, ...botTranslations } = getTranslation({ translations }, [language], true);
        const { adornments } = getDisplay(user, [language], palette);
        let bannerImageUrl = extractImageUrl(user?.bannerImage, user?.updated_at, BANNER_IMAGE_TARGET_SIZE_PX);
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
    const startActionEdit = useCallback(() => {
        actionData.onActionStart("Edit");
    }, [actionData]);
    const startActionShare = useCallback(() => {
        actionData.onActionStart("Share");
    }, [actionData]);

    const copyHandle = useCallback(() => {
        navigator.clipboard.writeText(`${window.location.origin}${LINKS.User}/${handle}`);
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }, [handle]);

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
            const { translations } = findBotDataForForm(bestLanguage, availableModels, user);
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
                        <IconCommon name={user?.isBot ? "Bot" : "User"} />
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
                            <IconCommon name="Ellipsis" fill="background.textSecondary" />
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
                        handle && <HandleText
                            variant="h6"
                            textAlign="center"
                            fontFamily="monospace"
                            onClick={copyHandle}
                        >@{handle}</HandleText>
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
                <StyledPageTabs
                    ariaLabel="user-tabs"
                    fullWidth
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />
                {currTab.key === UserPageTabOption.Details && (
                    <FormSection sx={detailsSectionStyle}>
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
                {currTab.key !== UserPageTabOption.Details ? <IconButton
                    aria-label={t("FilterList")}
                    onClick={toggleSearchFilters}
                >
                    <IconCommon name="Search" />
                </IconButton> : null}
                {permissions.canUpdate ? <IconButton
                    aria-label={t("Edit")}
                    onClick={startActionEdit}
                >
                    <IconCommon name="Edit" />
                </IconButton> : null}
                <IconButton
                    aria-label={t("Share")}
                    onClick={startActionShare}
                >
                    <IconCommon name="Export" />
                </IconButton>
                <IconButton
                    aria-label={t("AddToTeam")}
                    onClick={handleAddOrInvite}
                >
                    <IconCommon name="Add" />
                </IconButton>
                <IconButton
                    aria-label={t("MessageSend")}
                    onClick={handleStartChat}
                >
                    <IconCommon name="Comment" />
                </IconButton>
            </SideActionsButtons>
        </ScrollBox>
    );
}
