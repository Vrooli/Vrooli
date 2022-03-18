/**
 * Label that turns into a text input when clicked. 
 * Stores new text until committed.
 */
import { Box, IconButton, Stack, TextField } from '@mui/material';
import { ChangeEvent, useCallback, useEffect, useState } from 'react';
import {
    Cancel as CancelIcon,
    Done as DoneIcon,
} from '@mui/icons-material';
import { EditableLabelProps } from '../types';

export const EditableLabel = ({
    canEdit,
    handleUpdate,
    renderLabel,
    text,
    sxs,
}: EditableLabelProps) => {
    // Stores changed text before committing
    const [changedText, setChangedText] = useState<string>(text);
    useEffect(() => {
        setChangedText(text);
    }, [text]);
    const onTextChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setChangedText(e.target.value);
    }, []);

    // Used for editing the title of the routine
    const [active, setActive] = useState<boolean>(false);
    const toggleActive = useCallback((e: any) => {
        // e.stopPropagation();
        // e.preventDefault();
        if (!canEdit) return;
        setActive(!active)
    }, [active, canEdit]);
    const save = useCallback(() => {
        handleUpdate(changedText);
        setActive(false);
    }, [changedText]);
    const cancel = useCallback(() => {
        setChangedText(text);
        setActive(false);
    }, [text]);

    return active ?
        (<Stack direction="row" spacing={1} alignItems="center" sx={{ ...(sxs?.stack ?? {}) }}>
            {/* Component for editing title */}
            <TextField
                autoFocus
                variant="filled"
                id="title"
                name="title"
                autoComplete="routine-title"
                label="Title"
                value={changedText}
                onChange={onTextChange}
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
            <IconButton aria-label="confirm-title-change" onClick={save} sx={{ margin: 0, marginLeft: 0, padding: 0 }}>
                <DoneIcon sx={{ fill: '#40dd43' }} />
            </IconButton>
            <IconButton aria-label="cancel-title-change" onClick={cancel} sx={{ margin: 0, marginLeft: 0, padding: 0 }}>
                <CancelIcon sx={{ fill: '#ff2a2a' }} />
            </IconButton>
        </Stack>) :
        (<Box sx={{ 
            ...(sxs?.stack ?? {}),
            display: 'flex', 
            alignItems: 'center', 
            cursor: 'pointer', 
            paddingTop: 1, 
            paddingBottom: 1
        }} onClick={toggleActive}>
            {renderLabel(text)}
        </Box>)
};