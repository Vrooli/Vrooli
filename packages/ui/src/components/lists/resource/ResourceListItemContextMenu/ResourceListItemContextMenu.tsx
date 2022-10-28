import { ResourceListItemContextMenuProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';
import { ListMenu, SnackSeverity } from 'components';
import { CopyIcon, DeleteIcon, EditIcon, MoveLeftIcon, MoveLeftRightIcon, MoveRightIcon, ShareIcon, SvgComponent } from '@shared/icons';
import { getTranslation, PubSub } from 'utils';

export enum ResourceContextMenuOption {
    AddBefore = 'AddBefore',
    AddAfter = 'AddAfter',
    Copy = 'Copy',
    Delete = 'Delete',
    Edit = 'Edit',
    Move = 'Move',
    Share = 'Share',
}

const listOptionsMap: { [key in ResourceContextMenuOption]: [string, SvgComponent] } = {
    [ResourceContextMenuOption.AddBefore]: ['Add resource before', MoveLeftIcon],
    [ResourceContextMenuOption.AddAfter]: ['Add resource after', MoveRightIcon],
    [ResourceContextMenuOption.Copy]: ['Copy link', CopyIcon],
    [ResourceContextMenuOption.Edit]: ['Edit', EditIcon],
    [ResourceContextMenuOption.Delete]: ['Delete', DeleteIcon],
    [ResourceContextMenuOption.Move]: ['Move', MoveLeftRightIcon],
    [ResourceContextMenuOption.Share]: ['Share', ShareIcon],
}

const listOptions: ListMenuItemData<ResourceContextMenuOption>[] = Object.keys(listOptionsMap).map((o) => ({
    label: listOptionsMap[o][0],
    value: o as ResourceContextMenuOption,
    Icon: listOptionsMap[o][1]
}));

// Custom context menu for nodes
export const ResourceListItemContextMenu = ({
    canEdit,
    id,
    anchorEl,
    index,
    onClose,
    onAddBefore,
    onAddAfter,
    onEdit,
    onDelete,
    onMove,
    resource,
    zIndex,
}: ResourceListItemContextMenuProps) => {

    const onMenuItemSelect = (value: ResourceContextMenuOption) => {
        if (index === null || index < 0) return;
        switch (value) {
            case ResourceContextMenuOption.AddBefore:
                onAddBefore(index);
                break;
            case ResourceContextMenuOption.AddAfter:
                onAddAfter(index);
                break;
            case ResourceContextMenuOption.Copy:
                navigator.clipboard.writeText(resource?.link ?? '');
                PubSub.get().publishSnack({ message: 'Copied.', severity: SnackSeverity.Success });
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
            case ResourceContextMenuOption.Share:
                if (!resource?.link) return;
                const { title, description } = getTranslation(resource, []); //TODO languages
                navigator.share({
                    title: title ?? undefined,
                    text: description ?? undefined,
                    url: resource?.link
                });
                break;
        }
        onClose();
    }

    const listOptionsFiltered = canEdit ? listOptions : listOptions.filter(o => o.value === ResourceContextMenuOption.Share);

    return (
        <ListMenu
            id={id}
            anchorEl={anchorEl}
            title='Resource Options'
            data={listOptionsFiltered}
            onSelect={onMenuItemSelect}
            onClose={onClose}
            zIndex={zIndex}
        />
    )
}