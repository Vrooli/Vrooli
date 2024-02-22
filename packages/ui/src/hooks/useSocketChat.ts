import { ChatParticipant, DUMMY_ID, Session } from "@local/shared";
import { emitSocketEvent, onSocketEvent, socket } from "api";
import { useEffect } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { ChatShape } from "utils/shape/models/chat";
import { ChatMessageShape } from "utils/shape/models/chatMessage";

type UseWebSocketEventsProps = {
    addMessages: (newMessages: ChatMessageShape[]) => unknown;
    chat: ChatShape;
    editMessage: (editedMessage: ChatMessageShape) => unknown;
    removeMessages: (deletedIds: string[]) => unknown;
    session: Session | undefined;
    setUsersTyping: (updatedParticipants: ChatParticipant[]) => unknown;
    usersTyping: ChatParticipant[];
}

export const useSocketChat = ({
    addMessages,
    chat,
    editMessage,
    removeMessages,
    session,
    setUsersTyping,
    usersTyping,
}: UseWebSocketEventsProps) => {
    // Handle connection/disconnection
    useEffect(() => {
        if (!chat?.id || chat.id === DUMMY_ID) return;

        const onDisconnect = () => {
            PubSub.get().publish("snack", { messageKey: "ServerDisconnected", severity: "Error", id: "ServerDisconnected" });
        };
        const onReconnectAttempt = () => {
            PubSub.get().publish("snack", { messageKey: "ServerReconnectAttempt", severity: "Warning", id: "ServerReconnectAttempt" });
        };
        const onReconnect = () => {
            PubSub.get().publish("snack", { messageKey: "ServerReconnected", severity: "Success", id: "ServerReconnected" });
        };

        socket.on("disconnect", onDisconnect);
        socket.on("reconnect_attempt", onReconnectAttempt);
        socket.on("reconnect", onReconnect);

        emitSocketEvent("joinChatRoom", { chatId: chat.id }, (response) => {
            if (response.error) {
                PubSub.get().publish("snack", { messageKey: "ChatRoomJoinFailed", severity: "Error" });
            }
        });


        return () => {
            emitSocketEvent("leaveChatRoom", { chatId: chat.id }, (response) => {
                if (response.error) {
                    console.error("Failed to leave chat room", response.error);
                }
            });

            // Clean up event listeners
            socket.off("disconnect", onDisconnect);
            socket.off("reconnect_attempt", onReconnectAttempt);
            socket.off("reconnect", onReconnect);
        };
    }, [chat?.id]);

    // Handle incoming data
    useEffect(() => {
        /** Handle added, edited, and deleted messages */
        onSocketEvent("messages", ({ added, deleted, edited }) => {
            if (added) {
                addMessages(added);
            }
            if (deleted) {
                removeMessages(deleted);
            }
            if (edited) {
                for (const message of edited) {
                    editMessage(message);
                }
            }
        });
        /** Handle participant typing indicator */
        onSocketEvent("typing", ({ starting, stopping }) => {
            // Add every user that's typing
            const newTyping = [...usersTyping];
            for (const id of starting ?? []) {
                // Never add yourself
                if (id === getCurrentUser(session).id) continue;
                if (newTyping.some(p => p.user.id === id)) continue;
                const participant = chat.participants?.find(p => p.user.id === id);
                if (!participant) continue;
                newTyping.push(participant);
            }
            // Remove every user that stopped typing
            for (const id of stopping ?? []) {
                const index = newTyping.findIndex(p => p.user.id === id);
                if (index === -1) continue;
                newTyping.splice(index, 1);
            }
            setUsersTyping(newTyping);
        });
        // TODO add participants joining/leaving, making sure to update matching chat cache in cookies
        return () => {
            // Remove event handlers
            socket.off("messages");
            socket.off("typing");
        };
    }, [addMessages, editMessage, chat.participants, removeMessages, session, usersTyping, setUsersTyping]);
};
