import { CardGrid, PageTitle, SettingsTopBar, TIDCard } from 'components';
import { SettingsData, SettingsViewProps } from '../types';
import { HistoryIcon, LightModeIcon, LockIcon, NotificationsCustomizedIcon, ProfileIcon, VisibleIcon } from '@shared/icons';
import { useTranslation } from 'react-i18next';
import { useLocation } from '@shared/route';
import { useCallback } from 'react';
import { APP_LINKS } from '@shared/consts';


export const accountSettingsData: SettingsData[] = [
    {
        title: 'Profile',
        description: 'ProfileSettingsDescription',
        link: 'SettingsProfile',
        Icon: ProfileIcon,
    },
    {
        title: 'Privacy',
        description: 'PrivacySettingsDescription',
        link: 'SettingsPrivacy',
        Icon: VisibleIcon,
    },
    {
        title: 'Authentication',
        description: 'AuthenticationSettingsDescription',
        link: 'SettingsAuthentication',
        Icon: LockIcon,
    },
];

export const displaySettingsData: SettingsData[] = [
    {
        title: 'Display',
        description: 'DisplaySettingsDescription',
        link: 'SettingsDisplay',
        Icon: LightModeIcon,
    },
    {
        title: 'Notification',
        description: 'NotificationSettingsDescription',
        link: 'SettingsNotifications',
        Icon: NotificationsCustomizedIcon,
    },
    {
        title: 'Schedule',
        description: 'ScheduleDescription',
        link: 'SettingsSchedules',
        Icon: HistoryIcon,
    },
];

export const SettingsView = ({
    display = 'page',
    session,
}: SettingsViewProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const onSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        setLocation(APP_LINKS[newValue]);
    }, [setLocation]);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'Settings',
                }}
            />
            <PageTitle title={t(`Account`)} />
            <CardGrid minWidth={300}>
                {accountSettingsData.map(({ title, description, link, Icon }, index) => (
                    <TIDCard
                        buttonText={t('Open')}
                        description={t(description)}
                        Icon={Icon}
                        key={index}
                        onClick={() => onSelect(link)}
                        title={t(title, { count: 2 })}
                    />
                ))}
            </CardGrid>
            <PageTitle title={t(`Display`)} sxs={{ text: { paddingTop: 2 } }} />
            <CardGrid minWidth={300}>
                {displaySettingsData.map(({ title, description, link, Icon }, index) => (
                    <TIDCard
                        buttonText={t('Open')}
                        description={t(description)}
                        Icon={Icon}
                        key={index}
                        onClick={() => onSelect(link)}
                        title={t(title, { count: 2 })}
                    />
                ))}
            </CardGrid>
        </>
    )
}
