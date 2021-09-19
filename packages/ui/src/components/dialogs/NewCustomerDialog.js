import React from 'react';
import PropTypes from 'prop-types';
import { makeStyles } from '@material-ui/styles';
import {
    AppBar,
    Autocomplete,
    Button,
    Dialog,
    Grid,
    IconButton,
    Slide,
    TextField,
    Toolbar,
    Typography,
} from '@material-ui/core';
import {
    AddCircle as AddCircleIcon,
    Cancel as CancelIcon,
    Close as CloseIcon,
} from '@material-ui/icons';
import _ from 'lodash';
import { DEFAULT_PRONOUNS, addCustomerSchema } from '@local/shared';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { addCustomerMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { useMutation } from '@apollo/client';

const useStyles = makeStyles((theme) => ({
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

const Transition = React.forwardRef(function Transition(props, ref) {
    return <Slide direction="up" ref={ref} {...props} />;
});

function NewCustomerDialog({
    open = true,
    onClose,
}) {
    const classes = useStyles();
    // Stores the modified customer data before updating
    const [addCustomer] = useMutation(addCustomerMutation);

    const formik = useFormik({
        initialValues: {
            firstName: '',
            lastName: '',
            pronouns: '',
            business: '',
            email: '',
            phone: '',
        },
        validationSchema: addCustomerSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: addCustomer,
                data: { variables: { input: {
                    firstName: values.firstName,
                    lastName: values.lastName,
                    pronouns: values.pronouns,
                    business: {name: values.business},
                    emails: [{ emailAddress: values.email }],
                    phones: [{ number: values.phone }],
                } } },
                onSuccess: () => onClose(),
                successMessage: () => 'Customer created.'
            })
        },
    });


    let options = (
        <Grid className={classes.optionsContainer} container spacing={2}>
            <Grid item xs={12} sm={6}>
                <Button
                    fullWidth
                    startIcon={<AddCircleIcon />}
                    onClick={() => formik.handleSubmit()}
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
        <Dialog fullScreen open={open} onClose={onClose} TransitionComponent={Transition}>
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
                        <Grid item xs={12} sm={6}>
                            <TextField
                                fullWidth
                                autoFocus
                                id="firstName"
                                name="firstName"
                                autoComplete="fname"
                                label="First Name"
                                value={formik.values.firstName}
                                onChange={formik.handleChange}
                                error={formik.touched.firstName && Boolean(formik.errors.firstName)}
                                helperText={formik.touched.firstName && formik.errors.firstName}
                            />
                        </Grid>
                        <Grid item xs={12} sm={6}>
                            <TextField
                                fullWidth
                                id="lastName"
                                name="lastName"
                                autoComplete="lname"
                                label="Last Name"
                                value={formik.values.lastName}
                                onChange={formik.handleChange}
                                error={formik.touched.lastName && Boolean(formik.errors.lastName)}
                                helperText={formik.touched.lastName && formik.errors.lastName}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <Autocomplete
                                fullWidth
                                freeSolo
                                id="pronouns"
                                name="pronouns"
                                options={DEFAULT_PRONOUNS}
                                value={formik.values.pronouns}
                                onChange={(_, value) => formik.setFieldValue('pronouns', value)}
                                renderInput={(params) => (
                                    <TextField
                                        {...params}
                                        label="Pronouns"
                                        value={formik.values.pronouns}
                                        onChange={formik.handleChange}
                                        error={formik.touched.pronouns && Boolean(formik.errors.pronouns)}
                                        helperText={formik.touched.pronouns && formik.errors.pronouns}
                                    />
                                )}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                id="business"
                                name="business"
                                autoComplete="business"
                                label="Business"
                                value={formik.values.business}
                                onChange={formik.handleChange}
                                error={formik.touched.business && Boolean(formik.errors.business)}
                                helperText={formik.touched.business && formik.errors.business}
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
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                id="phone"
                                name="phone"
                                autoComplete="tel"
                                label="Phone Number"
                                value={formik.values.phone}
                                onChange={formik.handleChange}
                                error={formik.touched.phone && Boolean(formik.errors.phone)}
                                helperText={formik.touched.phone && formik.errors.phone}
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

NewCustomerDialog.propTypes = {
    product: PropTypes.object,
    open: PropTypes.bool,
    onClose: PropTypes.func.isRequired,
}

export { NewCustomerDialog };