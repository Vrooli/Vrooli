import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { useQuery } from "@apollo/client";
import { Box, List, ListItem, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { notificationFindMany } from "../../../api/generated/endpoints/notification_findMany";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { useDisplayApolloError } from "../../../utils/hooks/useDisplayApolloError";
import { useLocation } from "../../../utils/route";
export const NotificationsView = ({ display = "page", }) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [searchString, setSearchString] = useState("");
    const { data, refetch, loading, error } = useQuery(notificationFindMany, { variables: { input: { searchString } }, errorPolicy: "all" });
    const [notifications, setNotifications] = useState([]);
    useDisplayApolloError(error);
    useMemo(() => {
        if (data) {
            setNotifications(n => [...n, ...data.notifications.edges.map(e => e.node)]);
        }
    }, [data]);
    const hasItems = useMemo(() => notifications.length > 0, [notifications]);
    const onSelect = useCallback((notification) => {
    }, [setLocation]);
    const onDelete = useCallback((notification) => {
    }, []);
    const onMarkAsRead = useCallback((notification) => {
    }, []);
    const onMarkAllAsRead = useCallback(() => {
    }, []);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Notification",
                    titleVariables: { count: 2 },
                } }), _jsx(Box, { sx: {
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
                }, children: hasItems ? (_jsx(List, { sx: { padding: 0 }, children: notifications.map((notification) => (_jsx(ListItem, { children: notification.title }, notification.id))) })) : (_jsx(Typography, { variant: "h6", textAlign: "center", children: t("NoResults", { ns: "error" }) })) })] }));
};
//# sourceMappingURL=NotificationsView.js.map