import { ChatMessageShape, ChatMessageStatus, ChatSocketEventPayloads, ListObject, NavigableObject, ReactInput, ReactionFor, ReactionSummary, ReportFor, Success, endpointsReaction, getObjectUrl, getTranslation, noop } from "@local/shared";
import { Box, BoxProps, CircularProgress, CircularProgressProps, IconButton, Stack, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { Dispatch, RefObject, SetStateAction, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { SessionContext } from "../../contexts/session.js";
import { usePress } from "../../hooks/gestures.js";
import { MessageTree } from "../../hooks/messages.js";
import { useDeleter } from "../../hooks/objectActions.js";
import { useDimensions } from "../../hooks/useDimensions.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { IconCommon } from "../../icons/Icons.js";
import { Link, useLocation } from "../../route/router.js";
import { ProfileAvatar } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { extractImageUrl } from "../../utils/display/imageTools.js";
import { getDisplay, placeholderColor } from "../../utils/display/listTools.js";
import { fontSizeToPixels } from "../../utils/display/stringTools.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { BranchMap } from "../../utils/localStorage.js";
import { PubSub } from "../../utils/pubsub.js";
import { EmojiPicker } from "../EmojiPicker/EmojiPicker.js";
import { ReportButton } from "../buttons/ReportButton/ReportButton.js";
import { MarkdownDisplay } from "../text/MarkdownDisplay.js";
import { ChatBubbleProps } from "../types.js";

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
            ? theme.palette.error.main
            : theme.palette.success.main,
}));

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
    onDelete: () => unknown;
    onEdit: () => unknown;
    onRetry: () => unknown;
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
            <Tooltip title="Previous">
                <IconButton
                    size="small"
                    onClick={handlePrevious}
                    disabled={activeIndex <= 0}
                >
                    <IconCommon
                        decorative
                        fill="background.textSecondary"
                        name="ChevronLeft"
                    />
                </IconButton>
            </Tooltip>
            <Typography variant="body2" color="textSecondary">
                {activeIndex + 1}/{numSiblings}
            </Typography>
            <Tooltip title="Next">
                <IconButton
                    size="small"
                    onClick={handleNext}
                    disabled={activeIndex >= numSiblings - 1}
                >
                    <IconCommon
                        decorative
                        fill="background.textSecondary"
                        name="ChevronRight"
                    />
                </IconButton>
            </Tooltip>
        </NavigationBox>
    );
}

interface ChatBubbleOuterBoxProps extends BoxProps {
    showAll: boolean;
}
const ChatBubbleOuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "showAll",
})<ChatBubbleOuterBoxProps>(({ showAll }) => ({
    display: "flex",
    flexDirection: "column",
    padding: "8px",
    maxWidth: "100vw",
    ".chat-bubble-actions": {
        opacity: showAll ? 1 : 0,
        transition: "opacity 0.2s ease-in-out",
    },
    "&:hover .chat-bubble-actions": {
        opacity: 1,
    },
}));

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
    transition: "opacity 0.2s ease-in-out",
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
    // eslint-disable-next-line no-magic-numbers
    marginLeft: (isOwn || isMobile) ? 0 : theme.spacing(6),
    // eslint-disable-next-line no-magic-numbers
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
    onDelete,
    onEdit,
    onRetry,
}: ChatBubbleReactionsProps) {
    const { t } = useTranslation();
    const [progress, setProgress] = useState(0);
    const [isCompleted, setIsCompleted] = useState(false);

    useEffect(function chatStatusLoaderEffect() {
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
        return () => {
            if (timer) clearInterval(timer);
        };
    }, [status]);

    useEffect(function chatStatusTimeoutEffect() {
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

    function onReactionAdd(emoji: string) {
        handleReactionAdd(emoji);
    }

    if (status === "unsent") return null;

    // Render status indicator if message is sending/failed
    const statusIndicator = status === "sending" || isCompleted ? (
        <ChatBubbleSendingSpinner
            variant="determinate"
            value={progress}
            size={24}
            status={status}
        />
    ) : status === "failed" ? (
        <Tooltip title={t("Retry")}>
            <IconButton
                color="error"
                onClick={onRetry}
            >
                <IconCommon
                    decorative
                    name="Error"
                />
            </IconButton>
        </Tooltip>
    ) : null;

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
            <Stack direction="row" alignItems="center">
                {statusIndicator}
                <NavigationArrows
                    activeIndex={activeIndex}
                    numSiblings={numSiblings}
                    onIndexChange={handleActiveIndexChange}
                />
                <Box className="chat-bubble-actions">
                    <Tooltip title={t("Copy")}>
                        <IconButton
                            size="small"
                            onClick={handleCopy}
                        >
                            <IconCommon
                                decorative
                                fill="background.textSecondary"
                                name="Copy"
                            />
                        </IconButton>
                    </Tooltip>
                    {(isBot && isBotOnlyChat) && <Tooltip title={t("Retry")}>
                        <IconButton size="small" onClick={handleRegenerateResponse}>
                            <IconCommon
                                decorative
                                fill="background.textSecondary"
                                name="Refresh"
                            />
                        </IconButton>
                    </Tooltip>}
                    {isBot && <Tooltip title={t("Reply")}>
                        <IconButton size="small" onClick={handleReply}>
                            <IconCommon
                                decorative
                                fill="background.textSecondary"
                                name="Reply"
                            />
                        </IconButton>
                    </Tooltip>}
                    {isBot && <ReportButton forId={messageId} reportFor={ReportFor.ChatMessage} />}
                    {isOwn && status === "sent" && (
                        <>
                            <Tooltip title={t("Edit")}>
                                <IconButton
                                    onClick={onEdit}
                                    size="small"
                                >
                                    <IconCommon
                                        decorative
                                        fill="background.textSecondary"
                                        name="Edit"
                                    />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title={t("Delete")}>
                                <IconButton
                                    onClick={onDelete}
                                    size="small"
                                >
                                    <IconCommon
                                        decorative
                                        fill="background.textSecondary"
                                        name="Delete"
                                    />
                                </IconButton>
                            </Tooltip>
                        </>
                    )}
                </Box>
            </Stack>
        </ReactionsOuterBox>
    );
}

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

    function handleReactionAdd(emoji: string) {
        if (message.status !== "sent") return;
        const originalSummaries = message.reactionSummaries;
        // Add to summaries right away, so that the UI updates immediately
        const existingReaction = message.reactionSummaries.find((r) => r.emoji === emoji);
        if (existingReaction) {
            onDeleted({
                ...message,
                reactionSummaries: message.reactionSummaries.map((r) => r.emoji === emoji ? { ...r, count: r.count + 1 } : r),
            });
        } else {
            onDeleted({
                ...message,
                reactionSummaries: [...message.reactionSummaries, { __typename: "ReactionSummary", emoji, count: 1 }],
            });
        }
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
                onDeleted({ ...message, reactionSummaries: originalSummaries });
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
            <ChatBubbleOuterBox showAll={bubblePressed}>
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
                                <Typography variant="body2" color="textSecondary" marginLeft={0.5}>
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
                            <IconCommon
                                decorative
                                name={message.user?.isBot ? "Bot" : "User"}
                            />
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
                </Stack>
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
                    onDelete={handleDelete}
                    onEdit={startEdit}
                    onRetry={startRetry}
                />
            </ChatBubbleOuterBox>
        </>
    );
}

const SCROLL_THRESHOLD_PX = 200;
const HIDE_SCROLL_BUTTON_CLASS = "hide-scroll-button";

const ScrollToBottomIconButton = styled(IconButton)(({ theme }) => ({
    background: theme.palette.background.paper,
    position: "sticky",
    bottom: theme.spacing(2),
    left: "50%",
    transform: "translateX(-50%)",
    width: "36px",
    height: "36px",
    opacity: 0.8,
    transition: "opacity 0.2s ease-in-out !important",
    zIndex: 1,
    boxShadow: theme.shadows[2],
}));

export function ScrollToBottomButton({
    containerRef,
}: {
    containerRef: RefObject<HTMLElement>,
}) {
    const { palette } = useTheme();
    const buttonRef = useRef<HTMLButtonElement>(null);
    const lastScrollTop = useRef(0);
    const shouldAutoScroll = useRef(false);

    const scrollToBottom = useCallback(function scrollToBottomCallback() {
        const container = containerRef.current;
        if (!container) return;

        container.scroll({
            top: container.scrollHeight,
            behavior: "smooth",
        });
        shouldAutoScroll.current = true;
    }, [containerRef]);

    useEffect(() => {
        const container = containerRef.current;
        if (!container) return;

        // Create mutation observer to watch for content changes
        const mutationObserver = new MutationObserver(() => {
            if (shouldAutoScroll.current) {
                scrollToBottom();
            }
        });

        // Watch for changes to the container's content
        mutationObserver.observe(container, {
            childList: true,
            subtree: true,
            characterData: true,
        });

        // Handle scroll events to show/hide button and track scroll position
        function handleScroll() {
            if (!container || !buttonRef.current) return;

            const { scrollTop, scrollHeight, clientHeight } = container;

            // Detect scroll direction
            const isScrollingUp = scrollTop < lastScrollTop.current;
            lastScrollTop.current = scrollTop;

            // If scrolling up at all, disable auto-scroll
            if (isScrollingUp) {
                shouldAutoScroll.current = false;
            }

            // Only re-enable auto-scroll when manually scrolled to bottom
            const isAtBottom = scrollHeight - scrollTop - clientHeight === 0;
            if (isAtBottom) {
                shouldAutoScroll.current = true;
            }

            // Show/hide the scroll button based on being close to the bottom
            if (buttonRef.current) {
                const isNearBottom = scrollHeight - scrollTop - clientHeight < SCROLL_THRESHOLD_PX;
                if (isNearBottom) {
                    buttonRef.current.classList.add(HIDE_SCROLL_BUTTON_CLASS);
                } else {
                    buttonRef.current.classList.remove(HIDE_SCROLL_BUTTON_CLASS);
                }
            }
        }

        container.addEventListener("scroll", handleScroll);

        // Initial check
        handleScroll();

        return () => {
            mutationObserver.disconnect();
            container.removeEventListener("scroll", handleScroll);
        };
    }, [containerRef, scrollToBottom]);

    return (
        <ScrollToBottomIconButton
            onClick={scrollToBottom}
            ref={buttonRef}
            size="small"
        >
            <IconCommon
                decorative
                fill={palette.background.textSecondary}
                name="ArrowDown"
            />
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
    [`& .${HIDE_SCROLL_BUTTON_CLASS}`]: {
        opacity: 0,
    },
}));

interface InnerMessageListProps extends BoxProps {
    isEditingOrReplying: boolean;
}
const InnerMessageList = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isEditingOrReplying",
})<InnerMessageListProps>(({ isEditingOrReplying }) => ({
    filter: isEditingOrReplying ? "blur(20px)" : "none",
}));

export function ChatBubbleTree({
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

    const { dimensions, ref: outerBoxRef } = useDimensions();

    const messageData = useMemo<MessageRenderData[]>(() => {
        function renderMessage(withSiblings: string[], activeIndex: number): MessageRenderData[] {
            // Find information for current message
            const siblingId = withSiblings[activeIndex];
            const sibling = siblingId ? tree.getMap().get(siblingId) : null;
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
        return renderMessage(tree.getRoots(), 0);
    }, [branches, removeMessages, setBranches, tree, userId]);

    return (
        <OuterMessageList id={id} ref={outerBoxRef}>
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
            <ScrollToBottomButton containerRef={outerBoxRef} />
        </OuterMessageList>
    );
}

