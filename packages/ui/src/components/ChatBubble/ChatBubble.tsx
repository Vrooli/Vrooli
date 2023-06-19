import { BotIcon, ChatMessage, ChatMessageCreateInput, ChatMessageUpdateInput, endpointPostChatMessage, endpointPutChatMessage, UserIcon } from "@local/shared";
import { Avatar, Box, Grid, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { ChatBubbleStatus } from "components/ChatBubbleStatus/ChatBubbleStatus";
import { MarkdownInputBase } from "components/inputs/MarkdownInputBase/MarkdownInputBase";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { ChatBubbleProps } from "components/types";
import { useContext, useEffect, useMemo, useRef, useState } from "react";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { SessionContext } from "utils/SessionContext";
import { shapeChatMessage } from "utils/shape/models/chatMessage";

export const ChatBubble = ({
    index,
    isOwn,
    message,
    onUpdated,
    zIndex,
}: ChatBubbleProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    const [createMessage, { loading: isCreating, errors: createErrors }] = useLazyFetch<ChatMessageCreateInput, ChatMessage>(endpointPostChatMessage);
    const [updateMessage, { loading: isUpdating, errors: updateErrors }] = useLazyFetch<ChatMessageUpdateInput, ChatMessage>(endpointPutChatMessage);

    const [hasError, setHasError] = useState(false);
    useEffect(() => {
        if ((Array.isArray(createErrors) && createErrors.length > 0) || (Array.isArray(updateErrors) && updateErrors.length > 0)) {
            console.log("chatbubble setting error true");
            setHasError(true);
        }
    }, [createErrors, updateErrors]);
    console.log("chatbubble render", hasError, message);

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
        <Box
            key={index}
            sx={{ display: "flex", justifyContent: isOwn ? "flex-end" : "flex-start", p: 2 }}
        >
            {/* Avatar only display if it's not your message */}
            {!isOwn && (
                <Avatar
                    // src={message.user.avatar} TODO
                    alt={message.user?.name ?? message.user?.handle}
                    // onClick handlers...
                    sx={{
                        bgcolor: message.user.isBot ? "grey" : undefined,
                        boxShadow: 2,
                        cursor: "pointer",
                    }}
                >
                    {message.user.isBot ? <BotIcon width="75%" height="75%" /> : <UserIcon width="75%" height="75%" />}
                </Avatar>
            )}

            {/* Message bubble with reaction */}
            <Box sx={{
                ml: !isOwn ? 2 : 2,
                mr: isOwn ? 2 : 2,
                display: "block",
                maxWidth: "100%",
                position: "relative",
                overflowWrap: "break-word",
                wordWrap: "break-word",
            }}>
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
                    }}
                >
                    {editingText === undefined ? <MarkdownDisplay
                        content={getTranslation(message, getUserLanguages(session), true)?.text}
                        sx={{
                            whiteSpace: "pre-wrap",
                            wordWrap: "break-word",
                        }}
                    /> : <>
                        <MarkdownInputBase
                            maxChars={1500}
                            minRows={editingText?.split("\n").length ?? 1}
                            name="edit-message"
                            onChange={(updatedText) => setEditingText(updatedText)}
                            value={editingText ?? ""}
                            zIndex={zIndex}
                        />
                        <Grid container spacing={1} mt={2}>
                            <GridSubmitButtons
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
        </Box>
    );
};
