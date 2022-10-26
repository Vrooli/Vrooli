/**
 * Dialog for spreading the word about the site.
 */
import { APP_LINKS } from '@shared/consts';
import { Box, Dialog, Palette, Stack, Tooltip, useTheme } from '@mui/material';
import { ShareSiteDialogProps } from '../types';
import QRCode from "react-qr-code";
import { CopyIcon, EllipsisIcon, EmailIcon, LinkedInIcon, TwitterIcon } from '@shared/icons';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { PubSub, usePress } from 'utils';
import { ColorIconButton } from 'components/buttons';
import { SnackSeverity } from '../Snack/Snack';

// Invite link
const inviteLink = `https://vrooli.com${APP_LINKS.Start}`;
// Title for social media posts
const postTitle = 'Vrooli - Visual Work Routines';
// Invite message for social media posts
const postText = `The future of work in a decentralized world. ${inviteLink}`;

const buttonProps = (palette: Palette) => ({
    height: '48px',
    width: '48px',
})

const titleAria = 'share-site-dialog-title';

export const ShareSiteDialog = ({
    open,
    onClose,
    zIndex,
}: ShareSiteDialogProps) => {
    const { palette } = useTheme();

    const openLink = (link: string) => window.open(link, '_blank', 'noopener,noreferrer');

    const copyInviteLink = () => {
        navigator.clipboard.writeText(inviteLink);
        PubSub.get().publishSnack({ message: 'Copied to clipboard', severity: SnackSeverity.Success });
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
                        <ColorIconButton
                            onClick={copyInviteLink}
                            background={palette.secondary.main}
                            sx={buttonProps(palette)}
                        >
                            <CopyIcon fill={palette.secondary.contrastText} />
                        </ColorIconButton>
                    </Tooltip>
                    <Tooltip title="Share by email">
                        <ColorIconButton
                            onClick={() => openLink(`mailto:?subject=${encodeURIComponent(postTitle)}&body=${encodeURIComponent(postText)}`)}
                            background={palette.secondary.main}
                            sx={buttonProps(palette)}
                        >
                            <EmailIcon fill={palette.secondary.contrastText} />
                        </ColorIconButton>
                    </Tooltip>
                    <Tooltip title="Tweet about us">
                        <ColorIconButton
                            onClick={() => openLink(`https://twitter.com/intent/tweet?text=${encodeURIComponent(postText)}`)}
                            background={palette.secondary.main}
                            sx={buttonProps(palette)}
                        >
                            <TwitterIcon fill={palette.secondary.contrastText} />
                        </ColorIconButton>
                    </Tooltip>
                    <Tooltip title="Post on LinkedIn">
                        <ColorIconButton
                            onClick={() => openLink(`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(inviteLink)}&title=${encodeURIComponent(postTitle)}&summary=${encodeURIComponent(postText)}`)}
                            background={palette.secondary.main}
                            sx={buttonProps(palette)}
                        >
                            <LinkedInIcon fill={palette.secondary.contrastText} />
                        </ColorIconButton>
                    </Tooltip>
                    <Tooltip title="Share by another method">
                        <ColorIconButton
                            onClick={shareNative}
                            background={palette.secondary.main}
                            sx={buttonProps(palette)}
                        >
                            <EllipsisIcon fill={palette.secondary.contrastText} />
                        </ColorIconButton>
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
                        value="https://vrooli.com"
                    />
                </Box>
            </Box>
        </Dialog>
    )
}