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
import { BuildAction } from 'utils';

const listOptionsMap: {[index in Exclude<BuildAction, BuildAction.AddSubroutine | BuildAction.DeleteSubroutine | BuildAction.EditSubroutine | BuildAction.OpenSubroutine>]: [string, SvgIconComponent]}  = {
    [BuildAction.AddBeforeNode]: ['Add node before', MoveDownIcon],
    [BuildAction.AddAfterNode]: ['Add node after', MoveUpIcon],
    [BuildAction.DeleteNode]: ['Delete node', DeleteIcon],
    [BuildAction.MoveNode]: ['Move node', EditLocationIcon],
    [BuildAction.UnlinkNode]: ['Unlink node', EditIcon],
}

const listOptions: ListMenuItemData<BuildAction>[] = Object.keys(listOptionsMap).map(o => ({ 
    label: listOptionsMap[o][0],
    value: o as BuildAction,
    Icon: listOptionsMap[o][1]
}));

// Custom context menu for nodes
export const NodeContextMenu = ({
    id,
    anchorEl,
    handleClose,
    handleSelect,
    zIndex,
}: NodeContextMenuProps) => {
    return (
        <ListMenu
            id={id}
            anchorEl={anchorEl}
            title='Node Options'
            data={listOptions}
            onSelect={handleSelect}
            onClose={handleClose}
            zIndex={zIndex}
        />
    )
}