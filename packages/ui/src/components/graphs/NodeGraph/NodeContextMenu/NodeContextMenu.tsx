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

const listOptionsMap: {[x: string]: [string, SvgIconComponent]} = {
    'addBefore': ['Add node before', MoveDownIcon],
    'addAfter': ['Add node after', MoveUpIcon],
    'delete': ['Delete node', DeleteIcon],
    'edit': ['Edit node', EditIcon],
    'move': ['Move node', EditLocationIcon],
}

const listOptions: ListMenuItemData<string>[] = Object.keys(listOptionsMap).map(o => ({ 
    label: listOptionsMap[o][0],
    value: o,
    Icon: listOptionsMap[o][1]
}));

// Custom context menu for nodes
export const NodeContextMenu = ({
    id,
    anchorEl,
    node,
    onClose,
    onAddBefore,
    onAddAfter,
    onEdit,
    onDelete,
    onMove,
}: NodeContextMenuProps) => {
    const onMenuItemSelect = (value: string) => {
        switch (value) {
            case 'addBefore':
                onAddBefore(node);
                break;
            case 'addAfter':
                onAddAfter(node);
                break;
            case 'edit':
                onEdit(node);
                break;
            case 'delete':
                onDelete(node);
                break;
            case 'move':
                onMove(node);
                break;
        }
        onClose();
    }

    return (
        <ListMenu
            id={id}
            anchorEl={anchorEl}
            title='Node Options'
            data={listOptions}
            onSelect={onMenuItemSelect}
            onClose={onClose}
        />
    )
}