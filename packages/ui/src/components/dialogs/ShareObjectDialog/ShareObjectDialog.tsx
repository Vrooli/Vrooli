/**
 * Dialog for sharing an object
 */
import { Box, Dialog, IconButton, Palette, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { ShareObjectDialogProps } from '../types';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { useMemo, useState } from 'react';
import { getObjectSearchParams, getObjectSlug, getObjectUrlBase, ObjectType, usePress } from 'utils';
import QRCode from "react-qr-code";
import { CopyIcon, EllipsisIcon, EmailIcon, LinkedInIcon, TwitterIcon } from '@shared/icons';

// Title for social media posts
const postTitle: { [key in ObjectType]?: string } = {
    'Comment': 'Check out this comment on Vrooli',
    'Organization': 'Check out this organization on Vrooli',
    'Project': 'Check out this project on Vrooli',
    'Routine': 'Check out this routine on Vrooli',
    'Standard': 'Check out this standard on Vrooli',
    'User': 'Check out this user on Vrooli',
}

const buttonProps = (palette: Palette) => ({
    height: '48px',
    width: '48px',
    background: palette.secondary.main,
    '&:hover': {
        filter: `brightness(120%)`,
        background: palette.secondary.main,
    }
})

const openLink = (link: string) => window.open(link, '_blank', 'noopener,noreferrer');

const titleAria = 'share-object-dialog-title';

export const ShareObjectDialog = ({
    object,
    open,
    onClose,
    zIndex,
}: ShareObjectDialogProps) => {
    const { palette } = useTheme();

    const title = useMemo(() => object && object.__typename in postTitle ? postTitle[object.__typename] : 'Check out this object on Vrooli', [object]);
    const url = useMemo(() => object ? `${getObjectUrlBase(object)}/${getObjectSlug(object)}${getObjectSearchParams(object)}` : window.location.href.split('?')[0].split('#')[0], [object]);

    const [copied, setCopied] = useState<boolean>(false);
    const copyInviteLink = () => {
        navigator.clipboard.writeText(url);
        setCopied(true);
        setTimeout(() => setCopied(false), 5000);
    }

    /**
     * Opens navigator share dialog (if supported)
     */
    const shareNative = () => { navigator.share({ title, url }) }

    /**
    * When QR code is long-pressed in PWA, open copy/save photo dialog
    */
    const handleQRCodeLongPress = () => {
        const isPWA = window.matchMedia('(display-mode: standalone)').matches;
        if (!isPWA) return;
        // Find image using parent element's ID
        const qrCode = document.getElementById('qr-code-box')?.firstChild as HTMLImageElement;
        if (!qrCode) return;
        // Create file
        const file = new File([qrCode.src], 'qr-code.png', { type: 'image/png' });
        // Open save dialog
        const url = URL.createObjectURL(file);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'qr-code.png';
        a.click();
        URL.revokeObjectURL(url);
    }

    const pressEvents = usePress({
        onLongPress: handleQRCodeLongPress,
        onClick: handleQRCodeLongPress,
        onRightClick: handleQRCodeLongPress,
    });


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
                    boxShadow: 12,
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
            <Box sx={{ padding: 2 }}>
                <Stack direction="row" spacing={1} mb={2} display="flex" justifyContent="center" alignItems="center">
                    <Tooltip title="Copy invite link">
                        <IconButton onClick={copyInviteLink} sx={buttonProps(palette)}>
                            <CopyIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Share by email">
                        <IconButton onClick={() => openLink(`mailto:?subject=${encodeURIComponent(title)}&body=${encodeURIComponent(url)}`)} sx={buttonProps(palette)}>
                            <EmailIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Tweet about us">
                        <IconButton onClick={() => openLink(`https://twitter.com/intent/tweet?text=${encodeURIComponent(url)}`)} sx={buttonProps(palette)}>
                            <TwitterIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Post on LinkedIn">
                        <IconButton onClick={() => openLink(`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(url)}&title=${encodeURIComponent(title)}&summary=${encodeURIComponent(url)}`)} sx={buttonProps(palette)}>
                            <LinkedInIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Share by another method">
                        <IconButton onClick={shareNative} sx={buttonProps(palette)}>
                            <EllipsisIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <Box
                    id="qr-code-box"
                    {...pressEvents}
                    sx={{
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
                        value={window.location.href}
                    />
                </Box>
                {copied ? <Typography variant="h6" component="h4" textAlign="center" mb={1} mt={2}>🎉 Copied! 🎉</Typography> : null}
            </Box>
        </Dialog>
    )
}