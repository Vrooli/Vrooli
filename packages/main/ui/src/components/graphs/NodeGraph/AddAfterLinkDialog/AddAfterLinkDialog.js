import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { DialogContent, List, ListItem, ListItemText } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { getTranslation, getUserLanguages } from "../../../../utils/display/translationTools";
import { SessionContext } from "../../../../utils/SessionContext";
import { LargeDialog } from "../../../dialogs/LargeDialog/LargeDialog";
import { TopBar } from "../../../navigation/TopBar/TopBar";
const titleId = "add-after-link-dialog-title";
export const AddAfterLinkDialog = ({ isOpen, handleClose, handleSelect, nodeId, nodes, links, zIndex, }) => {
    const session = useContext(SessionContext);
    const getNodeName = useCallback((nodeId) => {
        const node = nodes.find(n => n.id === nodeId);
        return getTranslation(node, getUserLanguages(session), true).name;
    }, [nodes, session]);
    const linkOptions = useMemo(() => links.filter(l => l.from.id === nodeId), [links, nodeId]);
    const listOptions = linkOptions.map(o => ({
        label: `${getNodeName(o.from.id)} ⟶ ${getNodeName(o.to.id)}`,
        value: o,
    }));
    return (_jsxs(LargeDialog, { id: "add-link-after-dialog", onClose: handleClose, isOpen: isOpen, titleId: titleId, zIndex: zIndex, children: [_jsx(TopBar, { display: "dialog", onClose: handleClose, titleData: { titleId, titleKey: "LinkSelect" } }), _jsx(DialogContent, { children: _jsx(List, { children: listOptions.map(({ label, value }, index) => (_jsx(ListItem, { button: true, onClick: () => { handleSelect(value); handleClose(); }, children: _jsx(ListItemText, { primary: label }) }, index))) }) })] }));
};
//# sourceMappingURL=AddAfterLinkDialog.js.map