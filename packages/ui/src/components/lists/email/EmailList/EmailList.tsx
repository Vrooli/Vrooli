/**
 * Displays a list of emails for the user to manage
 */
import { EmailListProps } from '../types';
import { useCallback } from 'react';
import { Email } from 'types';
import { Box, IconButton, Stack, TextField, useTheme } from '@mui/material';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { Pubs, updateArray } from 'utils';
import { emailCreateMutation, deleteOneMutation, emailUpdateMutation, sendVerificationEmailMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { EmailListItem } from '../EmailListItem/EmailListItem';
import { DeleteOneType, emailCreateButton as validationSchema } from '@local/shared';
import { emailCreate, emailCreateVariables } from 'graphql/generated/emailCreate';
import { emailUpdate, emailUpdateVariables } from 'graphql/generated/emailUpdate';
import { deleteOne, deleteOneVariables } from 'graphql/generated/deleteOne';
import { sendVerificationEmail, sendVerificationEmailVariables } from 'graphql/generated/sendVerificationEmail';

export const EmailList = ({
    handleUpdate,
    numVerifiedWallets,
    list,
}: EmailListProps) => {
    const { palette } = useTheme();

    // Handle add
    const [addMutation, { loading: loadingAdd }] = useMutation<emailCreate, emailCreateVariables>(emailCreateMutation);
    const formik = useFormik({
        initialValues: {
            email: '',
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd) return;
            mutationWrapper({
                mutation: addMutation,
                input: {
                    emailAddress: values.email,
                    receivesAccountUpdates: true,
                    receivesBusinessUpdates: true,
                },
                onSuccess: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Please check your email to complete verification.' });
                    handleUpdate([...list, response.data.emailCreate]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            })
        },
    });

    const [updateMutation, { loading: loadingUpdate }] = useMutation<emailUpdate, emailUpdateVariables>(emailUpdateMutation);
    const onUpdate = useCallback((index: number, updatedEmail: Email) => {
        if (loadingUpdate) return;
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
    }, [handleUpdate, list, loadingUpdate, updateMutation]);

    const [deleteMutation, { loading: loadingDelete }] = useMutation<deleteOne, deleteOneVariables>(deleteOneMutation);
    const onDelete = useCallback((email: Email) => {
        if (loadingDelete) return;
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
                            input: { id: email.id, objectType: DeleteOneType.Email },
                            onSuccess: (response) => {
                                handleUpdate([...list.filter(w => w.id !== email.id)])
                            },
                        })
                    }
                },
                { text: 'Cancel', onClick: () => { } },
            ]
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numVerifiedWallets]);

    const [verifyMutation, { loading: loadingVerifyEmail }] = useMutation<sendVerificationEmail, sendVerificationEmailVariables>(sendVerificationEmailMutation);
    const sendVerificationEmail = useCallback((email: Email) => {
        if (loadingVerifyEmail) return;
        mutationWrapper({
            mutation: verifyMutation,
            input: { emailAddress: email.emailAddress },
            onSuccess: (response) => {
                PubSub.publish(Pubs.Snack, { message: 'Please check your email to complete verification.' });
            },
        })
    }, [loadingVerifyEmail, verifyMutation]);

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
                        background: palette.secondary.main,
                        borderRadius: '0 5px 5px 0',
                    }}>
                    <AddIcon />
                </IconButton>
            </Stack>
        </form>
    )
}