import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import ListItem from "@mui/material/ListItem";
import ListItemText from "@mui/material/ListItemText";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useTranslation } from "react-i18next";
import { getInstalledWalletProviders, walletDownloadUrls } from "../../utils/authentication/walletIntegration.js";
import { Dialog, DialogContent } from "./Dialog/Dialog.js";
import { type WalletInstallDialogProps, type WalletSelectDialogProps } from "./types.js";

function installExtension(url: string) {
    window.open(url, "_blank", "noopener,noreferrer");
}

export function WalletInstallDialog({
    onClose,
    open,
}: WalletInstallDialogProps) {
    const { t } = useTranslation();

    return (
        <Dialog
            isOpen={open}
            onClose={onClose}
            title={t("InstallWalletExtension")}
            size="md"
        >
            <DialogContent>
                {Object.entries(walletDownloadUrls).map(([walletProvider, [name, url]]) => {
                    function handleClick() {
                        installExtension(url);
                    }

                    return (
                        <ListItem button key={walletProvider} onClick={handleClick}>
                            <ListItemText primary={name} />
                        </ListItem>
                    );
                })}
            </DialogContent>
        </Dialog>
    );
}

const helpText =
    "All wallet extensions you have enabled should be shown here, as long as they support (CIP-0030)[https://cips.cardano.org/cips/cip30/].\n\nThis log in option only works for browsers that support Chromium extensions (Chrome, Brave, Opera, Vivaldi, etc. on desktop; Kiwi, Yandex, on Android).\n\nIf you need to download a wallet extension, we suggest [Nami](https://chrome.google.com/webstore/detail/nami/lpfcbjknijpeeillifnkikgncikgfhdo).\n\n**NOTE:** Working on support for Gero Wallet and Card Wallet.";
const walletSelectNameStyle = { marginLeft: "auto", marginRight: "10px" } as const;
const walletSelectImageBoxStyle = {
    marginRight: "auto",
    width: "48px",
    height: "48px",
} as const;
const walletSelectImageStyle = { width: "100%", height: "100%" } as const;

export function WalletSelectDialog({
    handleOpenInstall,
    onClose,
    open,
}: WalletSelectDialogProps) {
    const { t } = useTranslation();

    const walletsInfo = getInstalledWalletProviders();

    function handleClose() {
        onClose(null);
    }

    return (
        <Dialog
            isOpen={open}
            onClose={handleClose}
            title={t("InstalledWallets")}
            size="md"
        >
            <DialogContent>
                <Stack direction="column" spacing={2} paddingTop={2}>
                    {walletsInfo.map(([walletProvider, { name, icon }]) => {
                        function handleClick() {
                            onClose(walletProvider);
                        }

                        return (
                            <ListItem button key={walletProvider} onClick={handleClick}>
                                <Typography variant="h6" textAlign="center" sx={walletSelectNameStyle}>
                                    {name}
                                </Typography>
                                <Box sx={walletSelectImageBoxStyle}>
                                    <img src={icon} alt={name} style={walletSelectImageStyle} />
                                </Box>
                            </ListItem>
                        );
                    })}
                    {walletsInfo.length === 0 && (
                        <Typography variant="h6" textAlign="center">
                            {t("NoWalletsInstalled")}
                        </Typography>
                    )}
                    {/* Install new button */}
                    <Button
                        type="button"
                        fullWidth
                        onClick={handleOpenInstall}
                        variant="contained"
                    >
                        {t("InstallWallet")}
                    </Button>
                </Stack>
            </DialogContent>
        </Dialog>
    );
}
