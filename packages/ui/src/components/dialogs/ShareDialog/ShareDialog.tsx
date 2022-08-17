import { APP_LINKS } from '@shared/consts';
import { Box, Button, Dialog, Stack, Typography } from '@mui/material';
import { ShareDialogProps } from '../types';
import {
    ContentCopy as CopyIcon,
    Email as EmailIcon,
    LinkedIn as LinkedInIcon,
    Twitter as TwitterIcon
} from '@mui/icons-material';
import { useState } from 'react';

// Invite link
const inviteLink = `https://vrooli.com${APP_LINKS.Start}`;
// Title for social media posts
const postTitle = 'Vrooli - Visual Work Routines';
// Invite message for social media posts
const postText = `The future of work in a decentralized world. ${inviteLink}`;

const buttonProps = {
    height: "48px",
    background: "white",
    color: "black",
    borderRadius: "10px",
    width: "20em",
    display: "flex",
    marginBottom: "5px",
    transition: "0.3s ease-in-out",
    '&:hover': {
        filter: `brightness(120%)`,
        color: 'white',
        border: '1px solid white',
    }
}

export const ShareDialog = ({
    open,
    onClose,
    zIndex,
}: ShareDialogProps) => {
    const [copied, setCopied] = useState<boolean>(false);
    const openLink = (link: string) => window.open(link, '_blank', 'noopener,noreferrer');
    const copyInviteLink = () => {
        navigator.clipboard.writeText(inviteLink);
        setCopied(true);
        setTimeout(() => setCopied(false), 5000);
    }

    return (
        <Dialog
            onClose={onClose}
            open={open}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': {
                    overflow: 'hidden',
                    borderRadius: 2,
                    boxShadow: "0 0 35px 0 rgba(0,0,0,0.5)",
                    textAlign: "center",
                    padding: "1em",
                },
            }}
        >
            <Box sx={{ 
                padding: 2, 
                background: copied ? "#0e650b" : "#072781", 
                color: 'white',
                transition: 'background 0.2s ease-in-out',
            }}>
                <Typography variant="h4" component="h1" mb={1}>Spread the Word 🌍</Typography>
                <Stack direction="column" spacing={1} mb={2} sx={{ alignItems: 'center' }}>
                    <Button 
                        onClick={copyInviteLink} 
                        startIcon={<CopyIcon />}
                        sx={{ ...buttonProps, marginBottom: 0 }}
                    >Copy link</Button>
                    <Button 
                        onClick={() => openLink(`mailto:?subject=${encodeURIComponent(postTitle)}&body=${encodeURIComponent(postText)}`)} 
                        startIcon={<EmailIcon />}
                        sx={{ ...buttonProps }}
                    >Share by email</Button>
                    <Button 
                        onClick={() => openLink(`https://twitter.com/intent/tweet?text=${encodeURIComponent(postText)}`)}
                        startIcon={<TwitterIcon />} 
                        sx={{ ...buttonProps }}
                    >Tweet about us</Button>
                    <Button 
                        onClick={() => openLink(`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(inviteLink)}&title=${encodeURIComponent(postTitle)}&summary=${encodeURIComponent(postText)}`)} 
                        startIcon={<LinkedInIcon />}
                        sx={{ ...buttonProps }}
                    >Post on LinkedIn</Button>
                </Stack>
                { copied ? <Typography variant="h6" component="h4" textAlign="center" mb={1}>🎉 Copied! 🎉</Typography> : null}
            </Box>
        </Dialog>
    )
}