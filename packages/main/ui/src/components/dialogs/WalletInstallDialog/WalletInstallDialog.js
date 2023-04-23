import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { DialogContent, ListItem, ListItemText } from "@mui/material";
import { walletDownloadUrls } from "../../../utils/authentication/walletIntegration";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const installExtension = (url) => {
    window.open(url, "_blank", "noopener,noreferrer");
};
const titleId = "wallet-install-dialog-title";
export const WalletInstallDialog = ({ onClose, open, zIndex, }) => {
    return (_jsxs(LargeDialog, { id: "wallet-install-dialog", onClose: onClose, isOpen: open, titleId: titleId, zIndex: zIndex, children: [_jsx(DialogTitle, { id: titleId, title: "Install Wallet Extension", onClose: onClose }), _jsx(DialogContent, { children: Object.values(walletDownloadUrls).map((o, index) => (_jsx(ListItem, { button: true, onClick: () => installExtension(o[1]), children: _jsx(ListItemText, { primary: o[0] }) }, index))) })] }));
};
//# sourceMappingURL=WalletInstallDialog.js.map