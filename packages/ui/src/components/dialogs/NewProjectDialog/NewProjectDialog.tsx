import { makeStyles } from '@mui/styles';
import {
    AppBar,
    Button,
    Dialog,
    Grid,
    IconButton,
    TextField,
    Theme,
    Toolbar,
    Typography,
} from '@mui/material';
import {
    AddCircle as AddCircleIcon,
    Cancel as CancelIcon,
    Close as CloseIcon,
} from '@mui/icons-material';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { projectAddMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { useMutation } from '@apollo/client';
import { UpTransition } from 'components';
import { useCallback } from 'react';
import { addProjectSchema } from '@local/shared';
import { NewProjectDialogProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    appBar: {
        position: 'relative',
    },
    title: {
        textAlign: 'center',
    },
    optionsContainer: {
        padding: theme.spacing(2),
        background: theme.palette.primary.main,
    },
    container: {
        background: theme.palette.background.default,
        flex: 'auto',
        padding: theme.spacing(1),
        paddingBottom: '15vh',
    },
    bottom: {
        background: theme.palette.primary.main,
        position: 'fixed',
        bottom: '0',
        width: '-webkit-fill-available',
        zIndex: 1,
    },
    form: {
        width: '100%',
        marginTop: theme.spacing(3),
    },
}));

export const NewProjectDialog = ({
    open = true,
    onClose,
}: NewProjectDialogProps) => {
    const classes = useStyles();
    // Stores the modified customer data before updating
    const [projectAdd] = useMutation<any>(projectAddMutation);

    const formik = useFormik({
        initialValues: {
            name: '',
            description: '',
        },
        validationSchema: addProjectSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: projectAdd,
                input: {
                    name: values.name,
                    description: values.description,
                },
                onSuccess: () => onClose(),
                successMessage: () => 'Project created.'
            })
        },
    });

    const submit = useCallback(() => formik.handleSubmit(), [formik]);

    let options = (
        <Grid className={classes.optionsContainer} container spacing={2}>
            <Grid item xs={12} sm={6}>
                <Button
                    fullWidth
                    startIcon={<AddCircleIcon />}
                    onClick={submit}
                >Create</Button>
            </Grid>
            <Grid item xs={12} sm={6}>
                <Button
                    fullWidth
                    startIcon={<CancelIcon />}
                    onClick={onClose}
                >Cancel</Button>
            </Grid>
        </Grid>
    );

    return (
        <Dialog fullScreen disableScrollLock={true} open={open} onClose={onClose} TransitionComponent={UpTransition}>
            <AppBar className={classes.appBar}>
                <Toolbar>
                    <IconButton edge="start" color="inherit" onClick={onClose} aria-label="close">
                        <CloseIcon />
                    </IconButton>
                    <Grid container spacing={0}>
                        <Grid className={classes.title} item xs={12}>
                            <Typography variant="h5">
                                Create New Customer
                            </Typography>
                        </Grid>
                    </Grid>
                </Toolbar>
            </AppBar>
            <div className={classes.container}>
                <form className={classes.form}>
                    <Grid container spacing={2}>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                autoFocus
                                id="name"
                                name="name"
                                label="Project Name"
                                value={formik.values.name}
                                onChange={formik.handleChange}
                                error={formik.touched.name && Boolean(formik.errors.name)}
                                helperText={formik.touched.name && formik.errors.name}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                autoFocus
                                id="description"
                                name="description"
                                label="Description"
                                value={formik.values.description}
                                onChange={formik.handleChange}
                                error={formik.touched.description && Boolean(formik.errors.description)}
                                helperText={formik.touched.description && formik.errors.description}
                            />
                        </Grid>
                    </Grid>
                </form>
                <div className={classes.bottom}>
                    {options}
                </div>
            </div>
        </Dialog>
    );
}