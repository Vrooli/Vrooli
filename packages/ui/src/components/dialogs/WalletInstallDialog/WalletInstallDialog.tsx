import { DialogContent, ListItem, ListItemText } from "@mui/material";
import { useTranslation } from "react-i18next";
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
}: WalletInstallDialogProps) => {
    const { t } = useTranslation();

    return (
        <LargeDialog
            id="wallet-install-dialog"
            onClose={onClose}
            isOpen={open}
            titleId={titleId}
        >
            <DialogTitle
                id={titleId}
                title={t("InstallWalletExtension")}
                onClose={onClose}
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
