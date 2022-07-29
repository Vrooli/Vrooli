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
    Typography,
    useTheme
} from '@mui/material';
import { HelpButton } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { LinkDialogProps } from '../types';
import { Close as CloseIcon } from '@mui/icons-material';
import { Node, NodeLink } from 'types';
import { getTranslation, PubSub } from 'utils';
import { NodeType } from 'graphql/generated/globalTypes';
import { v4 as uuid } from 'uuid';

const helpText =
`This dialog allows you create new links between nodes, which specifies the order in which the nodes are executed.

In the future, links will also be able to specify conditions, which must be true in order for the path to be available.`;

export const LinkDialog = ({
    handleClose,
    handleDelete,
    isAdd,
    isOpen,
    language,
    link,
    nodeFrom,
    nodeTo,
    routine,
    zIndex,
}: LinkDialogProps) => {
    const { palette } = useTheme();

    // Selected "From" and "To" nodes
    const [fromNode, setFromNode] = useState<Node | null>(nodeFrom ?? null);
    const handleFromSelect = useCallback((node: Node) => {
        setFromNode(node);
    }, [setFromNode]);
    const [toNode, setToNode] = useState<Node | null>(nodeTo ?? null);
    const handleToSelect = useCallback((node: Node) => {
        setToNode(node);
    }, [setToNode]);
    useEffect(() => { setFromNode(nodeFrom ?? null); }, [nodeFrom, setFromNode]);
    useEffect(() => { setToNode(nodeTo ?? null); }, [nodeTo, setToNode]);

    /**
     * Before closing, clear inputs
     */
    const onClose = useCallback((newLink?: NodeLink) => {
        setFromNode(null);
        setToNode(null);
        handleClose(newLink);
    }, [handleClose, setFromNode, setToNode]);

    const addLink = useCallback(() => {
        if (!fromNode || !toNode) {
            PubSub.get().publishSnack({ message: 'Please select both from and to nodes', severity: 'error' });
            return;
        }
        onClose({
            __typename: 'NodeLink',
            id: uuid(),
            fromId: fromNode.id,
            toId: toNode.id,
            operation: null, //TODO
            whens: [], //TODO
        })
    }, [onClose, fromNode, toNode]);

    const closeDialog = useCallback(() => { onClose(undefined); }, [onClose]);

    /**
     * Title bar with help button and close icon
     */
    const titleBar = useMemo(() => (
        <Box sx={{
            background: palette.primary.dark,
            color: palette.primary.contrastText,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
        }}>
            <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto' }}>
                {isAdd ? 'Add Link' : 'Edit Link'}
                <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
            </Typography>
            <Box sx={{ marginLeft: 'auto' }}>
                <IconButton
                    edge="start"
                    onClick={(e) => { onClose() }}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </Box>
        </Box>
    ), [onClose, isAdd, palette.primary.contrastText, palette.primary.dark]);

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
    }, [fromNode, routine, toNode]);

    /**
     * Find the text to display for a node
     */
    const getNodeTitle = useCallback((node: Node) => {
        const title = getTranslation(node, 'title', [language]);
        if (title) return title;
        if (node.type === NodeType.Start) return 'Start';
        if (node.type === NodeType.End) return 'End';
        return 'Untitled';
    }, [language]);

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
                getOptionLabel={(option: Node) => getNodeTitle(option)}
                onChange={(_, value) => handleFromSelect(value as Node)}
                value={fromNode}
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
                getOptionLabel={(option: Node) => getNodeTitle(option)}
                onChange={(_, value) => handleToSelect(value as Node)}
                value={toNode}
                sx={{
                    minWidth: 200,
                    maxWidth: 350
                }}
                renderInput={(params) => <TextField {...params} label="To" />}
            />
        </Stack>
    ), [fromOptions, fromNode, toOptions, toNode, getNodeTitle, handleFromSelect, handleToSelect]);

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
    ) : null, [isAdd]);

    return (
        <Dialog
            open={isOpen}
            onClose={() => { handleClose() }}
            sx={{
                zIndex,
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