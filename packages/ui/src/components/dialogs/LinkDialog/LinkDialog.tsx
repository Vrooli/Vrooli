/**
 * Used to create/update a link between two routine nodes
 */
import {
    Autocomplete,
    Box,
    Button,
    Dialog,
    DialogContent,
    Grid,
    IconButton,
    Stack,
    TextField,
    Typography
} from '@mui/material';
import { HelpButton } from 'components';
import { useCallback, useMemo, useState } from 'react';
import { LinkDialogProps } from '../types';
import { Close as CloseIcon } from '@mui/icons-material';
import { Node } from 'types';
import { getTranslation, Pubs } from 'utils';
import { NodeType } from 'graphql/generated/globalTypes';

const helpText =
    `
TODO
`

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

    const addLink = useCallback(() => {
        if (!fromNode || !toNode) {
            PubSub.publish(Pubs.Snack, { message: 'Please select both from and to nodes', severity: 'error' });
            return;
        }
        handleClose({
            fromId: fromNode.id,
            toId: toNode.id,
        } as any)
    }, [handleClose, fromNode, toNode]);

    const closeDialog = useCallback(() => { handleClose(undefined); }, [handleClose]);

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
     * Calculate the "From" and "To" options
     */
    const { fromOptions, toOptions } = useMemo(() => {
        if (!routine) return { fromOptions: [], toOptions: [] };
        // Initialize options
        let fromNodes: Node[] = routine.nodes;
        let toNodes: Node[] = routine.nodes.filter((node: Node) => node.type !== NodeType.Start); // Can't link to start node
        const existingLinks = routine.nodeLinks;
        // If from node is already selected
        if (fromNode) {
            // Remove it from the "to" options
            toNodes = toNodes.filter(node => node.id !== fromNode.id);
            // Remove all links that already exist
            toNodes = toNodes.filter(node => !existingLinks.some(link => link.fromId === fromNode.id && link.toId === node.id));
        }
        // If to node is already selected
        if (toNode) {
            // Remove it from the "from" options
            fromNodes = fromNodes.filter(node => node.id !== toNode.id);
            // Remove all links that already exist
            fromNodes = fromNodes.filter(node => !existingLinks.some(link => link.fromId === node.id && link.toId === toNode.id));
        }
        return { fromOptions: fromNodes, toOptions: toNodes };
    }, [fromNode, toNode, routine]);

    /**
     * Container that displays "From" and "To" node selections, with right arrow inbetween
     */
    const nodeSelections = useMemo(() => (
        <Stack direction="row" justifyContent="center" alignItems="center">
            {/* From selector */}
            <Autocomplete
                disablePortal
                id="link-connect-from"
                options={fromOptions}
                getOptionLabel={(option: Node) => getTranslation(option, 'title', [language])}
                onChange={(_, value) => handleFromSelect(value as Node)}
                sx={{
                    minWidth: 200,
                    maxWidth: 350,
                }}
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
                options={toOptions}
                getOptionLabel={(option: Node) => getTranslation(option, 'title', [language])}
                onChange={(_, value) => handleToSelect(value as Node)}
                sx={{
                    minWidth: 200,
                    maxWidth: 350
                }}
                renderInput={(params) => <TextField {...params} label="To" />}
            />
        </Stack>
    ), [fromOptions, toOptions, language]);

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
            sx={{
                '& .MuiDialogContent-root': { overflow: 'visible' },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            {titleBar}
            <DialogContent>
                {nodeSelections}
                {conditions}
                {deleteOption}
                {/* Action buttons */}
                <Grid container sx={{ padding: 0, paddingTop: '24px' }}>
                    <Grid item xs={12} sm={6} sx={{ paddingRight: 1 }}>
                        <Button fullWidth type="submit" onClick={addLink}>Add</Button>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                        <Button fullWidth onClick={closeDialog} sx={{ paddingLeft: 1 }}>Cancel</Button>
                    </Grid>
                </Grid>
            </DialogContent>
        </Dialog>
    )
}