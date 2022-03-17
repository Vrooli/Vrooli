/**
 * Displays metadata about the routine.
 * On the left is a status indicator, which lets you know if the routine is valid or not.
 * In the middle is the title of the routine. Once clicked, the information bar converts to 
 * a text input field, which allows you to edit the title of the routine.
 * To the right is a button to switch to the metadata view/edit component. You can view/edit the 
 * title, descriptions, instructions, inputs, outputs, tags, etc.
 */
import { Box, IconButton, Menu, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Close as CloseIcon,
    Done as DoneIcon,
    Edit as EditIcon,
} from '@mui/icons-material';
import { getTranslation, BuildStatus } from 'utils';
import { BuildInfoContainerProps } from '../types';
import { HelpButton, BuildInfoDialog } from 'components';
import Markdown from 'markdown-to-jsx';
import { noSelect } from 'styles';

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
    // Stores changed title before committing
    const [changedTitle, setChangedTitle] = useState<string | null | undefined>(getTranslation(routine, 'title', [language], false));
    useEffect(() => {
        setChangedTitle(getTranslation(routine, 'title', [language], false));
    }, [routine?.translations, language]);

    // Used for editing the title of the routine
    const [titleActive, setTitleActive] = useState<boolean>(false);
    const toggleTitle = useCallback(() => setTitleActive(a => !a), []);
    const saveTitle = useCallback(() => { handleTitleUpdate(changedTitle ?? '') }, [changedTitle]);
    const cancelTitle = useCallback(() => {
        setChangedTitle(getTranslation(routine, 'title', [language], false));
        setTitleActive(false);
    }, [routine?.translations]);

    const titleObject = useMemo(() => {
        return titleActive ?
            (<Stack direction="row" spacing={1} alignItems="center">
                {/* Component for editing title */}
                <TextField
                    autoFocus
                    variant="filled"
                    id="title"
                    name="title"
                    autoComplete="routine-title"
                    label="Title"
                    value={changedTitle}
                    onChange={() => { }}
                    sx={{
                        marginTop: 1,
                        marginBottom: 1,
                        '& .MuiInputLabel-root': {
                            display: 'none',
                        },
                        '& .MuiInputBase-root': {
                            borderBottom: 'none',
                            borderRadius: '32px',
                            border: `2px solid green`,//TODO titleValid ? green : red
                            overflow: 'overlay',
                        },
                        '& .MuiInputBase-input': {
                            position: 'relative',
                            backgroundColor: '#ffffff94',
                            border: '1px solid #ced4da',
                            fontSize: 16,
                            width: 'auto',
                            padding: '8px 8px',
                        }
                    }}
                />
                {/* Buttons for confirm/cancel */}
                <IconButton aria-label="confirm-title-change" onClick={saveTitle}>
                    <DoneIcon sx={{ fill: '#40dd43' }} />
                </IconButton>
                <IconButton aria-label="cancel-title-change" onClick={cancelTitle} color="secondary">
                    <CloseIcon sx={{ fill: '#ff2a2a' }} />
                </IconButton>
            </Stack>) :
            (<Box sx={{ display: 'flex', alignItems: 'center', cursor: 'pointer', paddingTop: 1, paddingBottom: 1 }} onClick={toggleTitle}>
                <Typography
                    component="h2"
                    variant="h5"
                    textAlign="center"
                >{changedTitle ?? 'Loading...'}</Typography>
            </Box>)
    }, [titleActive, changedTitle]);

    /**
     * List of status messages converted to markdown. 
     * If one message, no bullet points. If multiple, bullet points.
     */
    const statusMarkdown = useMemo(() => {
        console.log('in statustoMarkdown', status);
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
            {titleObject}
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