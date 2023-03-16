import { Dialog, DialogContent, ListItem, ListItemText } from '@mui/material';
import { walletDownloadUrls } from 'utils/authentication/walletIntegration';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { WalletInstallDialogProps } from '../types';

const installExtension = (url: string) => {
    window.open(url, '_blank', 'noopener,noreferrer');
}

const titleId = 'wallet-install-dialog-title';

export const WalletInstallDialog = ({
    onClose,
    open,
    zIndex,
}: WalletInstallDialogProps) => {
    return (
        <Dialog
            open={open}
            onClose={onClose}
            aria-labelledby={titleId}
            sx={{ zIndex }}
        >
            <DialogTitle
                id={titleId}
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