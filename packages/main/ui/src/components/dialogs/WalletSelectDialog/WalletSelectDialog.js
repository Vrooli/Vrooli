import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Button, DialogContent, ListItem, Stack, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { getInstalledWalletProviders } from "../../../utils/authentication/walletIntegration";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const helpText = "All wallet extensions you have enabled should be shown here, as long as they support (CIP-0030)[https://cips.cardano.org/cips/cip30/].\n\nThis log in option only works for browsers that support Chromium extensions (Chrome, Brave, Opera, Vivaldi, etc. on desktop; Kiwi, Yandex, on Android).\n\nIf you need to download a wallet extension, we suggest [Nami](https://chrome.google.com/webstore/detail/nami/lpfcbjknijpeeillifnkikgncikgfhdo).\n\n**NOTE:** Working on support for Gero Wallet and Card Wallet.";
const titleId = "wallet-select-dialog-title";
export const WalletSelectDialog = ({ handleOpenInstall, onClose, open, zIndex, }) => {
    const { t } = useTranslation();
    const walletsInfo = getInstalledWalletProviders();
    const handleClose = () => {
        onClose(null);
    };
    return (_jsxs(LargeDialog, { id: "wallet-select-dialog", onClose: handleClose, isOpen: open, titleId: titleId, zIndex: zIndex, children: [_jsx(DialogTitle, { id: titleId, helpText: helpText, title: "Installed Wallets", onClose: handleClose }), _jsx(DialogContent, { children: _jsxs(Stack, { direction: "column", spacing: 2, paddingTop: 2, children: [walletsInfo.map(([walletProvider, { name, icon }]) => (_jsxs(ListItem, { button: true, onClick: () => onClose(walletProvider), children: [_jsx(Typography, { variant: "h6", textAlign: "center", sx: { marginLeft: "auto", marginRight: "10px" }, children: name }), _jsx(Box, { sx: {
                                        marginRight: "auto",
                                        width: "48px",
                                        height: "48px",
                                    }, children: _jsx("img", { src: icon, alt: name, style: { width: "100%", height: "100%" } }) })] }, walletProvider))), walletsInfo.length === 0 && (_jsx(Typography, { variant: "h6", textAlign: "center", children: t("NoWalletsInstalled") })), _jsx(Button, { type: "button", fullWidth: true, onClick: handleOpenInstall, children: t("InstallWallet") })] }) })] }));
};
//# sourceMappingURL=WalletSelectDialog.js.map