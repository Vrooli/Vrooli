import { useCallback, useMemo } from 'react';
import { SubroutineSelectOrCreateDialogProps } from '../types';
import { IsCompleteInput, IsInternalInput } from 'types';
import { uuidValidate } from '@shared/uuid';
import { RoutineVersion, SearchException, VisibilityType } from '@shared/consts';
import { SelectOrCreateDialog } from '../SelectOrCreateDialog/SelectOrCreateDialog';
import { useTranslation } from 'react-i18next';

export const SubroutineSelectOrCreateDialog = ({
    handleAdd,
    nodeId,
    owner,
    routineVersionId,
    session,
    ...params
}: SubroutineSelectOrCreateDialogProps) => {
    const { t } = useTranslation();

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
        if (owner) {
            const exception: SearchException = {
                field: owner.__typename,
                // Since exceptions support multiple data types, we must stringify the value
                value: JSON.stringify(owner.id),
            }
            incomplete.isCompleteExceptions = [exception];
            internal.isInternalExceptions = [exception];
        }
        return { ...excludeIds, ...incomplete, ...internal, visibility: VisibilityType.All };
    }, [owner, routineVersionId]);

    return <SelectOrCreateDialog
        {...params}
        handleAdd={handleCreated}
        help={t('SelectOrCreateSubroutineDialogHelp')}
        objectType='RoutineVersion'
        session={session}
        where={where}
    />
}