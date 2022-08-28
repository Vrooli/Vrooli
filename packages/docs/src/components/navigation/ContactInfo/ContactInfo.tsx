import { SOCIALS } from '@shared/consts';
import {
    BottomNavigation,
    BottomNavigationAction,
    Box,
    IconButton,
    Tooltip,
    useTheme,
} from '@mui/material';
import { DiscordIcon, GitHubIcon, TwitterIcon } from 'assets/img';
import { SvgProps } from 'assets/img/types';
import { ContactInfoProps } from '../types';

const contactInfo: [string, string, string, (props: SvgProps) => JSX.Element][] = [
    ['Find us on Twitter', 'Twitter', SOCIALS.Twitter, TwitterIcon],
    ['Join our Discord', 'Discord', SOCIALS.Discord, DiscordIcon],
    ['Source code', 'Code', SOCIALS.GitHub, GitHubIcon],
]

export const ContactInfo = ({
    className,
    ...props
}: ContactInfoProps) => {
    const { palette } = useTheme();

    const openLink = (e: React.MouseEvent, link: string) => {
        window.open(link, '_blank', 'noopener,noreferrer');
        e.preventDefault();
    }

    return (
        <Box sx={{ minWidth: 'fit-content', height: 'fit-content' }} {...props}>
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
                            onClick={(e) => openLink(e, link)}
                            icon={
                                <IconButton sx={{ background: palette.secondary.main }}>
                                    <Icon fill={palette.secondary.contrastText} />
                                </IconButton>
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