import { makeStyles } from '@material-ui/styles';
import { IconButton, Theme, Tooltip } from '@material-ui/core';
import { useMemo, useState } from 'react';
import { AddNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';
import { ListDialog } from 'components';
import { Add as AddIcon } from '@material-ui/icons';
import { NODE_TYPES } from '@local/shared';

const componentStyles = (theme: Theme) => ({
    root: {
        position: 'relative',
        display: 'block',
        width: '10vw',
        height: '10vw',
        backgroundColor: '#6daf72',
        color: 'white',
        borderRadius: '100%',
        boxShadow: '0px 0px 12px gray',
    },
    icon: {
        width: '80%',
        height: '80%',
    }
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

const optionsMap = {
    [NODE_TYPES.Combine]: 'Combine',
    [NODE_TYPES.Decision]: 'Decision',
    [NODE_TYPES.End]: 'End',
    [NODE_TYPES.Loop]: 'Loop',
    [NODE_TYPES.RoutineList]: 'Routine List',
    [NODE_TYPES.Redirect]: 'Redirect',
    [NODE_TYPES.Start]: 'Start',
}

export const AddNode = ({
    options = Object.values(NODE_TYPES),
    onAdd,
}: AddNodeProps) => {
    const classes = useStyles();
    const [dialogOpen, setDialogOpen] = useState(false);
    const openDialog = () => setDialogOpen(true);
    const closeDialog = () => setDialogOpen(false);
    const listOptions = useMemo(() => options.map(o => ({ label: optionsMap[o], value: o })), [options]);
    const dialog = useMemo(() => dialogOpen ? (
        <ListDialog
            title='Add Step'
            data={listOptions}
            onSelect={onAdd}
            onClose={closeDialog} />
    ) : null, [dialogOpen, listOptions, onAdd])

    return (
        <div>
            {dialog}
            <Tooltip placement={'top'} title='Insert step'>
                <IconButton className={classes.root} onClick={openDialog}>
                    <AddIcon className={classes.icon} />
                </IconButton>
            </Tooltip>
        </div>
    )
}