import { EMAIL, SOCIALS } from '@local/shared';
import {  
    BottomNavigation, 
    BottomNavigationAction, 
    IconButton, 
    Theme, 
    Tooltip 
} from '@mui/material';
import { 
    Email as EmailIcon,
    GitHub as GitHubIcon,
    Twitter as TwitterIcon,
} from '@mui/icons-material';
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
    iconButton: {
        background: theme.palette.secondary.main,
        fill: theme.palette.secondary.contrastText,
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
        ['Email Us', 'Email', EMAIL.Link, EmailIcon],
        ['Source code', 'Code', SOCIALS.GitHub, GitHubIcon],
    ]

    return (
        <div style={{ minWidth: 'fit-content', height: 'fit-content'}} {...props}>
            <BottomNavigation className={classes.nav} showLabels>
                {contactInfo.map(([tooltip, label, link, Icon]) => (
                    <Tooltip title={tooltip} placement="top">
                        <BottomNavigationAction className={classes.navAction} label={label} onClick={(e) => openLink(e, link)} icon={
                            <IconButton className={classes.iconButton}>
                                <Icon />
                            </IconButton>
                        } />
                    </Tooltip>
                ))}
            </BottomNavigation>
        </div>
    );
}