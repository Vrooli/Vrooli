import { makeStyles } from '@mui/styles';
import { Theme } from '@mui/material';
import { NodeGraphEdgeProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    root: {

    },
}));

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