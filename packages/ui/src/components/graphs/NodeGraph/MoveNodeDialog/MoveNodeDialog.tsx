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
    Stack,
    TextField,
    Typography,
    useTheme
} from '@mui/material';
import { CancelIcon, SaveIcon } from '@shared/icons';
import { DialogTitle } from 'components/dialogs';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { MoveNodeMenuProps } from '../types';

const helpText =
    `This dialog allows you to move a node to a new position.

Nodes are positioned using "column" and "row" coordinates.`

const titleAria = 'move-node-dialog-title';

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
            setToColumnIndex((columnIndex ?? 0) + 1);
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

    // Update to row index when available rows change
    useEffect(() => {
        if (availableRows.length > 0) {
            setToRowIndex(availableRows[0]);
        } else {
            setToRowIndex(0);
        }
    }, [availableRows, setToRowIndex]);

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
     * Container that displays "From" and "To" sections, with right arrow inbetween
     */
    const formContent = useMemo(() => (
        <Stack direction="row" justifyContent="center" alignItems="center" sx={{ color: palette.background.textPrimary }}>
            {/* "From" stack */}
            <Stack direction="column" spacing={2} justifyContent="center" alignItems="center">
                <Typography variant="h6">
                    From
                </Typography>
                {/* Column TextField (Disabled) */}
                <TextField
                    fullWidth
                    disabled
                    id="node-from-column"
                    label="Column"
                    value={fromColumnIndex + 1}
                />
                {/* Row TextField (Disabled) */}
                <TextField
                    fullWidth
                    disabled
                    id="node-from-row"
                    label="Row"
                    value={fromRowIndex + 1}
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
            <Stack direction="column" spacing={2} justifyContent="center" alignItems="center">
                <Typography variant="h6">
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
    ), [palette.background.textPrimary, fromColumnIndex, fromRowIndex, availableColumns, toColumnIndex, availableRows, toRowIndex, handleToColumnSelect, handleToRowSelect]);

    return (
        <Dialog
            open={isOpen}
            onClose={() => { handleClose() }}
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': { overflow: 'visible' },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                helpText={helpText}
                onClose={onClose}
                title="Move Node"
            />
            <DialogContent>
                {formContent}
                {/* Action buttons */}
                <Grid container sx={{ padding: 0, paddingTop: '24px' }}>
                    <Grid item xs={12} sm={6} sx={{ paddingRight: 1 }}>
                        <Button
                            fullWidth
                            type="submit"
                            onClick={moveNode}
                            startIcon={<SaveIcon />}
                        >Save</Button>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                        <Button
                            fullWidth
                            onClick={closeDialog}
                            sx={{ paddingLeft: 1 }}
                            startIcon={<CancelIcon />}
                        >Cancel</Button>
                    </Grid>
                </Grid>
            </DialogContent>
        </Dialog>
    )
}