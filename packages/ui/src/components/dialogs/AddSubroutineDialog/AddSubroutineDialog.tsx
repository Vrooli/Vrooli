import {
    Box,
    Button,
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    Typography,
    useTheme
} from '@mui/material';
import { BaseObjectDialog, HelpButton } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { AddSubroutineDialogProps } from '../types';
import {
    Add as CreateIcon,
    Close as CloseIcon
} from '@mui/icons-material';
import { Routine } from 'types';
import { routineDefaultSortOption, RoutineSortOptions, SearchList } from 'components/lists';
import { routineQuery, routinesQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { routine, routineVariables } from 'graphql/generated/routine';
import { RoutineCreate } from 'components/views/RoutineCreate/RoutineCreate';
import { validate as uuidValidate } from 'uuid';

const helpText =
    `This dialog allows you to connect a new or existing subroutine. Each subroutine becomes a page when executing the routine (or if it contains its own subroutines, then those subroutines become pages).`

export const AddSubroutineDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    language,
    nodeId,
    routineId,
    session,
}: AddSubroutineDialogProps) => {
    const { palette } = useTheme();

    // Create new routine dialog
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const handleCreateOpen = useCallback(() => { setIsCreateOpen(true); }, [setIsCreateOpen]);
    const handleCreated = useCallback((routine: Routine) => {
        setIsCreateOpen(false);
        handleAdd(nodeId, routine);
        handleClose();
    }, [handleAdd, handleClose, nodeId]);
    const handleCreateClose = useCallback(() => {
        setIsCreateOpen(false);
    }, [setIsCreateOpen]);

    // If routine selected from search, query for full data
    const [getRoutine, { data: routineData, loading }] = useLazyQuery<routine, routineVariables>(routineQuery);
    const handleRoutineSelect = useCallback((routine: Routine) => {
        getRoutine({ variables: { input: { id: routine.id } } });
    }, [getRoutine]);
    useEffect(() => {
        if (routineData?.routine) {
            handleAdd(nodeId, routineData.routine);
            handleClose();
        }
    }, [handleCreateClose, routineData]);

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
                    onClick={(e) => { handleClose() }}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </Box>
        </Box>
    ), [])

    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<string | undefined>(undefined);
    const [timeFrame, setTimeFrame] = useState<string | undefined>(undefined);

    return (
        <Dialog
            open={isOpen}
            onClose={handleClose}
            sx={{
                '& .MuiDialogContent-root': { overflow: 'visible', background: '#cdd6df' },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            {/* Popup for creating a new routine */}
            <BaseObjectDialog
                hasNext={false}
                hasPrevious={false}
                onAction={handleCreateClose}
                open={isCreateOpen}
                title={"Create Routine"}
            >
                <RoutineCreate
                    onCancel={handleCreateClose}
                    onCreated={handleCreated}
                    session={session}
                />
            </BaseObjectDialog>
            {titleBar}
            <DialogContent>
                <Stack direction="column" spacing={4}>
                    <Button
                        fullWidth
                        onClick={handleCreateOpen}
                        startIcon={<CreateIcon />}
                    >Create</Button>
                    <Box sx={{
                        alignItems: 'center',
                        display: 'flex',
                        justifyContent: 'space-between',
                    }}>
                        <Typography variant="h6" sx={{ marginLeft: 'auto', marginRight: 'auto' }}>Or</Typography>
                    </Box>
                    <SearchList
                        itemKeyPrefix='routine-list-item'
                        defaultSortOption={routineDefaultSortOption}
                        noResultsText={"None found. Maybe you should create one?"}
                        onObjectSelect={(newValue) => handleRoutineSelect(newValue)}
                        query={routinesQuery}
                        searchPlaceholder={'Select existing subroutine...'}
                        searchString={searchString}
                        session={session}
                        setSearchString={setSearchString}
                        setSortBy={setSortBy}
                        setTimeFrame={setTimeFrame}
                        sortOptions={RoutineSortOptions}
                        sortBy={sortBy}
                        take={20}
                        timeFrame={timeFrame}
                        where={uuidValidate(routineId) ? { excludeIds: [routineId] } : undefined}
                    />
                </Stack>
            </DialogContent>
        </Dialog>
    )
}