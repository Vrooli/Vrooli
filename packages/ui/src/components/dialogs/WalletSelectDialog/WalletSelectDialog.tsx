import { Box, Button, DialogContent, ListItem, Stack, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { getInstalledWalletProviders } from "utils/authentication/walletIntegration";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { WalletSelectDialogProps } from "../types";

const helpText =
    "All wallet extensions you have enabled should be shown here, as long as they support (CIP-0030)[https://cips.cardano.org/cips/cip30/].\n\nThis log in option only works for browsers that support Chromium extensions (Chrome, Brave, Opera, Vivaldi, etc. on desktop; Kiwi, Yandex, on Android).\n\nIf you need to download a wallet extension, we suggest [Nami](https://chrome.google.com/webstore/detail/nami/lpfcbjknijpeeillifnkikgncikgfhdo).\n\n**NOTE:** Working on support for Gero Wallet and Card Wallet.";

const titleId = "wallet-select-dialog-title";

export const WalletSelectDialog = ({
    handleOpenInstall,
    onClose,
    open,
    zIndex,
}: WalletSelectDialogProps) => {
    const { t } = useTranslation();

    const walletsInfo = getInstalledWalletProviders();

    const handleClose = () => {
        onClose(null);
    };

    return (
        <LargeDialog
            id="wallet-select-dialog"
            onClose={handleClose}
            isOpen={open}
            titleId={titleId}
            zIndex={zIndex}
        >
            <DialogTitle
                id={titleId}
                help={helpText}
                title={"Installed Wallets"}
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
                            <Typography variant="h6" textAlign="center" sx={{ marginLeft: "auto", marginRight: "10px" }}>
                                {name}
                            </Typography>
                            <Box sx={{
                                marginRight: "auto",
                                width: "48px",
                                height: "48px",
                            }}>
                                <img src={icon} alt={name} style={{ width: "100%", height: "100%" }} />
                            </Box>
                        </ListItem>
                    ))}
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
        </LargeDialog>
    );
};
