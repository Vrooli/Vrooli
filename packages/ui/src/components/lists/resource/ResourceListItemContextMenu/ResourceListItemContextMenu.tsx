import { ResourceListItemContextMenuProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import {
    Delete as DeleteIcon,
    MoveDown as MoveDownIcon,
    MoveUp as MoveUpIcon,
    OpenWith as MoveIcon,
} from '@mui/icons-material';
import { ListMenu } from 'components';
import { EditIcon } from '@shared/icons';

export enum ResourceContextMenuOption {
    AddBefore = 'AddBefore',
    AddAfter = 'AddAfter',
    Delete = 'Delete',
    Edit = 'Edit',
    Move = 'Move',
}

const listOptionsMap: { [key in ResourceContextMenuOption]: [string, any] } = {
    [ResourceContextMenuOption.AddBefore]: ['Add resource before', MoveDownIcon],
    [ResourceContextMenuOption.AddAfter]: ['Add resource after', MoveUpIcon],
    [ResourceContextMenuOption.Edit]: ['Edit resource', EditIcon],
    [ResourceContextMenuOption.Delete]: ['Delete resource', DeleteIcon],
    [ResourceContextMenuOption.Move]: ['Move resource', MoveIcon],
}

const listOptions: ListMenuItemData<ResourceContextMenuOption>[] = Object.keys(listOptionsMap).map((o) => ({
    label: listOptionsMap[o][0],
    value: o as ResourceContextMenuOption,
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
    zIndex,
}: ResourceListItemContextMenuProps) => {
    const onMenuItemSelect = (value: ResourceContextMenuOption) => {
        console.log('onMenuItemSelect', index);
        if (index === null || index < 0) return;
        switch (value) {
            case ResourceContextMenuOption.AddBefore:
                onAddBefore(index);
                break;
            case ResourceContextMenuOption.AddAfter:
                onAddAfter(index);
                break;
            case ResourceContextMenuOption.Delete:
                onDelete(index);
                break;
            case ResourceContextMenuOption.Edit:
                onEdit(index);
                break;
            case ResourceContextMenuOption.Move:
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
            zIndex={zIndex}
        />
    )
}