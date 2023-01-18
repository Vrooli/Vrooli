/**
 * Used to create/update a link between two routine nodes
 */
import {
    Autocomplete,
    Box,
    Dialog,
    DialogContent,
    Grid,
    Stack,
    TextField,
    Typography,
    useTheme,
} from '@mui/material';
import { DialogTitle, GridSubmitButtons, SnackSeverity } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { LinkDialogProps } from '../types';
import { getTranslation, NodeShape, PubSub } from 'utils';
import { uuid } from '@shared/uuid';
import { NodeType } from '@shared/consts';
import { useTranslation } from 'react-i18next';

const helpText =
    `This dialog allows you create new links between nodes, which specifies the order in which the nodes are executed.\n\nIn the future, links will also be able to specify conditions, which must be true in order for the path to be available.`;

const titleAria = "link-dialog-title";

export const LinkDialog = ({
    handleClose,
    handleDelete,
    isAdd,
    isOpen,
    language,
    link,
    nodeFrom,
    nodeTo,
    routineVersion,
    zIndex,
}: LinkDialogProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Selected "From" and "To" nodes
    const [fromNode, setFromNode] = useState<NodeShape | null>(nodeFrom ?? null);
    const handleFromSelect = useCallback((node: NodeShape) => {
        setFromNode(node);
    }, [setFromNode]);
    const [toNode, setToNode] = useState<NodeShape | null>(nodeTo ?? null);
    const handleToSelect = useCallback((node: NodeShape) => {
        setToNode(node);
    }, [setToNode]);
    useEffect(() => { setFromNode(nodeFrom ?? null); }, [nodeFrom, setFromNode]);
    useEffect(() => { setToNode(nodeTo ?? null); }, [nodeTo, setToNode]);

    const errors = useMemo(() => {
        const errors: { [key: string]: string } = {};
        if (!fromNode) {
            errors.fromNode = t(`error:FromNodeRequired`, { lng: language });
        }
        if (!toNode) {
            errors.toNode = t(`error:ToNodeRequired`, { lng: language });
        }
        return errors;
    }, [fromNode, language, t, toNode]);

    const addLink = useCallback(() => {
        if (!fromNode || !toNode) {
            PubSub.get().publishSnack({ messageKey: 'SelectFromAndToNodes', severity: SnackSeverity.Error });
            return;
        }
        handleClose({
            id: uuid(),
            from: { id: fromNode.id },
            operation: null, //TODO
            routineVersion: { id: routineVersion.id },
            to: { id: toNode.id },
            whens: [], //TODO
        })
        setFromNode(null);
        setToNode(null);
    }, [fromNode, toNode, handleClose, routineVersion.id]);

    /**
     * Calculate the "From" and "To" options
     */
    const { fromOptions, toOptions } = useMemo(() => {
        if (!routineVersion) return { fromOptions: [], toOptions: [] };
        // Initialize options
        let fromNodes: NodeShape[] = routineVersion.nodes.filter((node) => node.nodeType !== NodeType.Start) as NodeShape[]; // Can't link from end nodes
        let toNodes: NodeShape[] = routineVersion.nodes.filter((node) => node.nodeType !== NodeType.Start) as NodeShape[]; // Can't link to start node
        const existingLinks = routineVersion.nodeLinks;
        // If from node is already selected
        if (fromNode) {
            // Remove it from the "to" options
            toNodes = toNodes.filter(node => node.id !== fromNode.id);
            // Remove all links that already exist
            toNodes = toNodes.filter(node => !existingLinks.some(link => link.from.id === fromNode.id && link.to.id === node.id));
        }
        // If to node is already selected
        if (toNode) {
            // Remove it from the "from" options
            fromNodes = fromNodes.filter(node => node.id !== toNode.id);
            // Remove all links that already exist
            fromNodes = fromNodes.filter(node => !existingLinks.some(link => link.from.id === node.id && link.to.id === toNode.id));
        }
        return { fromOptions: fromNodes, toOptions: toNodes };
    }, [fromNode, routineVersion, toNode]);

    /**
     * Find the text to display for a node
     */
    const getNodeTitle = useCallback((node: NodeShape) => {
        const { name } = getTranslation(node, [language]);
        if (name) return name;
        if (node.nodeType === NodeType.Start) return t(`common:Start`, { lng: language });
        if (node.nodeType === NodeType.End) return t(`common:End`, { lng: language });
        return t(`common:Untitled`, { lng: language });
    }, [language, t]);

    /**
     * Container that displays "From" and "To" node selections, with right arrow inbetween
     */
    const nodeSelections = useMemo(() => (
        <Stack direction="row" justifyContent="center" alignItems="center" pt={2}>
            {/* From selector */}
            <Autocomplete
                disablePortal
                id="link-connect-from"
                options={fromOptions}
                getOptionLabel={(option: NodeShape) => getNodeTitle(option)}
                onChange={(_, value) => handleFromSelect(value as NodeShape)}
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
                <Typography variant="h6" textAlign="center" color={palette.background.textPrimary}>
                    ⮕
                </Typography>
            </Box>
            {/* To selector */}
            <Autocomplete
                disablePortal
                id="link-connect-to"
                options={toOptions}
                getOptionLabel={(option: NodeShape) => getNodeTitle(option)}
                onChange={(_, value) => handleToSelect(value as NodeShape)}
                value={toNode}
                sx={{
                    minWidth: 200,
                    maxWidth: 350
                }}
                renderInput={(params) => <TextField {...params} label="To" />}
            />
        </Stack>
    ), [fromOptions, fromNode, palette.background.textPrimary, toOptions, toNode, getNodeTitle, handleFromSelect, handleToSelect]);

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

    const handleCancel = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Don't close if data entered and clicked outside
        if ((fromNode !== nodeFrom || toNode !== nodeTo) && reason === 'backdropClick') return;
        // Otherwise, close
        setFromNode(null);
        setToNode(null);
        handleClose();
    }, [fromNode, handleClose, nodeFrom, nodeTo, toNode]);

    return (
        <Dialog
            open={isOpen}
            onClose={handleCancel}
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': { overflow: 'visible' },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                title={t(`common:${isAdd ? 'LinkAdd' : 'LinkEdit'}`, { lng: language })}
                helpText={helpText}
                onClose={handleCancel}
            />
            <DialogContent>
                {nodeSelections}
                {conditions}
                {deleteOption}
                {/* Action buttons */}
                <Grid container spacing={2} mt={2}>
                    <GridSubmitButtons
                        errors={errors}
                        isCreate={isAdd}
                        onCancel={handleCancel}
                        onSubmit={addLink}
                    />
                </Grid>
            </DialogContent>
        </Dialog>
    )
}