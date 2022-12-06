/**
 * Displays a list of push devices for the user to manage
 */
import { PushListProps } from '../types';
import { useCallback } from 'react';
import { Email, PushDevice } from 'types';
import { Box, Stack, TextField, useTheme } from '@mui/material';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { getDeviceName, PubSub, updateArray } from 'utils';
import { emailCreateMutation, deleteOneMutation, emailUpdateMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { PushListItem } from '../PushListItem/PushListItem';
import { emailCreateButton as validationSchema } from '@shared/validation';
import { emailCreateVariables, emailCreate_emailCreate } from 'graphql/generated/emailCreate';
import { emailUpdateVariables, emailUpdate_emailUpdate } from 'graphql/generated/emailUpdate';
import { deleteOneVariables, deleteOne_deleteOne } from 'graphql/generated/deleteOne';
import { AddIcon } from '@shared/icons';
import { SnackSeverity } from 'components/dialogs';
import { ColorIconButton } from 'components/buttons';
import { DeleteType } from '@shared/consts';

//TODO copied from emaillist. need to rewrite
export const PushList = ({
    handleUpdate,
    list,
}: PushListProps) => {
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
                    name: getDeviceName(),
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
        mutationWrapper<deleteOne_deleteOne, deleteOneVariables>({
            mutation: deleteMutation,
            input: { id: email.id, objectType: DeleteType.Email },
            onSuccess: () => {
                handleUpdate([...list.filter(w => w.id !== email.id)])
            },
        })
    }, [deleteMutation, handleUpdate, list, loadingDelete]);

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
                {/* Push device list */}
                {list.map((device: PushDevice, index) => (
                    <PushListItem
                        key={`push-${index}`}
                        data={device}
                        index={index}
                        handleUpdate={onUpdate}
                        handleDelete={onDelete}
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