import {
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { DialogTitle, ShareSiteDialog } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { UserSelectDialogProps } from '../types';
import { User } from 'types';
import { SearchList } from 'components/lists';
import { useLazyQuery } from '@apollo/client';
import { SearchType, userSearchSchema, removeSearchParams } from 'utils';
import { useLocation } from '@shared/route';
import { AddIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';

const helpText =
    `This dialog allows you to connect a user to an object.

The user will have to approve of the request.`

const titleAria = "select-or-invite-user-dialog-title"

export const UserSelectDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    session,
    zIndex,
}: UserSelectDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback(() => {
        // Clear search params
        removeSearchParams(setLocation, [
            ...userSearchSchema.fields.map(f => f.fieldName),
            'advanced',
            'sort',
            'time',
        ]);
        handleClose();
    }, [handleClose, setLocation]);

    // Share dialog
    const [shareDialogOpen, setShareDialogOpen] = useState(false);
    const openShareDialog = useCallback(() => { setShareDialogOpen(true) }, [setShareDialogOpen]);
    const closeShareDialog = useCallback(() => { setShareDialogOpen(false) }, []);

    // If user selected from search, query for full data
    const [getUser, { data: userData }] = useLazyQuery<user, userVariables>(userQuery);
    const fetchFullData = useCallback((user: User) => {
        // Query for full user data, if not already known (would be known if the same user was selected last time)
        if (userData?.user?.id === user.id) {
            handleAdd(userData?.user);
            onClose();
        } else {
            getUser({ variables: { input: { id: user.id } } });
        }
        // Return false so the list item does not navigate
        return false;
    }, [getUser, userData, handleAdd, onClose]);
    useEffect(() => {
        if (userData?.user) {
            handleAdd(userData.user);
            onClose();
        }
    }, [handleAdd, onClose, userData]);

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
                '& .MuiPaper-root': {
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
            {/* Invite user dialog */}
            <ShareSiteDialog
                onClose={closeShareDialog}
                open={shareDialogOpen}
                zIndex={200}
            />
            <DialogTitle
                ariaLabel={titleAria}
                title={'Add User'}
                helpText={helpText}
                onClose={onClose}
            />
            <Stack direction="column" spacing={2}>
                <Stack direction="row" alignItems="center" justifyContent="center">
                    <Typography component="h2" variant="h4">Users</Typography>
                    <Tooltip title="Add new" placement="top">
                        <IconButton
                            size="medium"
                            onClick={openShareDialog}
                            sx={{ padding: 1 }}
                        >
                            <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <SearchList
                    id="user-select-or-create-list"
                    itemKeyPrefix='user-list-item'
                    noResultsText={"No results. Maybe you should invite them?"}
                    beforeNavigation={fetchFullData}
                    searchType={SearchType.User}
                    searchPlaceholder={'Select existing users...'}
                    session={session}
                    take={20}
                    where={{ userId }}
                    zIndex={zIndex}
                />
            </Stack>
        </Dialog>
    )
}