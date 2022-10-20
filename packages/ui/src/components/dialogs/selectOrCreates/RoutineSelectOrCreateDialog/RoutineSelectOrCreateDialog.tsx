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
import { RoutineSelectOrCreateDialogProps } from '../types';
import { Routine } from 'types';
import { SearchList } from 'components/lists';
import { routineQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { routine, routineVariables } from 'graphql/generated/routine';
import { RoutineCreate } from 'components/views/Routine/RoutineCreate/RoutineCreate';
import { SearchType, routineSearchSchema, removeSearchParams } from 'utils';
import { useLocation } from '@shared/route';
import { AddIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';

const helpText =
    `This dialog allows you to connect a new or existing routine to an object.

If you do not own the routine, the owner will have to approve of the request.`

const titleAria = "select-or-create-routine-dialog-title"

export const RoutineSelectOrCreateDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    session,
    zIndex,
}: RoutineSelectOrCreateDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

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
        handleAdd(routine);
        onClose();
    }, [handleAdd, onClose]);
    const handleCreateClose = useCallback(() => {
        setIsCreateOpen(false);
    }, [setIsCreateOpen]);

    // If routine selected from search, query for full data
    const [getRoutine, { data: routineData }] = useLazyQuery<routine, routineVariables>(routineQuery);
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((routine: Routine) => {
        // Query for full routine data, if not already known (would be known if the same routine was selected last time)
        if (routineData?.routine?.id === routine.id) {
            handleAdd(routineData?.routine);
            onClose();
        } else {
            queryingRef.current = true;
            getRoutine({ variables: { input: { id: routine.id } } });
        }
        // Return false so the list item does not navigate
        return false;
    }, [getRoutine, routineData, handleAdd, onClose]);
    useEffect(() => {
        if (routineData?.routine && queryingRef.current) {
            handleAdd(routineData.routine);
            onClose();
        }
        queryingRef.current = false;
    }, [handleAdd, onClose, handleCreateClose, routineData]);

    return (
        <Dialog
            open={isOpen}
            onClose={onClose}
            scroll="body"
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': { overflow: 'visible', background: palette.background.default },
                '& .MuiDialog-paper': { 
                    overflow: 'visible',
                    width: 'min(100%, 600px)',
                }
            }}
        >
            {/* Popup for creating a new routine */}
            <BaseObjectDialog
                onAction={handleCreateClose}
                open={isCreateOpen}
                zIndex={zIndex + 1}
            >
                <RoutineCreate
                    onCreated={handleCreated}
                    onCancel={handleCreateClose}
                    session={session}
                    zIndex={zIndex + 1}
                />
            </BaseObjectDialog>
            <DialogTitle
                ariaLabel={titleAria}
                title={'Add Routine'}
                helpText={helpText}
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
                        id="routine-select-or-create-list"
                        itemKeyPrefix='routine-list-item'
                        noResultsText={"None found. Maybe you should create one?"}
                        beforeNavigation={fetchFullData}
                        searchType={SearchType.Routine}
                        searchPlaceholder={'Select existing routines...'}
                        session={session}
                        take={20}
                        where={{ userId }}
                        zIndex={zIndex}
                    />
                </Stack>
            </DialogContent>
        </Dialog>
    )
}