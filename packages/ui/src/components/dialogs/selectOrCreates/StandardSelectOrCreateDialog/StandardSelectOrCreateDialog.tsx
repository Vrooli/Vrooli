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
import { StandardSelectOrCreateDialogProps } from '../types';
import {
    Add as CreateIcon,
    Close as CloseIcon
} from '@mui/icons-material';
import { Standard } from 'types';
import { standardDefaultSortOption, StandardSortOptions, SearchList } from 'components/lists';
import { standardQuery, standardsQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { standard, standardVariables } from 'graphql/generated/standard';
import { StandardCreate } from 'components/views/StandardCreate/StandardCreate';

const helpText =
    `This dialog allows you to connect a new or existing standard to a routine input/output.
    
    Standards allow for interoperability between routines and external services adhering to the Vrooli protocol.`

export const StandardSelectOrCreateDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    session,
}: StandardSelectOrCreateDialogProps) => {
    const { palette } = useTheme();

    // Create new standard dialog
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const handleCreateOpen = useCallback(() => { setIsCreateOpen(true); }, [setIsCreateOpen]);
    const handleCreated = useCallback((standard: Standard) => {
        setIsCreateOpen(false);
        handleAdd(standard);
        handleClose();
    }, [handleAdd, handleClose]);
    const handleCreateClose = useCallback(() => {
        setIsCreateOpen(false);
    }, [setIsCreateOpen]);

    // If standard selected from search, query for full data
    const [getStandard, { data: standardData }] = useLazyQuery<standard, standardVariables>(standardQuery);
    const handeStandardSelect = useCallback((standard: Standard) => {
        // Query for full standard data, if not already known (would be known if the same standard was selected last time)
        if (standardData?.standard?.id === standard.id) {
            handleAdd(standardData?.standard);
            handleClose();
        } else {
            getStandard({ variables: { input: { id: standard.id } } });
        }
    }, [getStandard, standardData, handleAdd, handleClose]);
    useEffect(() => {
        if (standardData?.standard) {
            handleAdd(standardData.standard);
            handleClose();
        }
    }, [handleAdd, handleClose, handleCreateClose, standardData]);

    /**
     * Title bar with help button and close icon
     */
    const titleBar = useMemo(() => (
        <Box sx={{
            alignItems: 'center',
            background: palette.primary.dark,
            color: palette.primary.contrastText,
            display: 'flex',
            justifyContent: 'space-between',
            padding: 2,
        }}>
            <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto' }}>
                {'Add Standard'}
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
    ), [handleClose, palette.primary.contrastText, palette.primary.dark])

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
            {/* Popup for creating a new standard */}
            <BaseObjectDialog
                onAction={handleCreateClose}
                open={isCreateOpen}
                title={"Create Standard"}
            >
                <StandardCreate
                    onCreated={handleCreated}
                    onCancel={handleCreateClose}
                    session={session}
                />
            </BaseObjectDialog>
            {titleBar}
            <DialogContent>
                <Stack direction="column" spacing={4}>
                    <SearchList
                        itemKeyPrefix='standard-list-item'
                        defaultSortOption={standardDefaultSortOption}
                        noResultsText={"None found. Maybe you should create one?"}
                        onObjectSelect={(newValue) => handeStandardSelect(newValue)}
                        query={standardsQuery}
                        searchPlaceholder={'Select existing standard...'}
                        searchString={searchString}
                        setSearchString={setSearchString}
                        session={session}
                        setSortBy={setSortBy}
                        setTimeFrame={setTimeFrame}
                        sortBy={sortBy}
                        sortOptions={StandardSortOptions}
                        take={20}
                        timeFrame={timeFrame}
                    />
                    <Button
                        fullWidth
                        onClick={handleCreateOpen}
                        startIcon={<CreateIcon />}
                    >Create New</Button>
                </Stack>
            </DialogContent>
        </Dialog>
    )
}