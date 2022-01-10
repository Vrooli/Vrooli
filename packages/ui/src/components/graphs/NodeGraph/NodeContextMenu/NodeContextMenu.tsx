import { makeStyles } from '@mui/styles';
import { Popover, Theme } from '@mui/material';
import { NodeContextMenuProps } from '../types';
import { ListDialogItemData } from 'components/dialogs/types';
import {
    Delete as DeleteIcon,
    Edit as EditIcon,
    EditLocation as EditLocationIcon,
    MoveDown as MoveDownIcon,
    MoveUp as MoveUpIcon,
    SvgIconComponent,
} from '@mui/icons-material';
import { ListDialog } from 'components';

const useStyles = makeStyles((theme: Theme) => ({
    root: {

    },
}));

const listOptionsMap: {[x: string]: [string, SvgIconComponent]} = {
    'addBefore': ['Add node before', MoveDownIcon],
    'addAfter': ['Add node after', MoveUpIcon],
    'delete': ['Delete node', DeleteIcon],
    'edit': ['Edit node', EditIcon],
    'move': ['Move node', EditLocationIcon],
}

const listOptions: ListDialogItemData[] = Object.keys(listOptionsMap).map(o => ({ 
    label: listOptionsMap[o][0],
    value: o,
    Icon: listOptionsMap[o][1]
}));

// Custom context menu for nodes
export const NodeContextMenu = ({
    open,
    anchorEl,
    node,
    onClose,
    onAddBefore,
    onAddAfter,
    onEdit,
    onDelete,
    onMove,
}: NodeContextMenuProps) => {
    const classes = useStyles();
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
        <Popover
            className={classes.root}
            disableScrollLock={true}
            sx={{ pointerEvents: 'none' }}
            open={open}
            anchorEl={anchorEl}
            anchorOrigin={{
                vertical: 'top',
                horizontal: 'center',
            }}
            transformOrigin={{
                vertical: 'bottom',
                horizontal: 'center',
            }}
            onClose={onClose}
            disableRestoreFocus
        >
            <ListDialog
                title='Node Options'
                data={listOptions}
                onSelect={onMenuItemSelect}
                onClose={onClose}
            />
        </Popover>
    )
}