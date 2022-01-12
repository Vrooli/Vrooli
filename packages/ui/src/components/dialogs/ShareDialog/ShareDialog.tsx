import { APP_LINKS } from '@local/shared';
import { Box, Dialog, DialogTitle, List, ListItem, Typography } from '@mui/material';
import { useRef, useState } from 'react';
import { ShareDialogProps } from '../types';
import { 
    EmailIcon,
    InstagramIcon,
    LinkedinIcon,
    TwitterIcon,
} from 'assets/img';

// Invite link
const inviteLink = `https://vrooli.com/${APP_LINKS.Start}`;
// Title for social media posts
const postTitle = 'Vrooli - Visual Work Routines';
// Invite message for social media posts
const postText = `The future of work in a decentralized world. ${inviteLink}`;
// Social media share options
// In the form of [icon, label, link]
const shareOptions: Array<[any, string, string]> = [
    [EmailIcon, 'Email', `mailto:?subject=${encodeURIComponent(postTitle)}&body=${encodeURIComponent(postText)}`],
    [TwitterIcon, 'Twitter', `https://twitter.com/intent/tweet?text=${encodeURIComponent(postText)}`],
    [LinkedinIcon, 'LinkedIn', `https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(inviteLink)}&title=${encodeURIComponent(postTitle)}&summary=${encodeURIComponent(postText)}`],
    [InstagramIcon, 'Instagram', `https://www.instagram.com/intent/share/?url=${encodeURIComponent(inviteLink)}&text=${encodeURIComponent(postText)}`],
]

const shareListItems = shareOptions.map(([Icon, label, link]) => (
    <ListItem
        key={`share-list-item-${label}`}
        onClick={() => window.open(link, '_blank', 'noopener,noreferrer')}
    >
        <Box display="flex" alignItems="center">
            <Icon style={{width: '30px', height: '30px', fill: '#0b2684'}} />
            <Typography variant="body2">{label}</Typography>
        </Box>
    </ListItem>
))

export const ShareDialog = ({
    open,
    onClose
}: ShareDialogProps) => {
    const [copied, setCopied] = useState(false);

    const copyInviteLink = () => {
        navigator.clipboard.writeText(inviteLink);
        setCopied(true);
    }

    return (
        <Dialog
            onClose={onClose}
            open={open}
            sx={{
                zIndex: 10000,
                width: 'min(500px, 100vw)',
                textAlign: 'center',
                overflow: 'hidden',
            }}
        >
            <DialogTitle sx={{ background: (t) => t.palette.primary.light }}>
                Invite
            </DialogTitle>
            <Box sx={{ padding: 2 }}>
                <Typography variant="h5">Socials</Typography>
                <List>
                    {shareListItems}
                </List>
                <Typography variant="h5">Copy Link</Typography>
                <Box
                    onClick={copyInviteLink}
                    sx={{
                        border: '1px solid',
                        borderRadius: '12px',
                        borderColor: copied ? '#33e433' : 'black',
                        color: copied ? '#33e433' : 'black',
                        height: '1.5em',
                        textAlign: 'center',
                        verticalAlign: 'middle',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                    }}
                >
                    <Typography variant="body1" sx={{padding: 1}}>{inviteLink}</Typography>
                </Box>
                {
                    copied ? (
                        <Box sx={{
                            marginBottom: 1,
                        }}>
                            🎉Copied!🎉
                        </Box>
                    ) : null
                }
            </Box>
        </Dialog>
    )
}