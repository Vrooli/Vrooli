/**
 * Displays a list of emails for the user to manage
 */
import { EmailListProps } from '../types';
import { useCallback } from 'react';
import { Email } from 'types';
import { Box, IconButton, Stack, TextField } from '@mui/material';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { Pubs, updateArray } from 'utils';
import { emailCreateMutation, emailDeleteOneMutation, emailUpdateMutation, sendVerificationEmailMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { EmailListItem } from '../EmailListItem/EmailListItem';
import { emailCreateButton as validationSchema } from '@local/shared';

export const EmailList = ({
    handleUpdate,
    numVerifiedWallets,
    list,
}: EmailListProps) => {

    // Handle add
    const [addMutation, { loading: loadingAdd }] = useMutation<any>(emailCreateMutation);
    const formik = useFormik({
        initialValues: {
            email: '',
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            if (!formik.isValid) return;
            mutationWrapper({
                mutation: addMutation,
                input: {
                    emailAddress: values.email,
                    receivesAccountUpdates: true,
                    receivesBusinessUpdates: true,
                },
                onSuccess: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Please check your email to complete verification.' });
                    handleUpdate([...list, response.data.addEmail]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            })
        },
    });

    const [updateMutation, { loading: loadingUpdate }] = useMutation<any>(emailUpdateMutation);
    const onUpdate = useCallback((index: number, updatedEmail: Email) => {
        mutationWrapper({
            mutation: updateMutation,
            input: {
                id: updatedEmail.id,
                receivesAccountUpdates: updatedEmail.receivesAccountUpdates,
                receivesBusinessUpdates: updatedEmail.receivesBusinessUpdates,
            },
            onSuccess: (response) => {
                handleUpdate(updateArray(list, index, updatedEmail));
            },
        })
    }, [handleUpdate, list]);

    const [deleteMutation, { loading: loadingDelete }] = useMutation<any>(emailDeleteOneMutation);
    const onDelete = useCallback((email: Email) => {
        // Make sure that the user has at least one other authentication method 
        // (i.e. one other email or one other wallet)
        if (list.length <= 1 && numVerifiedWallets === 0) {
            PubSub.publish(Pubs.Snack, { message: 'Cannot delete your only authentication method!', severity: 'error' });
            return;
        }
        // Confirmation dialog
        PubSub.publish(Pubs.AlertDialog, {
            message: `Are you sure you want to delete email ${email.emailAddress}?`,
            buttons: [
                {
                    text: 'Yes', onClick: () => {
                        mutationWrapper({
                            mutation: deleteMutation,
                            input: { id: email.id },
                            onSuccess: (response) => {
                                handleUpdate([...list.filter(w => w.id !== email.id)])
                            },
                        })
                    }
                },
                { text: 'Cancel', onClick: () => { } },
            ]
        });
    }, [handleUpdate, list, numVerifiedWallets]);

    const [verifyMutation, { loading: loadingVerifyEmail }] = useMutation<any>(sendVerificationEmailMutation);
    const sendVerificationEmail = useCallback((email: Email) => {
        mutationWrapper({
            mutation: verifyMutation,
            input: { emailAddress: email.emailAddress },
            onSuccess: (response) => {
                PubSub.publish(Pubs.Snack, { message: 'Please check your email to complete verification.' });
            },
        })
    }, [verifyMutation]);

    return (
        <form onSubmit={formik.handleSubmit}>
            <Box sx={{
                overflow: 'overlay',
                border: '1px solid #e0e0e0',
                borderRadius: '8px',
                maxWidth: '1000px',
                marginLeft: 1,
                marginRight: 1,
            }}>
                {/* Email list */}
                {list.map((email: Email, index) => (
                    <EmailListItem
                        key={`email-${index}`}
                        data={email}
                        index={index}
                        handleUpdate={onUpdate}
                        handleDelete={onDelete}
                        handleVerify={sendVerificationEmail}
                    />
                ))}
            </Box>
            {/* Add new email */}
            <Stack direction="row" sx={{
                display: 'flex',
                justifyContent: 'center',
                paddingTop: 2,
                paddingBottom: 6,
            }}>
                <TextField
                    autoComplete='email'
                    fullWidth
                    id="email"
                    name="email"
                    label="New Email Address"
                    value={formik.values.email}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.email && Boolean(formik.errors.email)}
                    helperText={formik.touched.email && formik.errors.email}
                    sx={{
                        height: '56px',
                        maxWidth: '400px',
                    }}
                />
                <IconButton
                    aria-label='add-new-email-button'
                    onClick={() => { formik.handleSubmit() }}
                    sx={{
                        background: (t) => t.palette.secondary.main,
                        borderRadius: '0 5px 5px 0',
                    }}>
                    <AddIcon />
                </IconButton>
            </Stack>
        </form>
    )
}