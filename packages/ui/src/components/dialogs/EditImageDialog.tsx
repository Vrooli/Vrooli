import { useCallback, useEffect, useState } from 'react';
import { makeStyles } from '@material-ui/styles';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField,
    Theme
} from '@material-ui/core';

const useStyles = makeStyles((theme: Theme) => ({
    title: {
        background: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
    },
    button: {
        color: theme.palette.primary.main,
    }
}));

interface Props {
    open: boolean;
    data: any;
    onClose: () => any;
    onSave: (data: any) => any;
}

export const EditImageDialog = ({
    open,
    data,
    onClose,
    onSave,
}: Props) => {
    const classes = useStyles();
    const [alt, setAlt] = useState('')
    const [description, setDescription] = useState('')

    useEffect(() => {
        setAlt(data?.alt ?? '');
        setDescription(data?.description ?? '');
    }, [data])

    const save = useCallback(() => onSave({ alt: alt, description: description }), [alt, description, onSave]);

    return (
        <Dialog open={open} onClose={onClose} aria-labelledby="form-dialog-title">
            <DialogTitle className={classes.title} id="form-dialog-title">Edit Image Data</DialogTitle>
            <DialogContent>
                <TextField
                    fullWidth
                    variant="filled"
                    label="Alt"
                    value={alt}
                    onChange={e => setAlt(e.target.value)}
                />
                <TextField
                    fullWidth
                    variant="filled"
                    label="Description"
                    value={description}
                    onChange={e => setDescription(e.target.value)}
                />
            </DialogContent>
            <DialogActions>
                <Button className={classes.button} onClick={onClose} variant="text">
                    Cancel
                </Button>
                <Button className={classes.button} onClick={save} variant="text">
                    Save
                </Button>
            </DialogActions>
        </Dialog>
    );
}