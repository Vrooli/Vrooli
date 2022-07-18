import { Box, Dialog, IconButton, ListItem, ListItemText, Typography, useTheme } from '@mui/material';
import { WalletInstallDialogProps } from '../types';
import {
    Close as CloseIcon
} from '@mui/icons-material';
import { walletDownloadUrls } from 'utils/authentication';

const installExtension = (url: string) => {
    window.open(url, '_blank', 'noopener,noreferrer');
}

export const WalletInstallDialog = ({
    onClose,
    open,
    zIndex,
}: WalletInstallDialogProps) => {
    const { palette } = useTheme();

    return (
        <Dialog
            open={open}
            onClose={onClose}
        >
            <Box sx={{
                padding: 1,
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
            }}>
                <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto' }}>
                    Install Wallet Extension
                </Typography>
                <Box sx={{ marginLeft: 'auto' }}>
                    <IconButton
                        edge="start"
                        onClick={onClose}
                    >
                        <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                    </IconButton>
                </Box>
            </Box>
            {Object.values(walletDownloadUrls).map((o, index) => (
                <ListItem button key={index} onClick={() => installExtension(o[1])}>
                    <ListItemText primary={o[0]} />
                </ListItem>
            ))}
        </Dialog>
    )
}