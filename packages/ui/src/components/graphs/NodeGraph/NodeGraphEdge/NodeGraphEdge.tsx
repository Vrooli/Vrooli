import { makeStyles } from '@mui/styles';
import { Theme } from '@mui/material';
import { NodeGraphEdgeProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    root: {

    },
}));

// Displays a line between two nodes.
// If in editing mode, an "Add Node" button appears on the line. 
// This button always appears inbetween two node columns, to avoid collisions with nodes.
export const NodeGraphEdge = ({
    from,
    to,
    isEditable = true,
    scale = 1,
    onAdd,
}: NodeGraphEdgeProps) => {
    const classes = useStyles();

    return (
        <div className={classes.root}>TODO</div>
    )
}