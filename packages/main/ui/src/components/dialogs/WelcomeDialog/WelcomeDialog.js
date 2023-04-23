import { jsx as _jsx } from "react/jsx-runtime";
import { WelcomeView } from "../../../views/WelcomeView/WelcomeView";
import { LargeDialog } from "../LargeDialog/LargeDialog";
export const WelcomeDialog = ({ isOpen, onClose, }) => {
    return (_jsx(LargeDialog, { id: "welcome-dialog", isOpen: isOpen, onClose: onClose, zIndex: 10000, children: _jsx(WelcomeView, { display: "dialog" }) }));
};
//# sourceMappingURL=WelcomeDialog.js.map