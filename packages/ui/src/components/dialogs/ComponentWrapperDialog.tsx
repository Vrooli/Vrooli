// Wraps Organization, Actor, Project, or Routine page in a dialog.
// Used if components were navigated to, rather than directly loaded via url

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
import { deleteUserMutation, updateUserMutation } from 'graphql/mutation';
import { PUBS } from 'utils';
import PubSub from 'pubsub-js';
import { useMutation } from '@apollo/client';
import { UpTransition } from 'components';
import { updateUser } from 'graphql/generated/updateUser';
import { deleteUser } from 'graphql/generated/deleteUser';
import { User } from 'types';

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
    user: User | null;
    open?: boolean;
    onClose: () => any;
}

export const ActorDialog = ({
    user,
    open = true,
    onClose,
}: Props) => {
    const classes = useStyles();
    // Stores the modified user data before updating
    const [currUser, setCurrUser] = useState<User | null>(user);
    const [updateUserMut] = useMutation<updateUser>(updateUserMutation);
    const [deleteUserMut] = useMutation<deleteUser>(deleteUserMutation);

    useEffect(() => {
        setCurrUser(user);
    }, [user])

    const revert = () => setCurrUser(user);

    const statusToggleData = useMemo(() => {
        if (!currUser?.status || !(currUser?.status in statusToggle)) return ['', '', null, null];
        return statusToggle[currUser.status];
    }, [currUser])

    // Locks/unlocks/undeletes a user
    const toggleLock = useCallback(() => {
        mutationWrapper({
            mutation: updateUserMut,
            data: { variables: { id: currUser?.id, status: statusToggleData[2] } },
            successMessage: () => 'User updated.',
            errorMessage: () => 'Failed to update user.'
        })
    }, [currUser?.id, statusToggleData, updateUserMut])

    const deleteUser = useCallback(() => {
        mutationWrapper({
            mutation: deleteUserMut,
            data: { variables: { id: currUser?.id }},
            successMessage: () => 'User deleted.',
            onSuccess: onClose,
        })
    }, [currUser?.id, deleteUserMut, onClose])
    
    const confirmDelete = useCallback(() => {
        PubSub.publish(PUBS.AlertDialog, {
            message: `Are you sure you want to delete the account for ${currUser?.firstName} ${currUser?.lastName}?`,
            buttons: [
                { text: 'Yes', onClick: deleteUser },
                { text: 'No' },
            ]
        });
    }, [currUser, deleteUser])

    const updateUser = useCallback(() => {
        mutationWrapper({
            mutation: updateUserMut,
            data: { variables: { ...currUser }},
            successMessage: () => 'User updated.',
        })
    }, [currUser, updateUserMut])

    let changes_made = !isEqual(user, currUser);
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
                    disabled={!user?.id}
                    startIcon={statusToggleData[3]}
                    onClick={toggleLock}
                >{statusToggleData[1]}</Button>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <Button
                    fullWidth
                    disabled={!user?.id}
                    startIcon={<DeleteIcon />}
                    onClick={confirmDelete}
                >Delete</Button>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
                <Button
                    fullWidth
                    disabled={!changes_made}
                    startIcon={<UpdateIcon />}
                    onClick={updateUser}
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
                                {`${user?.firstName} ${user?.lastName}`}
                            </Typography>
                            <Typography variant="h6">
                                {user?.business?.name}
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