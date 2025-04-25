import { Notification, NotificationSearchInput, NotificationSearchResult, endpointsNotification } from "@local/shared";
import { useContext, useEffect } from "react";
import { create } from "zustand";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { SocketService } from "../api/socket.js";
import { SessionContext } from "../contexts/session.js";
import { checkIfLoggedIn } from "../utils/authentication/session.js";

interface NotificationsState {
    notifications: Notification[];
    isLoading: boolean;
    error: string | null;
    /** Fetch notifications from the server */
    fetchNotifications: (signal?: AbortSignal) => Promise<Notification[]>;
    /** Set notifications list */
    setNotifications: (notifications: Notification[] | ((prev: Notification[]) => Notification[])) => void;
    /** Add a new notification to the list */
    addNotification: (notification: Notification) => void;
    /** Clear all notifications and reset state */
    clearNotifications: () => void;
}

export const useNotificationsStore = create<NotificationsState>()((set, get) => ({
    notifications: [],
    isLoading: false,
    error: null,
    fetchNotifications: async (signal) => {
        // Avoid refetching if already fetched or in progress.
        if (get().notifications.length > 0 || get().isLoading) {
            return get().notifications;
        }
        set({ isLoading: true, error: null });
        try {
            const response = await fetchData<NotificationSearchInput, NotificationSearchResult>({
                ...endpointsNotification.findMany,
                inputs: {},
                signal,
            });
            if (response.errors) {
                ServerResponseParser.displayErrors(response.errors);
                set({ error: "Failed to fetch notifications", isLoading: false });
                throw new Error("Failed to fetch notifications");
            }
            const notifications = response.data?.edges.map(edge => edge.node) ?? [];

            set({ notifications, isLoading: false });
            return notifications;
        } catch (error) {
            if ((error as { name?: string }).name === "AbortError") return [];
            console.error("Error fetching notifications:", error);
            set({ error: "Error fetching notifications", isLoading: false });
            return [];
        }
    },
    setNotifications: (notifications) => {
        set((state) => ({
            notifications:
                typeof notifications === "function"
                    ? notifications(state.notifications)
                    : notifications,
        }));
    },
    addNotification: (notification) => {
        // Prepend new notifications to show the latest first.
        set((state) => ({
            notifications: [notification, ...state.notifications],
        }));
    },
    clearNotifications: () => {
        set({ notifications: [], isLoading: false, error: null });
    },
}));

/**
 * Custom hook to manage notification fetching and real-time updates.
 * This hook handles:
 * 1. Fetching initial notifications when the user logs in and the store is empty.
 * 2. Subscribing to real-time notification events via WebSocket.
 * 3. Clearing notifications from the store when the user logs out.
 *
 * Components needing the notification list should consume it directly from the store:
 * `const notifications = useNotificationsStore(state => state.notifications);`
 */
export function useNotifications(): void { // Return type changed to void
    const fetchNotifications = useNotificationsStore(state => state.fetchNotifications);
    const addNotification = useNotificationsStore(state => state.addNotification);
    const clearNotificationsStore = useNotificationsStore(state => state.clearNotifications);
    const getStoreState = useNotificationsStore.getState; // Function to get current store state

    const session = useContext(SessionContext);
    const isLoggedIn = checkIfLoggedIn(session);

    useEffect(function fetchExistingNotificationsEffect() {
        // Create AbortController specific to this effect run
        const abortController = new AbortController();

        if (!isLoggedIn) {
            // If user logs out and store has notifications, clear them.
            if (getStoreState().notifications.length > 0) {
                clearNotificationsStore();
            }
            // No fetch needed if not logged in
        } else {
            // Logged in: Check if we need to fetch based on current state
            const currentState = getStoreState();
            if (currentState.notifications.length === 0 && !currentState.isLoading && !currentState.error) {
                // Call fetchNotifications with the controller for this effect run
                fetchNotifications(abortController.signal).catch(error => {
                    // Prevent crashing on unhandled promise rejection if fetch fails
                    if ((error as { name?: string }).name !== "AbortError") {
                        console.error("Initial notification fetch failed:", error);
                    }
                });
            }
        }

        // Cleanup function: Aborts the controller associated with this specific effect run.
        // This runs on unmount or when isLoggedIn/session changes (deps of the effect).
        return () => {
            abortController.abort();
        };
        // Dependencies: Only re-run the effect logic when login status changes fundamentally.
        // Changes to isLoading, error, or notifications.length won't trigger cleanup/re-run.
    }, [isLoggedIn, fetchNotifications, clearNotificationsStore, session]);

    useEffect(function listenForNotificationsEffect() {
        if (!isLoggedIn) {
            return undefined; // Return undefined if no cleanup needed
        }

        const cleanupNotification = SocketService.get().onEvent("notification", (data) => {
            // Ensure user is still logged in when receiving the notification
            if (checkIfLoggedIn(session)) {
                addNotification(data);
                // Removed setLocalNotifications(prev => [data, ...prev]);
            }
        });

        return () => {
            cleanupNotification();
        };
    }, [isLoggedIn, addNotification, session]); // session needed for checkIfLoggedIn

    // The hook no longer returns the notification list.
    // Components should use useNotificationsStore directly.
    // return localNotifications; // Removed return
}
