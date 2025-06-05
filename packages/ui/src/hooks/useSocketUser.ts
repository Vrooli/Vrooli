import { type Session } from "@vrooli/shared";
import { useEffect } from "react";
import { SocketService } from "../api/socket.js";
import { getCurrentUser } from "../utils/authentication/session.js";

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
            cleanupApiCredits();
        };
    }, [session, setSession]);
}
