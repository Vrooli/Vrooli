import { Dialog, DialogContent, ListItem, ListItemText } from '@mui/material';
import { WalletInstallDialogProps } from '../types';
import { walletDownloadUrls } from 'utils/authentication';
import { DialogTitle } from 'components';

const installExtension = (url: string) => {
    window.open(url, '_blank', 'noopener,noreferrer');
}

const titleAria = 'wallet-install-dialog-title';

export const WalletInstallDialog = ({
    onClose,
    open,
    zIndex,
}: WalletInstallDialogProps) => {
    return (
        <Dialog
            open={open}
            onClose={onClose}
            aria-labelledby={titleAria}
            sx={{ zIndex }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                title={'Install Wallet Extension'}
                onClose={onClose}
            />
            <DialogContent>
                {Object.values(walletDownloadUrls).map((o, index) => (
                    <ListItem button key={index} onClick={() => installExtension(o[1])}>
                        <ListItemText primary={o[0]} />
                    </ListItem>
                ))}
            </DialogContent>
        </Dialog>
    )
}