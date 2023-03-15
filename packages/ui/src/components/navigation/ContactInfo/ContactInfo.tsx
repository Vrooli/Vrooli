import { LINKS, SOCIALS } from '@shared/consts';
import {
    BottomNavigation,
    BottomNavigationAction,
    Box,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { ArticleIcon, DiscordIcon, GitHubIcon, InfoIcon, StatsIcon, SvgComponent, TwitterIcon } from '@shared/icons';
import { ContactInfoProps } from '../types';
import { ColorIconButton } from 'components/buttons';
import { CopyrightBreadcrumbs } from 'components/breadcrumbs';
import { noSelect } from 'styles';
import { openLink, useLocation } from '@shared/route';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';

type NavActionListData = [string, string, string, SvgComponent]

export const ContactInfo = ({
    session,
    sx,
    ...props
}: ContactInfoProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { additionalInfo, contactInfo } = useMemo(() => {
        return {
            additionalInfo: [
                [t(`AboutUs`), 'About', LINKS.About, InfoIcon],
                [t(`DocumentationShort`), 'Docs', 'https://docs.vrooli.com', ArticleIcon],
                [t(`StatisticsShort`), 'Stats', LINKS.Stats, StatsIcon],
            ] as NavActionListData[],
            contactInfo: [
                [t(`ContactHelpTwitter`), 'Twitter', SOCIALS.Twitter, TwitterIcon],
                [t(`ContactHelpDiscord`), 'Discord', SOCIALS.Discord, DiscordIcon],
                [t(`ContactHelpCode`), 'Code', SOCIALS.GitHub, GitHubIcon],
            ] as NavActionListData[],
        }
    }, [t]);

    const handleLink = (e: React.MouseEvent<any>, link: string) => {
        e.preventDefault();
        openLink(setLocation, link);
    }

    return (
        <Box sx={{
            minWidth: 'fit-content',
            height: 'fit-content',
            background: palette.background.default,
            padding: 1,
            ...(sx ?? {})
        }} {...props}>
            <Typography variant="h6" textAlign="center" color={palette.background.textPrimary} sx={{ ...noSelect }}>{t('FindUsOn')}</Typography>
            <BottomNavigation
                showLabels
                sx={{
                    alignItems: 'baseline',
                    background: 'transparent',
                    height: 'fit-content',
                    padding: 1,
                    marginBottom: 2,
                }}>
                {contactInfo.map(([tooltip, label, link, Icon], index: number) => (
                    <Tooltip key={`contact-info-button-${index}`} title={tooltip} placement="top">
                        <BottomNavigationAction
                            label={label}
                            onClick={(e) => { e.preventDefault(); handleLink(e, link) }}
                            href={link}
                            icon={
                                <ColorIconButton background={palette.secondary.main} >
                                    <Icon fill={palette.secondary.contrastText} />
                                </ColorIconButton>
                            }
                            sx={{
                                alignItems: 'center',
                                color: palette.background.textPrimary,
                                overflowWrap: 'anywhere',
                            }}
                        />
                    </Tooltip>
                ))}
            </BottomNavigation>
            <Typography variant="h6" textAlign="center" color={palette.background.textPrimary} sx={{ ...noSelect }}>{t(`AdditionalResources`)}</Typography>
            <BottomNavigation
                showLabels
                sx={{
                    alignItems: 'baseline',
                    background: 'transparent',
                    height: 'fit-content',
                    padding: 1,
                }}>
                {additionalInfo.map(([tooltip, label, link, Icon], index: number) => (
                    <Tooltip key={`additional-info-button-${index}`} title={tooltip} placement="top">
                        <BottomNavigationAction
                            label={label}
                            onClick={(e) => { e.preventDefault(); handleLink(e, link) }}
                            href={link}
                            icon={
                                <ColorIconButton background={palette.secondary.main} >
                                    <Icon fill={palette.secondary.contrastText} />
                                </ColorIconButton>
                            }
                            sx={{
                                alignItems: 'center',
                                color: palette.background.textPrimary,
                                overflowWrap: 'anywhere',
                            }}
                        />
                    </Tooltip>
                ))}
            </BottomNavigation>
            <CopyrightBreadcrumbs sx={{ color: palette.background.textPrimary }} />
        </Box>
    );
}