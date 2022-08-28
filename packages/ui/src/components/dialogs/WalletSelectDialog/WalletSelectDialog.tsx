import { Box, Button, Dialog, DialogContent, ListItem, Stack, Typography } from '@mui/material';
import { WalletSelectDialogProps } from '../types';
import { getInstalledWalletProviders } from 'utils/authentication';
import { DialogTitle } from 'components';

const helpText =
    `All wallet extensions you have enabled should be shown here, as long as they support (CIP-0030)[https://cips.cardano.org/cips/cip30/]. 
    
This log in option only works for browsers that support Chromium extensions (Chrome, Brave, Opera, Vivaldi, etc. on desktop; Kiwi, Yandex, on Android).  

If you need to download a wallet extension, we suggest [Nami](https://chrome.google.com/webstore/detail/nami/lpfcbjknijpeeillifnkikgncikgfhdo). 

**NOTE:** Working on support for Gero Wallet and Card Wallet.`

const titleAria = 'wallet-select-dialog-title';

export const WalletSelectDialog = ({
    handleOpenInstall,
    onClose,
    open,
    zIndex,
}: WalletSelectDialogProps) => {
    const walletsInfo = getInstalledWalletProviders();

    const handleClose = () => {
        onClose(null);
    }

    return (
        <Dialog
            onClose={handleClose}
            open={open}
            aria-labelledby={titleAria}
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
            <DialogTitle
                ariaLabel={titleAria}
                helpText={helpText}
                title={'Installed Wallets'}
                onClose={handleClose}
            />
            <DialogContent>
                <Stack direction="column" spacing={2} paddingTop={2}>
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
            </DialogContent>
        </Dialog >
    )
}