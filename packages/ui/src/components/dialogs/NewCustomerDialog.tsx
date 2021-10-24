import { makeStyles } from '@material-ui/styles';
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
} from '@material-ui/core';
import {
    AddCircle as AddCircleIcon,
    Cancel as CancelIcon,
    Close as CloseIcon,
} from '@material-ui/icons';
import { addCustomerSchema } from '@local/shared';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { addCustomerMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { useMutation } from '@apollo/client';
import { UpTransition } from 'components';
import { addCustomer } from 'graphql/generated/addCustomer';
import { useCallback } from 'react';

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
    phoneInput: {
        width: '100%',
    },
}));

interface Props {
    open?: boolean;
    onClose: () => any;
}

export const NewCustomerDialog = ({
    open = true,
    onClose,
}: Props) => {
    const classes = useStyles();
    // Stores the modified customer data before updating
    const [addCustomer] = useMutation<addCustomer>(addCustomerMutation);

    const formik = useFormik({
        initialValues: {
            username: '',
            email: '',
        },
        validationSchema: addCustomerSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: addCustomer,
                data: { variables: { input: {
                    username: values.username,
                    emails: [{ emailAddress: values.email }],
                } } },
                onSuccess: () => onClose(),
                successMessage: () => 'Customer created.'
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
        <Dialog fullScreen open={open} onClose={onClose} TransitionComponent={UpTransition}>
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
                                id="username"
                                name="username"
                                autoComplete="username"
                                label="Username"
                                value={formik.values.username}
                                onChange={formik.handleChange}
                                error={formik.touched.username && Boolean(formik.errors.username)}
                                helperText={formik.touched.username && formik.errors.username}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                id="email"
                                name="email"
                                autoComplete="email"
                                label="Email Address"
                                value={formik.values.email}
                                onChange={formik.handleChange}
                                error={formik.touched.email && Boolean(formik.errors.email)}
                                helperText={formik.touched.email && formik.errors.email}
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