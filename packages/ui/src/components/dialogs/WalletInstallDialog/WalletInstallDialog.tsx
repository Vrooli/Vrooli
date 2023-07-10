import { DialogContent, ListItem, ListItemText } from "@mui/material";
import { walletDownloadUrls } from "utils/authentication/walletIntegration";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { WalletInstallDialogProps } from "../types";

const installExtension = (url: string) => {
    window.open(url, "_blank", "noopener,noreferrer");
};

const titleId = "wallet-install-dialog-title";

export const WalletInstallDialog = ({
    onClose,
    open,
    zIndex,
}: WalletInstallDialogProps) => {
    return (
        <LargeDialog
            id="wallet-install-dialog"
            onClose={onClose}
            isOpen={open}
            titleId={titleId}
            zIndex={zIndex}
        >
            <DialogTitle
                id={titleId}
                title={"Install Wallet Extension"}
                onClose={onClose}
                zIndex={zIndex + 1000}
            />
            <DialogContent>
                {Object.values(walletDownloadUrls).map((o, index) => (
                    <ListItem button key={index} onClick={() => installExtension(o[1])}>
                        <ListItemText primary={o[0]} />
                    </ListItem>
                ))}
            </DialogContent>
        </LargeDialog>
    );
};
