/**
 * Displays a list of emails for the user to manage
 */
import { EmailListProps } from '../types';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { Email } from 'types';
import { Box, Button, Stack, TextField } from '@mui/material';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { Pubs, updateArray } from 'utils';
import { emailCreateMutation, emailDeleteOneMutation, emailUpdateMutation } from 'graphql/mutation';
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
                onSuccess: (response) => { handleUpdate([...list, response.data.addEmail]); },
                onError: () => { formik.setSubmitting(false); },
            })
        },
    });

    const onAdd = useCallback((newEmail: Email) => {
        handleUpdate([...list, newEmail]);
    }, [handleUpdate, list]);

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
        mutationWrapper({
            mutation: deleteMutation,
            input: { id: email.id },
            onSuccess: (response) => {
                handleUpdate([...list.filter(w => w.id !== email.id)])
            },
        })
    }, [handleUpdate, list, numVerifiedWallets]);

    return (
        <>
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
                        handleVerify={() => { }}
                    />
                ))}
            </Box>
            {/* Add new email */}
            <Stack direction="row" sx={{
                alignItems: 'center',
                display: 'flex',
                justifyContent: 'center',
                paddingTop: 2,
                paddingBottom: 6,
            }}>
                <TextField
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
                <Button
                    fullWidth
                    startIcon={<AddIcon sx={{ width: '25px', height: '25px' }} />}
                    type="submit"
                    sx={{
                        height: '56px',
                        paddingRight: 0.5,
                        width: 'auto',
                    }}
                ></Button>
            </Stack>
        </>
    )
}