import { ResourceList } from '@shared/consts';
import { ResourceListHorizontal } from 'components/lists/resource';
import { useField } from 'formik';
import { useCallback } from 'react';
import { ResourceListHorizontalInputProps } from '../types';

export const ResourceListHorizontalInput = ({
    disabled = false,
    isCreate,
    isLoading = false,
    zIndex,
}: ResourceListHorizontalInputProps) => {
    const [field, , helpers] = useField('resourceList');

    const handleUpdate = useCallback((newList: ResourceList) => {
        helpers.setValue(newList);
    }, [helpers]);

    return (
        <ResourceListHorizontal
            list={field.value}
            canUpdate={!disabled}
            handleUpdate={handleUpdate}
            loading={isLoading}
            mutate={!isCreate}
            zIndex={zIndex}
        />
    );
};