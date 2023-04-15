/**
 * Displays a list of push devices for the user to manage
 */
import { Button, Stack, useTheme } from '@mui/material';
import { DeleteOneInput, DeleteType, PushDevice, PushDeviceCreateInput, PushDeviceUpdateInput, Success } from '@shared/consts';
import { AddIcon } from '@shared/icons';
import { pushDeviceValidation } from '@shared/validation';
import { deleteOneOrManyDeleteOne } from 'api/generated/endpoints/deleteOneOrMany_deleteOne';
import { pushDeviceCreate } from 'api/generated/endpoints/pushDevice_create';
import { pushDeviceUpdate } from 'api/generated/endpoints/pushDevice_update';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { ListContainer } from 'components/containers/ListContainer/ListContainer';
import { useFormik } from 'formik';
import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { getDeviceInfo } from 'utils/display/device';
import { PubSub } from 'utils/pubsub';
import { setupPush } from 'utils/push';
import { updateArray } from 'utils/shape/general';
import { PushListItem } from '../PushListItem/PushListItem';
import { PushListProps } from '../types';

//TODO copied from emaillist. need to rewrite
export const PushList = ({
    handleUpdate,
    list,
}: PushListProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

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
            <ListContainer
                emptyText={t(`NoPushDevices`, { ns: 'error' })}
                isEmpty={list.length === 0}
                sx={{ maxWidth: '500px' }}
            >
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
            </ListContainer>
            {/* Add new push-device */}
            <Stack direction="row" sx={{
                display: 'flex',
                justifyContent: 'center',
                paddingTop: 2,
                paddingBottom: 6,
            }}>
                <Button
                    disabled={loadingAdd}
                    fullWidth
                    onClick={setupPush}
                    startIcon={<AddIcon />}
                >{t('AddThisDevice')}</Button>
            </Stack>
        </form>
    )
}