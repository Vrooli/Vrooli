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
import { MoveNodeMenuProps } from '../types';
import { Close as CloseIcon } from '@mui/icons-material';

const helpText =
    `This dialog allows you to move a node to a new position.

Nodes are positioned using "column" and "row" coordinates.`

export const MoveNodeMenu = ({
    handleClose,
    isOpen,
    language,
    node,
    routine,
    zIndex,
}: MoveNodeMenuProps) => {
    const { palette } = useTheme();

    const [fromColumnIndex, setFromColumnIndex] = useState<number>(node?.columnIndex ?? 0);
    const [fromRowIndex, setFromRowIndex] = useState<number>(node?.rowIndex ?? 0);
    const [toColumnIndex, setToColumnIndex] = useState<number>(node?.columnIndex ?? 0);
    const [toRowIndex, setToRowIndex] = useState<number>(node?.rowIndex ?? 0);

    const handleToColumnSelect = useCallback((columnIndex: number | null) => {
        setFromColumnIndex(columnIndex ?? 0);
    }, [setFromColumnIndex]);
    const handleToRowSelect = useCallback((rowIndex: number | null) => {
        setFromRowIndex(rowIndex ?? 0);
    }, [setFromRowIndex]);

    useEffect(() => {
        if (node) {
            const { columnIndex, rowIndex } = node;
            setFromColumnIndex(columnIndex ?? 0);
            setToColumnIndex(columnIndex ?? 0);
            setFromRowIndex(rowIndex ?? 0);
            setToRowIndex(rowIndex ?? 0);
        }
    }, [node]);

    const availableColumns = useMemo(() => {
        const lowestColumn = 1; // Can't put in first column
        const highestColumn = routine.nodes.reduce((highest, node) => Math.max(highest, node.columnIndex), 0);
        return new Array(highestColumn - lowestColumn + 1).fill(0).map((_, index) => index + lowestColumn);
    }, [routine.nodes]);

    const availableRows = useMemo(() => {
        const nodesInColumn = routine.nodes.filter(node => node.columnIndex === fromColumnIndex);
        const highestRowNumber = nodesInColumn.reduce((highest, node) => Math.max(highest, node.rowIndex), 0);
        // Find all available numbers between 0 and highestRowNumber + 1
        const availableRows = new Array(highestRowNumber + 2)
            .fill(0).map((_, index) => index)
            .filter(rowIndex => nodesInColumn.every(node => node.rowIndex !== rowIndex));
        return availableRows;
    }, [fromColumnIndex, routine.nodes]);

    /**
     * Before closing, clear inputs
     */
    const onClose = useCallback((newPosition?: { columnIndex: number, rowIndex: number }) => {
        setFromColumnIndex(0);
        setToColumnIndex(0);
        setFromRowIndex(0);
        setToRowIndex(0);
        handleClose(newPosition);
    }, [handleClose]);

    const moveNode = useCallback(() => {
        if (toColumnIndex === fromColumnIndex && toRowIndex === fromRowIndex) {
            onClose();
        }
        onClose({ columnIndex: toColumnIndex, rowIndex: toRowIndex })
    }, [toColumnIndex, fromColumnIndex, toRowIndex, fromRowIndex, onClose]);

    const closeDialog = useCallback(() => { onClose(); }, [onClose]);

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
                'Move Node'
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
    ), [onClose, palette.primary.contrastText, palette.primary.dark]);

    /**
     * Container that displays "From" and "To" sections, with right arrow inbetween
     */
    const formContent = useMemo(() => (
        <Stack direction="row" justifyContent="center" alignItems="center">
            {/* "From" stack */}
            <Stack direction="column" justifyContent="center" alignItems="center">
                <Typography variant="h6" sx={{ color: palette.primary.contrastText }}>
                    From
                </Typography>
                {/* Column TextField (Disabled) */}
                <TextField
                    fullWidth
                    disabled
                    id="node-from-column"
                    label="From"
                    value={fromColumnIndex}
                />
                {/* Row TextField (Disabled) */}
                <TextField
                    fullWidth
                    disabled
                    id="node-to-column"
                    label="To"
                    value={toColumnIndex}
                />
            </Stack>
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
            {/* "To" stack */}
            <Stack direction="column" justifyContent="center" alignItems="center">
                <Typography variant="h6" sx={{ color: palette.primary.contrastText }}>
                    To
                </Typography>
                {/* Column selector */}
                <Autocomplete
                    disablePortal
                    id="node-to-column"
                    options={availableColumns}
                    getOptionLabel={(option: number) => (option + 1) + ''}
                    onChange={(_, value) => handleToColumnSelect(value)}
                    value={toColumnIndex}
                    sx={{
                        minWidth: 200,
                        maxWidth: 350,
                    }}
                    renderInput={(params) => <TextField {...params} label="Column" />}
                />
                {/* Row selector */}
                <Autocomplete
                    disablePortal
                    id="node-to-row"
                    options={availableRows}
                    getOptionLabel={(option: number) => (option + 1) + ''}
                    onChange={(_, value) => handleToRowSelect(value)}
                    value={toRowIndex}
                    sx={{
                        minWidth: 200,
                        maxWidth: 350,
                    }}
                    renderInput={(params) => <TextField {...params} label="Row" />}
                />
            </Stack>
        </Stack>
    ), [palette.primary.contrastText, fromColumnIndex, toColumnIndex, availableRows, availableColumns, toRowIndex, handleToRowSelect, handleToColumnSelect]);

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
                {formContent}
                {/* Action buttons */}
                <Grid container sx={{ padding: 0, paddingTop: '24px' }}>
                    <Grid item xs={12} sm={6} sx={{ paddingRight: 1 }}>
                        <Button fullWidth type="submit" onClick={moveNode}>Move</Button>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                        <Button fullWidth onClick={closeDialog} sx={{ paddingLeft: 1 }}>Cancel</Button>
                    </Grid>
                </Grid>
            </DialogContent>
        </Dialog>
    )
}