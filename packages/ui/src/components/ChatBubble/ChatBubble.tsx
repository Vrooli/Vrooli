import { ChatMessage, ChatMessageCreateInput, ChatMessageUpdateInput, endpointPostChatMessage, endpointPutChatMessage, ReactionSummary } from "@local/shared";
import { Avatar, Badge, Box, Grid, Menu, Stack, Typography, useTheme } from "@mui/material";
import CircularProgress from "@mui/material/CircularProgress";
import { green, red } from "@mui/material/colors";
import IconButton from "@mui/material/IconButton";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import EmojiPicker from "components/EmojiPicker";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { ChatBubbleProps } from "components/types";
import { SessionContext } from "contexts/SessionContext";
import { useLazyFetch } from "hooks/useLazyFetch";
import { AddIcon, BotIcon, EditIcon, ErrorIcon, ReportIcon, UserIcon } from "icons";
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

// const defaultReactions = ["👍", "👎", "❤️", "😂", "😮", "😢", "😡"];
// const EmojiPicker = ({ anchorEl, open, onClose, onSelect }: {
//     anchorEl: null | HTMLElement,
//     open: boolean,
//     onClose: () => void,
//     onSelect: (emoji: string) => void,
// }) => {
//     return (
//         <Menu
//             anchorEl={anchorEl}
//             open={open}
//             onClose={onClose}
//             PaperProps={{
//                 style: {
//                     maxWidth: "min(100vw, 400px)",
//                     overflowX: "hidden",
//                 },
//             }}
//         >
//             <Box display="flex" flexWrap="wrap" gap={1} p={1}>
//                 {defaultReactions.map((emoji, idx) => (
//                     <Box key={idx} onClick={() => {
//                         onSelect(emoji);
//                         onClose();
//                     }}>
//                         {emoji}
//                     </Box>
//                 ))}
//             </Box>
//         </Menu>
//     );
// };

/**
 * Displays message reactions and actions (i.e. refresh and report). 
 * Reactions are displayed as a list of emojis on the left, and bot actions are displayed
 * as a list of icons on the right.
 */
const ChatBubbleReactions = ({
    isBot,
    isOwn,
    reactions,
}: {
    isBot: boolean,
    isOwn: boolean,
    reactions: ReactionSummary[],
}) => {
    const [showEmojiPicker, setShowEmojiPicker] = useState(false);

    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const handleEmojiMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleEmojiMenuClose = () => {
        setAnchorEl(null);
    };
    const handleAddReaction = (emoji: string) => {
        // Add the reaction logic here
        console.log("Added reaction:", emoji);
        setShowEmojiPicker(false);
    };

    return (
        <Box display="flex" justifyContent="space-between" alignItems="center">
            <Stack direction="row" spacing={1} pl={6}>
                {reactions.map((reaction) => (
                    <Badge key={reaction.emoji} badgeContent={reaction.count} color="primary">
                        <IconButton size="small" disabled={isOwn} style={{ borderRadius: "50%", backgroundColor: "#f0f0f0" }}>
                            {reaction.emoji}
                        </IconButton>
                    </Badge>
                ))}
                {!isOwn && (
                    <IconButton
                        size="small"
                        style={{ borderRadius: "50%", backgroundColor: "#f0f0f0" }}
                        onClick={handleEmojiMenuOpen}
                    >
                        <AddIcon />
                    </IconButton>
                )}
                <Menu
                    anchorEl={anchorEl}
                    open={Boolean(anchorEl)}
                    onClose={handleEmojiMenuClose}
                    PaperProps={{
                        style: {
                            maxWidth: "min(100vw, 400px)",
                            overflowX: "hidden",
                        },
                    }}
                >
                    <EmojiPicker />
                </Menu>
            </Stack>
            {isBot && <Stack direction="row" spacing={1}>
                <IconButton size="small">
                    <ReportIcon />
                </IconButton>
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
            <ChatBubbleReactions isBot={message.user?.isBot ?? false} isOwn={isOwn} reactions={message.reactionSummaries} />
        </Stack>
    );
};
