/**
 * Displays a list of emails for the user to manage
 */
import { EmailListProps } from '../types';
import { useCallback } from 'react';
import { Email } from 'types';
import { Box, Stack, TextField, useTheme } from '@mui/material';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { PubSub, updateArray } from 'utils';
import { emailCreateMutation, deleteOneMutation, emailUpdateMutation, sendVerificationEmailMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { EmailListItem } from '../EmailListItem/EmailListItem';
import { emailCreateButton as validationSchema } from '@shared/validation';
import { DeleteOneType } from '@shared/consts';
import { emailCreateVariables, emailCreate_emailCreate } from 'graphql/generated/emailCreate';
import { emailUpdateVariables, emailUpdate_emailUpdate } from 'graphql/generated/emailUpdate';
import { deleteOneVariables, deleteOne_deleteOne } from 'graphql/generated/deleteOne';
import { sendVerificationEmailVariables, sendVerificationEmail_sendVerificationEmail } from 'graphql/generated/sendVerificationEmail';
import { AddIcon } from '@shared/icons';
import { SnackSeverity } from 'components/dialogs';
import { ColorIconButton } from 'components/buttons';

export const EmailList = ({
    handleUpdate,
    numVerifiedWallets,
    list,
}: EmailListProps) => {
    const { palette } = useTheme();

    // Handle add
    const [addMutation, { loading: loadingAdd }] = useMutation(emailCreateMutation);
    const formik = useFormik({
        initialValues: {
            emailAddress: '',
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd) return;
            mutationWrapper<emailCreate_emailCreate, emailCreateVariables>({
                mutation: addMutation,
                input: {
                    emailAddress: values.emailAddress,
                    receivesAccountUpdates: true,
                    receivesBusinessUpdates: true,
                },
                onSuccess: (data) => {
                    PubSub.get().publishSnack({ message: 'Please check your email to complete verification.', severity: SnackSeverity.Info });
                    handleUpdate([...list, data]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            })
        },
    });

    const [updateMutation, { loading: loadingUpdate }] = useMutation(emailUpdateMutation);
    const onUpdate = useCallback((index: number, updatedEmail: Email) => {
        if (loadingUpdate) return;
        mutationWrapper<emailUpdate_emailUpdate, emailUpdateVariables>({
            mutation: updateMutation,
            input: {
                id: updatedEmail.id,
                receivesAccountUpdates: updatedEmail.receivesAccountUpdates,
                receivesBusinessUpdates: updatedEmail.receivesBusinessUpdates,
            },
            onSuccess: () => {
                handleUpdate(updateArray(list, index, updatedEmail));
            },
        })
    }, [handleUpdate, list, loadingUpdate, updateMutation]);

    const [deleteMutation, { loading: loadingDelete }] = useMutation(deleteOneMutation);
    const onDelete = useCallback((email: Email) => {
        if (loadingDelete) return;
        // Make sure that the user has at least one other authentication method 
        // (i.e. one other email or one other wallet)
        if (list.length <= 1 && numVerifiedWallets === 0) {
            PubSub.get().publishSnack({ message: 'Cannot delete your only authentication method!', severity: SnackSeverity.Error });
            return;
        }
        // Confirmation dialog
        PubSub.get().publishAlertDialog({
            message: `Are you sure you want to delete email ${email.emailAddress}?`,
            buttons: [
                {
                    text: 'Yes', onClick: () => {
                        mutationWrapper<deleteOne_deleteOne, deleteOneVariables>({
                            mutation: deleteMutation,
                            input: { id: email.id, objectType: DeleteOneType.Email },
                            onSuccess: () => {
                                handleUpdate([...list.filter(w => w.id !== email.id)])
                            },
                        })
                    }
                },
                { text: 'Cancel', onClick: () => { } },
            ]
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numVerifiedWallets]);

    const [verifyMutation, { loading: loadingVerifyEmail }] = useMutation(sendVerificationEmailMutation);
    const sendVerificationEmail = useCallback((email: Email) => {
        if (loadingVerifyEmail) return;
        mutationWrapper<sendVerificationEmail_sendVerificationEmail, sendVerificationEmailVariables>({
            mutation: verifyMutation,
            input: { emailAddress: email.emailAddress },
            onSuccess: () => {
                PubSub.get().publishSnack({ message: 'Please check your email to complete verification.', severity: SnackSeverity.Info });
            },
        })
    }, [loadingVerifyEmail, verifyMutation]);

    return (
        <form onSubmit={formik.handleSubmit}>
            <Box id='email-list' sx={{
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
                    id="emailAddress"
                    name="emailAddress"
                    label="New Email Address"
                    value={formik.values.emailAddress}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.emailAddress && Boolean(formik.errors.emailAddress)}
                    helperText={formik.touched.emailAddress && formik.errors.emailAddress}
                    sx={{
                        height: '56px',
                        maxWidth: '400px',
                        '& .MuiInputBase-root': {
                            borderRadius: '5px 0 0 5px',
                        }
                    }}
                />
                <ColorIconButton
                    aria-label='add-new-email-button'
                    background={palette.secondary.main}
                    type='submit'
                    sx={{
                        borderRadius: '0 5px 5px 0',
                        height: '56px',
                    }}>
                    <AddIcon />
                </ColorIconButton>
            </Stack>
        </form>
    )
}