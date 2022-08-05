import { Box, Button, Dialog, IconButton, ListItem, Stack, Typography, useTheme } from '@mui/material';
import { HelpButton } from 'components/buttons';
import { WalletSelectDialogProps } from '../types';
import {
    Close as CloseIcon,
} from '@mui/icons-material';
import { getInstalledWalletProviders } from 'utils/authentication';

const helpText =
    `All wallet extensions you have enabled should be shown here, as long as they support (CIP-0030)[https://cips.cardano.org/cips/cip30/]. 
    
This log in option only works for browsers that support Chromium extensions (Chrome, Brave, Opera, Vivaldi, etc. on desktop; Kiwi, Yandex, on Android).  

If you need to download a wallet extension, we suggest [Nami](https://chrome.google.com/webstore/detail/nami/lpfcbjknijpeeillifnkikgncikgfhdo). 

**NOTE:** Working on support for Gero Wallet and Card Wallet.`

export const WalletSelectDialog = ({
    handleOpenInstall,
    onClose,
    open,
    zIndex,
}: WalletSelectDialogProps) => {
    const { palette } = useTheme();

    const walletsInfo = getInstalledWalletProviders();

    const handleClose = () => {
        onClose(null);
    }

    return (
        <Dialog
            onClose={handleClose}
            open={open}
            sx={{
                zIndex,
                '& .MuiPaper-root': {
                    minWidth: 'min(400px, 100%)',
                    margin: '0 auto',
                },
                '& .MuiDialog-paper': {
                    textAlign: 'center',
                    overflowX: 'hidden',
                }
            }}
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
                    Installed Wallets
                </Typography>
                <Box sx={{ marginLeft: 'auto' }}>
                    <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
                    <IconButton
                        edge="start"
                        onClick={handleClose}
                    >
                        <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                    </IconButton>
                </Box>
            </Box>
            <Stack direction="column" spacing={2} sx={{ padding: 2 }}>
                {walletsInfo.map(([walletProvider, { name, icon }]) => (
                    <ListItem
                        key={walletProvider}
                        button
                        onClick={() => onClose(walletProvider)}
                    >
                        <Typography variant="h6" textAlign="center" sx={{ marginLeft: 'auto', marginRight: '10px' }}>
                            {name}
                        </Typography>
                        <Box sx={{
                            marginRight: 'auto',
                            width: '48px',
                            height: '48px',
                        }}>
                            <img src={icon} alt={name} style={{ width: '100%', height: '100%' }} />
                        </Box>
                    </ListItem>
                ))}
                {walletsInfo.length === 0 && (
                    <Typography variant="h6" textAlign="center">
                        No wallets installed
                    </Typography>
                )}
                {/* Install new button */}
                <Button
                    type="button"
                    fullWidth
                    onClick={handleOpenInstall}
                >
                    Install New
                </Button>
            </Stack>
        </Dialog >
    )
}