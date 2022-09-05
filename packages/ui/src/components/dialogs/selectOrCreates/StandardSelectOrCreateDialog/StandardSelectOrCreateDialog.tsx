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
import { useCallback, useEffect, useState } from 'react';
import { StandardSelectOrCreateDialogProps } from '../types';
import { Standard } from 'types';
import { SearchList } from 'components/lists';
import { standardQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { standard, standardVariables } from 'graphql/generated/standard';
import { StandardCreate } from 'components/views/Standard/StandardCreate/StandardCreate';
import { SearchType, standardSearchSchema, removeSearchParams } from 'utils';
import { useLocation } from '@shared/route';
import { AddIcon } from '@shared/icons';

const helpText =
    `This dialog allows you to connect a new or existing standard to a routine input/output.
    
    Standards allow for interoperability between routines and external services adhering to the Vrooli protocol.`

const titleAria = 'select-or-create-standard-dialog-title';

export const StandardSelectOrCreateDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    session,
    zIndex,
}: StandardSelectOrCreateDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback(() => {
        // Clear search params
        removeSearchParams(setLocation, [
            ...standardSearchSchema.fields.map(f => f.fieldName),
            'advanced',
            'sort',
            'time',
        ]);
        handleClose();
    }, [handleClose, setLocation]);

    // Create new standard dialog
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const handleCreateOpen = useCallback(() => { setIsCreateOpen(true); }, [setIsCreateOpen]);
    const handleCreated = useCallback((standard: Standard) => {
        setIsCreateOpen(false);
        handleAdd(standard);
        onClose();
    }, [handleAdd, onClose]);
    const handleCreateClose = useCallback(() => {
        setIsCreateOpen(false);
    }, [setIsCreateOpen]);

    // If standard selected from search, query for full data
    const [getStandard, { data: standardData }] = useLazyQuery<standard, standardVariables>(standardQuery);
    const handeStandardSelect = useCallback((standard: Standard) => {
        // Query for full standard data, if not already known (would be known if the same standard was selected last time)
        if (standardData?.standard?.id === standard.id) {
            handleAdd(standardData?.standard);
            onClose();
        } else {
            getStandard({ variables: { input: { id: standard.id } } });
        }
    }, [getStandard, standardData, handleAdd, onClose]);
    useEffect(() => {
        if (standardData?.standard) {
            handleAdd(standardData.standard);
            onClose();
        }
    }, [handleAdd, onClose, handleCreateClose, standardData]);

    return (
        <Dialog
            open={isOpen}
            onClose={onClose}
            scroll="body"
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': { overflow: 'visible', background: palette.background.default },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            {/* Popup for creating a new standard */}
            <BaseObjectDialog
                onAction={handleCreateClose}
                open={isCreateOpen}
                zIndex={zIndex + 1}
            >
                <StandardCreate
                    onCreated={handleCreated}
                    onCancel={handleCreateClose}
                    session={session}
                    zIndex={zIndex + 1}
                />
            </BaseObjectDialog>
            <DialogTitle
                ariaLabel={titleAria}
                helpText={helpText}
                title={'Add Standard'}
                onClose={onClose}
            />
            <DialogContent>
                <Stack direction="column" spacing={2}>
                    <Stack direction="row" alignItems="center" justifyContent="center">
                        <Typography component="h2" variant="h4">Standards</Typography>
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
                        itemKeyPrefix='standard-list-item'
                        noResultsText={"None found. Maybe you should create one?"}
                        searchType={SearchType.Standard}
                        onObjectSelect={(newValue) => handeStandardSelect(newValue)}
                        searchPlaceholder={'Select existing standard...'}
                        session={session}
                        take={20}
                        zIndex={zIndex}
                    />
                </Stack>
            </DialogContent>
        </Dialog>
    )
}