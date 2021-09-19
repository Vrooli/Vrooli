import React, { useCallback, useMemo, useState } from 'react';
import PropTypes from 'prop-types'
import {
    Card,
    CardActions,
    CardContent,
    IconButton,
    Tooltip,
    Typography
} from '@material-ui/core';
import {
    Delete as DeleteIcon,
    DeleteForever as DeleteForeverIcon,
    Edit as EditIcon,
    Email as EmailIcon,
    Lock as LockIcon,
    LockOpen as LockOpenIcon,
    Phone as PhoneIcon,
} from '@material-ui/icons';
import { makeStyles } from '@material-ui/styles';
import { changeCustomerStatusMutation, deleteCustomerMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { ACCOUNT_STATUS } from '@local/shared';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useTheme } from '@emotion/react';
import { emailLink, mapIfExists, phoneLink, PUBS, PubSub, showPhone } from 'utils';
import { ListDialog } from 'components/dialogs';
import { cardStyles } from './styles';

const useStyles = makeStyles(cardStyles);

function CustomerCard({
    customer,
    status = ACCOUNT_STATUS.Deleted,
    onEdit,
}) {
    const classes = useStyles();
    const theme = useTheme();
    const [changeCustomerStatus] = useMutation(changeCustomerStatusMutation);
    const [deleteCustomer] = useMutation(deleteCustomerMutation);
    const [emailDialogOpen, setEmailDialogOpen] = useState(false);
    const [phoneDialogOpen, setPhoneDialogOpen] = useState(false);

    const callPhone = (phoneLink) => {
        setPhoneDialogOpen(false);
        if (phoneLink) window.location.href = phoneLink;
    }

    const sendEmail = (emailLink) => {
        setEmailDialogOpen(false);
        if (emailLink) window.open(emailLink, '_blank', 'noopener,noreferrer')
    }

    const status_map = useMemo(() => ({
        [ACCOUNT_STATUS.Deleted]: 'Deleted',
        [ACCOUNT_STATUS.Unlocked]: 'Unlocked',
        [ACCOUNT_STATUS.WaitingApproval]: 'Waiting Approval',
        [ACCOUNT_STATUS.SoftLock]: 'Soft Locked',
        [ACCOUNT_STATUS.HardLock]: 'Hard Locked',
    }), [theme])

    const edit = () => {
        onEdit(customer);
    }

    const modifyCustomer = useCallback((status, message) => {
        mutationWrapper({
            mutation: changeCustomerStatus,
            data: { variables: { id: customer?.id, status: status } },
            successCondition: (response) => response.changeCustomerStatus !== null,
            successMessage: () => message
        })
    }, [changeCustomerStatus, customer])

    const confirmPermanentDelete = useCallback(() => {
        PubSub.publish(PUBS.AlertDialog, {
            message: `Are you sure you want to permanently delete the account for ${customer.firstName} ${customer.lastName}? THIS ACTION CANNOT BE UNDONE!`,
            firstButtonText: 'Yes',
            firstButtonClicked: () => {
                mutationWrapper({
                    mutation: deleteCustomer,
                    data: { variables: { id: customer?.id } },
                    successCondition: (response) => response.deleteCustomer !== null,
                    successMessage: () => 'Customer permanently deleted.'
                })
            },
            secondButtonText: 'No',
        });
    }, [customer, deleteCustomer])

    const confirmDelete = useCallback(() => {
        PubSub.publish(PUBS.AlertDialog, {
            message: `Are you sure you want to delete the account for ${customer.firstName} ${customer.lastName}?`,
            firstButtonText: 'Yes',
            firstButtonClicked: () => modifyCustomer(ACCOUNT_STATUS.Deleted, 'Customer deleted.'),
            secondButtonText: 'No',
        });
    }, [customer, modifyCustomer])

    let edit_action = [edit, <EditIcon className={classes.icon} />, 'Edit customer']
    let unlock_action = [() => modifyCustomer(ACCOUNT_STATUS.Unlocked, 'Customer account unlocked.'), <LockOpenIcon className={classes.icon} />, 'Unlock customer account'];
    let lock_action = [() => modifyCustomer(ACCOUNT_STATUS.HardLock, 'Customer account locked.'), <LockIcon className={classes.icon} />, 'Lock customer account'];
    let undelete_action = [() => modifyCustomer(ACCOUNT_STATUS.Unlocked, 'Customer account restored.'), <LockOpenIcon className={classes.icon} />, 'Restore deleted account'];
    let delete_action = [confirmDelete, <DeleteIcon className={classes.icon} />, 'Delete user'];
    let permanent_delete_action = [confirmPermanentDelete, <DeleteForeverIcon className={classes.icon} />, 'Permanently delete user']

    let actions = [edit_action];
    // Actions for customer accounts
    if (!Array.isArray(customer?.roles) || !customer.roles.some(r => ['Owner', 'Admin'].includes(r.role.title))) {
        switch (customer?.status) {
            case ACCOUNT_STATUS.Unlocked:
                actions.push(lock_action);
                actions.push(delete_action)
                break;
            case ACCOUNT_STATUS.SoftLock:
            case ACCOUNT_STATUS.HardLock:
                actions.push(unlock_action);
                actions.push(delete_action)
                break;
            case ACCOUNT_STATUS.Deleted:
                actions.push(undelete_action);
                actions.push(permanent_delete_action);
            default:
                break;
        }
    }

    // Phone and email [label, value] pairs
    const phoneList = mapIfExists(customer, 'phones', (p) => ([showPhone(p.number), phoneLink(p.number)]));
    const emailList = mapIfExists(customer, 'emails', (e) => ([e.emailAddress, emailLink(e.emailAddress)]));

    return (
        <Card className={classes.cardRoot}>
            {phoneDialogOpen ? (
                <ListDialog
                    title={`Call ${customer?.fullName}`}
                    data={phoneList}
                    onClose={callPhone} />
            ) : null}
            {emailDialogOpen ? (
                <ListDialog
                    title={`Email ${customer?.fullName}`}
                    data={emailList}
                    onClose={sendEmail} />
            ) : null}
            <CardContent className={classes.content} onClick={() => onEdit(customer)}>
                <Typography gutterBottom variant="h6" component="h2">
                    {customer?.firstName} {customer?.lastName}
                </Typography>
                <p>Status: {status_map[customer?.status]}</p>
                <p>Business: {customer?.business?.name}</p>
                <p>Pronouns: {customer?.pronouns ?? 'Unset'}</p>
            </CardContent>
            <CardActions>
                {actions?.map((action, index) => 
                    <Tooltip key={`action-${index}`} title={action[2]} placement="bottom">
                        <IconButton onClick={action[0]}>
                            {action[1]}
                        </IconButton>
                    </Tooltip>
                )}
                {(phoneList?.length > 0) ?
                    (<Tooltip title="View phone numbers" placement="bottom">
                        <IconButton onClick={() => setPhoneDialogOpen(true)}>
                            <PhoneIcon className={classes.icon} />
                        </IconButton>
                    </Tooltip>)
                    : null}
                {(emailList?.length > 0) ?
                    (<Tooltip title="View emails" placement="bottom">
                        <IconButton onClick={() => setEmailDialogOpen(true)}>
                            <EmailIcon className={classes.icon} />
                        </IconButton>
                    </Tooltip>)
                    : null}
            </CardActions>
        </Card>
    );
}

CustomerCard.propTypes = {
    customer: PropTypes.object.isRequired,
    onEdit: PropTypes.func.isRequired,
}

export { CustomerCard };