/**
 * Displays a list of push devices for the user to manage
 */
import { PushListProps } from '../types';
import { useCallback } from 'react';
import { Box, Stack, TextField, useTheme } from '@mui/material';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { getDeviceInfo, PubSub, updateArray } from 'utils';
import { useFormik } from 'formik';
import { PushListItem } from '../PushListItem/PushListItem';
import { AddIcon } from '@shared/icons';
import { ColorIconButton } from 'components/buttons';
import { DeleteOneInput, DeleteType, PushDevice, PushDeviceCreateInput, PushDeviceUpdateInput, Success } from '@shared/consts';
import { pushDeviceValidation } from '@shared/validation';
import { pushDeviceCreate } from 'api/generated/endpoints/pushDevice_create';
import { pushDeviceUpdate } from 'api/generated/endpoints/pushDevice_update';
import { deleteOneOrManyDeleteOne } from 'api/generated/endpoints/deleteOneOrMany_deleteOne';

//TODO copied from emaillist. need to rewrite
export const PushList = ({
    handleUpdate,
    list,
}: PushListProps) => {
    const { palette } = useTheme();

    // Handle add
    const [addMutation, { loading: loadingAdd }] = useCustomMutation<PushDevice, PushDeviceCreateInput>(pushDeviceCreate);
    const formik = useFormik({
        initialValues: {
            endpoint: '',
            expires: '',
            keys: {
                auth: '',
                p256dh: '',
            },
        },
        enableReinitialize: true,
        validationSchema: pushDeviceValidation.create({}),
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd) return;
            mutationWrapper<PushDevice, PushDeviceCreateInput>({
                mutation: addMutation,
                input: {
                    endpoint: values.endpoint,
                    expires: values.expires,
                    keys: values.keys,
                    name: getDeviceInfo().deviceName,
                },
                onSuccess: (data) => {
                    PubSub.get().publishSnack({ messageKey: 'CompleteVerificationInEmail', severity: 'Info' });
                    handleUpdate([...list, data]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            })
        },
    });

    const [updateMutation, { loading: loadingUpdate }] = useCustomMutation<PushDevice, PushDeviceUpdateInput>(pushDeviceUpdate);
    const onUpdate = useCallback((index: number, updatedDevice: PushDevice) => {
        if (loadingUpdate) return;
        mutationWrapper<PushDevice, PushDeviceUpdateInput>({
            mutation: updateMutation,
            input: {
                id: updatedDevice.id,
                name: updatedDevice.name,
            },
            onSuccess: () => {
                handleUpdate(updateArray(list, index, updatedDevice));
            },
        })
    }, [handleUpdate, list, loadingUpdate, updateMutation]);

    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne);
    const onDelete = useCallback((device: PushDevice) => {
        if (loadingDelete) return;
        mutationWrapper<Success, DeleteOneInput>({
            mutation: deleteMutation,
            input: { id: device.id, objectType: DeleteType.Email },
            onSuccess: () => {
                handleUpdate([...list.filter(w => w.id !== device.id)])
            },
        })
    }, [deleteMutation, handleUpdate, list, loadingDelete]);

    return (
        <form onSubmit={formik.handleSubmit}>
            {list.length > 0 && <Box id='push-device-list' sx={{
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
            {/* Add new push-device */}
            <Stack direction="row" sx={{
                display: 'flex',
                justifyContent: 'center',
                paddingTop: 2,
                paddingBottom: 6,
            }}>
                {/* <TextField
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
                /> */}
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