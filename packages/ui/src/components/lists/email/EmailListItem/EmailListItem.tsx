// Used to display popular/search results of a particular object type
import { Box, IconButton, ListItem, ListItemText, Stack, Tooltip } from '@mui/material';
import { EmailListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback } from 'react';
import {
    Check as VerifyIcon,
    Delete as DeleteIcon,
} from '@mui/icons-material';

const Status = {
    NotVerified: '#a71c2d', // Red
    Verified: '#19972b', // Green
}

export function EmailListItem({
    handleDelete,
    handleUpdate,
    handleVerify,
    index,
    data,
}: EmailListItemProps) {

    const onDelete = useCallback(() => {
        handleDelete(data);
    }, [data, handleDelete]);

    const onVerify = useCallback(() => {
        handleVerify(data);
    }, [data, handleVerify]);

    return (
        <ListItem
            disablePadding
            sx={{
                display: 'flex',
                background: index % 2 === 0 ? '#c8d6e9' : '#e9e9e9',
                color: 'black',
                padding: 1,
            }}
        >
            {/* Left informational column */}
            <Stack direction="column" spacing={1} pl={2} sx={{ marginRight: 'auto' }}>
                <ListItemText
                    primary={data.emailAddress}
                    sx={{ ...multiLineEllipsis(1) }}
                />
                {/* Verified indicator */}
                <Box sx={{
                    borderRadius: 1,
                    border: `2px solid ${data.verified ? Status.Verified : Status.NotVerified}`,
                    color: data.verified ? Status.Verified : Status.NotVerified,
                    height: 'fit-content',
                    fontWeight: 'bold',
                    marginTop: 'auto',
                    marginBottom: 'auto',
                    textAlign: 'center',
                    padding: 0.25,
                    width: 'fit-content',
                }}>
                    {data.verified ? "Verified" : "Not Verified"}
                </Box>
            </Stack>
            {/* Right action buttons */}
            <Stack direction="row" spacing={1}>
                {!data.verified && <Tooltip title="Resend email verification">
                    <IconButton
                        onClick={onVerify}
                    >
                        <VerifyIcon sx={{ fill: (t) => t.palette.secondary.main }} />
                    </IconButton>
                </Tooltip>}
                <Tooltip title="Delete Email">
                    <IconButton
                        onClick={onDelete}
                    >
                        <DeleteIcon sx={{ fill: (t) => t.palette.secondary.main }} />
                    </IconButton>
                </Tooltip>
            </Stack>
        </ListItem>
    )
}