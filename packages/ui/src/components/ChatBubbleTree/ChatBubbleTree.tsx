import { ChatMessageShape, ChatMessageStatus, ChatSocketEventPayloads, ListObject, NavigableObject, ReactInput, ReactionFor, ReactionSummary, ReportFor, Success, endpointsReaction, getObjectUrl, getTranslation, noop } from "@local/shared";
import { Box, BoxProps, CircularProgress, CircularProgressProps, IconButton, IconButtonProps, Stack, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { green, red } from "@mui/material/colors";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { EmojiPicker } from "components/EmojiPicker/EmojiPicker";
import { ReportButton } from "components/buttons/ReportButton/ReportButton";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { ChatBubbleProps } from "components/types";
import { SessionContext } from "contexts";
import { usePress } from "hooks/gestures";
import { MessageTree } from "hooks/messages";
import { useDeleter } from "hooks/objectActions";
import { useDimensions } from "hooks/useDimensions";
import { useLazyFetch } from "hooks/useLazyFetch";
import { ArrowDownIcon, BotIcon, ChevronLeftIcon, ChevronRightIcon, CopyIcon, DeleteIcon, EditIcon, ErrorIcon, RefreshIcon, ReplyIcon, UserIcon } from "icons";
import { Dispatch, RefObject, SetStateAction, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "route";
import { ProfileAvatar } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { extractImageUrl } from "utils/display/imageTools";
import { getDisplay, placeholderColor } from "utils/display/listTools";
import { fontSizeToPixels } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import { BranchMap } from "utils/localStorage";
import { PubSub } from "utils/pubsub";

const PERCENTS = 100;
const STILL_SENDING_MAX_PERCENT = 90;
const SEND_STATUS_INTERVAL_MS = 50;
const HIDE_SEND_SUCCESS_DELAY_MS = 1000;
const TARGET_PROFILE_IMAGE_SIZE_PX = 50;

interface ChatBubbleSendingSpinnerProps extends CircularProgressProps {
    status: ChatMessageStatus;
}

const ChatBubbleSendingSpinner = styled(CircularProgress, {
    shouldForwardProp: (prop) => prop !== "status",
})<ChatBubbleSendingSpinnerProps>(({ status, theme }) => ({
    color: status === "sending"
        ? theme.palette.secondary.main
        : status === "failed"
            ? red[500]
            : green[500],
}));

const ChatBubbleErrorIconButton = styled(IconButton)(() => ({
    color: red[500],
}));
const ChatBubbleEditIconButton = styled(IconButton)(() => ({
    color: green[500],
}));
const ChatBubbleDeleteIconButton = styled(IconButton)(() => ({
    color: red[500],
}));

type ChatBubbleStatusProps = {
    onDelete: () => unknown;
    onEdit: () => unknown;
    onRetry: () => unknown;
    /** Indicates if the edit and delete buttons should be shown */
    showButtons: boolean;
    status: ChatMessageStatus;
}

/**
 * Displays a visual indicator for the status of a chat message (that you sent).
 * It shows a CircularProgress that progresses as the message is sending,
 * and changes color and icon based on the success or failure of the operation.
 */
export function ChatBubbleStatus({
    onDelete,
    onEdit,
    onRetry,
    showButtons,
    status,
}: ChatBubbleStatusProps) {
    const { t } = useTranslation();
    const [progress, setProgress] = useState(0);
    const [isCompleted, setIsCompleted] = useState(false);

    useEffect(function chatStatusLoaderEffect() {
        // Updates the progress value every 100ms, but stops at 90 if the message is still sending
        let timer: NodeJS.Timeout;
        if (status === "sending") {
            timer = setInterval(() => {
                setProgress((oldProgress) => {
                    if (oldProgress === PERCENTS) {
                        setIsCompleted(true);
                        return PERCENTS;
                    }
                    const diff = 3;
                    return Math.min(oldProgress + diff, status === "sending" ? STILL_SENDING_MAX_PERCENT : PERCENTS);
                });
            }, SEND_STATUS_INTERVAL_MS);
        }

        // Cleans up the interval when the component is unmounted or the message is not sending anymore
        return () => {
            if (timer) clearInterval(timer);
        };
    }, [status]);

    useEffect(function chatStatusTimeoutEffect() {
        // Resets the progress and completion state after a delay when the sending has completed
        if (isCompleted && status !== "sending") {
            const timer = setTimeout(() => {
                setProgress(0);
                setIsCompleted(false);
            }, HIDE_SEND_SUCCESS_DELAY_MS);
            return () => {
                clearTimeout(timer);
            };
        }
    }, [isCompleted, status]);

    // While the message is sending or has just completed, show a CircularProgress
    if (status === "sending" || isCompleted) {
        return (
            <ChatBubbleSendingSpinner
                variant="determinate"
                value={progress}
                size={24}
                status={status}
            />
        );
    }

    // If there was an error, show an ErrorIcon
    if (status === "failed") {
        return (
            <ChatBubbleErrorIconButton
                aria-label={t("Retry")}
                onClick={onRetry}
            >
                <ErrorIcon />
            </ChatBubbleErrorIconButton>
        );
    }
    // If allowed to show buttons, show edit and delete buttons
    if (showButtons) return (
        <>
            <ChatBubbleEditIconButton
                aria-label={t("Edit")}
                onClick={onEdit}
            >
                <EditIcon />
            </ChatBubbleEditIconButton>
            <ChatBubbleDeleteIconButton
                aria-label={t("Delete")}
                onClick={onDelete}
            >
                <DeleteIcon />
            </ChatBubbleDeleteIconButton>
        </>
    );
    // Otherwise, show nothing
    return null;
}

const NavigationBox = styled(Box)(() => ({
    display: "flex",
    alignItems: "center",
}));

export function NavigationArrows({
    activeIndex,
    numSiblings,
    onIndexChange,
}: {
    activeIndex: number,
    numSiblings: number,
    onIndexChange: (newIndex: number) => unknown,
}) {
    const { palette } = useTheme();

    // eslint-disable-next-line no-magic-numbers
    if (numSiblings < 2) {
        return null; // Do not render anything if there are less than 2 siblings
    }

    function handleActiveIndexChange(newIndex: number) {
        if (newIndex >= 0 && newIndex < numSiblings) {
            onIndexChange(newIndex);
        }
    }
    function handlePrevious() {
        handleActiveIndexChange(activeIndex - 1);
    }
    function handleNext() {
        handleActiveIndexChange(activeIndex + 1);
    }

    return (
        <NavigationBox>
            <IconButton
                size="small"
                onClick={handlePrevious}
                disabled={activeIndex <= 0}
                aria-label="left"
            >
                <ChevronLeftIcon fill={palette.background.textSecondary} />
            </IconButton>
            <Typography variant="body2" color="textSecondary">
                {activeIndex + 1}/{numSiblings}
            </Typography>
            <IconButton
                size="small"
                onClick={handleNext}
                disabled={activeIndex >= numSiblings - 1}
                aria-label="right"
            >
                <ChevronRightIcon fill={palette.background.textSecondary} />
            </IconButton>
        </NavigationBox>
    );
}

type ChatBubbleReactionsProps = {
    activeIndex: number,
    handleActiveIndexChange: (newIndex: number) => unknown,
    handleCopy: () => unknown,
    handleReactionAdd: (emoji: string) => unknown,
    handleReply: () => unknown,
    handleRegenerateResponse: () => unknown,
    isBot: boolean,
    isBotOnlyChat: boolean,
    isMobile: boolean;
    isOwn: boolean,
    numSiblings: number,
    messageId: string,
    reactions: ReactionSummary[],
    status: ChatMessageStatus,
}

interface ReactionsOuterBoxProps extends BoxProps {
    isOwn: boolean;
}

const ReactionsOuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isOwn",
})<ReactionsOuterBoxProps>(({ isOwn }) => ({
    display: "flex",
    alignItems: "center",
    flexDirection: "row",
    justifyContent: isOwn ? "flex-end" : "flex-start",
}));

interface ReactionsMainStackProps extends BoxProps {
    isMobile: boolean;
    isOwn: boolean;
}

const ReactionsMainStack = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isMobile" && prop !== "isOwn",
})<ReactionsMainStackProps>(({ isMobile, isOwn, theme }) => ({
    display: "flex",
    flexDirection: "row",
    padding: 0,
    marginLeft: (isOwn || isMobile) ? 0 : theme.spacing(6),
    marginRight: isOwn ? theme.spacing(6) : 0,
    paddingRight: isOwn ? theme.spacing(1) : 0,
    background: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
    borderRadius: "0 0 8px 8px",
    boxShadow: `1px 2px 3px rgba(0,0,0,0.2),
                1px 2px 2px rgba(0,0,0,0.14)`,
    overflow: "overlay",
}));

const reactionIconButtonStyle = { borderRadius: 0, background: "transparent" } as const;

/**
 * Displays message reactions and actions (i.e. refresh and report). 
 * Reactions are displayed as a list of emojis on the left, and bot actions are displayed
 * as a list of icons on the right.
 */
function ChatBubbleReactions({
    activeIndex,
    handleActiveIndexChange,
    handleCopy,
    handleReactionAdd,
    handleReply,
    handleRegenerateResponse,
    isBot,
    isBotOnlyChat,
    isMobile,
    isOwn,
    numSiblings,
    messageId,
    reactions,
    status,
}: ChatBubbleReactionsProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    function onReactionAdd(emoji: string) {
        handleReactionAdd(emoji);
    }

    if (status === "unsent") return null;
    return (
        <ReactionsOuterBox isOwn={isOwn}>
            <ReactionsMainStack isMobile={isMobile} isOwn={isOwn}>
                {reactions.map((reaction) => {
                    function handleClick() {
                        onReactionAdd(reaction.emoji);
                    }

                    return (
                        <Box key={reaction.emoji} display="flex" alignItems="center">
                            <IconButton
                                size="small"
                                disabled={isOwn}
                                onClick={handleClick}
                                style={reactionIconButtonStyle}
                            >
                                {reaction.emoji}
                            </IconButton>
                            <Typography variant="body2">
                                {reaction.count}
                            </Typography>
                        </Box>
                    );
                })}
                {!isOwn && <EmojiPicker onSelect={onReactionAdd} />}
            </ReactionsMainStack>
            <Stack direction="row">
                <Tooltip title={t("Copy")}>
                    <IconButton size="small" onClick={handleCopy}>
                        <CopyIcon fill={palette.background.textSecondary} />
                    </IconButton>
                </Tooltip>
                {(isBot && isBotOnlyChat) && <Tooltip title={t("Retry")}>
                    <IconButton size="small" onClick={handleRegenerateResponse}>
                        <RefreshIcon fill={palette.background.textSecondary} />
                    </IconButton>
                </Tooltip>}
                {isBot && <Tooltip title={t("Reply")}>
                    <IconButton size="small" onClick={handleReply}>
                        <ReplyIcon fill={palette.background.textSecondary} />
                    </IconButton>
                </Tooltip>}
                {isBot && <ReportButton forId={messageId} reportFor={ReportFor.ChatMessage} />}
                <NavigationArrows
                    activeIndex={activeIndex}
                    numSiblings={numSiblings}
                    onIndexChange={handleActiveIndexChange}
                />
            </Stack>
        </ReactionsOuterBox>
    );
}

const ChatBubbleOuterBox = styled(Box)(() => ({
    display: "flex",
    flexDirection: "column",
    padding: "8px",
    maxWidth: "100vw",
}));

interface ChatBubbleBoxProps extends BoxProps {
    isOwn: boolean;
    messageStatus: ChatMessageStatus | undefined;
}
const ChatBubbleBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isOwn" && prop !== "messageStatus",
})<ChatBubbleBoxProps>(({ isOwn, messageStatus, theme }) => ({
    padding: theme.spacing(1),
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
    marginLeft: isOwn ? "auto" : 0,
    marginRight: isOwn ? 0 : "auto",
    background: !isOwn && messageStatus === "failed"
        ? theme.palette.error.dark
        : isOwn
            ? theme.palette.mode === "light" ? "#88d17e" : "#1a5413"
            : theme.palette.background.paper,
    color: !isOwn && messageStatus === "failed"
        ? theme.palette.error.contrastText
        : theme.palette.background.textPrimary,
    borderRadius: isOwn ? "8px 8px 0 8px" : "8px 8px 8px 0",
    boxShadow: theme.spacing(2),
    minWidth: "50px",
    width: "unset",
    minHeight: "20x",
    transition: "width 0.3s ease-in-out",
}));

const UserNameDisplay = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    alignItems: "center",
    spacing: theme.spacing(0.5),
    marginBottom: theme.spacing(0.5),
    cursor: "pointer",
    "&:hover": { textDecoration: "underline" },
}));
const AdornmentBox = styled(Box)(() => ({
    width: fontSizeToPixels("0.85rem") * Number("1.5"),
    height: fontSizeToPixels("0.85rem") * Number("1.5"),
}));

const markdownDisplayStyle = {
    whiteSpace: "pre-wrap",
    wordWrap: "break-word",
    overflowWrap: "anywhere",
    minHeight: "unset",
} as const;
const profileAvatarStyle = {
    boxShadow: 2,
    cursor: "pointer",
    marginRight: 1,
} as const;

export function ChatBubble({
    activeIndex,
    chatWidth,
    isBotOnlyChat,
    isOwn,
    message,
    numSiblings,
    onActiveIndexChange,
    onDeleted,
    onEdit,
    onRegenerateResponse,
    onReply,
    onRetry,
}: ChatBubbleProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { breakpoints } = useTheme();
    const isMobile = useMemo(() => chatWidth <= breakpoints.values.sm, [breakpoints, chatWidth]);
    const profileColors = useMemo(() => placeholderColor(message.user?.id), [message.user?.id]);

    const [react] = useLazyFetch<ReactInput, Success>(endpointsReaction.createOne);

    const {
        handleDelete,
        DeleteDialogComponent,
    } = useDeleter({
        object: message as ListObject,
        objectType: "ChatMessage",
        onActionComplete: () => { onDeleted(message); },
    });

    // const shouldRetry = useRef(true);
    // useEffect(() => {
    //     if (message.user?.id === getCurrentUser(session).id && message.isUnsent && shouldRetry.current) {
    //         shouldRetry.current = false;
    //         fetchLazyWrapper<ChatMessageCreateInput, ChatMessage>({
    //             fetch: createMessage,
    //             inputs: shapeChatMessage.create({ ...message }),
    //             successCondition: (data) => data !== null,
    //             onSuccess: (data) => {
    //                 setEditingText(undefined);
    //                 console.log("chatbubble setting error false 1", data);
    //                 setHasError(false);
    //                 onUpdated({ ...data, isUnsent: false });
    //             },
    //         });
    //     }
    // }, [createMessage, message, message.isUnsent, onUpdated, session, shouldRetry]);

    function handleReactionAdd(emoji: string) {
        if (message.status !== "sent") return;
        const originalSummaries = message.reactionSummaries;
        // Add to summaries right away, so that the UI updates immediately
        // const existingReaction = message.reactionSummaries.find((r) => r.emoji === emoji);
        // if (existingReaction) {
        //     onUpdated({
        //         ...message,
        //         reactionSummaries: message.reactionSummaries.map((r) => r.emoji === emoji ? { ...r, count: r.count + 1 } : r),
        //     } as ChatBubbleProps["message"]);
        // } else {
        //     onUpdated({
        //         ...message,
        //         reactionSummaries: [...message.reactionSummaries, { __typename: "ReactionSummary", emoji, count: 1 }],
        //     } as ChatBubbleProps["message"]);
        // }
        // Send the request to the backend
        fetchLazyWrapper<ReactInput, Success>({
            fetch: react,
            inputs: {
                emoji,
                reactionFor: ReactionFor.ChatMessage,
                forConnect: message.id,
            },
            successCondition: (data) => data.success,
            onError: () => {
                // If the request fails, revert the UI changes
                // onUpdated({ ...message, reactionSummaries: originalSummaries } as ChatBubbleProps["message"]);
            },
        });
    }

    function startEdit() {
        onEdit(message);
    }
    function startRegenerateResponse() {
        onRegenerateResponse(message);
    }
    function startReply() {
        onReply(message);
    }
    function startRetry() {
        onRetry(message);
    }

    function handleCopy() {
        navigator.clipboard.writeText(getTranslation(message, getUserLanguages(session), true)?.text ?? "");
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }

    const { name, handle, adornments } = useMemo(() => {
        const { title, adornments } = getDisplay(message.user as ListObject);
        return {
            name: title,
            handle: message.user?.handle,
            adornments,
        };
    }, [message.user]);

    const [bubblePressed, setBubblePressed] = useState(false);
    function toggleBubblePressed() {
        if (!isMobile && bubblePressed) return;
        setBubblePressed(!bubblePressed);
    }
    const pressEvents = usePress({
        onLongPress: toggleBubblePressed,
        onClick: toggleBubblePressed,
    });
    useEffect(() => {
        function handleResize() {
            if (!isMobile && !bubblePressed) setBubblePressed(true);
            if (isMobile && bubblePressed) setBubblePressed(false);
        }
        window.addEventListener("resize", handleResize);
        return () => {
            window.removeEventListener("resize", handleResize);
        };
    }, [bubblePressed, isMobile]);

    const profileUrl = useMemo(function getProfileUrl() {
        return getObjectUrl(message.user as NavigableObject);
    }, [message.user]);
    const openProfile = useCallback(function openProfileCallback() {
        setLocation(profileUrl);
    }, [profileUrl, setLocation]);

    return (
        <>
            {DeleteDialogComponent}
            <ChatBubbleOuterBox key={message.id}>
                {!isOwn && (
                    <Link to={profileUrl}>
                        <UserNameDisplay>
                            <Typography variant="body2">
                                {name}
                            </Typography>
                            {adornments.map(({ Adornment, key }) => (
                                <AdornmentBox key={key}>
                                    {Adornment}
                                </AdornmentBox>
                            ))}
                            {handle && (
                                <Typography variant="body2" color="textSecondary">
                                    @{handle}
                                </Typography>
                            )}
                        </UserNameDisplay>
                    </Link>
                )}
                {/* Avatar, chat bubble, and status indicator */}
                <Stack direction="row" justifyContent={isOwn ? "flex-end" : "flex-start"}>
                    {!isOwn && !isMobile && (
                        <ProfileAvatar
                            src={extractImageUrl(message.user?.profileImage, message.user?.updated_at, TARGET_PROFILE_IMAGE_SIZE_PX)}
                            alt={message.user?.name ?? message.user?.handle ?? message?.user?.isBot ? "Bot" : "User"}
                            isBot={message.user?.isBot === true}
                            onClick={openProfile}
                            profileColors={profileColors}
                            sx={profileAvatarStyle}
                        >
                            {message.user?.isBot ? <BotIcon width="75%" height="75%" /> : <UserIcon width="75%" height="75%" />}
                        </ProfileAvatar>
                    )}
                    <ChatBubbleBox
                        {...pressEvents}
                        isOwn={isOwn}
                        messageStatus={message.status}
                    >
                        <MarkdownDisplay
                            content={getTranslation(message, getUserLanguages(session), true)?.text}
                            sx={markdownDisplayStyle}
                        />
                    </ChatBubbleBox>
                    {/* Status indicator and edit/retry buttons */}
                    {isOwn && (
                        <Box display="flex" alignItems="center">
                            <ChatBubbleStatus
                                onDelete={handleDelete}
                                onEdit={startEdit}
                                onRetry={startRetry}
                                showButtons={bubblePressed}
                                status={message.status ?? "sent"}
                            />
                        </Box>
                    )}
                </Stack>
                {/* Reactions */}
                <ChatBubbleReactions
                    activeIndex={activeIndex}
                    handleActiveIndexChange={onActiveIndexChange}
                    handleCopy={handleCopy}
                    handleReactionAdd={handleReactionAdd}
                    handleReply={startReply}
                    handleRegenerateResponse={startRegenerateResponse}
                    isBot={message.user?.isBot ?? false}
                    isBotOnlyChat={isBotOnlyChat}
                    isMobile={isMobile}
                    isOwn={isOwn}
                    numSiblings={numSiblings}
                    messageId={message.id}
                    reactions={message.reactionSummaries}
                    status={message.status ?? "sent"}
                />
            </ChatBubbleOuterBox>
        </>
    );
}

const SCROLL_POSITION_CHECK_INTERVAL_MS = 2_000;

interface ScrollToBottomIconButtonProps extends IconButtonProps {
    isVisible: boolean;
}

const ScrollToBottomIconButton = styled(IconButton, {
    shouldForwardProp: (prop) => prop !== "isVisible",
})<ScrollToBottomIconButtonProps>(({ isVisible, theme }) => ({
    background: theme.palette.background.paper,
    position: "absolute",
    bottom: theme.spacing(1),
    left: "50%",
    transform: "translateX(-50%)",
    width: "36px",
    height: "36px",
    // eslint-disable-next-line no-magic-numbers
    opacity: isVisible ? 0.8 : 0,
    transition: "opacity 0.2s ease-in-out !important",
}));

export function ScrollToBottomButton({
    containerRef,
}: {
    containerRef: RefObject<HTMLElement>,
}) {
    const { palette } = useTheme();
    const [isVisible, setIsVisible] = useState(false);

    useEffect(() => {
        function checkScrollPosition() {
            const scroll = containerRef.current;
            if (!scroll) return;

            // Threshold to determine "close to the bottom"
            // Adjust this value based on your needs
            const threshold = 100;
            const isCloseToBottom = scroll.scrollHeight - scroll.scrollTop - scroll.clientHeight < threshold;

            setIsVisible(!isCloseToBottom);
        }

        const scroll = containerRef.current;
        scroll?.addEventListener("scroll", checkScrollPosition);
        setTimeout(checkScrollPosition, SCROLL_POSITION_CHECK_INTERVAL_MS);
        return () => {
            scroll?.removeEventListener("scroll", checkScrollPosition);
        };
    }, [containerRef]); // Re-run effect if containerRef changes

    function scrollToBottom() {
        const scroll = containerRef.current;
        if (!scroll) return;
        // Directly using smooth scrolling to the calculated bottom
        scroll.scroll({
            top: scroll.scrollHeight - scroll.clientHeight,
            behavior: "smooth",
        });
    }

    return (
        <ScrollToBottomIconButton
            isVisible={isVisible}
            onClick={scrollToBottom}
            size="small"
        >
            <ArrowDownIcon fill={palette.background.textSecondary} />
        </ScrollToBottomIconButton>
    );
}

type MessageRenderData = {
    activeIndex: number;
    key: string;
    numSiblings: number;
    onActiveIndexChange: (newIndex: number) => unknown;
    onDeleted: (message: ChatMessageShape) => unknown;
    message: ChatMessageShape;
    isOwn: boolean;
}

type ChatBubbleTreeProps = {
    belowMessageList?: React.ReactNode,
    branches: BranchMap,
    handleEdit: (originalMessage: ChatMessageShape) => unknown,
    handleRegenerateResponse: (message: ChatMessageShape) => unknown,
    handleReply: (message: ChatMessageShape) => unknown,
    handleRetry: (message: ChatMessageShape) => unknown,
    isBotOnlyChat: boolean,
    id?: string;
    isEditingMessage: boolean,
    isReplyingToMessage: boolean,
    messageStream: ChatSocketEventPayloads["responseStream"] | null,
    removeMessages: (messageIds: string[]) => unknown,
    setBranches: Dispatch<SetStateAction<BranchMap>>,
    tree: MessageTree<ChatMessageShape>,
};

const OuterMessageList = styled(Box)(() => ({
    position: "relative",
    display: "flex",
    flexDirection: "column",
    overflowX: "hidden",
    overflowY: "auto",
    height: "100%",
    justifyContent: "space-between",
}));

interface InnerMessageListProps extends BoxProps {
    isEditingOrReplying: boolean;
}
const InnerMessageList = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isEditingOrReplying",
})<InnerMessageListProps>(({ isEditingOrReplying }) => ({
    filter: isEditingOrReplying ? "blur(20px)" : "none",
}));

const BelowMessageListButtonsBox = styled(Box)(({ theme }) => ({
    position: "sticky",
    bottom: 0,
    padding: theme.spacing(1),
    height: "52px",
}));

export function ChatBubbleTree({
    belowMessageList,
    branches,
    handleEdit,
    handleRegenerateResponse,
    handleReply,
    handleRetry,
    id,
    isBotOnlyChat,
    isEditingMessage,
    isReplyingToMessage,
    messageStream,
    removeMessages,
    setBranches,
    tree,
}: ChatBubbleTreeProps) {
    const session = useContext(SessionContext);
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const { dimensions, ref: dimRef } = useDimensions();

    const messageData = useMemo<MessageRenderData[]>(() => {
        function renderMessage(withSiblings: string[], activeIndex: number): MessageRenderData[] {
            // Find information for current message
            const siblingId = withSiblings[activeIndex];
            const sibling = siblingId ? tree.map.get(siblingId) : null;
            if (!sibling) return [];
            const isOwn = sibling.message.user?.id === userId;

            // Find information for next message
            // Check the stored branch data first
            let childId = branches[siblingId];
            // Fallback to first child if no branch data is found
            if (!childId) childId = sibling.children[0];
            const activeChildIndex = Math.max(sibling.children.findIndex(id => id === childId), 0);

            if (!sibling) return [];
            return [
                {
                    activeIndex,
                    key: sibling.message.id,
                    numSiblings: withSiblings.length,
                    onActiveIndexChange: (newIndex) => {
                        const siblingId = withSiblings[newIndex];
                        const parentId = sibling.message.parent?.id;
                        if (!siblingId) return;
                        if (!parentId) return; // TODO if root message, should reorder root
                        setBranches(prev => ({
                            ...prev,
                            [parentId]: siblingId,
                        }));
                    },
                    onDeleted: (message) => { removeMessages([message.id]); },
                    message: sibling.message,
                    isOwn,
                },
                ...childId ? renderMessage(sibling.children, activeChildIndex) : [],
            ];
        }
        return renderMessage(tree.roots, 0);
    }, [branches, removeMessages, setBranches, tree.map, tree.roots, userId]);

    return (
        <OuterMessageList id={id} ref={dimRef}>
            <InnerMessageList isEditingOrReplying={isEditingMessage || isReplyingToMessage}>
                {messageData.map((data) => (
                    <ChatBubble
                        key={data.key}
                        activeIndex={data.activeIndex}
                        chatWidth={dimensions.width}
                        isBotOnlyChat={isBotOnlyChat}
                        numSiblings={data.numSiblings}
                        onActiveIndexChange={data.onActiveIndexChange}
                        onDeleted={data.onDeleted}
                        onEdit={handleEdit}
                        onRegenerateResponse={handleRegenerateResponse}
                        onReply={handleReply}
                        onRetry={handleRetry}
                        message={data.message}
                        isOwn={data.isOwn}
                    />
                ))}
                {messageStream && (
                    <ChatBubble
                        key="streamingMessage"
                        activeIndex={0}
                        chatWidth={dimensions.width}
                        isBotOnlyChat={isBotOnlyChat}
                        numSiblings={1}
                        onActiveIndexChange={noop}
                        onDeleted={noop}
                        onEdit={noop}
                        onRegenerateResponse={noop}
                        onReply={noop}
                        onRetry={noop}
                        message={{
                            id: "streamingMessage",
                            reactionSummaries: [],
                            status: messageStream.__type === "error" ? "failed" : "sending",
                            translations: [{
                                language: "en",
                                text: messageStream.message,
                            }],
                            user: messageStream.botId && {
                                id: messageStream.botId,
                                isBot: true,
                            },
                            versionIndex: 0,
                        } as unknown as ChatMessageShape}
                        isOwn={false}
                    />
                )}
            </InnerMessageList>
            {!(isEditingMessage || isReplyingToMessage) && <BelowMessageListButtonsBox>
                <ScrollToBottomButton containerRef={dimRef} />
                {belowMessageList}
            </BelowMessageListButtonsBox>}
        </OuterMessageList>
    );
}
