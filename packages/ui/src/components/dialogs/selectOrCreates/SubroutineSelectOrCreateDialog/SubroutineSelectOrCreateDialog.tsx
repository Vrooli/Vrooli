import { RoutineVersion, SearchException, VisibilityType } from '@shared/consts';
import { uuidValidate } from '@shared/uuid';
import { useField } from 'formik';
import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { IsCompleteInput, IsInternalInput } from 'types';
import { OwnerShape } from 'utils/shape/models/types';
import { SelectOrCreateDialog } from '../SelectOrCreateDialog/SelectOrCreateDialog';
import { SubroutineSelectOrCreateDialogProps } from '../types';

export const SubroutineSelectOrCreateDialog = ({
    handleAdd,
    nodeId,
    routineVersionId,
    ...params
}: SubroutineSelectOrCreateDialogProps) => {
    const { t } = useTranslation();

    const [ownerField] = useField<OwnerShape | null | undefined>('root.owner');

    const handleCreated = useCallback((item: any) => {
        handleAdd(nodeId, item as RoutineVersion);
    }, [handleAdd, nodeId]);

    /**
     * Query conditions change depending on a few factors
     */
    const where = useMemo(() => {
        // If no routineVersionId, then we are creating a new routine
        if (!routineVersionId || !uuidValidate(routineVersionId)) return { visibility: VisibilityType.All };
        // Ignore current routine
        const excludeIds = { excludeIds: [routineVersionId] };
        // Don't include incomplete/internal routines, unless they're your own
        const incomplete: IsCompleteInput = { isComplete: true };
        const internal: IsInternalInput = { isInternal: false };
        if (ownerField.value) {
            const exception: SearchException = {
                field: ownerField.value.__typename,
                // Since exceptions support multiple data types, we must stringify the value
                value: JSON.stringify(ownerField.value.id),
            }
            incomplete.isCompleteExceptions = [exception];
            internal.isInternalExceptions = [exception];
        }
        return { ...excludeIds, ...incomplete, ...internal, visibility: VisibilityType.All };
    }, [ownerField.value, routineVersionId]);

    return <SelectOrCreateDialog
        {...params}
        handleAdd={handleCreated}
        help={t('SelectOrCreateSubroutineDialogHelp')}
        objectType='RoutineVersion'
        where={where}
    />
}