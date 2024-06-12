import { DUMMY_ID, endpointPostReact, getObjectUrl, ListObject, LlmTaskInfo, NavigableObject, ReactInput, ReactionFor, ReactionSummary, ReportFor, Success } from "@local/shared";
import { Avatar, Box, Chip, Grid, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import CircularProgress from "@mui/material/CircularProgress";
import { green, red } from "@mui/material/colors";
import IconButton from "@mui/material/IconButton";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ReportButton } from "components/buttons/ReportButton/ReportButton";
import { EmojiPicker } from "components/EmojiPicker/EmojiPicker";
import { RichInputBase } from "components/inputs/RichInput/RichInput";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { ChatBubbleProps } from "components/types";
import { SessionContext } from "contexts/SessionContext";
import { useDeleter } from "hooks/useDeleter";
import { useLazyFetch } from "hooks/useLazyFetch";
import { usePopover } from "hooks/usePopover";
import usePress from "hooks/usePress";
import { AddIcon, BotIcon, ChevronLeftIcon, ChevronRightIcon, CopyIcon, DeleteIcon, EditIcon, ErrorIcon, PlayIcon, RefreshIcon, ReplyIcon, SearchIcon, SuccessIcon, UserIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { isTaskStale } from "utils/display/chatTools";
import { extractImageUrl } from "utils/display/imageTools";
import { getDisplay } from "utils/display/listTools";
import { displayDate, fontSizeToPixels } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { ChatMessageStatus } from "utils/shape/models/chatMessage";

/**
 * Displays a visual indicator for the status of a chat message (that you sent).
 * It shows a CircularProgress that progresses as the message is sending,
 * and changes color and icon based on the success or failure of the operation.
 */
export const ChatBubbleStatus = ({
    isEditing,
    onDelete,
    onEdit,
    onRetry,
    showButtons,
    status,
}: {
    isEditing: boolean;
    onDelete: () => unknown;
    onEdit: () => unknown;
    onRetry: () => unknown;
    /** Indicates if the edit and delete buttons should be shown */
    showButtons: boolean;
    status: ChatMessageStatus;
}) => {
    const { palette } = useTheme();
    console.log("in chatbubblestatus", useTranslation);
    const { t } = useTranslation();
    const [progress, setProgress] = useState(0);
    const [isCompleted, setIsCompleted] = useState(false);

    useEffect(() => {
        // Updates the progress value every 100ms, but stops at 90 if the message is still sending
        let timer: NodeJS.Timeout;
        if (status === "sending") {
            timer = setInterval(() => {
                setProgress((oldProgress) => {
                    if (oldProgress === 100) {
                        setIsCompleted(true);
                        return 100;
                    }
                    const diff = 3;
                    return Math.min(oldProgress + diff, status === "sending" ? 90 : 100);
                });
            }, 50);
        }

        // Cleans up the interval when the component is unmounted or the message is not sending anymore
        return () => {
            if (timer) clearInterval(timer);
        };
    }, [status]);

    useEffect(() => {
        // Resets the progress and completion state after a delay when the sending has completed
        if (isCompleted && status !== "sending") {
            const timer = setTimeout(() => {
                setProgress(0);
                setIsCompleted(false);
            }, 1000);
            return () => {
                clearTimeout(timer);
            };
        }
    }, [isCompleted, status]);

    // While the message is sending or has just completed, show a CircularProgress
    if (status === "sending" || isCompleted) {
        return (
            <CircularProgress
                variant="determinate"
                value={progress}
                size={24}
                sx={{
                    color: status === "sending"
                        ? palette.secondary.main
                        : status === "failed"
                            ? red[500]
                            : green[500],
                }}
            />
        );
    }

    // If editing, don't show any icon
    if (isEditing) {
        return null;
    }
    // If there was an error, show an ErrorIcon
    if (status === "failed") {
        return (
            <IconButton
                aria-label={t("Retry")}
                onClick={() => { onRetry(); }}
                sx={{ color: red[500] }}
            >
                <ErrorIcon />
            </IconButton>
        );
    }
    // If allowed to show buttons, show edit and delete buttons
    if (showButtons) return (
        <>
            <IconButton
                aria-label={t("Edit")}
                onClick={onEdit}
                sx={{ color: green[500] }}
            >
                <EditIcon />
            </IconButton>
            <IconButton
                aria-label={t("Delete")}
                onClick={onDelete}
                sx={{ color: red[500] }}
            >
                <DeleteIcon />
            </IconButton>
        </>
    );
    // Otherwise, show nothing
    return null;
};

export const NavigationArrows = ({
    activeIndex,
    numSiblings,
    onIndexChange,
}: {
    activeIndex: number,
    numSiblings: number,
    onIndexChange: (newIndex: number) => unknown,
}) => {
    const { palette } = useTheme();

    if (numSiblings < 2) {
        return null; // Do not render anything if there are less than 2 siblings
    }

    const handleActiveIndexChange = (newIndex: number) => {
        if (newIndex >= 0 && newIndex < numSiblings) {
            onIndexChange(newIndex);
        }
    };

    return (
        <div style={{ display: "flex", alignItems: "center" }}>
            <IconButton
                size="small"
                onClick={() => handleActiveIndexChange(activeIndex - 1)}
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
                onClick={() => handleActiveIndexChange(activeIndex + 1)}
                disabled={activeIndex >= numSiblings - 1}
                aria-label="right"
            >
                <ChevronRightIcon fill={palette.background.textSecondary} />
            </IconButton>
        </div>
    );
};

type ChatBubbleReactionsProps = {
    activeIndex: number,
    handleActiveIndexChange: (newIndex: number) => unknown,
    handleCopy,
    handleReactionAdd: (emoji: string) => unknown,
    handleReply: () => unknown,
    handleRetry: () => unknown,
    isBot: boolean,
    isLastMessage: boolean,
    isOwn: boolean,
    numSiblings: number,
    messageId: string,
    reactions: ReactionSummary[],
    status: ChatMessageStatus,
}

/**
 * Displays message reactions and actions (i.e. refresh and report). 
 * Reactions are displayed as a list of emojis on the left, and bot actions are displayed
 * as a list of icons on the right.
 */
const ChatBubbleReactions = ({
    activeIndex,
    handleActiveIndexChange,
    handleCopy,
    handleReactionAdd,
    handleReply,
    handleRetry,
    isBot,
    isLastMessage,
    isOwn,
    numSiblings,
    messageId,
    reactions,
    status,
}: ChatBubbleReactionsProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [anchorEl, openReactionMenu, closeReactionMenu] = usePopover();
    const onReactionAdd = (emoji: string) => {
        closeReactionMenu();
        handleReactionAdd(emoji);
    };

    if (status === "unsent") return null;
    return (
        <Box
            display="flex"
            alignItems="center"
            flexDirection="row"
            justifyContent={isOwn ? "flex-end" : "flex-start"}
        >
            <Stack
                direction="row"
                spacing={1}
                ml={isOwn ? 0 : 6}
                mr={isOwn ? 6 : 0}
                pr={isOwn ? 1 : 0}
                sx={{
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                    borderRadius: "0 0 8px 8px",
                    boxShadow: `1px 2px 3px rgba(0,0,0,0.2),
                    1px 2px 2px rgba(0,0,0,0.14)`,
                    overflow: "overlay",
                }}
            >
                {reactions.map((reaction) => (
                    <Box key={reaction.emoji} display="flex" alignItems="center">
                        <IconButton
                            size="small"
                            disabled={isOwn}
                            onClick={() => { onReactionAdd(reaction.emoji); }}
                            style={{ borderRadius: 0, background: "transparent" }}
                        >
                            {reaction.emoji}
                        </IconButton>
                        <Typography variant="body2">
                            {reaction.count}
                        </Typography>
                    </Box>
                ))}
                {!isOwn && (
                    <IconButton
                        size="small"
                        style={{ borderRadius: 0, background: "transparent" }}
                        onClick={openReactionMenu}
                    >
                        <AddIcon />
                    </IconButton>
                )}
                <EmojiPicker
                    anchorEl={anchorEl}
                    onClose={closeReactionMenu}
                    onSelect={onReactionAdd}
                />
            </Stack>
            <Stack direction="row">
                <Tooltip title={t("Copy")}>
                    <IconButton size="small" onClick={handleCopy}>
                        <CopyIcon fill={palette.background.textSecondary} />
                    </IconButton>
                </Tooltip>
                {(isBot || (isOwn && isLastMessage)) && <Tooltip title={t("Retry")}>
                    <IconButton size="small" onClick={handleRetry}>
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
        </Box >
    );
};



/** Displays a suggested, active, or finished task that is associated with the message */
export const TaskChip = ({
    taskInfo,
    onTaskClick,
}: {
    taskInfo: LlmTaskInfo,
    onTaskClick: (task: LlmTaskInfo) => unknown,
}) => {
    const { label, status, resultLabel, resultLink, task } = taskInfo;
    const [, setLocation] = useLocation();

    const isStale = isTaskStale(taskInfo);
    const canPress =
        isStale // Has been running or canceling for too long
        || !["Completed", "Canceling"].includes(status)
        || (status === "Completed" && resultLink); // Result can be opened

    const getStatusColor = () => {
        if (isStale) return "warning";
        switch (status) {
            case "Running":
            case "Canceling":
                return "primary";
            case "Completed":
                return "success";
            case "Failed":
                return "error";
            default:
                return "default";
        }
    };

    const getStatusIcon = () => {
        if (isStale || !task) return <ErrorIcon />;
        switch (status) {
            case "Running":
            case "Canceling":
                return <CircularProgress size={20} color="inherit" />;
            case "Completed":
                return <SuccessIcon />;
            case "Failed":
                return <ErrorIcon />;
            default:
                // Base Icon style on task type
                if (task.endsWith("Add"))
                    return <AddIcon />;
                if (task.endsWith("Delete"))
                    return <DeleteIcon />;
                if (task.endsWith("Find"))
                    return <SearchIcon />;
                if (task.endsWith("Update"))
                    return <EditIcon />;
                return <PlayIcon />;
        }
    };

    const getStatusTooltip = () => {
        if (isStale) return `Task is stale: ${label}`;
        switch (status) {
            case "Suggested":
                return `Press to start task: ${label}`;
            case "Running":
                return `Task is running: ${label} (Started: ${displayDate(taskInfo.lastUpdated)})`;
            case "Canceling":
                return `Task is canceling: ${label}`;
            case "Completed":
                return `Task completed: ${resultLabel || label}`;
            case "Failed":
                return `Task failed: ${label}`;
            default:
                return `Task: ${label}`;
        }
    };

    const handleTaskClick = () => {
        // If the result link is available, open it
        if (resultLink) {
            openLink(setLocation, resultLink);
        } else {
            onTaskClick(taskInfo);
        }
    };

    return (
        <Tooltip title={getStatusTooltip()}>
            {/* Wrap in span so tooltip displayed when disabled */}
            <span>
                <Chip
                    label={resultLabel || label}
                    onClick={() => canPress ? handleTaskClick() : undefined}
                    color={getStatusColor()}
                    icon={getStatusIcon()}
                    style={{
                        transition: "all 0.3s ease",
                        cursor: canPress ? "pointer" : "default",
                        borderRadius: "4px",
                        border: status === "Suggested" ? "1px solid" : "none",
                        paddingLeft: "4px",
                    }}
                    disabled={!canPress}
                />
            </span>
        </Tooltip>
    );
};

export const ChatBubble = ({
    activeIndex,
    chatWidth,
    isBotOnlyChat,
    isLastMessage,
    isOwn,
    message,
    numSiblings,
    onActiveIndexChange,
    onDeleted,
    onReply,
    onRetry,
    onTaskClick,
    onUpdated,
    tasks,
}: ChatBubbleProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { breakpoints, palette } = useTheme();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);
    const isMobile = useMemo(() => chatWidth <= breakpoints.values.sm, [breakpoints, chatWidth]);

    const [react] = useLazyFetch<ReactInput, Success>(endpointPostReact);

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

    const [editingText, setEditingText] = useState<string | undefined>(undefined);
    const isEditing = Boolean(editingText);
    const startEditing = useCallback(() => {
        if (message.status && message.status !== "sent") return;
        setEditingText(getTranslation(message, getUserLanguages(session), true)?.text ?? "");
    }, [message, session]);
    const finishEditing = () => {
        if (message.status !== "sent" || editingText === undefined || editingText.trim() === "") return;
        if (isBotOnlyChat) {
            //TODO
        }
        // Otherwise, update the existing message
        else {
            const updatedMessage = {
                ...message,
                translations: [
                    ...message.translations.filter((t) => t.language !== lng),
                    {
                        __typename: "ChatMessageTranslation" as const,
                        id: DUMMY_ID,
                        language: lng,
                        ...message.translations.find((t) => t.language === lng),
                        text: editingText,
                    },
                ],
            };
            onUpdated(updatedMessage);
            // TODO need these after update complete
            // setEditingText(undefined);
            // setHasError(false);
        }
    };

    const handleReactionAdd = (emoji: string) => {
        if (message.status !== "sent") return;
        const originalSummaries = message.reactionSummaries;
        // Add to summaries right away, so that the UI updates immediately
        const existingReaction = message.reactionSummaries.find((r) => r.emoji === emoji);
        if (existingReaction) {
            onUpdated({
                ...message,
                reactionSummaries: message.reactionSummaries.map((r) => r.emoji === emoji ? { ...r, count: r.count + 1 } : r),
            } as ChatBubbleProps["message"]);
        } else {
            onUpdated({
                ...message,
                reactionSummaries: [...message.reactionSummaries, { __typename: "ReactionSummary", emoji, count: 1 }],
            } as ChatBubbleProps["message"]);
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
                onUpdated({ ...message, reactionSummaries: originalSummaries } as ChatBubbleProps["message"]);
            },
        });
    };

    const handleCopy = () => {
        navigator.clipboard.writeText(getTranslation(message, getUserLanguages(session), true)?.text ?? "");
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    };

    const { name, handle, adornments } = useMemo(() => {
        const { title, adornments } = getDisplay(message.user as ListObject);
        return {
            name: title,
            handle: message.user?.handle,
            adornments,
        };
    }, [message.user]);

    const [bubblePressed, setBubblePressed] = useState(false);
    const toggleBubblePressed = () => {
        if (!isMobile && bubblePressed) return;
        setBubblePressed(!bubblePressed);
    };
    const pressEvents = usePress({
        onLongPress: toggleBubblePressed,
        onClick: toggleBubblePressed,
    });
    useEffect(() => {
        const handleResize = () => {
            if (!isMobile && !bubblePressed) setBubblePressed(true);
            if (isMobile && bubblePressed) setBubblePressed(false);
        };
        window.addEventListener("resize", handleResize);
        return () => {
            window.removeEventListener("resize", handleResize);
        };
    }, [bubblePressed, isMobile]);

    return (
        <>
            {DeleteDialogComponent}
            <Box
                key={message.id}
                sx={{
                    display: "flex",
                    flexDirection: "column",
                    padding: "8px",
                    maxWidth: "100vw",
                }}
            >
                {/* User name display if it's not your message */}
                {!isOwn && (
                    <Stack direction="row" alignItems="center" spacing={0.5} mb={0.5}>
                        <Typography variant="body2">
                            {name}
                        </Typography>
                        {adornments.map(({ Adornment, key }, index) => (
                            <Box key={key} sx={{
                                width: fontSizeToPixels("0.85rem") * Number("1.5"),
                                height: fontSizeToPixels("0.85rem") * Number("1.5"),
                            }}>
                                {Adornment}
                            </Box>
                        ))}
                        {handle && (
                            <Typography variant="body2" color="textSecondary">
                                @{handle}
                            </Typography>
                        )}
                    </Stack>
                )}
                {/* Avatar, chat bubble, and status indicator */}
                <Stack direction="row" justifyContent={isOwn ? "flex-end" : "flex-start"}>
                    {!isOwn && (
                        <Avatar
                            src={extractImageUrl(message.user?.profileImage, message.user?.updated_at, 50)}
                            alt={message.user?.name ?? message.user?.handle ?? message?.user?.isBot ? "Bot" : "User"}
                            onClick={() => { setLocation(getObjectUrl(message.user as NavigableObject)); }}
                            sx={{
                                bgcolor: message.user?.isBot ? "grey" : undefined,
                                boxShadow: 2,
                                cursor: "pointer",
                                marginRight: 1,
                                // Bots show up as squares, to distinguish them from users
                                ...(message.user?.isBot ? { borderRadius: "8px" } : {}),
                            }}
                        >
                            {message.user?.isBot ? <BotIcon width="75%" height="75%" /> : <UserIcon width="75%" height="75%" />}
                        </Avatar>
                    )}
                    <Box
                        {...pressEvents}
                        sx={{
                            p: 1,
                            pl: isEditing ? 0 : 2,
                            pr: isEditing ? 0 : 2,
                            ml: isOwn ? "auto" : 0,
                            mr: isOwn ? 0 : "auto",
                            background: !isOwn && message.status === "failed"
                                ? palette.error.dark
                                : (isOwn && !isEditing)
                                    ? palette.mode === "light" ? "#88d17e" : "#1a5413"
                                    : palette.background.paper,
                            color: !isOwn && message.status === "failed"
                                ? palette.error.contrastText
                                : palette.background.textPrimary,
                            borderRadius: isOwn ? "8px 8px 0 8px" : "8px 8px 8px 0",
                            boxShadow: 2,
                            minWidth: "50px",
                            width: editingText !== undefined ? "100%" : "unset",
                            minHeight: "20x",
                            transition: "width 0.3s ease-in-out",
                        }}
                    >
                        {editingText === undefined ? <MarkdownDisplay
                            content={getTranslation(message, getUserLanguages(session), true)?.text}
                            sx={{
                                whiteSpace: "pre-wrap",
                                wordWrap: "break-word",
                                overflowWrap: "anywhere",
                                minHeight: "unset",
                            }}
                        /> : <>
                            <RichInputBase
                                fullWidth
                                maxChars={1500}
                                minRows={editingText?.split("\n").length ?? 1}
                                maxRows={10}
                                name="edit-message"
                                onChange={(updatedText) => setEditingText(updatedText)}
                                value={editingText ?? ""}
                            />
                            <Grid container spacing={1} mt={2}>
                                <BottomActionsButtons
                                    disabledCancel={message.status !== "sent"}
                                    disabledSubmit={!["unsent", "editing"].includes(message.status ?? "sent")}
                                    display="page"
                                    errors={{}}
                                    isCreate={false}
                                    onCancel={() => {
                                        setEditingText(undefined);
                                    }}
                                    onSubmit={() => {
                                        finishEditing();
                                    }}
                                />
                            </Grid>
                        </>
                        }
                    </Box>
                    {/* Status indicator and edit/retry buttons */}
                    {isOwn && (
                        <Box display="flex" alignItems="center">
                            <ChatBubbleStatus
                                isEditing={isEditing}
                                onDelete={handleDelete}
                                onEdit={() => {
                                    startEditing();
                                }}
                                onRetry={() => {
                                    //TODO
                                }}
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
                    handleReply={() => { onReply(message); }}
                    handleRetry={() => { onRetry(message); }}
                    isBot={message.user?.isBot ?? false}
                    isLastMessage={isLastMessage}
                    isOwn={isOwn}
                    numSiblings={numSiblings}
                    messageId={message.id}
                    reactions={message.reactionSummaries}
                    status={message.status ?? "sent"}
                />
                {/* Tasks associated with message */}
                {tasks && tasks.length > 0 && (
                    <Box sx={{ display: "flex", flexWrap: "wrap", gap: "10px", marginTop: "8px" }}>
                        {tasks.map((taskInfo) => (
                            <TaskChip key={taskInfo.id} taskInfo={taskInfo} onTaskClick={onTaskClick} />
                        ))}
                    </Box>
                )}
            </Box>
        </>
    );
};
