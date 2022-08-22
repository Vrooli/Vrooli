import {
    Box,
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { BaseObjectDialog, HelpButton, routineSearchSchema } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { SubroutineSelectOrCreateDialogProps } from '../types';
import {
    Add as CreateIcon,
    Close as CloseIcon
} from '@mui/icons-material';
import { IsCompleteInput, IsInternalInput, Routine } from 'types';
import { SearchList } from 'components/lists';
import { routineQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { routine, routineVariables } from 'graphql/generated/routine';
import { RoutineCreate } from 'components/views/Routine/RoutineCreate/RoutineCreate';
import { validate as uuidValidate } from 'uuid';
import { parseSearchParams, stringifySearchParams, SearchType } from 'utils';
import { useLocation } from '@shared/route';

const helpText =
    `This dialog allows you to connect a new or existing subroutine. Each subroutine becomes a page when executing the routine (or if it contains its own subroutines, then those subroutines become pages).`

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
        // Find all search fields
        const searchFields = [
            ...routineSearchSchema.fields.map(f => f.fieldName),
            'advanced',
            'sort',
            'time',
        ];
        // Find current search params
        const params = parseSearchParams(window.location.search);
        // Remove all search params that are advanced search fields
        Object.keys(params).forEach(key => {
            if (searchFields.includes(key)) {
                delete params[key];
            }
        });
        // Update URL
        setLocation(stringifySearchParams(params), { replace: true });
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
    const handleRoutineSelect = useCallback((routine: Routine) => {
        getRoutine({ variables: { input: { id: routine.id } } });
    }, [getRoutine]);
    useEffect(() => {
        if (routineData?.routine) {
            handleAdd(nodeId, routineData.routine);
            onClose();
        }
    }, [handleAdd, onClose, handleCreateClose, nodeId, routineData]);

    /**
     * Title bar with help button and close icon
     */
    const titleBar = useMemo(() => (
        <Box sx={{
            background: palette.primary.dark,
            color: palette.primary.contrastText,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: 2,
        }}>
            <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto' }}>
                {'Add Subroutine'}
                <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
            </Typography>
            <Box sx={{ marginLeft: 'auto' }}>
                <IconButton
                    edge="start"
                    onClick={(e) => { onClose() }}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </Box>
        </Box>
    ), [onClose, palette.primary.contrastText, palette.primary.dark])

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
            {titleBar}
            <DialogContent>
                <Stack direction="column" spacing={2}>
                    <Stack direction="row" alignItems="center" justifyContent="center">
                        <Typography component="h2" variant="h4">Routines</Typography>
                        <Tooltip title="Add new" placement="top">
                            <IconButton
                                size="large"
                                onClick={handleCreateOpen}
                                sx={{ padding: 1 }}
                            >
                                <CreateIcon color="secondary" sx={{ width: '1.5em', height: '1.5em' }} />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    <SearchList
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