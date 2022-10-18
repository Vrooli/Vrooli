/**
 * Dialog for sharing an object
 */
import { Box, Button, Dialog, Stack, Typography, useTheme } from '@mui/material';
import { ShareObjectDialogProps } from '../types';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { useState } from 'react';
import { ObjectType } from 'utils';
import QRCode from "react-qr-code";
import { CopyIcon, EmailIcon, LinkedInIcon, TwitterIcon } from '@shared/icons';

// Title for social media posts
const postTitle: { [key in ObjectType]?: string } = {
    'Comment': 'Check out this comment on Vrooli',
    'Organization': 'Check out this organization on Vrooli',
    'Project': 'Check out this project on Vrooli',
    'Routine': 'Check out this routine on Vrooli',
    'Standard': 'Check out this standard on Vrooli',
    'User': 'Check out this user on Vrooli',
}

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

const openLink = (link: string) => window.open(link, '_blank', 'noopener,noreferrer');

const titleAria = 'share-object-dialog-title';

export const ShareObjectDialog = ({
    objectType,
    open,
    onClose,
    zIndex,
}: ShareObjectDialogProps) => {
    const { palette } = useTheme();

    const [copied, setCopied] = useState<boolean>(false);

    /**
     * Get URL minus search and hash
     */
    const getLink = () => window.location.href.split('?')[0].split('#')[0];
    const copyInviteLink = () => {
        navigator.clipboard.writeText(getLink());
        setCopied(true);
        setTimeout(() => setCopied(false), 5000);
    }

    /**
     * Opens navigator share dialog (if supported)
     */
    const shareNative = () => {
        if (navigator.share) {
            navigator.share({
                title: postTitle[objectType],
                url: getLink(),
            })
        }
    }

    return (
        <Dialog
            onClose={onClose}
            open={open}
            aria-labelledby={titleAria}
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
            <DialogTitle
                ariaLabel={titleAria}
                title="Share"
                onClose={onClose}
            />
            <Box sx={{
                padding: 2,
                background: copied ? "#0e650b" : palette.background.default,
                color: 'white',
                transition: 'background 0.2s ease-in-out',
            }}>
                <Stack direction="column" spacing={1} mb={2} sx={{ alignItems: 'center' }}>
                    <Button
                        onClick={copyInviteLink}
                        startIcon={<CopyIcon fill='black' />}
                        sx={{ ...buttonProps, marginBottom: 0 }}
                    >Copy link</Button>
                    <Button
                        onClick={() => openLink(`mailto:?subject=${encodeURIComponent(postTitle[objectType] ?? '')}&body=${encodeURIComponent(getLink())}`)}
                        startIcon={<EmailIcon fill='black' />}
                        sx={{ ...buttonProps }}
                    >Share by email</Button>
                    <Button
                        onClick={() => openLink(`https://twitter.com/intent/tweet?text=${encodeURIComponent(getLink())}`)}
                        startIcon={<TwitterIcon fill='black' />}
                        sx={{ ...buttonProps }}
                    >Tweet</Button>
                    <Button
                        onClick={() => openLink(`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(getLink())}&title=${encodeURIComponent(postTitle[objectType] ?? '')}&summary=${encodeURIComponent(getLink())}`)}
                        startIcon={<LinkedInIcon fill='black' />}
                        sx={{ ...buttonProps }}
                    >Post on LinkedIn</Button>
                    <Button
                        onClick={shareNative}
                        sx={{ ...buttonProps }}
                    >Other</Button>
                    <Box sx={{
                        width: '200px',
                        height: '200px',

                    }}>
                        <QRCode
                            size={200}
                            style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                            value={window.location.href}
                        />
                    </Box>
                </Stack>
                {copied ? <Typography variant="h6" component="h4" textAlign="center" mb={1}>🎉 Copied! 🎉</Typography> : null}
            </Box>
        </Dialog>
    )
}