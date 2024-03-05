import { Session } from "@local/shared";
import { emitSocketEvent, onSocketEvent, socket } from "api";
import { useEffect } from "react";
import { getCurrentUser } from "utils/authentication/session";

export const useSocketUser = (
    session: Session | undefined,
    setSession: (session: Session) => unknown,
) => {
    // Handle connection/disconnection
    useEffect(() => {
        const { id } = getCurrentUser(session);
        if (!id) return;

        emitSocketEvent("joinUserRoom", { userId: id }, (response) => {
            if (response.error) {
                console.error("Failed to join user room", response.error);
            }
        });

        return () => {
            emitSocketEvent("leaveUserRoom", { userId: id }, (response) => {
                if (response.error) {
                    console.error("Failed to leave user room", response.error);
                }
            });
        };
    }, [session]);

    // Handle incoming data
    useEffect(() => {
        onSocketEvent("notification", (data) => {
            console.log("got notification", data); //TODO
        });
        onSocketEvent("apiCredits", ({ credits }) => {
            const { id } = getCurrentUser(session);
            if (!id) return;
            setSession({
                ...session,
                users: session?.users?.map((user) =>
                    user.id === id ? { ...user, credits } : user,
                ),
            } as Session);
        });
        return () => {
            // Remove event handlers
            socket.off("notification");
            socket.off("apiCredits");
        };
    }, [session, setSession]);
};