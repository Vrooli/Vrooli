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
import { OrganizationSelectOrCreateDialogProps } from '../types';
import {
    Add as CreateIcon,
    Close as CloseIcon
} from '@mui/icons-material';
import { Organization } from 'types';
import { organizationDefaultSortOption, OrganizationSortOptions, SearchList } from 'components/lists';
import { organizationQuery, organizationsQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { organization, organizationVariables } from 'graphql/generated/organization';
import { OrganizationCreate } from 'components/views/OrganizationCreate/OrganizationCreate';

const helpText =
    `This dialog allows you to connect a new or existing organization to an object.
    
    This will assign ownership of the object to that organization, instead of your own account.
    
    You must be an admin or owner of the organization for it to appear in the list`

export const OrganizationSelectOrCreateDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    session,
}: OrganizationSelectOrCreateDialogProps) => {
    const { palette } = useTheme();

    // Create new organization dialog
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const handleCreateOpen = useCallback(() => { setIsCreateOpen(true); }, [setIsCreateOpen]);
    const handleCreated = useCallback((organization: Organization) => {
        setIsCreateOpen(false);
        handleAdd(organization);
        handleClose();
    }, [handleAdd, handleClose]);
    const handleCreateClose = useCallback(() => {
        setIsCreateOpen(false);
    }, [setIsCreateOpen]);

    // If organization selected from search, query for full data
    const [getOrganization, { data: organizationData }] = useLazyQuery<organization, organizationVariables>(organizationQuery);
    const handleOrganizationSelect = useCallback((organization: Organization) => {
        // Query for full organization data, if not already known (would be known if the same organization was selected last time)
        if (organizationData?.organization?.id === organization.id) {
            handleAdd(organizationData?.organization);
            handleClose();
        } else {
            getOrganization({ variables: { input: { id: organization.id } } });
        }
    }, [getOrganization, organizationData, handleAdd, handleClose]);
    useEffect(() => {
        if (organizationData?.organization) {
            handleAdd(organizationData.organization);
            handleClose();
        }
    }, [handleAdd, handleClose, handleCreateClose, organizationData]);

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
                {'Add Organization'}
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
            {/* Popup for creating a new organization */}
            <BaseObjectDialog
                hasPrevious={false}
                hasNext={false}
                onAction={handleCreateClose}
                open={isCreateOpen}
                title={"Create Organization"}
            >
                <OrganizationCreate
                    onCreated={handleCreated}
                    onCancel={handleCreateClose}
                    session={session}
                />
            </BaseObjectDialog>
            {titleBar}
            <DialogContent>
                <Stack direction="column" spacing={4}>
                    <SearchList
                        itemKeyPrefix='organization-list-item'
                        defaultSortOption={organizationDefaultSortOption}
                        noResultsText={"None found. Maybe you should create one?"}
                        onObjectSelect={(newValue) => handleOrganizationSelect(newValue)}
                        query={organizationsQuery}
                        searchPlaceholder={'Select existing organization...'}
                        searchString={searchString}
                        setSearchString={setSearchString}
                        session={session}
                        setSortBy={setSortBy}
                        setTimeFrame={setTimeFrame}
                        sortBy={sortBy}
                        sortOptions={OrganizationSortOptions}
                        take={20}
                        timeFrame={timeFrame}
                        where={{ userId: session?.id,}}
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