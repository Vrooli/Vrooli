/**
 * Displays metadata about the routine orchestration.
 * On the left is a status indicator, which lets you know if the routine is valid or not.
 * In the middle is the title of the routine. Once clicked, the information bar converts to 
 * a text input field, which allows you to edit the title of the routine.
 * To the right is a button to switch to the metadata view/edit component. You can view/edit the 
 * title, descriptions, instructions, inputs, outputs, tags, etc.
 */
import { Box, IconButton, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Close as CloseIcon,
    Done as DoneIcon,
    Edit as EditIcon,
} from '@mui/icons-material';
import { OrchestrationStatus } from 'utils';
import { OrchestrationInfoContainerProps } from '../types';
import helpMarkdown from './OrchestratorHelp.md';
import { HelpButton, OrchestrationInfoDialog } from 'components';

/**
 * Status indicator and slider change color to represent orchestration's status
 */
const STATUS_COLOR = {
    [OrchestrationStatus.Incomplete]: '#cde22c', // Yellow
    [OrchestrationStatus.Invalid]: '#ff6a6a', // Red
    [OrchestrationStatus.Valid]: '#00d51e', // Green
}
const STATUS_LABEL = {
    [OrchestrationStatus.Incomplete]: 'Incomplete',
    [OrchestrationStatus.Invalid]: 'Invalid',
    [OrchestrationStatus.Valid]: 'Valid',
}

const TERTIARY_COLOR = '#95f3cd';

export const OrchestrationInfoContainer = ({
    canEdit,
    handleStartEdit,
    isEditing,
    status,
    routine,
    handleRoutineUpdate,
    handleTitleUpdate,
}: OrchestrationInfoContainerProps) => {
    // Stores changed title before committing
    const [changedTitle, setChangedTitle] = useState<string | null | undefined>(routine?.title);
    useEffect(() => {
        setChangedTitle(routine?.title);
    }, [routine?.title]);

    // Used for editing the title of the routine
    const [titleActive, setTitleActive] = useState<boolean>(false);
    const toggleTitle = useCallback(() => setTitleActive(a => !a), []);
    const saveTitle = useCallback(() => { handleTitleUpdate(changedTitle ?? '') }, [changedTitle]);
    const cancelTitle = useCallback(() => {
        setChangedTitle(routine?.title);
        setTitleActive(false);
    }, [routine?.title]);

    // Parse markdown from .md file
    const [helpText, setHelpText] = useState<string>('');
    useEffect(() => {
        fetch(helpMarkdown).then((r) => r.text()).then((text) => { setHelpText(text) });
    }, []);

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

    return (
        <Stack
            id="orchestration-information-bar"
            direction="row"
            spacing={2}
            width="100%"
            justifyContent="space-between"
            sx={{
                zIndex: 2,
                background: (t) => t.palette.primary.light,
                color: (t) => t.palette.primary.contrastText,
            }}
        >
            {/* Status indicator */}
            <Tooltip title={status.details}>
                <Box sx={{
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
                {/* Switch to routine metadata page */}
                <OrchestrationInfoDialog
                    sxs={{ icon: { fill: TERTIARY_COLOR, marginRight: 1 } }}
                    isEditing={isEditing}
                    routineInfo={routine}
                    handleUpdate={handleRoutineUpdate}
                />
            </Box>
        </Stack>
    )
};