import { Box, IconButton, Stack, Tooltip, Typography } from '@mui/material';
import { HelpButton } from 'components';
import { InputOutputContainerProps } from '../types';
import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { updateArray } from 'utils';
import { InputOutputListItem } from '../InputOutputListItem/InputOutputListItem';
import { RoutineInput, RoutineInputList, RoutineOutput, Standard } from 'types';
import { AddStandardDialog } from 'components/dialogs';

const helpText =
    `TODO`

type Point = {
    x: number;
    y: number;
}

type EdgePositions = {
    top: number;
    left: number;
    width: number;
    height: number;
    from: Point;
    to: Point;
}

interface AddButtonProps {
    index: number;
    isInput: boolean;
    handleAdd: (index: number, newItem: RoutineInput | RoutineOutput) => void;
}
/**
 * Add button for adding new inputs/outputs
 */
const AddButton = ({ index, isInput, handleAdd }: AddButtonProps) => (
    <Tooltip placement="top" title="Add">
        <IconButton
            id={`add-${isInput ? 'input' : 'output'}-item-${index}`}
            color="inherit"
            onClick={() => handleAdd(index, {
                description: '',
                isRequired: true,
                name: '',
                standard: null,
            } as any)}
            aria-label="close"
            sx={{
                zIndex: 1,
                width: 'fit-content',
                margin: '5px auto !important',
                padding: '0',
                marginBottom: '16px !important',
                display: 'flex',
                alignItems: 'center',
                backgroundColor: '#6daf72',
                color: 'white',
                borderRadius: '100%',
                '&:hover': {
                    backgroundColor: '#6daf72',
                    filter: `brightness(110%)`,
                    transition: 'filter 0.2s',
                },
            }}
        >
            <AddIcon />
        </IconButton>
    </Tooltip>
)

export const InputOutputContainer = ({
    handleUpdate,
    isEditing,
    isInput,
    list,
    session,
}: InputOutputContainerProps) => {
    // Store open/close state of each list item
    const [isOpenArray, setIsOpenArray] = useState<boolean[]>([]);

    /**
     * Open item at index, and close all others
     */
    const handleOpen = useCallback((index: number) => {
        console.log('handleOpen', index, isOpenArray);
        setIsOpenArray(isOpenArray.map((_, i) => i === index ? true : false));
    }, [isOpenArray]);

    /**
     * Close all items
     */
    const handleClose = useCallback((index: number) => {
        setIsOpenArray(isOpenArray.map((_) => false));
    }, [isOpenArray]);

    const onAdd = useCallback((index: number, newItem: RoutineInput | RoutineOutput) => {
        console.log('onadd', newItem)
        const newIsOpenArray = new Array(list.length + 1).fill(false);
        newIsOpenArray[index + 1] = true;
        setIsOpenArray(newIsOpenArray);
        console.log('oldlistttt', list)
        const newList = [...list];
        newList.splice(index + 1, 0, newItem as any);
        console.log('newlistttttt', newList)
        handleUpdate(newList as any);
    }, [handleUpdate, list, isOpenArray]);

    const onUpdate = useCallback((index: number, updatedItem: RoutineInput | RoutineOutput) => {
        handleUpdate(updateArray(list, index, updatedItem));
    }, [handleUpdate, list]);

    const onDelete = useCallback((index: number) => {
        console.log('ondelete', index)
        setIsOpenArray(isOpenArray.filter((_, i) => i !== index));
        handleUpdate((list as RoutineInputList).filter((_, i) => i !== index));
    }, [handleUpdate, list, isOpenArray]);

    // Store dimensions of edge
    const [lineDims, setLineDims] = useState<EdgePositions | null>(null);

    /**
     * Calculates absolute position of the center of an object by its ID, relative to this component
     * @returns x and y coordinates of the center point, as well as the start and end x
     */
    const getPoint = (id: string): { x: number, y: number } | null => {
        // Find graph and node
        const container = document.getElementById(`${isInput ? 'input' : 'output'}-container`);
        const item = document.getElementById(id);
        if (!container || !item) {
            console.error('Could not find item to connect to edge', id);
            return null;
        }
        const itemRect = item.getBoundingClientRect();
        const containerRect = container.getBoundingClientRect();
        return {
            x: container.scrollLeft + itemRect.left + (itemRect.width / 2) - containerRect.left,
            y: container.scrollTop + itemRect.top + (itemRect.height / 2) - containerRect.top,
        }
    }

    /**
     * Calculate the start and end position of the line connecting the inputs/outputs. 
     * This signifies that the inputs/outputs are connected and ordered.
     */
    const calculateLineDims = useCallback(() => {
        if (!list.length) {
            setLineDims(null);
            return;
        }
        const padding = 20;
        // From first item
        const fromPoint = getPoint(`${isInput ? 'input' : 'output'}-item-0`);
        // To add button after last item
        const toPoint = getPoint(`add-${isInput ? 'input' : 'output'}-item-${list.length - 1}`);
        if (fromPoint && toPoint) {
            const top = Math.min(fromPoint.y, toPoint.y) - padding;
            const left = Math.min(fromPoint.x, toPoint.x) - padding;
            const width = Math.abs(fromPoint.x - toPoint.x) + (padding * 2);
            const height = Math.abs(fromPoint.y - toPoint.y) + (padding * 2);
            const from = {
                x: padding,
                y: padding
            }
            const to = {
                x: padding,
                y: height - padding,
            }
            setLineDims({ top, left, width, height, from, to });
        }
        else setLineDims(null);
    }, [isInput, list, setLineDims]);

    /**
     * Continually update line position
     */
    const lineRef = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        let delta = 1000; // Milliseconds
        if (list.length > 0) {
            delta = 100;
        }
        lineRef.current = setInterval(() => { calculateLineDims(); }, delta);
        calculateLineDims();
        return () => {
            if (lineRef.current) clearInterval(lineRef.current);
        }
    }, [calculateLineDims, list]);

    /**
     * The line connecting the inputs/outputs
     */
    const edge = useMemo(() => {
        if (!lineDims) return null;
        return (
            <svg
                width={lineDims.width}
                height={lineDims.height}
                style={{
                    zIndex: 0, // Display behind nodes
                    position: "absolute",
                    pointerEvents: "none",
                    top: lineDims.top,
                    left: lineDims.left,
                }}
            >
                {/* Straight line */}
                <line x1={lineDims.from.x} y1={lineDims.from.y} x2={lineDims.to.x} y2={lineDims.to.y} stroke="#9e3984" strokeWidth={4} />
            </svg>
        )
    }, [lineDims])

    // Add standard dialog
    const [addStandardIndex, setAddStandardIndex] = useState<number | null>(null);
    const handleOpenStandardSelect = useCallback((index: number) => { setAddStandardIndex(index); }, []);
    const closeAddStandardDialog = useCallback(() => { setAddStandardIndex(null); }, []);
    const handleAddStandard = useCallback((standard: Standard) => {
        if (addStandardIndex === null) return;
        // Add standard to item at index
        handleUpdate(updateArray(list, addStandardIndex, {
            ...list[addStandardIndex],
            standard
        }));
        closeAddStandardDialog();
    }, [addStandardIndex, closeAddStandardDialog, handleUpdate, list, addStandardIndex]);
    const handleRemoveStandard = useCallback((index: number) => {
        // Remove standard from item at index
        handleUpdate(updateArray(list, index, {
            ...list[index],
            standard: null
        }));
    }, [handleUpdate, list]);

    return (
        <Box id={`${isInput ? 'input' : 'output'}-container`} sx={{ position: 'relative' }}>
            {/* Popup for adding/connecting a new standard */}
            <AddStandardDialog 
                isOpen={addStandardIndex !== null}
                handleAdd={handleAddStandard}
                handleClose={closeAddStandardDialog}
                session={session}
            />
            <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                {/* Title */}
                <Typography component="h2" variant="h5" textAlign="left">{isInput ? 'Inputs' : 'Outputs'}</Typography>
                {/* Help button */}
                <HelpButton markdown={helpText} />
            </Stack>
            <Stack direction="column">
                {/* List of inputs. If editing, display delete icon next to each and an add button at the bottom */}
                {list.map((item, index) => (
                    <Fragment key={index}>
                        <InputOutputListItem
                            key={`input-item-${index}`}
                            index={index}
                            isInput={isInput}
                            isOpen={isOpenArray.length > index && isOpenArray[index]}
                            item={item}
                            isEditing={isEditing}
                            handleOpen={handleOpen}
                            handleClose={handleClose}
                            handleDelete={onDelete}
                            handleUpdate={onUpdate}
                            handleOpenStandardSelect={handleOpenStandardSelect}
                            handleRemoveStandard={handleRemoveStandard}
                        />
                        <AddButton
                            key={`add-input-item-${index}`}
                            index={index}
                            isInput={isInput}
                            handleAdd={onAdd}
                        />
                    </Fragment>
                ))}
                {/* If no items to display, still show add button */}
                {list.length === 0 && <AddButton
                    index={0}
                    isInput={isInput}
                    handleAdd={onAdd}
                />}
            </Stack>
            {/* Line between first item and last add button, so mimic node/edge design of build page */}
            {edge}
        </Box>
    )
}