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
    resource,
    onClose,
    onAddBefore,
    onAddAfter,
    onEdit,
    onDelete,
    onMove,
}: ResourceListItemContextMenuProps) => {
    console.log('in resource list item context menu', { id, anchorEl, resource, onClose, onAddBefore, onAddAfter, onEdit, onDelete, onMove })
    const onMenuItemSelect = (value: string) => {
        if (!resource) return;
        switch (value) {
            case 'addBefore':
                onAddBefore(resource);
                break;
            case 'addAfter':
                onAddAfter(resource);
                break;
            case 'edit':
                onEdit(resource);
                break;
            case 'delete':
                onDelete(resource);
                break;
            case 'move':
                onMove(resource);
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