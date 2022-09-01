import { IconButton, Stack, Tooltip, useTheme } from '@mui/material';
import { ContentCollapse } from 'components';
import { InputOutputContainerProps } from '../types';
import { Fragment, useCallback, useEffect, useMemo, useState } from 'react';
import { InputShape, OutputShape, updateArray } from 'utils';
import { InputOutputListItem } from '../InputOutputListItem/InputOutputListItem';
import { RoutineInput, RoutineOutput } from 'types';
import { v4 as uuid } from 'uuid';
import { AddIcon } from '@shared/icons';
import { ReorderInputDialog } from 'components/dialogs/ReorderInputDialog/ReorderInputDialog';

const inputHelpText =
    `Inputs specify the information required to complete a routine. 
    
For example, a routine for creating a business entity might require its name, mission statement, wallet address, and a logo.

An input may also be associated with a standard, which specifies the shape of data that is expected to be entered. 
This is used to generate a form component when executing the routine manually, or to ensure interoperability with other rotines 
when executing the routine automatically.

Inputs and outputs with the same identifier and standard can be reused in different subroutines.`

const outputHelpText =
    `Outputs specify the information that is generated by a routine.  
    
For example, a routine for creating a business entity might generate a governance token.

Outputs may either be sent to the Vrooli protocol, or stored as a reference to another location. The governance token, for example, 
would be sent directly to the business's wallet address. This means the output would be JSON indicating the object type and token's address.

Outputs and inputs with the same identifier and standard can be reused in different subroutines.`

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
            onClick={() => handleAdd(index, {} as any)}
            aria-label="add item"
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
    language,
    list,
    session,
    zIndex,
}: InputOutputContainerProps) => {
    const { palette } = useTheme();

    const sortedList = useMemo(() => {
        return list;
        // TODO when index added in schema.prisma, sort by index
        //return list.sort((a, b) => a. - b.index);
    } , [list]);

    const type = useMemo(() => isInput ? 'input' : 'output', [isInput]);
    // Store open/close state of each sortedList item
    const [isOpenArray, setIsOpenArray] = useState<boolean[]>([]);
    useEffect(() => {
        if (sortedList.length > 0 && sortedList.length !== isOpenArray.length) {
            setIsOpenArray(sortedList.map(() => false));
        }
    }, [sortedList, isOpenArray]);

    /**
     * Open item at index, and close all others
     */
    const handleOpen = useCallback((index: number) => {
        setIsOpenArray(isOpenArray.map((_, i) => i === index ? true : false));
    }, [isOpenArray]);

    /**
     * Close all items
     */
    const handleClose = useCallback((index: number) => {
        setIsOpenArray(isOpenArray.map((_) => false));
    }, [isOpenArray]);

    const onItemAdd = useCallback((index: number, newItem: RoutineInput | RoutineOutput) => {
        const newIsOpenArray = new Array(sortedList.length + 1).fill(false);
        newIsOpenArray[Math.min(index + 1, sortedList.length)] = true;
        setIsOpenArray(newIsOpenArray);
        const newList = [...sortedList];
        let newItemFormatted: RoutineInput | RoutineOutput = {
            ...newItem,
            id: uuid(),
            name: newItem.name || `${isInput ? 'Input' : 'Output'} ${sortedList.length + 1}`,
            standard: newItem.standard || null,
            translations: newItem.translations ? newItem.translations : [{
                id: uuid(),
                language,
                description: ''
            }] as any,
        };
        if (isInput && (newItem as RoutineInput).isRequired !== true && (newItem as RoutineInput).isRequired !== false) (newItemFormatted as RoutineInput).isRequired = true;
        // Add new item to sortedList at index (splice does not work)
        const listStart = newList.slice(0, index);
        const listEnd = newList.slice(index);
        const combined = [...listStart, newItemFormatted, ...listEnd];
        // newList.splice(index + 1, 0, newItemFormatted);
        handleUpdate(combined as any);
    }, [sortedList, language, isInput, handleUpdate]);

    const onItemReorder = useCallback((fromIndex: number, toIndex: number) => {
        console.log('onItemReorder', fromIndex, toIndex);
        const newList = [...sortedList];
        console.log('original sortedList', newList);
        const [removed] = newList.splice(fromIndex, 1);
        newList.splice(toIndex, 0, removed);
        console.log('new sortedList', newList);
        handleUpdate(newList as any);
    }, [handleUpdate, sortedList]);

    const onItemUpdate = useCallback((index: number, updatedItem: InputShape | OutputShape) => {
        handleUpdate(updateArray(sortedList, index, updatedItem));
    }, [handleUpdate, sortedList]);

    const onItemDelete = useCallback((index: number) => {
        setIsOpenArray(isOpenArray.filter((_, i) => i !== index));
        handleUpdate([...sortedList.filter((_, i) => i !== index)]);
    }, [handleUpdate, sortedList, isOpenArray]);

    // Move item dialog
    const [moveStartIndex, setMoveStartIndex] = useState<number>(-1);
    const openMoveDialog = useCallback((index: number) => { setMoveStartIndex(index); }, []);
    const closeMoveDialog = useCallback((toIndex?: number) => {
        console.log('close move dialog', toIndex);
        if (toIndex !== undefined && toIndex >= 0) {
            onItemReorder(moveStartIndex, toIndex);
            // If the item was open, open the new item
            if (isOpenArray[moveStartIndex]) {
                handleOpen(toIndex);
            }
        }
        setMoveStartIndex(-1);
    }, [onItemReorder, moveStartIndex, isOpenArray, handleOpen]);

    return (
        <>
            {/* Dialog for reordering item */}
            <ReorderInputDialog
                isInput={isInput}
                handleClose={closeMoveDialog}
                listLength={sortedList.length}
                startIndex={moveStartIndex}
                zIndex={zIndex + 1}
            />
            {/* Main content */}
            <ContentCollapse
                id={`${type}-container`}
                helpText={isInput ? inputHelpText : outputHelpText}
                title={isInput ? 'Inputs' : 'Outputs'}
                sxs={{
                    titleContainer: {
                        display: 'flex',
                        justifyContent: 'center'
                    },
                    root: {
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                        border: `1px solid ${palette.background.textPrimary}`,
                        borderRadius: 2,
                        padding: 0,
                        overflow: 'overlay',
                    }
                }}
            >
                <Stack direction="column" sx={{
                    background: palette.background.default,
                }}>
                    {/* List of inputs. If editing, display delete icon next to each and an add button at the bottom */}
                    {sortedList.map((item, index) => (
                        <Fragment key={index}>
                            <InputOutputListItem
                                key={`${type}-item-${item.id}`}
                                index={index}
                                isInput={isInput}
                                isOpen={isOpenArray.length > index && isOpenArray[index]}
                                item={item}
                                isEditing={isEditing}
                                handleOpen={handleOpen}
                                handleClose={handleClose}
                                handleDelete={onItemDelete}
                                handleReorder={openMoveDialog}
                                handleUpdate={onItemUpdate}
                                language={language}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Fragment>
                    ))}
                    {/* Show add button at bottom of sortedList */}
                    {isEditing && <AddButton
                        key={`add-${type}-item-${sortedList.length}`}
                        index={sortedList.length}
                        isInput={isInput}
                        handleAdd={onItemAdd}
                    />}
                </Stack>
            </ContentCollapse>
        </>
    )
}