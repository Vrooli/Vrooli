import { useCallback } from 'react';
import { PageContainer, PageTitle } from 'components';
import { ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, StandardIcon, SvgComponent } from '@shared/icons';
import { Box, Button, Typography, useTheme } from '@mui/material';
import { NotificationsPageProps } from '../types';
import { useTranslation } from 'react-i18next';
import { getUserLanguages } from 'utils';
import { CommonKey } from 'types';
import { APP_LINKS } from '@shared/consts';
import { useLocation } from '@shared/route';
import { useQuery } from '@apollo/client';

export const NotificationsPage = ({
    session
}: NotificationsPageProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const { data, refetch, loading } = useQuery<notifications, notificationsVariables>(notificationsQuery, { variables: { input: { searchString } }, errorPolicy: 'all' });

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

            </List>
        </PageContainer>
    )
};
