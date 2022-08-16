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
import { BaseObjectDialog, HelpButton, standardSearchSchema } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { StandardSelectOrCreateDialogProps } from '../types';
import {
    Add as CreateIcon,
    Close as CloseIcon
} from '@mui/icons-material';
import { Standard } from 'types';
import { SearchList } from 'components/lists';
import { standardQuery, standardsQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { standard, standardVariables } from 'graphql/generated/standard';
import { StandardCreate } from 'components/views/StandardCreate/StandardCreate';
import { ObjectType, parseSearchParams, stringifySearchParams } from 'utils';
import { useLocation } from '@local/route';

const helpText =
    `This dialog allows you to connect a new or existing standard to a routine input/output.
    
    Standards allow for interoperability between routines and external services adhering to the Vrooli protocol.`

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
        // Find all search fields
        const searchFields = [
            ...standardSearchSchema.fields.map(f => f.fieldName),
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
                    onClick={(e) => { onClose() }}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </Box>
        </Box>
    ), [onClose, palette.primary.contrastText, palette.primary.dark])

    return (
        <Dialog
            open={isOpen}
            onClose={onClose}
            scroll="body"
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
                title={"Create Standard"}
                zIndex={zIndex + 1}
            >
                <StandardCreate
                    onCreated={handleCreated}
                    onCancel={handleCreateClose}
                    session={session}
                    zIndex={zIndex + 1}
                />
            </BaseObjectDialog>
            {titleBar}
            <DialogContent>
                <Stack direction="column" spacing={2}>
                    <Stack direction="row" alignItems="center" justifyContent="center">
                        <Typography component="h2" variant="h4">Standards</Typography>
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
                        itemKeyPrefix='standard-list-item'
                        noResultsText={"None found. Maybe you should create one?"}
                        objectType={ObjectType.Standard}
                        onObjectSelect={(newValue) => handeStandardSelect(newValue)}
                        query={standardsQuery}
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