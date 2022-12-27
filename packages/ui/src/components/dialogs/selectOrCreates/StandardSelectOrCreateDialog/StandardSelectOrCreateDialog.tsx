import {
    Dialog,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { BaseObjectDialog, DialogTitle } from 'components';
import { useCallback, useEffect, useRef, useState } from 'react';
import { StandardSelectOrCreateDialogProps } from '../types';
import { Standard } from 'types';
import { SearchList } from 'components/lists';
import { useLazyQuery } from '@apollo/client';
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
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((standard: Standard) => {
        // Query for full standard data, if not already known (would be known if the same standard was selected last time)
        if (standardData?.standard?.id === standard.id) {
            handleAdd(standardData?.standard);
            onClose();
        } else {
            queryingRef.current = true;
            getStandard({ variables: { input: { id: standard.id } } });
        }
        // Return false so the list item does not navigate
        return false;
    }, [getStandard, standardData, handleAdd, onClose]);
    useEffect(() => {
        if (standardData?.standard && queryingRef.current) {
            handleAdd(standardData.standard);
            onClose();
        }
        queryingRef.current = false;
    }, [handleAdd, onClose, handleCreateClose, standardData]);

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
                    id="standard-select-or-create-list"
                    itemKeyPrefix='standard-list-item'
                    noResultsText={"None found. Maybe you should create one?"}
                    searchType={SearchType.Standard}
                    beforeNavigation={fetchFullData}
                    searchPlaceholder={'Select existing standard...'}
                    session={session}
                    take={20}
                    zIndex={zIndex}
                />
            </Stack>
        </Dialog>
    )
}