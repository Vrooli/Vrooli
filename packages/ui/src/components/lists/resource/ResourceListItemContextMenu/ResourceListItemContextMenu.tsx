import { ResourceListItemContextMenuProps } from '../types';
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

const listOptionsMap: { [x: string]: [string, SvgIconComponent] } = {
    'addBefore': ['Add resource before', MoveDownIcon],
    'addAfter': ['Add resource after', MoveUpIcon],
    'delete': ['Delete resource', DeleteIcon],
    'edit': ['Edit resource', EditIcon],
    'move': ['Move resource', EditLocationIcon],
}

const listOptions: ListMenuItemData<string>[] = Object.keys(listOptionsMap).map(o => ({
    label: listOptionsMap[o][0],
    value: o,
    Icon: listOptionsMap[o][1]
}));

// Custom context menu for nodes
export const ResourceListItemContextMenu = ({
    id,
    anchorEl,
    index,
    onClose,
    onAddBefore,
    onAddAfter,
    onEdit,
    onDelete,
    onMove,
}: ResourceListItemContextMenuProps) => {
    const onMenuItemSelect = (value: string) => {
        if (!index) return;
        switch (value) {
            case 'addBefore':
                onAddBefore(index);
                break;
            case 'addAfter':
                onAddAfter(index);
                break;
            case 'edit':
                onEdit(index);
                break;
            case 'delete':
                onDelete(index);
                break;
            case 'move':
                onMove(index);
                break;
        }
        onClose();
    }

    return (
        <ListMenu
            id={id}
            anchorEl={anchorEl}
            title='Resource Options'
            data={listOptions}
            onSelect={onMenuItemSelect}
            onClose={onClose}
        />
    )
}