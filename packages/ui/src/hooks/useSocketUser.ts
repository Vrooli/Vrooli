import { Session } from "@local/shared";
import { SocketService } from "api/socket.js";
import { useEffect } from "react";
import { getCurrentUser } from "utils/authentication/session.js";

export function useSocketUser(
    session: Session | undefined,
    setSession: (session: Session) => unknown,
) {
    // Handle connection/disconnection
    useEffect(() => {
        const { id } = getCurrentUser(session);
        if (!id) return;

        SocketService.get().emitEvent("joinUserRoom", { userId: id }, (response) => {
            if (response.error) {
                console.error("Failed to join user room", response.error);
            }
        });

        return () => {
            SocketService.get().emitEvent("leaveUserRoom", { userId: id }, (response) => {
                if (response.error) {
                    console.error("Failed to leave user room", response.error);
                }
            });
        };
    }, [session]);

    // Handle incoming data
    useEffect(() => {
        const cleanupNotification = SocketService.get().onEvent("notification", (data) => {
            console.log("got notification", data); //TODO
        });
        const cleanupApiCredits = SocketService.get().onEvent("apiCredits", ({ credits }) => {
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
            cleanupNotification();
            cleanupApiCredits();
        };
    }, [session, setSession]);
}
