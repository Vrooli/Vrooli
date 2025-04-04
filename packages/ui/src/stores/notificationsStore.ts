import { Notification, NotificationSearchInput, NotificationSearchResult, endpointsNotification } from "@local/shared";
import { useEffect, useState } from "react";
import { create } from "zustand";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { SocketService } from "../api/socket.js";

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
}));

export function useNotifications() {
    const [localNotifications, setLocalNotifications] = useState<Notification[]>([]);
    const fetchNotifications = useNotificationsStore(state => state.fetchNotifications);
    const addNotification = useNotificationsStore(state => state.addNotification);

    useEffect(function fetchExistingNotificationsEffect() {
        const abortController = new AbortController();
        async function loadNotifications() {
            const notifications = await fetchNotifications(abortController.signal);
            setLocalNotifications(notifications);
        }
        loadNotifications();
        return () => {
            abortController.abort();
        };
    }, [fetchNotifications]);

    useEffect(function listenForNotificationsEffect() {
        const cleanupNotification = SocketService.get().onEvent("notification", (data) => {
            addNotification(data);
            setLocalNotifications(prev => [data, ...prev]);
        });
        return () => {
            cleanupNotification();
        };
    }, [addNotification]);

    return localNotifications;
}
