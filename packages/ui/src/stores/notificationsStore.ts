import { Notification, NotificationSearchInput, NotificationSearchResult, endpointsNotification } from "@local/shared";
import { useContext, useEffect, useState } from "react";
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

export function useNotifications() {
    const [localNotifications, setLocalNotifications] = useState<Notification[]>([]);
    const fetchNotifications = useNotificationsStore(state => state.fetchNotifications);
    const addNotification = useNotificationsStore(state => state.addNotification);
    const clearNotificationsStore = useNotificationsStore(state => state.clearNotifications);
    const storeNotifications = useNotificationsStore(state => state.notifications);

    const session = useContext(SessionContext);
    const isLoggedIn = checkIfLoggedIn(session);

    useEffect(function fetchExistingNotificationsEffect() {
        if (!isLoggedIn) {
            setLocalNotifications([]);
            if (storeNotifications.length > 0) {
                clearNotificationsStore();
            }
            return;
        }

        const abortController = new AbortController();
        async function loadNotifications() {
            if (storeNotifications.length === 0) {
                const fetchedNotifications = await fetchNotifications(abortController.signal);
                if (!abortController.signal.aborted && checkIfLoggedIn(session)) {
                    setLocalNotifications(fetchedNotifications);
                }
            } else {
                setLocalNotifications(storeNotifications);
            }
        }
        loadNotifications();
        return () => {
            abortController.abort();
        };
    }, [isLoggedIn, fetchNotifications, clearNotificationsStore, session, storeNotifications]);

    useEffect(function listenForNotificationsEffect() {
        if (!isLoggedIn) {
            return;
        }

        const cleanupNotification = SocketService.get().onEvent("notification", (data) => {
            if (checkIfLoggedIn(session)) {
                addNotification(data);
                setLocalNotifications(prev => [data, ...prev]);
            }
        });

        return () => {
            cleanupNotification();
        };
    }, [isLoggedIn, addNotification, session]);

    return localNotifications;
}
