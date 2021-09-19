import React, { useState, useEffect, useCallback, useMemo } from 'react';
import PropTypes from 'prop-types';
import { makeStyles } from '@material-ui/styles';
import {
    AppBar,
    Button,
    Dialog,
    Grid,
    IconButton,
    Slide,
    Toolbar,
    Typography,
} from '@material-ui/core';
import {
    AddCircle as AddCircleIcon,
    Close as CloseIcon,
    Delete as DeleteIcon,
    Lock as LockIcon,
    LockOpen as LockOpenIcon,
    Restore as RestoreIcon,
    Update as UpdateIcon
} from '@material-ui/icons';
import _ from 'lodash';
import { ACCOUNT_STATUS } from '@local/shared';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { deleteCustomerMutation, updateCustomerMutation } from 'graphql/mutation';
import { PUBS, PubSub } from 'utils';

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
}));

const Transition = React.forwardRef(function Transition(props, ref) {
    return <Slide direction="up" ref={ref} {...props} />;
});

// Associates account states with a dynamic action button
// curr_account_value: [curr_account_label, toggled_account_label, toggled_account_value, toggle_icon]
const statusToggle = {
    [ACCOUNT_STATUS.Deleted]: ['Deleted', 'Undelete', ACCOUNT_STATUS.Unlocked, (<AddCircleIcon />)],
    [ACCOUNT_STATUS.Unlocked]: ['Unlocked', 'Lock', ACCOUNT_STATUS.HardLock, (<LockIcon />)],
    [ACCOUNT_STATUS.SoftLock]: ['Soft Locked (password timeout)', 'Unlock', ACCOUNT_STATUS.Unlocked, (<LockOpenIcon />)],
    [ACCOUNT_STATUS.HardLock]: ['Hard Locked', 'Unlock', ACCOUNT_STATUS.Unlocked, (<LockOpenIcon />)]
}

function CustomerDialog({
    customer,
    open = true,
    onClose,
}) {
    const classes = useStyles();
    // Stores the modified customer data before updating
    const [currCustomer, setCurrCustomer] = useState(customer);

    useEffect(() => {
        setCurrCustomer(customer);
    }, [customer])

    const revert = () => setCurrCustomer(customer);

    const statusToggleData = useMemo(() => {
        if (!(currCustomer?.status in statusToggle)) return ['', '', null, null];
        return statusToggle[currCustomer.status];
    }, [currCustomer])

    // Locks/unlocks/undeletes a user
    const toggleLock = useCallback(() => {
        mutationWrapper({
            mutation: updateCustomerMutation,
            data: { variables: { id: currCustomer.id, status: statusToggleData[2] } },
            successMessage: () => 'Customer updated.',
            errorMessage: () => 'Failed to update customer.'
        })
    }, [currCustomer, statusToggleData])

    const deleteCustomer = useCallback(() => {
        mutationWrapper({
            mutation: deleteCustomerMutation,
            data: { variables: { id: currCustomer?.id }},
            successMessage: () => 'Customer deleted.',
            onSuccess: onClose,
        })
    }, [currCustomer?.id, onClose])
    
    const confirmDelete = useCallback(() => {
        PubSub.publish(PUBS.AlertDialog, {
            message: `Are you sure you want to delete the account for ${currCustomer.firstName} ${currCustomer.lastName}?`,
            firstButtonText: 'Yes',
            firstButtonClicked: deleteCustomer,
            secondButtonText: 'No',
        });
    }, [currCustomer, deleteCustomer])

    const updateCustomer = useCallback(() => {
        mutationWrapper({
            mutation: updateCustomerMutation,
            data: { variables: { ...currCustomer }},
            successMessage: () => 'Customer updated.',
        })
    }, [currCustomer])

    let changes_made = !_.isEqual(customer, currCustomer);
    let options = (
        <Grid className={classes.optionsContainer} container spacing={2}>
            <Grid item xs={12} sm={6} md={3}>
                <Button
                    fullWidth
                    disabled={!changes_made}
                    startIcon={<RestoreIcon />}
                    onClick={revert}
                >Revert</Button>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <Button
                    fullWidth
                    disabled={!customer?.id}
                    startIcon={statusToggleData[3]}
                    onClick={toggleLock}
                >{statusToggleData[1]}</Button>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <Button
                    fullWidth
                    disabled={!customer?.id}
                    startIcon={<DeleteIcon />}
                    onClick={confirmDelete}
                >Delete</Button>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <Button
                    fullWidth
                    disabled={!changes_made}
                    startIcon={<UpdateIcon />}
                    onClick={updateCustomer}
                >Update</Button>
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
                                {customer?.fullName}
                            </Typography>
                            <Typography variant="h6">
                                {customer?.business?.name}
                            </Typography>
                        </Grid>
                    </Grid>
                </Toolbar>
            </AppBar>
            <div className={classes.container}>
    
                <div className={classes.bottom}>
                    {options}
                </div>
            </div>
        </Dialog>
    );
}

CustomerDialog.propTypes = {
    product: PropTypes.object,
    open: PropTypes.bool,
    onClose: PropTypes.func.isRequired,
}

export { CustomerDialog };