import {
    Dialog,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { BaseObjectDialog, DialogTitle } from 'components';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { SubroutineSelectOrCreateDialogProps } from '../types';
import { IsCompleteInput, IsInternalInput, Routine } from 'types';
import { SearchList } from 'components/lists';
import { routineQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { routine, routineVariables } from 'graphql/generated/routine';
import { RoutineCreate } from 'components/views/Routine/RoutineCreate/RoutineCreate';
import { uuidValidate } from '@shared/uuid';
import { SearchType, routineSearchSchema, removeSearchParams } from 'utils';
import { useLocation } from '@shared/route';
import { AddIcon } from '@shared/icons';
import { SearchException, VisibilityType } from 'graphql/generated/globalTypes';

const helpText =
    `This dialog allows you to connect a new or existing subroutine. Each subroutine becomes a page when executing the routine (or if it contains its own subroutines, then those subroutines become pages).`

const titleAria = 'select-or-create-subroutine-dialog-title';

export const SubroutineSelectOrCreateDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    owner,
    nodeId,
    routineId,
    session,
    zIndex,
}: SubroutineSelectOrCreateDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback(() => {
        // Clear search params
        removeSearchParams(setLocation, [
            ...routineSearchSchema.fields.map(f => f.fieldName),
            'advanced',
            'sort',
            'time',
        ]);
        handleClose();
    }, [handleClose, setLocation]);

    // Create new routine dialog
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const handleCreateOpen = useCallback(() => { setIsCreateOpen(true); }, [setIsCreateOpen]);
    const handleCreated = useCallback((routine: Routine) => {
        setIsCreateOpen(false);
        handleAdd(nodeId, routine);
        onClose();
    }, [handleAdd, onClose, nodeId]);
    const handleCreateClose = useCallback(() => {
        setIsCreateOpen(false);
    }, [setIsCreateOpen]);

    // If routine selected from search, query for full data
    const [getRoutine, { data: routineData }] = useLazyQuery<routine, routineVariables>(routineQuery);
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((routine: Routine) => {
        queryingRef.current = true;
        getRoutine({ variables: { input: { id: routine.id } } });
        // Return false so the list item does not navigate
        return false;
    }, [getRoutine]);
    useEffect(() => {
        if (routineData?.routine && queryingRef.current) {
            handleAdd(nodeId, routineData.routine);
            onClose();
        }
        queryingRef.current = false;
    }, [handleAdd, onClose, handleCreateClose, nodeId, routineData]);

    /**
     * Query conditions change depending on a few factors
     */
    const where = useMemo(() => {
        // If no routineId, then we are creating a new routine
        if (!routineId || !uuidValidate(routineId)) return { visibility: VisibilityType.All };
        // Ignore current routine
        const excludeIds = { excludeIds: [routineId] };
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
    }, [owner, routineId]);

    return (
        <Dialog
            open={isOpen}
            onClose={onClose}
            scroll="body"
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': {
                    overflow: 'visible',
                    minWidth: 'min(600px, 100%)',
                },
                '& .MuiDialog-paperScrollBody': {
                    overflow: 'visible',
                    background: palette.background.default,
                    margin: { xs: 0, sm: 2, md: 4 },
                    maxWidth: { xs: '100%!important', sm: 'calc(100% - 64px)' },
                    minHeight: { xs: '100vh', sm: 'auto' },
                    display: { xs: 'block', sm: 'inline-block' },
                },
                // Remove ::after element that is added to the dialog
                '& .MuiDialog-container::after': {
                    content: 'none',
                },
            }}
        >
            {/* Popup for creating a new routine */}
            <BaseObjectDialog
                onAction={handleCreateClose}
                open={isCreateOpen}
                zIndex={zIndex + 1}
            >
                <RoutineCreate
                    isSubroutine={true}
                    onCancel={handleCreateClose}
                    onCreated={handleCreated}
                    session={session}
                    zIndex={zIndex + 1}
                />
            </BaseObjectDialog>
            <DialogTitle
                ariaLabel={titleAria}
                helpText={helpText}
                title={'Add Subroutine'}
                onClose={onClose}
            />
            <Stack direction="column" spacing={2}>
                <Stack direction="row" alignItems="center" justifyContent="center">
                    <Typography component="h2" variant="h4">Routines</Typography>
                    <Tooltip title="Add new" placement="top">
                        <IconButton
                            size="medium"
                            onClick={handleCreateOpen}
                            sx={{ padding: 1 }}
                        >
                            <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <SearchList
                    canSearch={Boolean(nodeId)} // Can only query when a node is selected
                    id="subroutine-select-or-create-list"
                    itemKeyPrefix='routine-list-item'
                    noResultsText={"None found. Maybe you should create one?"}
                    searchType={SearchType.Routine}
                    beforeNavigation={fetchFullData}
                    searchPlaceholder={'Select existing subroutine...'}
                    session={session}
                    take={20}
                    where={where}
                    zIndex={zIndex}
                />
            </Stack>
        </Dialog>
    )
}