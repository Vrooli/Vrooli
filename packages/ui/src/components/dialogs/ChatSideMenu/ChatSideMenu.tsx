import { Chat, ChatPageTabOption, InboxPageTabOption, LINKS, ListObject, MyStuffPageTabOption, SearchPageTabOption, funcFalse, getObjectUrl, getObjectUrlBase, noop } from "@local/shared";
import { Box, BoxProps, Button, Divider, IconButton, SwipeableDrawer, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { ChatBubbleTree } from "components/ChatBubbleTree/ChatBubbleTree";
import { PageTabs } from "components/PageTabs/PageTabs";
import { ChatMessageInput } from "components/inputs/ChatMessageInput/ChatMessageInput";
import { SiteSearchBar } from "components/inputs/search";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ObjectListActions } from "components/lists/types";
import { ActiveChatContext, SessionContext } from "contexts";
import { useMessageActions, useMessageInput, useMessageTree } from "hooks/messages";
import { useIsLeftHanded } from "hooks/subscriptions";
import { UseChatTaskReturn, useChatTasks } from "hooks/tasks";
import { useFindMany } from "hooks/useFindMany";
import { useHistoryState } from "hooks/useHistoryState";
import { useSideMenu } from "hooks/useSideMenu";
import { useSocketChat } from "hooks/useSocketChat";
import { PageTab, useTabs } from "hooks/useTabs";
import { useWindowSize } from "hooks/useWindowSize";
import { AddIcon, ArrowRightIcon, ChatNewIcon, CloseIcon, CommentIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ArgsType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { LEFT_DRAWER_WIDTH } from "utils/consts";
import { getUserLanguages } from "utils/display/translationTools";
import { CHAT_SIDE_MENU_ID, PubSub, SideMenuPayloads } from "utils/pubsub";
import { ChatTabsInfo, TabParam, chatTabParams } from "utils/search/objectToSearch";

const SHORT_TAKE = 10;
const emptyArray: readonly [] = [];
const drawerPaperProps = { id: CHAT_SIDE_MENU_ID } as const;
const MESSAGE_LIST_ID = "chatSideMenuMessage";
/** Routes where the main content is already a chat, so the side chat should not be shown */
const CHAT_IS_MAIN_CONTENT_ROUTES = [
    LINKS.Chat,
    LINKS.Home,
] as string[];

const NewChatIconButton = styled(IconButton)(({ theme }) => ({
    position: "absolute",
    bottom: theme.spacing(1),
    right: theme.spacing(1),
    background: theme.palette.secondary.main,
    width: "36px",
    height: "36px",
}));

type ChatTabProps = ActiveChatContext & {
    taskInfo: UseChatTaskReturn;
}

function ChatTab({
    chat,
    isBotOnlyChat,
    isLoading,
    participants,
    resetActiveChat,
    setParticipants,
    setUsersTyping,
    taskInfo,
    usersTyping,
}: ChatTabProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [message, setMessage] = useHistoryState<string>(`${MESSAGE_LIST_ID}-message`, "");
    const messageTree = useMessageTree(chat?.id);
    const messageActions = useMessageActions({
        activeTask: taskInfo.activeTask,
        addMessages: messageTree.addMessages,
        chat,
        contexts: taskInfo.contexts[taskInfo.activeTask.taskId] || [],
        editMessage: messageTree.editMessage,
        isBotOnlyChat,
        language: languages[0],
        setMessage,
    });
    const messageInput = useMessageInput({
        id: MESSAGE_LIST_ID,
        languages,
        message,
        postMessage: messageActions.postMessage,
        putMessage: messageActions.putMessage,
        replyToMessage: messageActions.replyToMessage,
        setMessage,
    });
    const { messageStream } = useSocketChat({
        addMessages: messageTree.addMessages,
        chat,
        editMessage: messageTree.editMessage,
        participants,
        removeMessages: messageTree.removeMessages,
        setParticipants,
        setUsersTyping,
        usersTyping,
    });

    return (
        <>
            <ChatBubbleTree
                belowMessageList={<Tooltip title={t("NewChat")} placement="top">
                    <NewChatIconButton
                        aria-label="new chat"
                        onClick={resetActiveChat}
                    >
                        <ChatNewIcon fill="white" height={20} width={20} />
                    </NewChatIconButton>
                </Tooltip>}
                branches={messageTree.branches}
                handleEdit={messageInput.startEditingMessage}
                handleRegenerateResponse={messageActions.regenerateResponse}
                handleReply={messageInput.startReplyingToMessage}
                handleRetry={messageActions.retryPostMessage}
                isBotOnlyChat={isBotOnlyChat}
                isEditingMessage={Boolean(messageInput.messageBeingEdited)}
                isReplyingToMessage={Boolean(messageInput.messageBeingRepliedTo)}
                messageStream={messageStream}
                removeMessages={messageTree.removeMessages}
                setBranches={messageTree.setBranches}
                tree={messageTree.tree}
            />
            <ChatMessageInput
                disabled={!chat}
                display={"partial"}
                isLoading={isLoading}
                message={message}
                messageBeingEdited={messageInput.messageBeingEdited}
                messageBeingRepliedTo={messageInput.messageBeingRepliedTo}
                participantsAll={participants}
                participantsTyping={usersTyping}
                messagesCount={messageTree.messagesCount}
                placeholder={t("MessagePlaceholder")}
                setMessage={setMessage}
                stopEditingMessage={messageInput.stopEditingMessage}
                stopReplyingToMessage={messageInput.stopReplyingToMessage}
                submitMessage={messageInput.submitMessage}
                taskInfo={taskInfo}
            />
        </>
    );
}

/**
 * Tab for chat view, which is not available on all pages 
 * (namely ones where the main content is already a chat)
 */
const chatViewTab = {
    color: (palette) => palette.primary.contrastText,
    Icon: CommentIcon,
    key: "Chat",
    titleKey: "Chat",
    where: () => ({}),
} as const;

const NoResultsText = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    fontStyle: "italic",
    padding: theme.spacing(1),
    textAlign: "center",
}));

const TabsBox = styled(Box)(({ theme }) => ({
    background: theme.palette.primary.main,
}));

const SizedDrawer = styled(SwipeableDrawer)(() => ({
    width: LEFT_DRAWER_WIDTH,
    flexShrink: 0,
    "& .MuiDrawer-paper": {
        width: LEFT_DRAWER_WIDTH,
        boxSizing: "border-box",
    },
    "& > .MuiDrawer-root": {
        "& > .MuiPaper-root": {
        },
    },
}));

interface TabContentBoxProps extends BoxProps {
    currTabKey: PageTab<ChatTabsInfo>["key"];
}

const TabContentBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "currTabKey",
})<TabContentBoxProps>(({ currTabKey, theme }) => ({
    display: "flex",
    flexDirection: "column",
    height: "inherit",
    overflow: "auto",
    background: currTabKey === "Chat" ? theme.palette.background.default : theme.palette.background.paper,
}));

const searchBarStyle = {
    root: {
        width: "100%",
        "& > .MuiAutocomplete-popper": {
            display: "none",
        },
    },
} as const;

// TODO improve prompts so it searches for actual prompts, rather than just standards. Then update it so when pressed, it adds prompt to chat input
export function ChatSideMenu() {
    const { t } = useTranslation();
    const [location, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();
    console.log("qqqq user", session, userId);

    // Put here so that we can track tasks even when the chat tab is not open
    const chatContext = useContext(ActiveChatContext);
    const taskInfo = useChatTasks({ chatId: chatContext.chat?.id });

    const allTabParams = useMemo(() => {
        const baseTabs = [...chatTabParams];
        const canShowChatTab = !CHAT_IS_MAIN_CONTENT_ROUTES.some(route =>
            location.pathname === route || location.pathname.startsWith(`${route}/`),
        );
        if (canShowChatTab) {
            baseTabs.push(chatViewTab as TabParam<ChatTabsInfo>);
        }
        return baseTabs;
    }, [location.pathname]);
    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "chat-side-tabs", tabParams: allTabParams, display: "dialog" });
    useEffect(function leaveChatTab() {
        const canShowChatTab = !CHAT_IS_MAIN_CONTENT_ROUTES.some(route =>
            location.pathname === route || location.pathname.startsWith(`${route}/`),
        );
        if (!canShowChatTab && currTab.key === "Chat") {
            // Switch to the first non-ChatView tab
            const firstNonChatViewTab = tabs.find(tab => tab.key !== "Chat");
            if (firstNonChatViewTab) {
                handleTabChange(undefined, firstNonChatViewTab);
            }
        }
    }, [currTab.key, handleTabChange, location.pathname, tabs]);

    // Handle opening and closing
    const onEvent = useCallback(function onEventCallback({ data }: SideMenuPayloads["chat-side-menu"]) {
        if (!data) return;
        if (data.tab) {
            const matchingTab = tabs.find(tab => tab.key === data.tab);
            if (matchingTab) {
                handleTabChange(undefined, matchingTab);
            }
        }
    }, [handleTabChange, tabs]);
    const { isOpen, close } = useSideMenu({
        id: CHAT_SIDE_MENU_ID,
        isMobile,
        onEvent,
    });
    // When moving between mobile/desktop, publish current state
    useEffect(() => {
        PubSub.get().publish("sideMenu", { id: CHAT_SIDE_MENU_ID, isOpen });
    }, [breakpoints, isOpen]);
    const handleClose = useCallback(() => { close(); }, [close]);

    const addButtonData = useMemo<{ [key in Exclude<ChatPageTabOption, "Chat">]: (() => unknown) }>(() => ({
        History: () => { setLocation(`${getObjectUrlBase({ __typename: "Chat" })}/add`); },
        Prompt: () => { setLocation(`${getObjectUrlBase({ __typename: "Standard" })}/add`); },
        Routine: () => { setLocation(`${getObjectUrlBase({ __typename: "Routine" })}/add`); },
    }), [setLocation]);

    const more1ButtonData = useMemo<{ [key in Exclude<ChatPageTabOption, "Chat">]: (() => unknown) }>(() => ({
        History: () => { setLocation(`${LINKS.Inbox}?type="${InboxPageTabOption.Message}"`); },
        Prompt: () => { setLocation(`${LINKS.Search}?type="${SearchPageTabOption.Prompt}"`); },
        Routine: () => { setLocation(`${LINKS.Search}?type="${SearchPageTabOption.Routine}"`); },
    }), [setLocation]);

    const more2ButtonData = useMemo<{ [key in Exclude<ChatPageTabOption, "Chat">]: (() => unknown) }>(() => ({
        History: noop, //TODO
        Prompt: () => { setLocation(`${LINKS.MyStuff}?type="${MyStuffPageTabOption.Standard}"`); },
        Routine: () => { setLocation(`${LINKS.MyStuff}?type="${MyStuffPageTabOption.Routine}"`); },
    }), [setLocation]);

    // The "Routine" and "Prompt" tabs have two search results, so we'll have two search hooks
    const { where1, where2 } = useMemo(() => {
        if (typeof where !== "function") {
            return {
                where1: undefined,
                where2: undefined,
            };
        }
        const whereResult = where();
        if (Object.prototype.hasOwnProperty.call(whereResult, "My")) {
            return {
                where1: whereResult.My,
                where2: whereResult.Public,
            };
        }
        return {
            where1: whereResult,
            where2: undefined,
        };
    }, [where]);
    const {
        allData: allData1,
        loading: loading1,
        removeItem: removeItem1,
        searchString,
        setSearchString: setSearchString1,
        updateItem: updateItem1,
    } = useFindMany<ListObject>({
        canSearch: () => currTab.key !== "Chat",
        controlsUrl: false,
        searchType,
        take: SHORT_TAKE,
        where: where1,
    });
    const {
        allData: allData2,
        loading: loading2,
        removeItem: removeItem2,
        setSearchString: setSearchString2,
        updateItem: updateItem2,
    } = useFindMany<ListObject>({
        controlsUrl: false,
        searchType,
        take: SHORT_TAKE,
        where: where2,
    });
    const onAction1 = useCallback((action: keyof ObjectListActions<ListObject>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted":
                removeItem1(...(data as ArgsType<ObjectListActions<ListObject>["Deleted"]>));
                break;
            case "Updated":
                updateItem1(...(data as ArgsType<ObjectListActions<ListObject>["Updated"]>));
                break;
        }
    }, [removeItem1, updateItem1]);
    const onAction2 = useCallback((action: keyof ObjectListActions<ListObject>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted":
                removeItem2(...(data as ArgsType<ObjectListActions<ListObject>["Deleted"]>));
                break;
            case "Updated":
                updateItem2(...(data as ArgsType<ObjectListActions<ListObject>["Updated"]>));
                break;
        }
    }, [removeItem2, updateItem2]);
    const { title1, title2 } = useMemo(() => {
        if (!["Routine", "Prompt"].includes(currTab.key)) {
            return {
                title1: currTab.label,
                title2: undefined,
            };
        }
        return {
            title1: t(`${currTab.key}My`, { count: 2, defaultValue: currTab.label }),
            title2: t(`${currTab.key}Public`, { count: 2, defaultValue: currTab.label }),
        };
    }, [currTab, t]);
    const handleSearchStringChange = useCallback(function handleSearchCallback(newString: string) {
        setSearchString1(newString);
        setSearchString2(newString);
    }, [setSearchString1, setSearchString2]);

    const handleItemClick = useCallback((data: ListObject) => {
        // Chats can often be opened in the side menu instead of navigated to
        const chatTab = tabs.find(tab => tab.key === "Chat");
        const isActiveChat =
            chatTab &&
            data.__typename === "Chat" &&
            // Every participant is either you or a bot
            (data as Chat).participants?.every(p => p.user?.isBot === true || p.user?.id === userId);
        if (chatTab && isActiveChat) {
            chatContext.setActiveChat(data as Chat);
            handleTabChange(undefined, chatTab);
        }
        // Otherwise, navigate to item's page
        //TODO there will be cases for other objects. Routines can open as suggested tasks, and prompts as chat prompts
        else {
            setLocation(getObjectUrl(data));
        }
    }, [chatContext, tabs, handleTabChange, setLocation, userId]);

    return (
        <>
            <SizedDrawer
                // Displays opposite of main side menu
                anchor={isLeftHanded ? "right" : "left"}
                open={isOpen}
                onOpen={noop}
                onClose={handleClose}
                PaperProps={drawerPaperProps}
                variant={isMobile ? "temporary" : "persistent"}
            >
                {/* Menu title */}
                <Box
                    display="flex"
                    flexDirection="row"
                    alignItems="center"
                    justifyContent="space-between"
                    height="64px" // Matches Navbar height
                    bgcolor={palette.primary.dark}
                    color={palette.primary.contrastText}
                    p={1}
                >
                    <IconButton
                        aria-label="close"
                        onClick={handleClose}
                    >
                        <CloseIcon fill={palette.primary.contrastText} width="32px" height="32px" />
                    </IconButton>
                    <SiteSearchBar
                        id={"search-bar-chat-side-menu"}
                        isNested={true}
                        placeholder={"Search"}
                        value={searchString}
                        onChange={handleSearchStringChange}
                        onInputChange={noop}
                        sxs={searchBarStyle}
                    />
                </Box>
                <TabsBox>
                    <PageTabs
                        ariaLabel="chat-side-menu-tabs"
                        currTab={currTab}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                    />
                </TabsBox>
                <Divider />
                <TabContentBox currTabKey={currTab.key}>
                    {currTab.key === "Chat" ? (
                        <ChatTab
                            {...chatContext}
                            taskInfo={taskInfo}
                        />
                    ) : (
                        <>
                            <Box>
                                <Typography variant="h5" p={1}>{title1}</Typography>
                                <Divider />
                                <ObjectList
                                    canNavigate={funcFalse}
                                    dummyItems={new Array(SHORT_TAKE).fill(searchType)}
                                    handleToggleSelect={noop}
                                    hideUpdateButton={true}
                                    isSelecting={false}
                                    items={allData1}
                                    keyPrefix={`chat-search-${currTab.key}-list-item`}
                                    loading={loading1}
                                    onAction={onAction1}
                                    onClick={handleItemClick}
                                    selectedItems={emptyArray}
                                />
                                {allData1.length === 0 && <NoResultsText variant="body1">
                                    {t("NoResults", { ns: "error" })}
                                </NoResultsText>}
                                <Box display="flex" alignItems="center" justifyContent="space-between" pb={4}>
                                    <Button
                                        onClick={addButtonData[currTab.key]}
                                        startIcon={<AddIcon />}
                                        variant="text"
                                    >
                                        {t("Add")}
                                    </Button>
                                    <Button
                                        endIcon={<ArrowRightIcon />}
                                        onClick={more1ButtonData[currTab.key]}
                                        variant="text"
                                    >
                                        {t("More")}
                                    </Button>
                                </Box>
                            </Box>
                            {where2 && <>
                                <Box>
                                    <Typography variant="h5" p={1}>{title2}</Typography>
                                    <Divider />
                                    <ObjectList
                                        canNavigate={funcFalse}
                                        dummyItems={new Array(SHORT_TAKE).fill(searchType)}
                                        handleToggleSelect={noop}
                                        hideUpdateButton={true}
                                        isSelecting={false}
                                        items={allData2}
                                        keyPrefix={`chat-search-${searchType}-list-item`}
                                        loading={loading2}
                                        onAction={onAction2}
                                        onClick={handleItemClick}
                                        selectedItems={emptyArray}
                                    />
                                    {allData2.length === 0 && <NoResultsText variant="body1">
                                        {t("NoResults", { ns: "error" })}
                                    </NoResultsText>}
                                    <Box display="flex" alignItems="center" justifyContent="space-between" pb={4}>
                                        <Button
                                            onClick={addButtonData[currTab.key]}
                                            startIcon={<AddIcon />}
                                            variant="text"
                                        >
                                            {t("Add")}
                                        </Button>
                                        <Button
                                            endIcon={<ArrowRightIcon />}
                                            onClick={more2ButtonData[currTab.key]}
                                            variant="text"
                                        >
                                            {t("More")}
                                        </Button>
                                    </Box>
                                </Box>
                            </>}
                        </>
                    )}
                </TabContentBox>
            </SizedDrawer>
        </>
    );
}
