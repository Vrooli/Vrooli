import {
    Box,
    Button,
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    Typography
} from '@mui/material';
import { BaseObjectDialog, HelpButton } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { AddStandardDialogProps } from '../types';
import {
    Add as CreateIcon,
    Close as CloseIcon
} from '@mui/icons-material';
import { Standard } from 'types';
import { standardDefaultSortOption, StandardListItem, standardOptionLabel, StandardSortOptions, SearchList } from 'components/lists';
import { standardQuery, standardsQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { standard, standardVariables } from 'graphql/generated/standard';
import { StandardCreate } from 'components/views/StandardCreate/StandardCreate';

const helpText =
    `This dialog allows you to connect a new or existing standard to a routine input/output.
    
    Standards allow for interoperability between routines and external services adhering to the Vrooli protocol.`

export const AddStandardDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    session,
}: AddStandardDialogProps) => {

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
    const [getStandard, { data: standardData, loading }] = useLazyQuery<standard, standardVariables>(standardQuery);
    const handeStandardSelect = useCallback((standard: Standard) => {
        getStandard({ variables: { input: { id: standard.id } } });
    }, [getStandard]);
    useEffect(() => {
        if (standardData?.standard) {
            handleAdd(standardData.standard);
            handleClose();
        }
    }, [standardData, handleCreateClose]);

    /**
     * Title bar with help button and close icon
     */
    const titleBar = useMemo(() => (
        <Box sx={{
            background: (t) => t.palette.primary.dark,
            color: (t) => t.palette.primary.contrastText,
            display: 'flex',
            alignItems: 'center',
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
                    <CloseIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
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
            {/* Popup for creating a new standard */}
            <BaseObjectDialog
                title={"Create Standard"}
                open={isCreateOpen}
                hasPrevious={false}
                hasNext={false}
                onAction={handleCreateClose}
                session={session}
            >
                <StandardCreate
                    session={session}
                    onCreated={handleCreated}
                    onCancel={handleCreateClose}
                />
            </BaseObjectDialog>
            {titleBar}
            <DialogContent>
                <Stack direction="column" spacing={4}>
                    <Button
                        fullWidth
                        startIcon={<CreateIcon />}
                        onClick={handleCreateOpen}
                    >Create</Button>
                    <Box sx={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                    }}>
                        <Typography variant="h6" sx={{ marginLeft: 'auto', marginRight: 'auto' }}>Or</Typography>
                    </Box>
                    <SearchList
                        searchPlaceholder={'Select existing standard...'}
                        sortOptions={StandardSortOptions}
                        defaultSortOption={standardDefaultSortOption}
                        query={standardsQuery}
                        take={20}
                        searchString={searchString}
                        sortBy={sortBy}
                        timeFrame={timeFrame}
                        noResultsText={"None found. Maybe you should create one?"}
                        setSearchString={setSearchString}
                        setSortBy={setSortBy}
                        setTimeFrame={setTimeFrame}
                        listItemFactory={(node: Standard, index: number) => (
                            <StandardListItem
                                key={`standard-list-item-${index}`}
                                index={index}
                                session={session}
                                data={node}
                                onClick={(_e, selected: Standard) => handeStandardSelect(selected)}
                            />)}
                        getOptionLabel={standardOptionLabel}
                        onObjectSelect={(newValue) => handeStandardSelect(newValue)}
                        session={session}
                    />
                </Stack>
            </DialogContent>
        </Dialog>
    )
}