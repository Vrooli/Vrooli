import {
    Dialog,
    DialogContent,
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
import { validate as uuidValidate } from 'uuid';
import { SearchType, routineSearchSchema, removeSearchParams } from 'utils';
import { useLocation } from '@shared/route';
import { AddIcon } from '@shared/icons';

const helpText =
    `This dialog allows you to connect a new or existing subroutine. Each subroutine becomes a page when executing the routine (or if it contains its own subroutines, then those subroutines become pages).`

const titleAria = 'select-or-create-subroutine-dialog-title';

export const SubroutineSelectOrCreateDialog = ({
    handleAdd,
    handleClose,
    isOpen,
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
    const handleRoutineSelect = useCallback((routine: Routine) => {
        queryingRef.current = true;
        getRoutine({ variables: { input: { id: routine.id } } });
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
        const validId: boolean = uuidValidate(routineId);
        // Ignore current routine
        const excludeIds = validId ? { excludeIds: [routineId] } : {};
        // Don't include incomplete/internal routines, unless they're your own
        const incomplete: IsCompleteInput = { isComplete: true, isCompleteExceptions: [] };
        const internal: IsInternalInput = { isInternal: false, isInternalExceptions: [] };
        if (validId) {
            const except = {
                id: routineId,
                relation: 'parent',
            }
            incomplete.isCompleteExceptions.push(except);
            internal.isInternalExceptions.push(except);
        }
        if (session.userId) {
            const except = {
                id: session.userId,
                relation: 'user',
            }
            incomplete.isCompleteExceptions.push(except);
            internal.isInternalExceptions.push(except);
        }
        return { ...excludeIds, ...incomplete, ...internal };
    }, [routineId, session.userId]);

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
                    background: palette.background.default,
                },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            {/* Popup for creating a new routine */}
            <BaseObjectDialog
                onAction={handleCreateClose}
                open={isCreateOpen}
                zIndex={zIndex + 1}
            >
                <RoutineCreate
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
            <DialogContent>
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
                        id="subroutine-select-or-create-list"
                        itemKeyPrefix='routine-list-item'
                        noResultsText={"None found. Maybe you should create one?"}
                        searchType={SearchType.Routine}
                        onObjectSelect={(newValue) => handleRoutineSelect(newValue)}
                        searchPlaceholder={'Select existing subroutine...'}
                        session={session}
                        take={20}
                        where={where}
                        zIndex={zIndex}
                    />
                </Stack>
            </DialogContent>
        </Dialog>
    )
}