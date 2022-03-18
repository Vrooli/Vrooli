/**
 * Displays metadata about the routine.
 * On the left is a status indicator, which lets you know if the routine is valid or not.
 * In the middle is the title of the routine. Once clicked, the information bar converts to 
 * a text input field, which allows you to edit the title of the routine.
 * To the right is a button to switch to the metadata view/edit component. You can view/edit the 
 * title, descriptions, instructions, inputs, outputs, tags, etc.
 */
import { Box, IconButton, Menu, Stack, Tooltip, Typography } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import {
    Close as CloseIcon,
    Edit as EditIcon,
} from '@mui/icons-material';
import { getTranslation, BuildStatus } from 'utils';
import { BuildInfoContainerProps } from '../types';
import { HelpButton, BuildInfoDialog } from 'components';
import Markdown from 'markdown-to-jsx';
import { noSelect } from 'styles';
import { EditableLabel } from 'components/inputs';

const helpText =
    `## What am I looking at?
Lorem ipsum dolor sit amet consectetur adipisicing elit. 


## How does it work?
Lorem ipsum dolor sit amet consectetur adipisicing elit.
`

/**
 * Status indicator and slider change color to represent routine's status
 */
const STATUS_COLOR = {
    [BuildStatus.Incomplete]: '#cde22c', // Yellow
    [BuildStatus.Invalid]: '#ff6a6a', // Red
    [BuildStatus.Valid]: '#00d51e', // Green
}
const STATUS_LABEL = {
    [BuildStatus.Incomplete]: 'Incomplete',
    [BuildStatus.Invalid]: 'Invalid',
    [BuildStatus.Valid]: 'Valid',
}

const TERTIARY_COLOR = '#95f3cd';

export const BuildInfoContainer = ({
    canEdit,
    handleRoutineAction,
    handleRoutineUpdate,
    handleStartEdit,
    handleTitleUpdate,
    isEditing,
    language,
    routine,
    session,
    status,
}: BuildInfoContainerProps) => {

    /**
     * List of status messages converted to markdown. 
     * If one message, no bullet points. If multiple, bullet points.
     */
    const statusMarkdown = useMemo(() => {
        if (status.messages.length === 0) return 'Routine is valid.';
        if (status.messages.length === 1) {
            return status.messages[0];
        }
        return status.messages.map((s) => {
            return `* ${s}`;
        }).join('\n');
    }, [status]);

    const [statusMenuAnchorEl, setStatusMenuAnchorEl] = useState(null);
    const statusMenuOpen = Boolean(statusMenuAnchorEl);
    const openStatusMenu = useCallback((event) => {
        if (!statusMenuAnchorEl) setStatusMenuAnchorEl(event.currentTarget);
    }, [statusMenuAnchorEl])
    const closeStatusMenu = () => {
        setStatusMenuAnchorEl(null);
    };

    /**
     * Menu displayed when status is clicked
     */
    const statusMenu = useMemo(() => {
        return (
            <Box>
                <Box sx={{ background: (t) => t.palette.primary.dark }}>
                    <IconButton edge="start" color="inherit" onClick={closeStatusMenu} aria-label="close">
                        <CloseIcon sx={{ fill: 'white', marginLeft: '0.5em' }} />
                    </IconButton>
                </Box>
                <Box sx={{ padding: 1 }}>
                    <Markdown>{statusMarkdown}</Markdown>
                </Box>
            </Box>
        )
    }, [statusMarkdown])

    return (
        <Stack
            id="build-routine-information-bar"
            direction="row"
            spacing={2}
            width="100%"
            justifyContent="space-between"
            sx={{
                zIndex: 2,
                paddingTop: '10vh',
                background: (t) => t.palette.primary.light,
                color: (t) => t.palette.primary.contrastText,
            }}
        >
            {/* Status indicator */}
            <Tooltip title='Press for details'>
                <Box onClick={openStatusMenu} sx={{
                    ...noSelect,
                    cursor: 'pointer',
                    borderRadius: 1,
                    border: `2px solid ${STATUS_COLOR[status.code]}`,
                    color: STATUS_COLOR[status.code],
                    height: 'fit-content',
                    fontWeight: 'bold',
                    fontSize: 'larger',
                    padding: 0.5,
                    marginTop: 'auto',
                    marginBottom: 'auto',
                    marginLeft: 2,
                }}>{STATUS_LABEL[status.code]}</Box>
            </Tooltip>
            <Menu
                id='status-menu'
                open={statusMenuOpen}
                disableScrollLock={true}
                anchorEl={statusMenuAnchorEl}
                onClose={closeStatusMenu}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'center',
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'center',
                }}
                sx={{
                    '& .MuiPopover-paper': {
                        background: (t) => t.palette.background.paper,
                        maxWidth: 'min(100vw, 400px)',
                    },
                    '& .MuiMenu-list': {
                        padding: 0,
                    }
                }}
            >
                {statusMenu}
            </Menu>
            {/* Title */}
            <EditableLabel
                canEdit={isEditing}
                handleUpdate={handleTitleUpdate}
                renderLabel={(t) => (
                    <Typography
                        component="h2"
                        variant="h5"
                        textAlign="center"
                    >{t}</Typography>
                )}
                text={getTranslation(routine, 'title', [language], false) ?? 'Loading...'}
            />
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
                {/* Edit button */}
                {canEdit && !isEditing ? (
                    <IconButton aria-label="confirm-title-change" onClick={handleStartEdit} >
                        <EditIcon sx={{ fill: TERTIARY_COLOR }} />
                    </IconButton>
                ) : null}
                {/* Help button */}
                <HelpButton markdown={helpText} sxRoot={{ margin: "auto", marginRight: 1 }} sx={{ color: TERTIARY_COLOR }} />
                {/* Display routine description, insturctionss, etc. */}
                <BuildInfoDialog
                    handleAction={handleRoutineAction}
                    handleUpdate={handleRoutineUpdate}
                    isEditing={isEditing}
                    language={language}
                    routine={routine}
                    session={session}
                    sxs={{ icon: { fill: TERTIARY_COLOR, marginRight: 1 } }}
                />
            </Box>
        </Stack>
    )
};