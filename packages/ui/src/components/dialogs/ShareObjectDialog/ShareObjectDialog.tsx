/**
 * Dialog for sharing an object
 */
import { Box, Dialog, Palette, Stack, Tooltip, useTheme } from '@mui/material';
import { ShareObjectDialogProps } from '../types';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { useMemo } from 'react';
import { getObjectUrl, ObjectType, PubSub, usePress } from 'utils';
import QRCode from "react-qr-code";
import { CopyIcon, EllipsisIcon, EmailIcon, LinkedInIcon, TwitterIcon } from '@shared/icons';
import { ColorIconButton } from 'components/buttons';
import { SnackSeverity } from '../Snack/Snack';

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
    const url = useMemo(() => object ? getObjectUrl(object) : window.location.href.split('?')[0].split('#')[0], [object]);

    const copyInviteLink = () => {
        navigator.clipboard.writeText(url);
        PubSub.get().publishSnack({ message: 'Copied to clipboard', severity: SnackSeverity.Success });
    }

    /**
     * Opens navigator share dialog (if supported)
     */
    const shareNative = () => { navigator.share({ title, url }) }

    /**
    * When QR code is long-pressed in standalone mode (i.e. app is downloaded), open copy/save photo dialog
    */
    const handleQRCodeLongPress = () => {
        const isStandalone = window.matchMedia('(display-mode: standalone)').matches;
        if (!isStandalone) return;
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
                            onClick={() => openLink(`mailto:?subject=${encodeURIComponent(title)}&body=${encodeURIComponent(url)}`)}
                            background={palette.secondary.main}
                            sx={buttonProps(palette)}
                        >
                            <EmailIcon fill={palette.secondary.contrastText} />
                        </ColorIconButton>
                    </Tooltip>
                    <Tooltip title="Tweet about us">
                        <ColorIconButton
                            onClick={() => openLink(`https://twitter.com/intent/tweet?text=${encodeURIComponent(url)}`)}
                            background={palette.secondary.main}
                            sx={buttonProps(palette)}
                        >
                            <TwitterIcon fill={palette.secondary.contrastText} />
                        </ColorIconButton>
                    </Tooltip>
                    <Tooltip title="Post on LinkedIn">
                        <ColorIconButton
                            onClick={() => openLink(`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(url)}&title=${encodeURIComponent(title)}&summary=${encodeURIComponent(url)}`)}
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
                        value={window.location.href}
                    />
                </Box>
            </Box>
        </Dialog>
    )
}