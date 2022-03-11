/**
 * Used to create/update a link between two routine nodes
 */
import {
    Autocomplete,
    Box,
    Dialog,
    DialogContent,
    IconButton,
    Link,
    Menu,
    Stack,
    TextField,
    Typography
} from '@mui/material';
import { HelpButton } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { LinkDialogProps } from '../types';
import { Close as CloseIcon } from '@mui/icons-material';
import helpMarkdown from './LinkDialogHelp.md';
import { Node } from 'types';
import { getTranslation } from 'utils';

export const LinkDialog = ({
    handleClose,
    handleDelete,
    isAdd,
    isOpen,
    language,
    link,
    routine,
}: LinkDialogProps) => {
    // Selected "From" and "To" nodes
    const [fromNode, setFromNode] = useState<Node | null>(null);
    const handleFromSelect = useCallback((node: Node) => {
        setFromNode(node);
    }, [setFromNode]);
    const [toNode, setToNode] = useState<Node | null>(null);
    const handleToSelect = useCallback((node: Node) => {
        setToNode(node);
    }, [setToNode]);

    // Parse markdown from .md file
    const [helpText, setHelpText] = useState<string>('');
    useEffect(() => {
        fetch(helpMarkdown).then((r) => r.text()).then((text) => { setHelpText(text) });
    }, []);

    /**
     * Title bar with help button and close icon
     */
    const titleBar = useMemo(() => (
        <Box sx={{
            background: (t) => t.palette.primary.dark,
            color: (t) => t.palette.primary.contrastText,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
        }}>
            <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto' }}>
                {isAdd ? 'Add Link' : 'Edit Link'}
            </Typography>
            <Box sx={{ marginLeft: 'auto' }}>
                <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
                <IconButton
                    edge="start"
                    onClick={(e) => { handleClose() }}
                >
                    <CloseIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                </IconButton>
            </Box>
        </Box>
    ), [])

    /**
     * Container that displays "From" and "To" node selections, with right arrow inbetween
     */
    const nodeSelections = useMemo(() => (
        <Stack direction="row" justifyContent="center" alignItems="center">
            {/* From selector */}
            <Autocomplete
                disablePortal
                id="link-connect-from"
                options={routine?.nodes ?? []}
                getOptionLabel={(option: Node) => getTranslation(option, 'title', [language])}
                sx={{ maxWidth: 300 }}
                renderInput={(params) => <TextField {...params} label="From" />}
            />
            {/* Right arrow */}
            <Box sx={{
                width: '3em',
                height: '3em',
                color: 'black',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
            }}>
                <Typography variant="h6" textAlign="center">
                    ⮕
                </Typography>
            </Box>
            {/* To selector */}
            <Autocomplete
                disablePortal
                id="link-connect-to"
                options={routine?.nodes ?? []}
                getOptionLabel={(option: Node) => getTranslation(option, 'title', [language])}
                sx={{ maxWidth: 300 }}
                renderInput={(params) => <TextField {...params} label="To" />}
            />
        </Stack>
    ), [fromNode, handleFromSelect, handleToSelect, language, routine, toNode]);

    /**
     * Container for creating link conditions.
     * If any conditions are set, they will be displayed when running the routine
     */
    const conditions = useMemo(() => (null), []); //TODO

    /**
     * Delete node option if editing (i.e. not a new link)
     */
    const deleteOption = useMemo(() => isAdd ? (
        <></>
    ) : null, [])

    return (
        <Dialog
            open={isOpen}
            onClose={() => { handleClose() }}
        >
            {titleBar}
            <DialogContent>
                {nodeSelections}
                {conditions}
                {deleteOption}
            </DialogContent>
        </Dialog>
    )
}