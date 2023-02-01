/**
 * Displays a list of emails for the user to manage
 */
import { EmailListProps } from '../types';
import { useCallback } from 'react';
import { Box, Stack, TextField, useTheme } from '@mui/material';
import { useMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { PubSub } from 'utils';
import { useFormik } from 'formik';
import { EmailListItem } from '../EmailListItem/EmailListItem';
import { AddIcon } from '@shared/icons';
import { SnackSeverity } from 'components/dialogs';
import { ColorIconButton } from 'components/buttons';
import { DeleteOneInput, DeleteType, Email, EmailCreateInput, SendVerificationEmailInput, Success } from '@shared/consts';
import { emailValidation } from '@shared/validation';
import { emailCreate, emailVerify } from 'api/generated/endpoints/email';
import { deleteOneOrManyDeleteOne } from 'api/generated/endpoints/deleteOneOrMany';

export const EmailList = ({
    handleUpdate,
    numVerifiedWallets,
    list,
}: EmailListProps) => {
    const { palette } = useTheme();

    // Handle add
    const [addMutation, { loading: loadingAdd }] = useMutation<Email, EmailCreateInput, 'emailCreate'>(emailCreate, 'emailCreate');
    const formik = useFormik({
        initialValues: {
            emailAddress: '',
        },
        enableReinitialize: true,
        validationSchema: emailValidation.create(),
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd) return;
            mutationWrapper<Email, EmailCreateInput>({
                mutation: addMutation,
                input: {
                    emailAddress: values.emailAddress,
                },
                onSuccess: (data) => {
                    PubSub.get().publishSnack({ messageKey: 'CompleteVerificationInEmail', severity: SnackSeverity.Info });
                    handleUpdate([...list, data]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            })
        },
    });

    const [deleteMutation, { loading: loadingDelete }] = useMutation<Success, DeleteOneInput, 'deleteOne'>(deleteOneOrManyDeleteOne, 'deleteOne');
    const onDelete = useCallback((email: Email) => {
        if (loadingDelete) return;
        // Make sure that the user has at least one other authentication method 
        // (i.e. one other email or one other wallet)
        if (list.length <= 1 && numVerifiedWallets === 0) {
            PubSub.get().publishSnack({ messageKey: 'MustLeaveVerificationMethod', severity: SnackSeverity.Error });
            return;
        }
        // Confirmation dialog
        PubSub.get().publishAlertDialog({
            messageKey: 'EmailDeleteConfirm',
            messageVariables: { emailAddress: email.emailAddress },
            buttons: [
                {
                    labelKey: 'Yes', 
                    onClick: () => {
                        mutationWrapper<Success, DeleteOneInput>({
                            mutation: deleteMutation,
                            input: { id: email.id, objectType: DeleteType.Email },
                            onSuccess: () => {
                                handleUpdate([...list.filter(w => w.id !== email.id)])
                            },
                        })
                    }
                },
                { labelKey: 'Cancel', onClick: () => { } },
            ]
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numVerifiedWallets]);

    const [verifyMutation, { loading: loadingVerifyEmail }] = useMutation<Success, SendVerificationEmailInput, 'sendVerificationEmail'>(emailVerify, 'sendVerificationEmail');
    const sendVerificationEmail = useCallback((email: Email) => {
        if (loadingVerifyEmail) return;
        mutationWrapper<Success, SendVerificationEmailInput>({
            mutation: verifyMutation,
            input: { emailAddress: email.emailAddress },
            onSuccess: () => {
                PubSub.get().publishSnack({ messageKey: 'CompleteVerificationInEmail', severity: SnackSeverity.Info });
            },
        })
    }, [loadingVerifyEmail, verifyMutation]);

    return (
        <form onSubmit={formik.handleSubmit}>
            {list.length > 0 && <Box id='email-list' sx={{
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
                        handleDelete={onDelete}
                        handleVerify={sendVerificationEmail}
                    />
                ))}
            </Box>}
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