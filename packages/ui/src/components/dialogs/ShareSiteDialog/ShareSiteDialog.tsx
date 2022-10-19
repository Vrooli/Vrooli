/**
 * Dialog for spreading the word about the site.
 */
import { APP_LINKS } from '@shared/consts';
import { Box, Dialog, IconButton, Palette, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { ShareSiteDialogProps } from '../types';
import { useState } from 'react';
import QRCode from "react-qr-code";
import { CopyIcon, EllipsisIcon, EmailIcon, LinkedInIcon, TwitterIcon } from '@shared/icons';
import { DialogTitle } from '../DialogTitle/DialogTitle';

// Invite link
const inviteLink = `https://vrooli.com${APP_LINKS.Start}`;
// Title for social media posts
const postTitle = 'Vrooli - Visual Work Routines';
// Invite message for social media posts
const postText = `The future of work in a decentralized world. ${inviteLink}`;

const buttonProps = (palette: Palette) => ({
    height: '48px',
    width: '48px',
    background: palette.secondary.main,
    // color: palette.secondary.contrastText,
    // display: "flex",
    // marginBottom: "5px",
    // transition: "0.3s ease-in-out",
    '&:hover': {
        filter: `brightness(120%)`,
        background: palette.secondary.main,
    }
})

const titleAria = 'share-site-dialog-title';

export const ShareSiteDialog = ({
    open,
    onClose,
    zIndex,
}: ShareSiteDialogProps) => {
    const { palette } = useTheme();

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
                    minWidth: 'min(400px, 100%)',
                },
            }}
        >
            <DialogTitle ariaLabel={titleAria} title="Spread the Word 🌍" onClose={onClose} />
            <Box sx={{ padding: 2 }}>
                <Stack direction="row" spacing={1} mb={2} display="flex" justifyContent="center" alignItems="center">
                    <Tooltip title="Copy invite link">
                        <IconButton onClick={copyInviteLink} sx={buttonProps(palette)}>
                            <CopyIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Share by email">
                        <IconButton onClick={() => openLink(`mailto:?subject=${encodeURIComponent(postTitle)}&body=${encodeURIComponent(postText)}`)} sx={buttonProps(palette)}>
                            <EmailIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Tweet about us">
                        <IconButton onClick={() => openLink(`https://twitter.com/intent/tweet?text=${encodeURIComponent(postText)}`)} sx={buttonProps(palette)}>
                            <TwitterIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Post on LinkedIn">
                        <IconButton onClick={() => openLink(`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(inviteLink)}&title=${encodeURIComponent(postTitle)}&summary=${encodeURIComponent(postText)}`)} sx={buttonProps(palette)}>
                            <LinkedInIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Share by another method">
                        <IconButton onClick={shareNative} sx={buttonProps(palette)}>
                            <EllipsisIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <Box sx={{
                    width: '210px',
                    height: '210px',
                    background: palette.secondary.main,
                    borderRadius: 1,
                    padding: 0.5,
                    marginLeft: 'auto',
                    marginRight: 'auto',
                }}>
                    <QRCode
                        size={200}
                        style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                        value="https://vrooli.com"
                    />
                </Box>
                {copied ? <Typography variant="h6" component="h4" textAlign="center" mb={1} mt={2}>🎉 Copied! 🎉</Typography> : null}
            </Box>
        </Dialog>
    )
}