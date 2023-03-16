import { Stack, Tooltip, useTheme } from '@mui/material';
import { AddIcon } from '@shared/icons';
import { uuid } from '@shared/uuid';
import { ColorIconButton } from 'components/buttons/ColorIconButton/ColorIconButton';
import { ContentCollapse } from 'components/containers/ContentCollapse/ContentCollapse';
import { ReorderInputDialog } from 'components/dialogs/ReorderInputDialog/ReorderInputDialog';
import { Fragment, useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { updateArray } from 'utils/shape/general';
import { RoutineVersionInputShape } from 'utils/shape/models/routineVersionInput';
import { RoutineVersionOutputShape } from 'utils/shape/models/routineVersionOutput';
import { InputOutputListItem } from '../InputOutputListItem/InputOutputListItem';
import { InputOutputContainerProps } from '../types';

interface AddButtonProps {
    index: number;
    isInput: boolean;
    handleAdd: (index: number, newItem: RoutineVersionInputShape | RoutineVersionOutputShape) => void;
}
/**
 * Add button for adding new inputs/outputs
 */
const AddButton = ({ index, isInput, handleAdd }: AddButtonProps) => (
    <Tooltip placement="top" title="Add">
        <ColorIconButton
            id={`add-${isInput ? 'input' : 'output'}-item-${index}`}
            color="inherit"
            onClick={() => handleAdd(index, {} as any)}
            aria-label="add item"
            background="#6daf72"
            sx={{
                zIndex: 1,
                width: 'fit-content',
                margin: '5px auto !important',
                padding: '0',
                marginBottom: '16px !important',
                display: 'flex',
                alignItems: 'center',
                borderRadius: '100%',
            }}
        >
            <AddIcon fill='white' />
        </ColorIconButton>
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
    const { t } = useTranslation();

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

    const onItemAdd = useCallback((index: number, newItem: RoutineVersionInputShape | RoutineVersionOutputShape) => {
        const newIsOpenArray = new Array(sortedList.length + 1).fill(false);
        newIsOpenArray[Math.min(index + 1, sortedList.length)] = true;
        setIsOpenArray(newIsOpenArray);
        const newList = [...sortedList];
        let newItemFormatted: RoutineVersionInputShape | RoutineVersionOutputShape = {
            ...newItem,
            id: uuid(),
            name: newItem.name || t(`${isInput ? 'Input' : 'Output'}NameDefault`, { number: sortedList.length + 1 }),
            standardVersion: newItem.standardVersion || null,
            translations: newItem.translations ? newItem.translations : [{
                id: uuid(),
                language,
                description: ''
            }] as any,
        };
        if (isInput && (newItem as RoutineVersionInputShape).isRequired !== true && (newItem as RoutineVersionInputShape).isRequired !== false) (newItemFormatted as RoutineVersionInputShape).isRequired = true;
        // Add new item to sortedList at index (splice does not work)
        const listStart = newList.slice(0, index);
        const listEnd = newList.slice(index);
        const combined = [...listStart, newItemFormatted, ...listEnd];
        // newList.splice(index + 1, 0, newItemFormatted);
        handleUpdate(combined as any);
    }, [sortedList, t, isInput, language, handleUpdate]);

    const onItemReorder = useCallback((fromIndex: number, toIndex: number) => {
        const newList = [...sortedList];
        const [removed] = newList.splice(fromIndex, 1);
        newList.splice(toIndex, 0, removed);
        handleUpdate(newList as any);
    }, [handleUpdate, sortedList]);

    const onItemUpdate = useCallback((index: number, updatedItem: RoutineVersionInputShape | RoutineVersionOutputShape) => {
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
                helpText={t(`${isInput ? 'Input' : 'Output'}ContainerHelp`)}
                title={isInput ? 'Inputs' : 'Outputs'}
                sxs={{
                    titleContainer: {
                        display: 'flex',
                        justifyContent: 'center'
                    },
                    root: {
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                        boxShadow: 2,
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