import {
    Dialog,
    DialogTitle,
    IconButton,
    List,
    ListItem,
    ListItemButton,
    ListItemIcon,
    ListItemText,
    Theme
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import { HelpButton } from 'components';
import { useMemo } from 'react';
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

    const items = useMemo(() => data?.map(({ label, value, Icon, helpData }, index) => {
        console.log('in list dialog item', { label, value, Icon, helpData })
        const itemText = <ListItemText primary={label} />;
        const itemIcon = Icon ? (
                <ListItemIcon>
                    <Icon />
                </ListItemIcon>
        ) : null;
        const helpIcon = helpData ? (
            <IconButton edge="end">
                <HelpButton {...helpData} />
            </IconButton>
        ) : null;
        return (
            <ListItem button onClick={() => { onSelect(value); onClose(); }} key={index}>
                {itemIcon}
                {itemText}
                {helpIcon}
            </ListItem>
        )
    }), [data, onClose, onSelect])

    return (
        <Dialog
            PaperProps={{
                className: classes.root,
            }}
            onClose={onClose}
            disableScrollLock={true}
            aria-labelledby="simple-dialog-title"
            open={open}
            {...props}>
            <DialogTitle className={classes.title} id="simple-dialog-title">{title}</DialogTitle>
            <List>
                {items}
            </List>
        </Dialog>
    );
}

export { ListDialog };