import { ChatMessage, ChatMessageCreateInput, ChatMessageUpdateInput, endpointPostChatMessage, endpointPostReact, endpointPutChatMessage, ReactInput, ReactionFor, ReactionSummary, ReportFor, Success } from "@local/shared";
import { Avatar, Box, Grid, Stack, Typography, useTheme } from "@mui/material";
import CircularProgress from "@mui/material/CircularProgress";
import { green, red } from "@mui/material/colors";
import IconButton from "@mui/material/IconButton";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ReportButton } from "components/buttons/ReportButton/ReportButton";
import { EmojiPicker } from "components/EmojiPicker/EmojiPicker";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { ChatBubbleProps } from "components/types";
import { SessionContext } from "contexts/SessionContext";
import { useDisplayServerError } from "hooks/useDisplayServerError";
import { useLazyFetch } from "hooks/useLazyFetch";
import { AddIcon, BotIcon, EditIcon, ErrorIcon, UserIcon } from "icons";
import { useContext, useEffect, useMemo, useRef, useState } from "react";
import { extractImageUrl } from "utils/display/imageTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { shapeChatMessage } from "utils/shape/models/chatMessage";

/**
 * Displays a visual indicator for the status of a chat message (that you sent).
 * It shows a CircularProgress that progresses as the message is sending,
 * and changes color and icon based on the success or failure of the operation.
 */
const ChatBubbleStatus = ({
    hasError,
    isEditing,
    isSending,
    onEdit,
    onRetry,
}: {
    isEditing: boolean;
    /** Indicates if the message is still sending */
    isSending: boolean;
    /** Indicates if there has been an error in sending the message */
    hasError: boolean;
    onEdit: () => unknown;
    onRetry: () => unknown;
}) => {
    const [progress, setProgress] = useState(0);
    const [isCompleted, setIsCompleted] = useState(false);

    useEffect(() => {
        // Updates the progress value every 100ms, but stops at 90 if the message is still sending
        let timer: NodeJS.Timeout;
        if (isSending) {
            timer = setInterval(() => {
                setProgress((oldProgress) => {
                    if (oldProgress === 100) {
                        setIsCompleted(true);
                        return 100;
                    }
                    const diff = 3;
                    return Math.min(oldProgress + diff, isSending ? 90 : 100);
                });
            }, 50);
        }

        // Cleans up the interval when the component is unmounted or the message is not sending anymore
        return () => {
            if (timer) clearInterval(timer);
        };
    }, [isSending]);

    useEffect(() => {
        // Resets the progress and completion state after a delay when the sending has completed
        if (isCompleted && !isSending) {
            const timer = setTimeout(() => {
                setProgress(0);
                setIsCompleted(false);
            }, 1000);
            return () => {
                clearTimeout(timer);
            };
        }
    }, [isCompleted, isSending]);

    // While the message is sending or has just completed, show a CircularProgress
    if (isSending || isCompleted) {
        return (
            <CircularProgress
                variant="determinate"
                value={progress}
                size={24}
                sx={{ color: isSending ? "secondary.main" : hasError ? red[500] : green[500] }}
            />
        );
    }

    // If editing, don't show any icon
    if (isEditing) {
        return null;
    }
    // If there was an error, show an ErrorIcon
    if (hasError) {
        return (
            <IconButton onClick={() => { onRetry(); }} sx={{ color: red[500] }}>
                <ErrorIcon />
            </IconButton>
        );
    }
    // Otherwise, show an EditIcon
    return (
        <IconButton onClick={() => { onEdit(); }} sx={{ color: green[500] }}>
            <EditIcon />
        </IconButton>
    );
};

/**
 * Displays message reactions and actions (i.e. refresh and report). 
 * Reactions are displayed as a list of emojis on the left, and bot actions are displayed
 * as a list of icons on the right.
 */
const ChatBubbleReactions = ({
    handleReactionAdd,
    isBot,
    isOwn,
    messageId,
    reactions,
}: {
    handleReactionAdd: (emoji: string) => unknown,
    isBot: boolean,
    isOwn: boolean,
    messageId: string,
    reactions: ReactionSummary[],
}) => {
    const { palette } = useTheme();

    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const handleEmojiMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleEmojiMenuClose = () => {
        setAnchorEl(null);
    };
    const onReactionAdd = (emoji: string) => {
        setAnchorEl(null);
        handleReactionAdd(emoji);
    };

    return (
        <Box
            display="flex"
            justifyContent="space-between"
            alignItems="center"
            flexDirection={isOwn ? "row-reverse" : "row"}
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
                    borderRadius: 1,
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
                        onClick={handleEmojiMenuOpen}
                    >
                        <AddIcon />
                    </IconButton>
                )}
                <EmojiPicker
                    anchorEl={anchorEl}
                    onClose={handleEmojiMenuClose}
                    onSelect={onReactionAdd}
                />
            </Stack>
            {isBot && <Stack direction="row" spacing={1}>
                <ReportButton forId={messageId} reportFor={ReportFor.ChatMessage} />
            </Stack>}
        </Box>
    );
};

export const ChatBubble = ({
    index,
    isOwn,
    message,
    onUpdated,
}: ChatBubbleProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);
    console.log("chat bubble", message);

    const [createMessage, { loading: isCreating, errors: createErrors }] = useLazyFetch<ChatMessageCreateInput, ChatMessage>(endpointPostChatMessage);
    const [updateMessage, { loading: isUpdating, errors: updateErrors }] = useLazyFetch<ChatMessageUpdateInput, ChatMessage>(endpointPutChatMessage);
    const [react, { loading: isReacting, errors: reactErrors }] = useLazyFetch<ReactInput, Success>(endpointPostReact);
    useDisplayServerError(createErrors ?? updateErrors ?? reactErrors);

    const [hasError, setHasError] = useState(false);
    useEffect(() => {
        if ((Array.isArray(createErrors) && createErrors.length > 0) || (Array.isArray(updateErrors) && updateErrors.length > 0)) {
            setHasError(true);
        }
    }, [createErrors, updateErrors]);

    const shouldRetry = useRef(true);
    useEffect(() => {
        if (message.isUnsent && shouldRetry.current) {
            shouldRetry.current = false;
            fetchLazyWrapper<ChatMessageCreateInput, ChatMessage>({
                fetch: createMessage,
                inputs: shapeChatMessage.create({ ...message, isFork: false }),
                successCondition: (data) => data !== null,
                onSuccess: (data) => {
                    setEditingText(undefined);
                    console.log("chatbubble setting error false 1", data);
                    setHasError(false);
                    onUpdated({ ...data, isUnsent: false });
                },
            });
        }
    }, [createMessage, message, message.isUnsent, onUpdated, shouldRetry]);

    const [editingText, setEditingText] = useState<string | undefined>(undefined);
    const startEditing = () => {
        if (message.isUnsent) return;
        setEditingText(getTranslation(message, getUserLanguages(session), true)?.text ?? "");
    };
    const finishEditing = () => {
        if (message.isUnsent || editingText === undefined || editingText.trim() === "") return;
        // TODO when talking to a bot, should create new fork. When talking to a user,
        // should update the message instead. Need to figure out what to do in chat groups, 
        // with mixed bots and users.
        fetchLazyWrapper<ChatMessageCreateInput, ChatMessage>({
            fetch: createMessage,
            inputs: shapeChatMessage.create({
                ...message,
                isFork: true,
                fork: { id: message.id },
                translations: [
                    ...message.translations.filter((t) => t.language !== lng),
                    {
                        ...message.translations.find((t) => t.language === lng),
                        text: editingText,
                    },
                ] as any,
            }),
            successCondition: (data) => data !== null,
            onSuccess: (data) => {
                setEditingText(undefined);
                console.log("chatbubble setting error false 2", data);
                setHasError(false);
                onUpdated({ ...data, isUnsent: false });
            },
        });
    };

    const handleReactionAdd = (emoji: string) => {
        if (message.isUnsent) return;
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

    return (
        <Stack
            key={index}
            direction="column"
            spacing={1}
            p={2}
        >
            {/* User name display if it's not your message */}
            {!isOwn && (
                <Typography variant="body2">
                    {message.user?.name ?? message.user?.handle}
                </Typography>
            )}
            {/* Avatar, chat bubble, and status indicator */}
            <Stack direction="row" spacing={1} justifyContent={isOwn ? "flex-end" : "flex-start"}>
                {!isOwn && (
                    <Avatar
                        src={extractImageUrl(message.user?.profileImage, message.user?.updated_at, 50)}
                        alt={message.user?.name ?? message.user?.handle}
                        // onClick handlers...
                        sx={{
                            bgcolor: message.user?.isBot ? "grey" : undefined,
                            boxShadow: 2,
                            cursor: "pointer",
                            // Bots show up as squares, to distinguish them from users
                            ...(message.user?.isBot ? { borderRadius: "8px" } : {}),
                        }}
                    >
                        {message.user?.isBot ? <BotIcon width="75%" height="75%" /> : <UserIcon width="75%" height="75%" />}
                    </Avatar>
                )}
                <Box
                    sx={{
                        p: 1,
                        pl: 2,
                        pr: 2,
                        ml: isOwn ? "auto" : 0,
                        mr: isOwn ? 0 : "auto",
                        backgroundColor: isOwn ?
                            palette.mode === "light" ? "#88d17e" : "#1a5413" :
                            palette.background.paper,
                        color: palette.background.textPrimary,
                        borderRadius: "8px",
                        boxShadow: 2,
                        width: editingText !== undefined ? "100%" : "unset",
                    }}
                >
                    {editingText === undefined ? <MarkdownDisplay
                        content={getTranslation(message, getUserLanguages(session), true)?.text}
                        sx={{
                            whiteSpace: "pre-wrap",
                            wordWrap: "break-word",
                            minHeight: "unset",
                        }}
                    /> : <>
                        <RichInputBase
                            fullWidth
                            maxChars={1500}
                            minRows={editingText?.split("\n").length ?? 1}
                            name="edit-message"
                            onChange={(updatedText) => setEditingText(updatedText)}
                            value={editingText ?? ""}
                        />
                        <Grid container spacing={1} mt={2}>
                            <BottomActionsButtons
                                disabledCancel={isCreating || isUpdating}
                                disabledSubmit={isCreating || isUpdating}
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
                            isEditing={Boolean(editingText)}
                            isSending={isCreating || isUpdating}
                            hasError={hasError}
                            onEdit={() => {
                                startEditing();
                            }}
                            onRetry={() => {
                                shouldRetry.current = true;
                                onUpdated({ ...message, isUnsent: true });
                            }}
                        />
                    </Box>
                )}
            </Stack>
            {/* Reactions */}
            <ChatBubbleReactions
                handleReactionAdd={handleReactionAdd}
                isBot={message.user?.isBot ?? false}
                isOwn={isOwn}
                messageId={message.id}
                reactions={message.reactionSummaries}
            />
        </Stack>
    );
};
