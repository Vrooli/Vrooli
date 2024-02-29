import { Chat, DUMMY_ID, RegenerateResponseInput, Success, endpointPostRegenerateResponse, uuid } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { SessionContext } from "contexts/SessionContext";
import { useCallback, useContext } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { ChatShape } from "utils/shape/models/chat";
import { ChatMessageShape } from "utils/shape/models/chatMessage";
import { useLazyFetch } from "./useLazyFetch";

type UseMessageActionsProps = {
    chat: ChatShape;
    handleChatUpdate: (updatedChat?: ChatShape) => Promise<Chat>;
    language: string;
};

/**
 * Performs various chat message actions by 
 * updating the server database and local tree structure
 */
export const useMessageActions = ({
    chat,
    handleChatUpdate,
    language,
}: UseMessageActionsProps) => {
    const session = useContext(SessionContext);

    /** Commit a new message */
    const postMessage = useCallback((text: string) => {
        const newMessage: ChatMessageShape = {
            __typename: "ChatMessage" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            status: "sending",
            versionIndex: 0,
            chat: {
                __typename: "Chat" as const,
                id: chat.id,
            },
            reactionSummaries: [],
            translations: [{
                __typename: "ChatMessageTranslation" as const,
                id: DUMMY_ID,
                language,
                text,
            }],
            user: {
                __typename: "User" as const,
                id: getCurrentUser(session).id ?? "",
                isBot: false,
                name: getCurrentUser(session).name ?? undefined,
            },
        };
        PubSub.get().publish("chatMessage", {
            chatId: chat.id,
            data: { newMessage },
        });
        handleChatUpdate({
            ...chat,
            messages: [...(chat.messages ?? []), newMessage],
        }).then(() => {
            PubSub.get().publish("chatMessage", {
                chatId: chat.id,
                data: { updatedMessage: { ...newMessage, status: "sent" } },
            });
        }).catch(() => {
            PubSub.get().publish("chatMessage", {
                chatId: chat.id,
                data: { updatedMessage: { ...newMessage, status: "failed" } },
            });
        });
    }, [chat, language, handleChatUpdate, session]);

    /** Commit an existing message */
    const putMessage = useCallback((updatedMessage: ChatMessageShape) => {
        const isOwn = updatedMessage.user?.id === getCurrentUser(session).id;
        const existingMessage = chat.messages?.find((message) => message.id === updatedMessage.id);
        if (!existingMessage) return;
        PubSub.get().publish("chatMessage", {
            chatId: chat.id,
            data: { updatedMessage },
        });
        handleChatUpdate({
            ...chat,
            messages: (chat.messages ?? []).map((message) => {
                if (message.id === updatedMessage.id) {
                    return updatedMessage;
                }
                return message;
            }),
        }).catch(() => {
            PubSub.get().publish("snack", { messageKey: "ActionFailed", severity: "Error" });
            // If own message, mark as fail
            if (isOwn) {
                PubSub.get().publish("chatMessage", {
                    chatId: chat.id,
                    data: { updatedMessage: { ...updatedMessage, status: "failed" } },
                });
            }
            // Otherwise, reverse
            else {
                PubSub.get().publish("chatMessage", {
                    chatId: chat.id,
                    data: { updatedMessage: existingMessage },
                });
            }
        });
    }, [chat, handleChatUpdate, session]);

    const [regenerate] = useLazyFetch<RegenerateResponseInput, Success>(endpointPostRegenerateResponse);
    const regenerateResponse = useCallback((message: ChatMessageShape) => {
        fetchLazyWrapper<RegenerateResponseInput, Success>({
            fetch: regenerate,
            inputs: { messageId: message.id },
            successCondition: (data) => data && data.success === true,
            errorMessage: () => ({ messageKey: "ActionFailed" }),
        });
    }, [regenerate]);

    return {
        postMessage,
        putMessage,
        regenerateResponse,
    };
};
