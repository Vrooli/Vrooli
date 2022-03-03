/**
 * Used to create/update a link between two routine nodes
 */
import {
    Autocomplete,
    Box,
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

export const LinkDialog = ({
    anchorEl,
    handleClose,
    handleDelete,
    isAdd,
    routine,
}: LinkDialogProps) => {
    const open = Boolean(anchorEl);

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
            padding: 0.5,
        }}>
            <Stack direction="row" justifyContent="center" alignItems="center">
                <Typography component="h2" variant="h4" textAlign="center">
                    {isAdd ? 'Add Link' : 'Edit Link'}
                </Typography>
                <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
                <IconButton
                    edge="end"
                    onClick={(e) => { handleClose(null) }}
                >
                    <CloseIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                </IconButton>
            </Stack>
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
                getOptionLabel={(option: Node) => option.title}
                sx={{ maxWidth: 300 }}
                renderInput={(params) => <TextField {...params} label="From" />}
            />
            {/* Right arrow */}
            <Box sx={{
                width: '1em',
                height: '1em',
                background: (t) => t.palette.primary.dark,
                color: (t) => t.palette.primary.contrastText,
                borderRadius: '50%',
                margin: 0.5,
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
                getOptionLabel={(option: Node) => option.title}
                sx={{ maxWidth: 300 }}
                renderInput={(params) => <TextField {...params} label="To" />}
            />
        </Stack>
    ), [fromNode, handleFromSelect, handleToSelect, routine, toNode]);

    /**
     * Container for creating link conditions.
     * If any conditions are set, they will be displayed when running the routine
     */
    const conditions = useMemo(() => (null), []); //TODO

    /**
     * Delete node option if editing (i.e. not a new link)
     */
    const deleteOption = useMemo(() => isAdd ? (
        <Link onClick={handleDelete}>
            <Typography
                sx={{
                    color: (t) => t.palette.secondary.dark,
                    display: 'flex',
                    alignItems: 'center',
                }}
            >
                Already have an account? Log in
            </Typography>
        </Link>
    ) : null, [])

    return (
        <Menu
            id='link-dialog'
            disableScrollLock={true}
            autoFocus={true}
            open={open}
            anchorEl={anchorEl}
            anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'center',
            }}
            transformOrigin={{
                vertical: 'top',
                horizontal: 'center',
            }}
            onClose={() => { handleClose(null) }}
            sx={{
                '& .MuiMenu-paper': {
                    background: (t) => t.palette.background.paper
                },
                '& .MuiMenu-list': {
                    paddingTop: '0',
                }
            }}
        >
            {titleBar}
            {nodeSelections}
            {conditions}
            {deleteOption}
        </Menu>
    )
}