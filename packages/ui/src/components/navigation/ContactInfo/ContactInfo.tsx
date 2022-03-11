import { EMAIL, SOCIALS } from '@local/shared';
import {  
    BottomNavigation, 
    BottomNavigationAction, 
    Box, 
    IconButton, 
    Theme, 
    Tooltip 
} from '@mui/material';
import { DiscordIcon, GitHubIcon, TwitterIcon } from 'assets/img';
import { makeStyles } from '@mui/styles';
import { ContactInfoProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    tableHead: {
        background: theme.palette.primary.main,
    },
    tableHeadCell: {
        color: theme.palette.primary.contrastText,
    },
    tableRow: {
        background: theme.palette.background.paper,
    },
    nav: {
        alignItems: 'baseline',
        background: 'transparent',
        height: 'fit-content',
    },
    navAction: {
        alignItems: 'center',
        color: theme.palette.primary.contrastText,
        overflowWrap: 'anywhere',
    },
}));

export const ContactInfo = ({
    className,
    ...props
}: ContactInfoProps) => {
    const classes = useStyles();

    const openLink = (e, link) => {
        window.open(link, '_blank', 'noopener,noreferrer');
        e.preventDefault();
    }

    const contactInfo = [
        ['Find us on Twitter', 'Twitter', SOCIALS.Twitter, TwitterIcon],
        ['Join our Discord', 'Discord', SOCIALS.Discord, DiscordIcon],
        ['Source code', 'Code', SOCIALS.GitHub, GitHubIcon],
    ]

    return (
        <Box sx={{ minWidth: 'fit-content', height: 'fit-content' }} {...props}>
            <BottomNavigation className={classes.nav} showLabels>
                {contactInfo.map(([tooltip, label, link, Icon], index: number) => (
                    <Tooltip key={`contact-info-button-${index}`} title={tooltip} placement="top">
                        <BottomNavigationAction className={classes.navAction} label={label} onClick={(e) => openLink(e, link)} icon={
                            <IconButton sx={{background: (t) => t.palette.secondary.main}}>
                                <Icon fill="#1e581f"/>
                            </IconButton>
                        } />
                    </Tooltip>
                ))}
            </BottomNavigation>
        </Box>
    );
}