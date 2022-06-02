import { Box, IconButton, Stack, Tooltip, Typography } from '@mui/material';
import { BaseEdge, HelpButton } from 'components';
import { InputOutputContainerProps } from '../types';
import { Fragment, useCallback, useEffect, useState } from 'react';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { updateArray } from 'utils';
import { InputOutputListItem } from '../InputOutputListItem/InputOutputListItem';
import { RoutineInput, RoutineInputList, RoutineOutput } from 'types';

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
    language,
    list,
    session,
    zIndex,
}: InputOutputContainerProps) => {
    // Store open/close state of each list item
    const [isOpenArray, setIsOpenArray] = useState<boolean[]>([]);
    useEffect(() => {
        if (list.length > 0 && list.length !== isOpenArray.length) {
            setIsOpenArray(list.map(() => false));
        }
    }, [list, isOpenArray]);

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

    const onAdd = useCallback((index: number, newItem: RoutineInput | RoutineOutput) => {
        const newIsOpenArray = new Array(list.length + 1).fill(false);
        newIsOpenArray[Math.min(index + 1, list.length)] = true;
        setIsOpenArray(newIsOpenArray);
        const newList = [...list];
        let newItemFormatted = {
            ...newItem,
            name: newItem.name || `${isInput ? 'Input' : 'Output'} ${list.length + 1}`,
            standard: newItem.standard || undefined,
            translations: newItem.translations ? newItem.translations : [{
                language,
                description: ''
            }],
        } as any;
        if (isInput && (newItem as RoutineInput).isRequired !== true && (newItem as RoutineInput).isRequired !== false) newItemFormatted.isRequired = true;
        newList.splice(index + 1, 0, newItemFormatted);
        handleUpdate(newList as any);
    }, [list, language, isInput, handleUpdate]);

    const onUpdate = useCallback((index: number, updatedItem: RoutineInput | RoutineOutput) => {
        console.log('input output container onupdate', index, updatedItem)
        handleUpdate(updateArray(list, index, updatedItem));
    }, [handleUpdate, list]);

    const onDelete = useCallback((index: number) => {
        setIsOpenArray(isOpenArray.filter((_, i) => i !== index));
        handleUpdate((list as RoutineInputList).filter((_, i) => i !== index));
    }, [handleUpdate, list, isOpenArray]);

    return (
        <Box id={`${isInput ? 'input' : 'output'}-container`} sx={{ position: 'relative' }}>
            <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                {/* Title */}
                <Typography component="h2" variant="h5" textAlign="left">{isInput ? 'Inputs' : 'Outputs'}</Typography>
                {/* Help button */}
                <HelpButton markdown={isInput ? inputHelpText : outputHelpText} />
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
                            language={language}
                            session={session}
                            zIndex={zIndex}
                        />
                        {isEditing && <AddButton
                            key={`add-input-item-${index}`}
                            index={index}
                            isInput={isInput}
                            handleAdd={onAdd}
                        />}
                    </Fragment>
                ))}
                {/* If no items to display, still show add button */}
                {list.length === 0 && isEditing && <AddButton
                    index={list.length}
                    isInput={isInput}
                    handleAdd={onAdd}
                />}
            </Stack>
            {/* Edges displayed between items (and add button) is actually one edge, since it will be a 
                straight line anyway */}
            {((isEditing && list.length > 0) || (!isEditing && list.length > 1)) && <BaseEdge
                containerId={`${isInput ? 'input' : 'output'}-container`}
                fromId={`${isInput ? 'input' : 'output'}-item-0`}
                isEditing={isEditing}
                thiccness={30}
                timeBetweenDraws={100}
                toId={`add-${isInput ? 'input' : 'output'}-item-${list.length - 1}`}
            />}
        </Box>
    )
}