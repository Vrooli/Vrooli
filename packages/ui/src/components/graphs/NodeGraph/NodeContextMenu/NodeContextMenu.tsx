import { AddEndNodeAfterIcon, AddIncomingLinkIcon, AddOutgoingLinkIcon, AddRoutineListAfterIcon, AddRoutineListBeforeIcon, DeleteIcon, DeleteNodeIcon, EditIcon, MoveNodeIcon, SvgComponent, UnlinkNodeIcon } from "@local/shared";
import { ListMenu } from "components/dialogs/ListMenu/ListMenu";
import { ListMenuItemData } from "components/dialogs/types";
import { useMemo } from "react";
import { BuildAction } from "utils/consts";
import { NodeContextMenuProps } from "../types";

const allOptionsMap: { [index in Exclude<BuildAction, BuildAction.AddSubroutine>]?: [string, SvgComponent] } = {
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

const listOptions: ListMenuItemData<BuildAction>[] = Object.keys(allOptionsMap).map(o => ({
    label: allOptionsMap[o][0],
    value: o as BuildAction,
    Icon: allOptionsMap[o][1],
}));

// Custom context menu for nodes
export const NodeContextMenu = ({
    id,
    anchorEl,
    availableActions,
    handleClose,
    handleSelect,
    zIndex,
}: NodeContextMenuProps) => {
    const availableOptions = useMemo(() => listOptions.filter(o => availableActions.includes(o.value)), [availableActions]);

    return (
        <ListMenu
            id={id}
            anchorEl={anchorEl}
            data={availableOptions}
            onSelect={handleSelect}
            onClose={handleClose}
            zIndex={zIndex}
        />
    );
};
