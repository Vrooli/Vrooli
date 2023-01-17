import { useCallback, useMemo, useState } from 'react';
import { PageContainer, PageTitle } from 'components';
import { ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, StandardIcon, SvgComponent } from '@shared/icons';
import { Box, Button, List, ListItem, Typography, useTheme } from '@mui/material';
import { NotificationsPageProps } from '../types';
import { useTranslation } from 'react-i18next';
import { getUserLanguages } from 'utils';
import { CommonKey, Wrap } from 'types';
import { APP_LINKS, NotificationSearchInput, NotificationSearchResult } from '@shared/consts';
import { useLocation } from '@shared/route';
import { useQuery } from '@apollo/client';
import { notificationEndpoint } from 'graphql/endpoints';

export const NotificationsPage = ({
    session
}: NotificationsPageProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [searchString, setSearchString] = useState('');
    const { data, refetch, loading } = useQuery<Wrap<NotificationSearchResult, 'notifications'>, Wrap<NotificationSearchInput, 'input'>>(notificationEndpoint.findMany[0], { variables: { input: { searchString } }, errorPolicy: 'all' });
    const [notifications, setNotifications] = useState<Notification[]>([]);
    useMemo(() => {
        if (data) {
            setNotifications(n => [...n, ...data.notifications.edges.map(e => e.node)]);
        }
    }, [data]);

    const onSelect = useCallback((notification: Notification) => {
        setLocation(`${APP_LINKS[objectType]}/add`);
    }, [setLocation]);

    const onDelete = useCallback((notification: Notification) => {
    }, []);

    const onMarkAsRead = useCallback((notification: Notification) => {
    }, []);

    const onMarkAllAsRead = useCallback(() => {
    }, []);

    return (
        <PageContainer>
            <List>
                {notifications.map((notification) => (
                    <ListItem key={notification.id}>
                        {notification.text}
                    </ListItem>
                ))}
            </List>
        </PageContainer>
    )
};
