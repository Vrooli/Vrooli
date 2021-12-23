import { NODE_TYPES } from '@local/shared';
import {
    Dialog,
    DialogTitle,
    List,
    ListItem,
    ListItemText,
    Theme
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { ListDialogProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        background: theme.palette.background.paper,
        minWidth: 'min(90vw, 300px)',
    },
    title: {
        textAlign: 'center',
        background: theme.palette.primary.dark,
        color: theme.palette.primary.contrastText,
    },
}));

const ListDialog = ({
    open = true,
    onSelect,
    onClose,
    title = 'Select Item',
    data,
    ...props
}: ListDialogProps) => {
    const classes = useStyles();

    return (
        <Dialog
            PaperProps={{
                className: classes.root,
            }}
            onClose={onClose}
            aria-labelledby="simple-dialog-title"
            open={open}
            {...props}>
            <DialogTitle className={classes.title} id="simple-dialog-title">{title}</DialogTitle>
            <List>
                {data?.map(({ label, value }, index) => (
                    <ListItem button onClick={() => {onSelect(value); onClose();}} key={index}>
                        <ListItemText primary={label} />
                    </ListItem>
                ))}
            </List>
        </Dialog>
    );
}

export { ListDialog };