/**
 * Dialog for spreading the word about the site.
 */
import { APP_LINKS } from '@shared/consts';
import { Box, Button, Dialog, Stack, Typography } from '@mui/material';
import { ShareSiteDialogProps } from '../types';
import { useState } from 'react';
import QRCode from "react-qr-code";
import { CopyIcon, EmailIcon, LinkedInIcon, TwitterIcon } from '@shared/icons';

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

export const ShareSiteDialog = ({
    open,
    onClose,
    zIndex,
}: ShareSiteDialogProps) => {
    const [copied, setCopied] = useState<boolean>(false);
    const openLink = (link: string) => window.open(link, '_blank', 'noopener,noreferrer');
    const copyInviteLink = () => {
        navigator.clipboard.writeText(inviteLink);
        setCopied(true);
        setTimeout(() => setCopied(false), 5000);
    }

    /**
     * Opens navigator share dialog (if supported)
     */
    const shareNative = () => {
        navigator.share({
            title: postTitle,
            text: postText,
            url: inviteLink,
        })
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
                    boxShadow: 12,
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
                        startIcon={<CopyIcon fill="black" />}
                        sx={{ ...buttonProps, marginBottom: 0 }}
                    >Copy link</Button>
                    <Button
                        onClick={() => openLink(`mailto:?subject=${encodeURIComponent(postTitle)}&body=${encodeURIComponent(postText)}`)}
                        startIcon={<EmailIcon fill="black" />}
                        sx={{ ...buttonProps }}
                    >Share by email</Button>
                    <Button
                        onClick={() => openLink(`https://twitter.com/intent/tweet?text=${encodeURIComponent(postText)}`)}
                        startIcon={<TwitterIcon fill="black" />}
                        sx={{ ...buttonProps }}
                    >Tweet about us</Button>
                    <Button
                        onClick={() => openLink(`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(inviteLink)}&title=${encodeURIComponent(postTitle)}&summary=${encodeURIComponent(postText)}`)}
                        startIcon={<LinkedInIcon fill="black" />}
                        sx={{ ...buttonProps }}
                    >Post on LinkedIn</Button>
                    <Button
                        onClick={shareNative}
                        sx={{ ...buttonProps }}
                    >Other</Button>
                    <Box sx={{
                        width: '220px',
                        height: '220px',
                        background: 'white',
                        padding: '10px',
                    }}>
                        <QRCode
                            size={200}
                            style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                            value="https://vrooli.com"
                        />
                    </Box>
                </Stack>
                {copied ? <Typography variant="h6" component="h4" textAlign="center" mb={1}>🎉 Copied! 🎉</Typography> : null}
            </Box>
        </Dialog>
    )
}