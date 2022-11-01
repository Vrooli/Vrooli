import { APP_LINKS, LANDING_LINKS, LANDING_URL, SOCIALS } from '@shared/consts';
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
import { openLink } from 'utils';
import { useLocation } from '@shared/route';

const contactInfo: [string, string, string, SvgComponent][] = [
    ['Find us on Twitter', 'Twitter', SOCIALS.Twitter, TwitterIcon],
    ['Join our Discord', 'Discord', SOCIALS.Discord, DiscordIcon],
    ['Source code', 'Code', SOCIALS.GitHub, GitHubIcon],
]

const additionalInfo: [string, string, string, SvgComponent][] = [
    ['About Us', 'About', `${LANDING_URL}${LANDING_LINKS.AboutUs}`, InfoIcon],
    ['Documentation', 'Docs', 'https://docs.vrooli.com', ArticleIcon],
    ['Site Stats', 'Stats', APP_LINKS.Stats, StatsIcon],
]

export const ContactInfo = ({
    sx,
    ...props
}: ContactInfoProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

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
            <Typography variant="h6" textAlign="center" color={palette.background.textPrimary} sx={{ ...noSelect }}>Find us on...</Typography>
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
            <Typography variant="h6" textAlign="center" color={palette.background.textPrimary} sx={{ ...noSelect }}>Additional resources</Typography>
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