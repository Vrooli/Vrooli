import { jsx as _jsx } from "react/jsx-runtime";
import { AddEndNodeAfterIcon, AddIncomingLinkIcon, AddOutgoingLinkIcon, AddRoutineListAfterIcon, AddRoutineListBeforeIcon, DeleteIcon, DeleteNodeIcon, EditIcon, MoveNodeIcon, UnlinkNodeIcon } from "@local/icons";
import { useMemo } from "react";
import { ListMenu } from "../../../dialogs/ListMenu/ListMenu";
const allOptionsMap = {
    AddIncomingLink: ["Add incoming link", AddIncomingLinkIcon],
    AddOutgoingLink: ["Add outgoing link", AddOutgoingLinkIcon],
    AddListBeforeNode: ["Add routine list before", AddRoutineListBeforeIcon],
    AddListAfterNode: ["Add routine list after", AddRoutineListAfterIcon],
    AddEndAfterNode: ["Add end node after", AddEndNodeAfterIcon],
    DeleteNode: ["Delete node", DeleteNodeIcon],
    MoveNode: ["Move node", MoveNodeIcon],
    UnlinkNode: ["Unlink node", UnlinkNodeIcon],
    EditSubroutine: ["Edit subroutine", EditIcon],
    DeleteSubroutine: ["Delete subroutine", DeleteIcon],
};
const listOptions = Object.keys(allOptionsMap).map(o => ({
    label: allOptionsMap[o][0],
    value: o,
    Icon: allOptionsMap[o][1],
}));
export const NodeContextMenu = ({ id, anchorEl, availableActions, handleClose, handleSelect, zIndex, }) => {
    const availableOptions = useMemo(() => listOptions.filter(o => availableActions.includes(o.value)), [availableActions]);
    return (_jsx(ListMenu, { id: id, anchorEl: anchorEl, data: availableOptions, onSelect: handleSelect, onClose: handleClose, zIndex: zIndex }));
};
//# sourceMappingURL=NodeContextMenu.js.map