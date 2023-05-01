import { useQuery } from "@apollo/client";
import { DeleteIcon, DeleteOneInput, DeleteType, FindByIdInput, Notification, NotificationSearchInput, NotificationSearchResult, openLink, Success, useLocation, VisibleIcon } from "@local/shared";
import { Box, Button, Chip, IconButton, List, ListItem, ListItemText, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { mutationWrapper, useCustomMutation } from "api";
import { deleteOneOrManyDeleteOne } from "api/generated/endpoints/deleteOneOrMany_deleteOne";
import { notificationFindMany } from "api/generated/endpoints/notification_findMany";
import { notificationMarkAllAsRead } from "api/generated/endpoints/notification_markAllAsRead";
import { notificationMarkAsRead } from "api/generated/endpoints/notification_markAsRead";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Wrap } from "types";
import { useDisplayApolloError } from "utils/hooks/useDisplayApolloError";
import { NotificationsViewProps } from "../types";

export const NotificationsView = ({
    display = "page",
}: NotificationsViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [searchString, setSearchString] = useState("");
    const { data, refetch, loading, error } = useQuery<Wrap<NotificationSearchResult, "notifications">, Wrap<NotificationSearchInput, "input">>(notificationFindMany, { variables: { input: { searchString } }, errorPolicy: "all" });
    const [notifications, setNotifications] = useState<Notification[]>([]);
    useMemo(() => {
        if (data) {
            setNotifications(n => [...n, ...data.notifications.edges.map(e => e.node)]);
        }
    }, [data]);
    const hasItems = useMemo(() => notifications.length > 0, [notifications]);

    const [deleteMutation, { loading: loadingDelete, error: deleteError }] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne);
    const [markAsReadMutation, { loading: isMarkLoading, error: markError }] = useCustomMutation<Success, FindByIdInput>(notificationMarkAsRead);
    const [markAllAsReadMutation, { loading: isMarkAllLoading, error: markAllError }] = useCustomMutation<Success, undefined>(notificationMarkAllAsRead);
    useDisplayApolloError(error ?? deleteError ?? markError ?? markAllError);

    const onSelect = useCallback((notification: Notification) => {
        if (notification.link) {
            openLink(setLocation, notification.link);
        }
    }, [setLocation]);

    const onDelete = useCallback((notification: Notification) => {
        mutationWrapper<Success, DeleteOneInput>({
            mutation: deleteMutation,
            input: { id: notification.id, objectType: DeleteType.Notification },
            successCondition: (data) => data.success,
            onSuccess: () => {
                setNotifications(n => n.filter(n => n.id !== notification.id));
            },
        });
    }, [deleteMutation]);

    const onMarkAsRead = useCallback((notification: Notification) => {
        mutationWrapper<Success, FindByIdInput>({
            mutation: markAsReadMutation,
            input: { id: notification.id },
            successCondition: (data) => data.success,
            onSuccess: () => {
                setNotifications(n => n.map(n => n.id === notification.id ? { ...n, isRead: true } : n));
            },
        });
    }, [markAsReadMutation]);

    const onMarkAllAsRead = useCallback(() => {
        mutationWrapper<Success, any>({
            mutation: markAllAsReadMutation,
            successCondition: (data) => data.success,
            onSuccess: () => {
                setNotifications(n => n.map(n => ({ ...n, isRead: true })));
            },
        });
    }, [markAllAsReadMutation]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: "Notification",
                    titleVariables: { count: 2 },
                }}
            />
            <Button
                onClick={onMarkAllAsRead}
                disabled={!hasItems}
                sx={{
                    marginLeft: "auto",
                    marginRight: "auto",
                    marginTop: 2,
                    marginBottom: 2,
                    display: "block",
                }}
            >Mark all as read</Button>
            <Box sx={{
                marginTop: 2,
                maxWidth: "1000px",
                marginLeft: "auto",
                marginRight: "auto",
                ...(hasItems ? {
                    boxShadow: 12,
                    background: palette.background.paper,
                    borderRadius: "8px",
                    overflow: "overlay",
                    display: "block",
                } : {}),
            }}>
                {
                    hasItems ? (
                        <List sx={{ padding: 0 }}>
                            {notifications.map((notification) => (
                                <ListItem
                                    key={notification.id}
                                    onClick={() => onSelect(notification)}
                                    sx={{
                                        display: "flex",
                                        background: palette.background.paper,
                                        filter: notification.isRead ? "brightness(0.8)" : "none",
                                    }}
                                >
                                    {/* Left informational column */}
                                    <Stack direction="column" spacing={1} pl={2} sx={{ marginRight: "auto" }}>
                                        <ListItemText primary={notification.title} />
                                        {notification.description && (
                                            <ListItemText
                                                primary={notification.description}
                                                sx={{ color: palette.background.textSecondary }}
                                            />
                                        )}
                                        {notification.category && (
                                            <Chip
                                                label={notification.category}
                                                variant="outlined"
                                                size="small"
                                                sx={{
                                                    backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0",
                                                    color: "white",
                                                    width: "fit-content",
                                                }}
                                            />
                                        )}
                                    </Stack>
                                    {/* Right-aligned icons */}
                                    <Stack direction="row" spacing={1}>
                                        {!notification.isRead && <Tooltip title={"Mark as read"}>
                                            <IconButton edge="end" size="small" onClick={() => onMarkAsRead(notification)}>
                                                <VisibleIcon fill={palette.secondary.main} />
                                            </IconButton>
                                        </Tooltip>}
                                        <Tooltip title="Delete">
                                            <IconButton edge="end" size="small" onClick={() => onDelete(notification)}>
                                                <DeleteIcon fill={palette.secondary.main} />
                                            </IconButton>
                                        </Tooltip>
                                    </Stack>
                                </ListItem>
                            ))}
                        </List>
                    ) : (<Typography variant="h6" textAlign="center">{t("NoResults", { ns: "error" })}</Typography>)
                }
            </Box>
        </>
    );
};
