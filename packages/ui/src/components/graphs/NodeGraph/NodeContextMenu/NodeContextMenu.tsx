import { NodeContextMenuProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import {
    Delete as DeleteIcon,
    Edit as EditIcon,
    EditLocation as EditLocationIcon,
    MoveDown as MoveDownIcon,
    MoveUp as MoveUpIcon,
    SvgIconComponent,
} from '@mui/icons-material';
import { ListMenu } from 'components';

export enum NodeContextMenuOptions {
    AddBefore = 'add-before',
    AddAfter = 'add-after',
    Delete = 'delete',
    Edit = 'edit',
    Move = 'move',
    Unlink = 'unlink',
}

const listOptionsMap: {[index in NodeContextMenuOptions]: [string, SvgIconComponent]}  = {
    [NodeContextMenuOptions.AddBefore]: ['Add node before', MoveDownIcon],
    [NodeContextMenuOptions.AddAfter]: ['Add node after', MoveUpIcon],
    [NodeContextMenuOptions.Delete]: ['Delete node', DeleteIcon],
    [NodeContextMenuOptions.Edit]: ['Edit node', EditIcon],
    [NodeContextMenuOptions.Move]: ['Move node', EditLocationIcon],
    [NodeContextMenuOptions.Unlink]: ['Unlink node', EditIcon],
}

const listOptions: ListMenuItemData<NodeContextMenuOptions>[] = Object.keys(listOptionsMap).map(o => ({ 
    label: listOptionsMap[o][0],
    value: o as NodeContextMenuOptions,
    Icon: listOptionsMap[o][1]
}));

// Custom context menu for nodes
export const NodeContextMenu = ({
    id,
    anchorEl,
    handleClose,
    handleContextItemSelect,
}: NodeContextMenuProps) => {
    return (
        <ListMenu
            id={id}
            anchorEl={anchorEl}
            title='Node Options'
            data={listOptions}
            onSelect={handleContextItemSelect}
            onClose={handleClose}
        />
    )
}