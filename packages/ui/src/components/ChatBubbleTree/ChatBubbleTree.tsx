import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import type { BoxProps, CircularProgressProps } from "@mui/material";
import { styled, useTheme } from "@mui/material";
import { ReactionFor, ReportFor, endpointsChatMessage, endpointsReaction, getObjectUrl, getTranslation, noop, type ChatMessageShape, type ChatMessageStatus, type ChatSocketEventPayloads, type ListObject, type NavigableObject, type ReactInput, type ReactionSummary, type StreamErrorPayload, type Success } from "@vrooli/shared";
import { type ChatMessageRunConfig } from "@vrooli/shared";
import React, { useCallback, useContext, useEffect, useMemo, useRef, useState, type RefObject, lazy, Suspense, forwardRef } from "react";
import { VariableSizeList as List, type ListChildComponentProps } from "react-window";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { SessionContext } from "../../contexts/session.js";
import { usePress } from "../../hooks/gestures.js";
import { type MessageTree } from "../../hooks/messages.js";
import { useDeleter } from "../../hooks/objectActions.js";
import { useDimensions } from "../../hooks/useDimensions.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { IconCommon } from "../../icons/Icons.js";
import { Link, useLocation } from "../../route/router.js";
import { ProfileAvatar } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { extractImageUrl } from "../../utils/display/imageTools.js";
import { getDisplay, placeholderColor } from "../../utils/display/listTools.js";
import { fontSizeToPixels } from "../../utils/display/stringTools.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { type BranchMap } from "../../utils/localStorage.js";
import { PubSub } from "../../utils/pubsub.js";
import { EmojiPicker } from "../EmojiPicker/EmojiPicker.js";
import { ReportButton } from "../buttons/ReportButton.js";

import { MarkdownDisplay } from "../text/MarkdownDisplay.js";

import { type ChatBubbleProps } from "../types.js";

// Lazy load heavy components
const RunExecutorContainer = lazy(() => import("../execution/RunExecutorContainer.js").then(module => ({ default: module.RunExecutorContainer })));

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
    messageCreatedAt: string,
    reactions: ReactionSummary[],
    status: ChatMessageStatus,
    onDelete: () => unknown;
    onEdit: () => unknown;
    onRetry: () => unknown;
    messageStreamError?: StreamErrorPayload;
}

const NavigationBox = styled(Box)(() => ({
    display: "flex",
    alignItems: "center",
}));

// Memoized NavigationArrows component
const NavigationArrows = React.memo(function NavigationArrows({
    activeIndex,
    numSiblings,
    onIndexChange,
}: {
    activeIndex: number,
    numSiblings: number,
    onIndexChange: (newIndex: number) => unknown,
}) {
    const handleActiveIndexChange = useCallback((newIndex: number) => {
        if (newIndex >= 0 && newIndex < numSiblings) {
            onIndexChange(newIndex);
        }
    }, [numSiblings, onIndexChange]);

    const handlePrevious = useCallback(() => {
        handleActiveIndexChange(activeIndex - 1);
    }, [activeIndex, handleActiveIndexChange]);

    const handleNext = useCallback(() => {
        handleActiveIndexChange(activeIndex + 1);
    }, [activeIndex, handleActiveIndexChange]);

    // eslint-disable-next-line no-magic-numbers
    if (numSiblings < 2) {
        return null; // Do not render anything if there are less than 2 siblings
    }

    const previousDisabled = activeIndex <= 0;
    const nextDisabled = activeIndex >= numSiblings - 1;

    return (
        <NavigationBox>
            <Tooltip title="Previous">
                <span>
                    <IconButton
                        size="small"
                        onClick={handlePrevious}
                        disabled={previousDisabled}
                    >
                        <IconCommon
                            decorative
                            fill="background.textSecondary"
                            name="ChevronLeft"
                        />
                    </IconButton>
                </span>
            </Tooltip>
            <Typography variant="body2" color="textSecondary">
                {activeIndex + 1}/{numSiblings}
            </Typography>
            <Tooltip title="Next">
                <span>
                    <IconButton
                        size="small"
                        onClick={handleNext}
                        disabled={nextDisabled}
                    >
                    <IconCommon
                        decorative
                        fill="background.textSecondary"
                        name="ChevronRight"
                    />
                </IconButton>
                </span>
            </Tooltip>
        </NavigationBox>
    );
}, (prevProps, nextProps) => {
    return prevProps.activeIndex === nextProps.activeIndex &&
           prevProps.numSiblings === nextProps.numSiblings &&
           prevProps.onIndexChange === nextProps.onIndexChange;
});

export { NavigationArrows };

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
    width: "100%",
    marginRight: isOwn ? "auto" : 0,
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
    flexWrap: "wrap",
    padding: 0,
    // Reactions should align with the start of the message content
    marginLeft: 0,
    marginRight: 0,
    paddingRight: 0,
    // Match the chat bubble color
    background: isOwn
        ? theme.palette.mode === "light" ? "#a5b5bf" : "#21341f"
        : theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
    // No rounded corners on top (attached to bubble), rounded on bottom
    borderRadius: "0 0 8px 8px",
    overflow: "visible",
}));

const reactionIconButtonStyle = { borderRadius: 0, background: "transparent" } as const;

/**
 * Displays message reactions and actions (i.e. refresh and report). 
 * Reactions are displayed as a list of emojis on the left, and bot actions are displayed
 * as a list of icons on the right.
 */
const ChatBubbleReactions = React.memo(function ChatBubbleReactions({
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
    messageCreatedAt,
    reactions,
    status,
    onDelete,
    onEdit,
    onRetry,
    messageStreamError,
}: ChatBubbleReactionsProps) {
    const { t } = useTranslation();
    const [progress, setProgress] = useState(0);
    const [isCompleted, setIsCompleted] = useState(false);
    const [showAllReactions, setShowAllReactions] = useState(false);
    

    const formattedTime = useMemo(() => {
        if (!messageCreatedAt) return "";
        try {
            const date = new Date(messageCreatedAt);
            // Check if the date is valid before formatting
            if (isNaN(date.getTime())) {
                console.error("Invalid date received for message:", messageId, messageCreatedAt);
                return "";
            }
            return date.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
        } catch (error) {
            console.error("Error formatting date:", error);
            return "";
        }
    }, [messageCreatedAt, messageId]);

    useEffect(function chatStatusLoaderEffect() {
        let timer: NodeJS.Timeout | undefined;
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
            if (timer) {
                clearInterval(timer);
                timer = undefined;
            }
        };
    }, [status]);

    useEffect(function chatStatusTimeoutEffect() {
        let timer: NodeJS.Timeout | undefined;
        if (isCompleted && status !== "sending") {
            timer = setTimeout(() => {
                setProgress(0);
                setIsCompleted(false);
            }, HIDE_SEND_SUCCESS_DELAY_MS);
        }
        return () => {
            if (timer) {
                clearTimeout(timer);
                timer = undefined;
            }
        };
    }, [isCompleted, status]);

    const onReactionAdd = useCallback((emoji: string) => {
        handleReactionAdd(emoji);
    }, [handleReactionAdd]);

    const handleShowAllReactions = useCallback(() => {
        setShowAllReactions(!showAllReactions);
    }, [showAllReactions]);

    if (status === "unsent") return null;

    // Configuration for reaction display
    const MAX_REACTIONS_COLLAPSED = 5;
    const displayedReactions = useMemo(() => {
        if (!Array.isArray(reactions)) return [];
        if (showAllReactions || reactions.length <= MAX_REACTIONS_COLLAPSED) {
            return reactions;
        }
        return reactions.slice(0, MAX_REACTIONS_COLLAPSED);
    }, [reactions, showAllReactions]);

    const hiddenReactionsCount = reactions.length - displayedReactions.length;

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

    // Show retry button for bot responses with retryable errors
    const retryForBotResponse = isBot && status === "failed" && messageStreamError?.retryable ? (
        <Tooltip title={t("Regenerate response")}>
            <IconButton
                color="secondary"
                onClick={handleRegenerateResponse}
                size="small"
            >
                <IconCommon
                    decorative
                    name="Refresh"
                />
            </IconButton>
        </Tooltip>
    ) : null;

    return (
        <ReactionsOuterBox isOwn={isOwn}>
            <ReactionsMainStack isMobile={isMobile} isOwn={isOwn}>
                {displayedReactions.map((reaction) => (
                    <Box key={reaction.emoji} display="flex" alignItems="center">
                        <IconButton
                            size="small"
                            disabled={isOwn}
                            onClick={() => onReactionAdd(reaction.emoji)}
                            style={reactionIconButtonStyle}
                        >
                            {reaction.emoji}
                        </IconButton>
                        <Typography variant="body2">
                            {reaction.count}
                        </Typography>
                    </Box>
                ))}
                {hiddenReactionsCount > 0 && (
                    <Tooltip title={showAllReactions ? t("Show fewer reactions") : t("Show all {{count}} reactions", { count: reactions.length })}>
                        <IconButton
                            size="small"
                            onClick={handleShowAllReactions}
                            style={reactionIconButtonStyle}
                        >
                            <IconCommon
                                decorative
                                name={showAllReactions ? "Invisible" : "Ellipsis"}
                            />
                        </IconButton>
                    </Tooltip>
                )}
                {!isOwn && <EmojiPicker onSelect={onReactionAdd} />}
            </ReactionsMainStack>
            <Stack direction="row" alignItems="center">
                {statusIndicator}
                {retryForBotResponse}
                <NavigationArrows
                    activeIndex={activeIndex}
                    numSiblings={numSiblings}
                    onIndexChange={handleActiveIndexChange}
                />
                <Box className="chat-bubble-actions">
                    {/* Conditionally render timestamp and buttons based on isOwn */}
                    {isOwn ? (
                        <>
                            {/* Timestamp first for own messages */}
                            {formattedTime && (
                                <Typography variant="caption" color="textSecondary" sx={{ mx: 1, whiteSpace: "nowrap" }}>
                                    {formattedTime}
                                </Typography>
                            )}
                            {/* Action Buttons for own messages */}
                            <Tooltip title={t("Copy")}>
                                <IconButton size="small" onClick={handleCopy}>
                                    <IconCommon decorative fill="background.textSecondary" name="Copy" />
                                </IconButton>
                            </Tooltip>
                            {status === "sent" && (
                                <>
                                    <Tooltip title={t("Edit")}>
                                        <IconButton onClick={onEdit} size="small">
                                            <IconCommon decorative fill="background.textSecondary" name="Edit" />
                                        </IconButton>
                                    </Tooltip>
                                    <Tooltip title={t("Delete")}>
                                        <IconButton onClick={onDelete} size="small">
                                            <IconCommon decorative fill="background.textSecondary" name="Delete" />
                                        </IconButton>
                                    </Tooltip>
                                </>
                            )}
                        </>
                    ) : (
                        <>
                            {/* Action Buttons first for other messages */}
                            <Tooltip title={t("Copy")}>
                                <IconButton size="small" onClick={handleCopy}>
                                    <IconCommon decorative fill="background.textSecondary" name="Copy" />
                                </IconButton>
                            </Tooltip>
                            {(isBot && isBotOnlyChat) && (
                                <Tooltip title={t("Retry")}>
                                    <IconButton size="small" onClick={handleRegenerateResponse}>
                                        <IconCommon decorative fill="background.textSecondary" name="Refresh" />
                                    </IconButton>
                                </Tooltip>
                            )}
                            {isBot && (
                                <Tooltip title={t("Reply")}>
                                    <IconButton size="small" onClick={handleReply}>
                                        <IconCommon decorative fill="background.textSecondary" name="Reply" />
                                    </IconButton>
                                </Tooltip>
                            )}
                            {isBot && <ReportButton forId={messageId} reportFor={ReportFor.ChatMessage} />}
                            {/* Timestamp last for other messages */}
                            {formattedTime && (
                                <Typography variant="caption" color="textSecondary" sx={{ ml: 1, whiteSpace: "nowrap" }}>
                                    {formattedTime}
                                </Typography>
                            )}
                        </>
                    )}
                </Box>
            </Stack>
        </ReactionsOuterBox>
    );
}, (prevProps, nextProps) => {
    // Custom comparison function for React.memo
    return prevProps.activeIndex === nextProps.activeIndex &&
           prevProps.numSiblings === nextProps.numSiblings &&
           prevProps.messageId === nextProps.messageId &&
           prevProps.messageCreatedAt === nextProps.messageCreatedAt &&
           prevProps.status === nextProps.status &&
           prevProps.isBot === nextProps.isBot &&
           prevProps.isBotOnlyChat === nextProps.isBotOnlyChat &&
           prevProps.isMobile === nextProps.isMobile &&
           prevProps.isOwn === nextProps.isOwn &&
           JSON.stringify(prevProps.reactions) === JSON.stringify(nextProps.reactions) &&
           prevProps.messageStreamError === nextProps.messageStreamError;
});

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
            ? theme.palette.mode === "light" ? "#a5b5bf" : "#21341f"
            : theme.palette.background.paper,
    color: !isOwn && messageStatus === "failed"
        ? theme.palette.error.contrastText
        : theme.palette.background.textPrimary,
    borderRadius: "8px 8px 8px 0", // Always rounded on top and bottom-right, straight on bottom-left for reactions
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

export const ChatBubble = React.memo(function ChatBubble({
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
    messageStreamError,
}: ChatBubbleProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { breakpoints } = useTheme();
    const isMobile = useMemo(() => chatWidth <= breakpoints.values.sm, [breakpoints, chatWidth]);
    const profileColors = useMemo(() => placeholderColor(message.user?.id), [message.user?.id]);

    const [react] = useLazyFetch<ReactInput, Success>(endpointsReaction.createOne);
    const [regenerateResponse] = useLazyFetch(endpointsChatMessage.regenerateResponse);

    const {
        handleDelete,
        DeleteDialogComponent,
    } = useDeleter({
        object: message as ListObject,
        objectType: "ChatMessage",
        onActionComplete: () => { onDeleted(message); },
    });

    const handleReactionAdd = useCallback((emoji: string) => {
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
    }, [message, onDeleted, react]);

    const startEdit = useCallback(() => {
        onEdit(message);
    }, [onEdit, message]);

    const startRegenerateResponse = useCallback(() => {
        // If this is a retryable error, use the regenerateResponse endpoint
        if (messageStreamError?.retryable && message.user?.isBot) {
            fetchLazyWrapper({
                fetch: regenerateResponse,
                inputs: { messageId: message.id },
                onError: (error) => {
                    console.error("Failed to regenerate response:", error);
                    PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error" });
                },
            });
        } else {
            // Otherwise, use the standard regenerate flow
            onRegenerateResponse(message);
        }
    }, [messageStreamError, message, regenerateResponse, onRegenerateResponse]);

    const startReply = useCallback(() => {
        onReply(message);
    }, [onReply, message]);

    const startRetry = useCallback(() => {
        onRetry(message);
    }, [onRetry, message]);

    const handleCopy = useCallback(() => {
        navigator.clipboard.writeText(getTranslation(message, getUserLanguages(session), true)?.text ?? "");
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }, [message, session]);

    const { name, handle, adornments } = useMemo(() => {
        const { title, adornments } = getDisplay(message.user as ListObject);
        return {
            name: title,
            handle: message.user?.handle,
            adornments,
        };
    }, [message.user]);

    const [bubblePressed, setBubblePressed] = useState(false);
    const toggleBubblePressed = useCallback(() => {
        setBubblePressed(!bubblePressed);
    }, [bubblePressed]);
    
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
                            src={extractImageUrl(message.user?.profileImage, message.user?.updatedAt, TARGET_PROFILE_IMAGE_SIZE_PX)}
                            alt={message.user?.name ?? message.user?.handle ?? message?.user?.isBot ? "Bot" : "User"}
                            isBot={message.user?.isBot === true}
                            onClick={openProfile}
                            profileColors={profileColors}
                            sx={profileAvatarStyle}
                        >
                            <IconCommon
                                decorative
                                name={message.user?.isBot ? "Bot" : "Profile"}
                            />
                        </ProfileAvatar>
                    )}
                    <Box sx={{ 
                        display: "flex", 
                        flexDirection: "column", 
                        alignItems: isOwn ? "flex-end" : "flex-start",
                        width: "fit-content",
                        maxWidth: "100%",
                    }}>
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
                    messageCreatedAt={message.createdAt}
                    reactions={message.reactionSummaries}
                    status={message.status ?? "sent"}
                    onDelete={handleDelete}
                    onEdit={startEdit}
                    onRetry={startRetry}
                    messageStreamError={messageStreamError}
                        />
                        {/* Render routine executors if runs exist in message config */}
                        {message.config?.runs && message.config.runs.length > 0 && (
                            <Suspense fallback={<CircularProgress size={24} />}>
                                <RunExecutorContainer
                                    runs={message.config.runs as ChatMessageRunConfig[]}
                                    chatWidth={chatWidth}
                                    messageId={message.id}
                                />
                            </Suspense>
                        )}
                    </Box>
                </Stack>
            </ChatBubbleOuterBox>
        </>
    );
});

const SCROLL_THRESHOLD_PX = 200;
const HIDE_SCROLL_BUTTON_CLASS = "hide-scroll-button";

const ScrollToBottomIconButton = styled(IconButton)(({ theme }) => ({
    background: theme.palette.background.paper,
    position: "absolute",
    bottom: theme.spacing(2),
    left: "50%",
    transform: "translateX(-50%)",
    width: "36px",
    height: "36px",
    opacity: 0.9,
    transition: "opacity 0.2s ease-in-out",
    zIndex: 10,
    boxShadow: theme.shadows[3],
    "&:hover": {
        opacity: 1,
        background: theme.palette.background.paper,
    },
}));

const ScrollToBottomButton = React.memo(function ScrollToBottomButton({
    containerRef,
}: {
    containerRef: RefObject<HTMLElement>,
}) {
    const { palette } = useTheme();
    const buttonRef = useRef<HTMLButtonElement>(null);
    const lastScrollTop = useRef(0);
    const shouldAutoScroll = useRef(false);
    const [isNearBottom, setIsNearBottom] = useState(true);

    const scrollToBottom = useCallback(function scrollToBottomCallback() {
        const container = containerRef.current;
        if (!container) return;

        container.scroll({
            top: container.scrollHeight,
            behavior: "smooth",
        });
        shouldAutoScroll.current = true;
    }, [containerRef]);

    // Throttled scroll handler for better performance
    const handleScroll = useCallback(() => {
        const container = containerRef.current;
        if (!container) return;

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

        // Update button visibility state
        const nearBottom = scrollHeight - scrollTop - clientHeight < SCROLL_THRESHOLD_PX;
        setIsNearBottom(nearBottom);
    }, [containerRef]);

    // Throttled mutation callback
    const handleMutation = useCallback(() => {
        if (shouldAutoScroll.current) {
            scrollToBottom();
        }
    }, [scrollToBottom]);

    useEffect(() => {
        const container = containerRef.current;
        if (!container) return;

        let scrollTimer: NodeJS.Timeout;
        let mutationTimer: NodeJS.Timeout;

        const SCROLL_THROTTLE_MS = 16; // ~60fps
        const MUTATION_THROTTLE_MS = 100;

        // Throttled scroll handler
        const throttledScrollHandler = () => {
            if (scrollTimer) clearTimeout(scrollTimer);
            scrollTimer = setTimeout(handleScroll, SCROLL_THROTTLE_MS);
        };

        // Throttled mutation handler
        const throttledMutationHandler = () => {
            if (mutationTimer) clearTimeout(mutationTimer);
            mutationTimer = setTimeout(handleMutation, MUTATION_THROTTLE_MS);
        };

        // Create mutation observer to watch for content changes
        const mutationObserver = new MutationObserver(throttledMutationHandler);

        // Watch for changes to the container's content
        mutationObserver.observe(container, {
            childList: true,
            subtree: true,
            characterData: true,
        });

        container.addEventListener("scroll", throttledScrollHandler, { passive: true });

        // Initial check
        handleScroll();

        return () => {
            if (scrollTimer) clearTimeout(scrollTimer);
            if (mutationTimer) clearTimeout(mutationTimer);
            mutationObserver.disconnect();
            container.removeEventListener("scroll", throttledScrollHandler);
        };
    }, [containerRef, handleScroll, handleMutation]);

    return (
        <ScrollToBottomIconButton
            onClick={scrollToBottom}
            ref={buttonRef}
            size="small"
            className={isNearBottom ? HIDE_SCROLL_BUTTON_CLASS : ""}
        >
            <IconCommon
                decorative
                fill={palette.background.textSecondary}
                name="ArrowDown"
            />
        </ScrollToBottomIconButton>
    );
});

export { ScrollToBottomButton };

type MessageRenderData = {
    activeIndex: number;
    key: string;
    numSiblings: number;
    updateBranchesAndLocation: (parentId: string, newActiveChildId: string) => unknown;
    onDeleted: (message: ChatMessageShape) => unknown;
    message: ChatMessageShape;
    isOwn: boolean;
    messageStreamError?: StreamErrorPayload;
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
    updateBranchesAndLocation: (parentId: string, newActiveChildId: string) => unknown;
    tree: MessageTree<ChatMessageShape>,
};

const OuterMessageList = styled(Box)(() => ({
    position: "relative",
    display: "flex",
    flexDirection: "column",
    margin: "0 auto",
    maxWidth: "800px",
    overflowX: "hidden",
    overflowY: "hidden", // Changed to hidden since List handles scrolling
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

// Row component for virtualized list
interface VirtualizedMessageRowProps {
    index: number;
    style: React.CSSProperties;
    data: {
        messageData: MessageRenderData[];
        streamingMessage: ChatMessageShape | null;
        messageStream: ChatSocketEventPayloads["responseStream"] | null;
        dimensions: { width: number; height: number };
        isBotOnlyChat: boolean;
        handleEdit: (message: ChatMessageShape) => unknown;
        handleRegenerateResponse: (message: ChatMessageShape) => unknown;
        handleReply: (message: ChatMessageShape) => unknown;
        handleRetry: (message: ChatMessageShape) => unknown;
        treeMap: Map<string, any>;
        treeRoots: string[];
        updateBranchesAndLocation: (parentId: string, newActiveChildId: string) => unknown;
        setItemSize: (index: number, size: number) => void;
    };
}

const VirtualizedMessageRow = React.memo(function VirtualizedMessageRow({ index, style, data }: VirtualizedMessageRowProps) {
    const {
        messageData,
        streamingMessage,
        messageStream,
        dimensions,
        isBotOnlyChat,
        handleEdit,
        handleRegenerateResponse,
        handleReply,
        handleRetry,
        treeMap,
        treeRoots,
        updateBranchesAndLocation,
        setItemSize,
    } = data;

    const rowRef = useRef<HTMLDivElement>(null);

    // Measure and update row height
    useEffect(() => {
        if (rowRef.current) {
            const height = rowRef.current.getBoundingClientRect().height;
            if (height > 0 && setItemSize) {
                setItemSize(index, height);
            }
        }
    }, [index, setItemSize]);

    // Handle streaming message as last item
    if (index === messageData.length && streamingMessage && messageStream) {
        return (
            <div ref={rowRef} style={{...style, height: 'auto'}}>
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
                    message={streamingMessage}
                    isOwn={false}
                    messageStreamError={messageStream.error}
                />
            </div>
        );
    }

    const messageItem = messageData[index];
    if (!messageItem) return null;

    return (
        <div ref={rowRef} style={{...style, height: 'auto'}}>
            <ChatBubble
                key={messageItem.key}
                activeIndex={messageItem.activeIndex}
                chatWidth={dimensions.width}
                isBotOnlyChat={isBotOnlyChat}
                numSiblings={messageItem.numSiblings}
                onActiveIndexChange={(newIndex: number) => {
                    const messageNode = treeMap.get(messageItem.message.id);
                    const parentId = messageNode?.message.parent?.id;
                    const siblings = parentId ? treeMap.get(parentId)?.children ?? [] : treeRoots;
                    const newSiblingId = siblings[newIndex];
                    if (parentId && newSiblingId) {
                        updateBranchesAndLocation(parentId, newSiblingId);
                    }
                }}
                onDeleted={messageItem.onDeleted}
                onEdit={handleEdit}
                onRegenerateResponse={handleRegenerateResponse}
                onReply={handleReply}
                onRetry={handleRetry}
                message={messageItem.message}
                isOwn={messageItem.isOwn}
                messageStreamError={messageItem.messageStreamError}
            />
        </div>
    );
});

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
    updateBranchesAndLocation,
    tree,
}: ChatBubbleTreeProps) {
    const session = useContext(SessionContext);
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const { palette } = useTheme();

    const { dimensions, ref: outerBoxRef } = useDimensions();
    const listRef = useRef<List>(null);
    
    // Track accumulated message for streaming
    const [accumulatedMessage, setAccumulatedMessage] = useState("");
    
    // Update accumulated message when stream changes
    useEffect(() => {
        if (!messageStream) {
            setAccumulatedMessage("");
            return;
        }
        
        if (messageStream.__type === "stream" && messageStream.chunk) {
            setAccumulatedMessage(prev => prev + (prev ? " " : "") + messageStream.chunk);
        } else if (messageStream.__type === "end" && messageStream.finalMessage) {
            setAccumulatedMessage(messageStream.finalMessage);
        } else if (messageStream.__type === "error") {
            // Keep the accumulated message on error
        }
    }, [messageStream]);

    // Memoize tree map to avoid repeated computations
    const treeMap = useMemo(() => tree.getMap(), [tree]);
    const treeRoots = useMemo(() => tree.getRoots(), [tree]);
    
    const messageData = useMemo<MessageRenderData[]>(() => {
        const result: MessageRenderData[] = [];
        
        function renderMessage(withSiblings: string[], activeIndex: number): void {
            // Find information for current message
            const siblingId = withSiblings[activeIndex];
            const sibling = siblingId ? treeMap.get(siblingId) : null;
            if (!sibling) return;
            
            const isOwn = sibling.message.user?.id === userId;

            // Add current message to result
            result.push({
                activeIndex,
                key: sibling.message.id,
                numSiblings: withSiblings.length,
                updateBranchesAndLocation,
                onDeleted: (message) => { removeMessages([message.id]); },
                message: sibling.message,
                isOwn,
                messageStreamError: undefined, // Regular messages don't have stream errors
            });

            // Find information for next message
            // Check the stored branch data first
            let childId = branches[siblingId];
            // Fallback to first child if no branch data is found
            if (!childId) childId = sibling.children[0];
            
            if (childId && sibling.children.length > 0) {
                const activeChildIndex = Math.max(sibling.children.findIndex(id => id === childId), 0);
                renderMessage(sibling.children, activeChildIndex);
            }
        }
        
        renderMessage(treeRoots, 0);
        return result;
    }, [branches, treeMap, treeRoots, updateBranchesAndLocation, removeMessages, userId]);

    // Create streaming message object outside of render
    const streamingMessage = useMemo(() => {
        if (!messageStream) return null;
        
        return {
            id: "streamingMessage",
            reactionSummaries: [],
            status: messageStream.__type === "error" ? "failed" : "sending",
            translations: [{
                language: "en",
                text: accumulatedMessage || "",
            }],
            user: messageStream.botId ? {
                id: messageStream.botId,
                isBot: true,
            } : undefined,
            versionIndex: 0,
        } as unknown as ChatMessageShape;
    }, [messageStream, accumulatedMessage]);

    // Calculate item count including streaming message
    const itemCount = messageData.length + (streamingMessage ? 1 : 0);

    // Height cache for variable size list
    const itemHeights = useRef<{ [index: number]: number }>({});
    const estimatedItemSize = 250; // Fixed estimate - react-window will measure and correct
    
    const getItemSize = useCallback((index: number) => {
        // Return measured height if available, otherwise use estimate
        return itemHeights.current[index] || estimatedItemSize;
    }, []);

    const setItemSize = useCallback((index: number, size: number) => {
        itemHeights.current[index] = size;
        if (listRef.current) {
            listRef.current.resetAfterIndex(index);
        }
    }, []);

    // Track scroll position for scroll-to-bottom button
    const [isNearBottom, setIsNearBottom] = useState(true);
    
    // Auto-scroll to bottom when new messages arrive (only if already near bottom)
    useEffect(() => {
        if (listRef.current && itemCount > 0 && isNearBottom) {
            listRef.current.scrollToItem(itemCount - 1, "end");
        }
    }, [itemCount, isNearBottom]);
    
    // Handle scroll events from the virtualized list
    const handleScroll = useCallback(({ scrollOffset, scrollUpdateWasRequested }: { scrollOffset: number; scrollUpdateWasRequested: boolean }) => {
        if (!listRef.current || scrollUpdateWasRequested) return;
        
        // For variable size lists, we can check if we're near the last item
        // by seeing which items are currently visible
        const list = listRef.current;
        const lastItemIndex = itemCount - 1;
        
        // Check if the last item is visible or nearly visible
        // @ts-ignore - accessing private method but it's the most reliable way
        const isLastItemVisible = list._instanceProps?.visibleStopIndex >= lastItemIndex - 2;
        
        setIsNearBottom(isLastItemVisible);
    }, [itemCount]);
    
    // Scroll to bottom handler
    const scrollToBottom = useCallback(() => {
        if (listRef.current && itemCount > 0) {
            listRef.current.scrollToItem(itemCount - 1, "end");
            setIsNearBottom(true);
        }
    }, [itemCount]);

    // Data for virtualized rows
    const rowData = useMemo(() => ({
        messageData,
        streamingMessage,
        messageStream,
        dimensions,
        isBotOnlyChat,
        handleEdit,
        handleRegenerateResponse,
        handleReply,
        handleRetry,
        treeMap,
        treeRoots,
        updateBranchesAndLocation,
        setItemSize,
    }), [messageData, streamingMessage, messageStream, dimensions, isBotOnlyChat, 
         handleEdit, handleRegenerateResponse, handleReply, handleRetry, 
         treeMap, treeRoots, updateBranchesAndLocation, setItemSize]);

    return (
        <OuterMessageList id={id} ref={outerBoxRef}>
            <InnerMessageList isEditingOrReplying={isEditingMessage || isReplyingToMessage}>
                {dimensions.height > 0 && (
                    <List
                        ref={listRef}
                        height={dimensions.height}
                        width={dimensions.width}
                        itemCount={itemCount}
                        itemSize={getItemSize}
                        itemData={rowData}
                        overscanCount={5}
                        onScroll={handleScroll}
                    >
                        {VirtualizedMessageRow}
                    </List>
                )}
            </InnerMessageList>
            {!isNearBottom && (
                <ScrollToBottomIconButton
                    onClick={scrollToBottom}
                    size="small"
                >
                    <IconCommon
                        decorative
                        fill={palette.background.textSecondary}
                        name="ArrowDown"
                    />
                </ScrollToBottomIconButton>
            )}
        </OuterMessageList>
    );
}

