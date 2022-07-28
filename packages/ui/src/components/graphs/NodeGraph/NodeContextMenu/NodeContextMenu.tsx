import { NodeContextMenuProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import {
    Delete as DeleteIcon,
    Edit as EditIcon,
    EditLocation as EditLocationIcon,
    FileOpen as OpenIcon,
    MoveDown as MoveDownIcon,
    MoveUp as MoveUpIcon,
    SvgIconComponent,
} from '@mui/icons-material';
import { ListMenu } from 'components';
import { BuildAction } from 'utils';
import { useMemo } from 'react';

const allOptionsMap: { [index in Exclude<BuildAction, BuildAction.AddSubroutine>]: [string, SvgIconComponent] } = {
    [BuildAction.AddListBeforeNode]: ['Add routine list before', MoveDownIcon],
    [BuildAction.AddListAfterNode]: ['Add routine list after', MoveUpIcon],
    [BuildAction.AddEndAfterNode]: ['Add end node after', MoveUpIcon],
    [BuildAction.DeleteNode]: ['Delete node', DeleteIcon],
    [BuildAction.MoveNode]: ['Move node', EditLocationIcon],
    [BuildAction.UnlinkNode]: ['Unlink node', EditIcon],
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