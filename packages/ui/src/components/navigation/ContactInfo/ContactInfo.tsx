import { SOCIALS } from '@shared/consts';
import {
    BottomNavigation,
    BottomNavigationAction,
    Box,
    Tooltip,
    useTheme,
} from '@mui/material';
import { DiscordIcon, GitHubIcon, SvgComponent, TwitterIcon } from '@shared/icons';
import { ContactInfoProps } from '../types';
import { ColorIconButton } from 'components/buttons';

const contactInfo: [string, string, string, SvgComponent][] = [
    ['Find us on Twitter', 'Twitter', SOCIALS.Twitter, TwitterIcon],
    ['Join our Discord', 'Discord', SOCIALS.Discord, DiscordIcon],
    ['Source code', 'Code', SOCIALS.GitHub, GitHubIcon],
]

export const ContactInfo = ({
    sx,
    ...props
}: ContactInfoProps) => {
    const { palette } = useTheme();

    const openLink = (e: React.MouseEvent<any>, link: string) => {
        window.open(link, '_blank', 'noopener,noreferrer');
        e.preventDefault();
    }

    return (
        <Box sx={{ minWidth: 'fit-content', height: 'fit-content', ...(sx ?? {}) }} {...props}>
            <BottomNavigation
                showLabels
                sx={{
                    alignItems: 'baseline',
                    background: 'transparent',
                    height: 'fit-content',
                    padding: 1,
                }}>
                {contactInfo.map(([tooltip, label, link, Icon], index: number) => (
                    <Tooltip key={`contact-info-button-${index}`} title={tooltip} placement="top">
                        <BottomNavigationAction
                            label={label}
                            onClick={(e) => { e.preventDefault(); openLink(e, link) }}
                            href={link}
                            icon={
                                <ColorIconButton background={palette.secondary.main} >
                                    <Icon fill={palette.secondary.contrastText} />
                                </ColorIconButton>
                            }
                            sx={{
                                alignItems: 'center',
                                color: palette.primary.contrastText,
                                overflowWrap: 'anywhere',
                            }}
                        />
                    </Tooltip>
                ))}
            </BottomNavigation>
        </Box>
    );
}