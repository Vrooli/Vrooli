import { useQuery } from '@apollo/client';
import { Box, List, ListItem, Typography, useTheme } from '@mui/material';
import { Notification, NotificationSearchInput, NotificationSearchResult } from '@shared/consts';
import { useLocation } from '@shared/route';
import { notificationFindMany } from 'api/generated/endpoints/notification_findMany';
import { TopBar } from 'components/navigation/TopBar/TopBar';
import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Wrap } from 'types';
import { useDisplayApolloError } from 'utils/hooks/useDisplayApolloError';
import { NotificationsViewProps } from '../types';

export const NotificationsView = ({
    display = 'page',
}: NotificationsViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [searchString, setSearchString] = useState('');
    const { data, refetch, loading, error } = useQuery<Wrap<NotificationSearchResult, 'notifications'>, Wrap<NotificationSearchInput, 'input'>>(notificationFindMany, { variables: { input: { searchString } }, errorPolicy: 'all' });
    const [notifications, setNotifications] = useState<Notification[]>([]);
    useDisplayApolloError(error);
    useMemo(() => {
        if (data) {
            setNotifications(n => [...n, ...data.notifications.edges.map(e => e.node)]);
        }
    }, [data]);
    const hasItems = useMemo(() => notifications.length > 0, [notifications]);

    const onSelect = useCallback((notification: Notification) => {
        // setLocation(`${LINKS[objectType]}/add`);
    }, [setLocation]);

    const onDelete = useCallback((notification: Notification) => {
    }, []);

    const onMarkAsRead = useCallback((notification: Notification) => {
    }, []);

    const onMarkAllAsRead = useCallback(() => {
    }, []);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Notification',
                    titleVariables: { count: 2 },
                }}
            />
            <Box sx={{
                marginTop: 2,
                maxWidth: '1000px',
                marginLeft: 'auto',
                marginRight: 'auto',
                ...(hasItems ? {
                    boxShadow: 12,
                    background: palette.background.paper,
                    borderRadius: '8px',
                    overflow: 'overlay',
                    display: 'block',
                } : {}),
            }}>
                {
                    hasItems ? (
                        <List sx={{ padding: 0 }}>
                            {notifications.map((notification) => (
                                <ListItem key={notification.id}>
                                    {notification.title}
                                </ListItem>
                            ))}
                        </List>
                    ) : (<Typography variant="h6" textAlign="center">{t(`NoResults`, { ns: 'error' })}</Typography>)
                }
            </Box>
        </>
    )
};
