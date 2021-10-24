import { useCallback, useMemo, useState } from 'react';
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
import { AccountStatus } from '@local/shared';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { emailLink, mapIfExists, phoneLink, PUBS, showPhone } from 'utils';
import PubSub from 'pubsub-js';
import { ListDialog } from 'components';
import { cardStyles } from './styles';
import { changeCustomerStatus } from 'graphql/generated/changeCustomerStatus';
import { deleteCustomer } from 'graphql/generated/deleteCustomer';
import { Customer } from 'types';

const useStyles = makeStyles(cardStyles);

interface Props {
    customer: Customer;
    onEdit: (customer: Customer) => void;
}

const CustomerCard = ({
    customer,
    onEdit,
}: Props) => {
    const classes = useStyles();
    const [changeCustomerStatus] = useMutation<changeCustomerStatus>(changeCustomerStatusMutation);
    const [deleteCustomer] = useMutation<deleteCustomer>(deleteCustomerMutation);
    const [phoneDialogOpen, setPhoneDialogOpen] = useState(false);
    const [emailDialogOpen, setEmailDialogOpen] = useState(false);

    const openPhoneDialog = () => setPhoneDialogOpen(true);
    const openEmailDialog = () => setEmailDialogOpen(true);

    const callPhone = (phoneLink?: string | null) => {
        setPhoneDialogOpen(false);
        if (phoneLink) window.location.href = phoneLink;
    }

    const sendEmail = (emailLink?: string | null) => {
        setEmailDialogOpen(false);
        if (emailLink) window.open(emailLink, '_blank', 'noopener,noreferrer')
    }

    const editCustomer = useCallback(() => onEdit(customer), [customer, onEdit]);

    const status_map = useMemo(() => ({
        [AccountStatus.DELETED]: 'Deleted',
        [AccountStatus.UNLOCKED]: 'Unlocked',
        [AccountStatus.SOFT_LOCKED]: 'Soft Locked',
        [AccountStatus.HARD_LOCKED]: 'Hard Locked',
    }), [])

    const modifyCustomer = useCallback((status, message) => {
        mutationWrapper({
            mutation: changeCustomerStatus,
            data: { variables: { id: customer.id, status: status } },
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
                    data: { variables: { id: customer.id } },
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
            firstButtonClicked: () => modifyCustomer(AccountStatus.DELETED, 'Customer deleted.'),
            secondButtonText: 'No',
        });
    }, [customer, modifyCustomer])

    type Action = [(data: any) => any, any, string, any?, string?];
    let edit_action: Action = [editCustomer, <EditIcon className={classes.icon} />, 'Edit customer']
    let unlock_action: Action = [() => modifyCustomer(AccountStatus.UNLOCKED, 'Customer account unlocked.'), <LockOpenIcon className={classes.icon} />, 'Unlock customer account'];
    let lock_action: Action = [() => modifyCustomer(AccountStatus.HARD_LOCKED, 'Customer account locked.'), <LockIcon className={classes.icon} />, 'Lock customer account'];
    let undelete_action: Action = [() => modifyCustomer(AccountStatus.UNLOCKED, 'Customer account restored.'), <LockOpenIcon className={classes.icon} />, 'Restore deleted account'];
    let delete_action: Action = [confirmDelete, <DeleteIcon className={classes.icon} />, 'Delete user'];
    let permanent_delete_action: Action = [confirmPermanentDelete, <DeleteForeverIcon className={classes.icon} />, 'Permanently delete user']

    let actions = [edit_action];
    // Actions for customer accounts (i.e. not an owner or admin)
    if (!(customer.roles?.some(r => ['Owner', 'Admin'].includes(r.role.title)) || false)) {
        switch (customer.status) {
            case AccountStatus.UNLOCKED:
                actions.push(lock_action);
                actions.push(delete_action)
                break;
            case AccountStatus.SOFT_LOCKED:
            case AccountStatus.HARD_LOCKED:
                actions.push(unlock_action);
                actions.push(delete_action)
                break;
            case AccountStatus.DELETED:
                actions.push(undelete_action);
                actions.push(permanent_delete_action);
                break;
            default:
                break;
        }
    }

    const phoneList = mapIfExists(customer, 'phones', (p: any) => ({ label: showPhone(p.number), value: phoneLink(p.number)}));
    const emailList = mapIfExists(customer, 'emails', (e: any) => ({ label: e.emailAddress, value: emailLink(e.emailAddress)}));

    return (
        <Card className={classes.cardRoot}>
            {phoneDialogOpen ? (
                <ListDialog
                    title={`Call ${customer.firstName} ${customer.lastName}`}
                    data={phoneList}
                    onClose={callPhone} />
            ) : null}
            {emailDialogOpen ? (
                <ListDialog
                    title={`Email ${customer.firstName} ${customer.lastName}`}
                    data={emailList}
                    onClose={sendEmail} />
            ) : null}
            <CardContent className={classes.content} onClick={editCustomer}>
                <Typography gutterBottom variant="h6" component="h2">
                    {customer.firstName} {customer.lastName}
                </Typography>
                <p>Status: {status_map[customer.status]}</p>
                <p>Business: {customer.business?.name}</p>
                <p>Pronouns: {customer.pronouns ?? 'Unset'}</p>
            </CardContent>
            <CardActions>
                {actions?.map((action, index) => 
                    <Tooltip key={`action-${index}`} title={action[2]} placement="bottom">
                        <IconButton onClick={action[0]}>
                            {action[1]}
                        </IconButton>
                    </Tooltip>
                )}
                {(phoneList && phoneList?.length > 0) ?
                    (<Tooltip title="View phone numbers" placement="bottom">
                        <IconButton onClick={openPhoneDialog}>
                            <PhoneIcon className={classes.icon} />
                        </IconButton>
                    </Tooltip>)
                    : null}
                {(emailList && emailList?.length > 0) ?
                    (<Tooltip title="View emails" placement="bottom">
                        <IconButton onClick={openEmailDialog}>
                            <EmailIcon className={classes.icon} />
                        </IconButton>
                    </Tooltip>)
                    : null}
            </CardActions>
        </Card>
    );
}

export { CustomerCard };