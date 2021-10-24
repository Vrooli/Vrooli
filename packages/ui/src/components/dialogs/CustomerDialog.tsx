import { useState, useEffect, useCallback, useMemo } from 'react';
import { makeStyles } from '@material-ui/styles';
import {
    AppBar,
    Button,
    Dialog,
    Grid,
    IconButton,
    Theme,
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
import isEqual from 'lodash/isEqual';
import { AccountStatus } from '@local/shared';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { deleteCustomerMutation, updateCustomerMutation } from 'graphql/mutation';
import { PUBS } from 'utils';
import PubSub from 'pubsub-js';
import { useMutation } from '@apollo/client';
import { UpTransition } from 'components';
import { updateCustomer } from 'graphql/generated/updateCustomer';
import { deleteCustomer } from 'graphql/generated/deleteCustomer';
import { Customer } from 'types';

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
}));

// Associates account states with a dynamic action button
// curr_account_value: [curr_account_label, toggled_account_label, toggled_account_value, toggle_icon]
const statusToggle = {
    [AccountStatus.DELETED]: ['Deleted', 'Undelete', AccountStatus.UNLOCKED, (<AddCircleIcon />)],
    [AccountStatus.UNLOCKED]: ['Unlocked', 'Lock', AccountStatus.HARD_LOCKED, (<LockIcon />)],
    [AccountStatus.SOFT_LOCKED]: ['Soft Locked (password timeout)', 'Unlock', AccountStatus.UNLOCKED, (<LockOpenIcon />)],
    [AccountStatus.HARD_LOCKED]: ['Hard Locked', 'Unlock', AccountStatus.UNLOCKED, (<LockOpenIcon />)]
}

interface Props {
    customer: Customer | null;
    open?: boolean;
    onClose: () => any;
}

export const CustomerDialog = ({
    customer,
    open = true,
    onClose,
}: Props) => {
    const classes = useStyles();
    // Stores the modified customer data before updating
    const [currCustomer, setCurrCustomer] = useState<Customer | null>(customer);
    const [updateCustomerMut] = useMutation<updateCustomer>(updateCustomerMutation);
    const [deleteCustomerMut] = useMutation<deleteCustomer>(deleteCustomerMutation);

    useEffect(() => {
        setCurrCustomer(customer);
    }, [customer])

    const revert = () => setCurrCustomer(customer);

    const statusToggleData = useMemo(() => {
        if (!currCustomer?.status || !(currCustomer?.status in statusToggle)) return ['', '', null, null];
        return statusToggle[currCustomer.status];
    }, [currCustomer])

    // Locks/unlocks/undeletes a user
    const toggleLock = useCallback(() => {
        mutationWrapper({
            mutation: updateCustomerMut,
            data: { variables: { id: currCustomer?.id, status: statusToggleData[2] } },
            successMessage: () => 'Customer updated.',
            errorMessage: () => 'Failed to update customer.'
        })
    }, [currCustomer?.id, statusToggleData, updateCustomerMut])

    const deleteCustomer = useCallback(() => {
        mutationWrapper({
            mutation: deleteCustomerMut,
            data: { variables: { id: currCustomer?.id }},
            successMessage: () => 'Customer deleted.',
            onSuccess: onClose,
        })
    }, [currCustomer?.id, deleteCustomerMut, onClose])
    
    const confirmDelete = useCallback(() => {
        PubSub.publish(PUBS.AlertDialog, {
            message: `Are you sure you want to delete the account for ${currCustomer?.firstName} ${currCustomer?.lastName}?`,
            firstButtonText: 'Yes',
            firstButtonClicked: deleteCustomer,
            secondButtonText: 'No',
        });
    }, [currCustomer, deleteCustomer])

    const updateCustomer = useCallback(() => {
        mutationWrapper({
            mutation: updateCustomerMut,
            data: { variables: { ...currCustomer }},
            successMessage: () => 'Customer updated.',
        })
    }, [currCustomer, updateCustomerMut])

    let changes_made = !isEqual(customer, currCustomer);
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
        <Dialog fullScreen open={open} onClose={onClose} TransitionComponent={UpTransition}>
            <AppBar className={classes.appBar}>
                <Toolbar>
                    <IconButton edge="start" color="inherit" onClick={onClose} aria-label="close">
                        <CloseIcon />
                    </IconButton>
                    <Grid container spacing={0}>
                        <Grid className={classes.title} item xs={12}>
                            <Typography variant="h5">
                                {`${customer?.firstName} ${customer?.lastName}`}
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