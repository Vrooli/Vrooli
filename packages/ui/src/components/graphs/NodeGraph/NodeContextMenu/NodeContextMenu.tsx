import { NodeContextMenuProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import {
    Delete as DeleteIcon,
    Edit as EditIcon,
    FileOpen as OpenIcon,
} from '@mui/icons-material';
import { ListMenu } from 'components';
import { BuildAction } from 'utils';
import { useMemo } from 'react';
import { 
    AddEndNodeAfterIcon,
    AddIncomingLinkIcon, 
    AddOutgoingLinkIcon, 
    AddRoutineListAfterIcon, 
    AddRoutineListBeforeIcon, 
    DeleteNodeIcon, 
    MoveNodeIcon, 
    UnlinkNodeIcon 
} from '@shared/icons';

const allOptionsMap: { [index in Exclude<BuildAction, BuildAction.AddSubroutine>]: [string, any] } = {
    [BuildAction.AddIncomingLink]: ['Add incoming link', AddIncomingLinkIcon],
    [BuildAction.AddOutgoingLink]: ['Add outgoing link', AddOutgoingLinkIcon],
    [BuildAction.AddListBeforeNode]: ['Add routine list before', AddRoutineListBeforeIcon],
    [BuildAction.AddListAfterNode]: ['Add routine list after', AddRoutineListAfterIcon],
    [BuildAction.AddEndAfterNode]: ['Add end node after', AddEndNodeAfterIcon],
    [BuildAction.DeleteNode]: ['Delete node', DeleteNodeIcon],
    [BuildAction.MoveNode]: ['Move node', MoveNodeIcon],
    [BuildAction.UnlinkNode]: ['Unlink node', UnlinkNodeIcon],
    [BuildAction.OpenSubroutine]: ['Open subroutine', OpenIcon],
    [BuildAction.EditSubroutine]: ['Edit subroutine', EditIcon],
    [BuildAction.DeleteSubroutine]: ['Delete subroutine', DeleteIcon],
}

const listOptions: ListMenuItemData<BuildAction>[] = Object.keys(allOptionsMap).map(o => ({
    label: allOptionsMap[o][0],
    value: o as BuildAction,
    Icon: allOptionsMap[o][1]
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
            title='Node Options'
            data={availableOptions}
            onSelect={handleSelect}
            onClose={handleClose}
            zIndex={zIndex}
        />
    )
}