import { Button, Stack, useTheme } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { routineVersionIOInitialValues } from "forms/RoutineVersionIOForm/RoutineVersionIOForm";
import { AddIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { DragDropContext, Draggable, Droppable, DropResult } from "react-beautiful-dnd";
import { useTranslation } from "react-i18next";
import { SessionContext } from "utils/SessionContext";
import { updateArray } from "utils/shape/general";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { InputOutputListItem } from "../InputOutputListItem/InputOutputListItem";
import { InputOutputContainerProps } from "../types";

export const InputOutputContainer = ({
    handleUpdate,
    isEditing,
    isInput,
    language,
    list,
    zIndex,
}: InputOutputContainerProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const sortedList = useMemo(() => {
        console.log("in sorted listtt", list);
        return list ?? [];
        // TODO when index added in schema.prisma, sort by index
        //return list.sort((a, b) => a. - b.index);
    }, [list]);

    const type = useMemo(() => isInput ? "input" : "output", [isInput]);
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
        const newItemFormatted: RoutineVersionInputShape | RoutineVersionOutputShape = routineVersionIOInitialValues(session, isInput, {
            ...newItem,
            name: newItem.name || t(`${isInput ? "Input" : "Output"}NameDefault`, { number: sortedList.length + 1 }),
        } as any);
        if (isInput && (newItem as RoutineVersionInputShape).isRequired !== true && (newItem as RoutineVersionInputShape).isRequired !== false) (newItemFormatted as RoutineVersionInputShape).isRequired = true;
        // Add new item to sortedList at index (splice does not work)
        const listStart = newList.slice(0, index);
        const listEnd = newList.slice(index);
        const combined = [...listStart, newItemFormatted, ...listEnd];
        // newList.splice(index + 1, 0, newItemFormatted);
        handleUpdate(combined as any);
    }, [sortedList, session, isInput, t, handleUpdate]);

    const onDragEnd = (result: DropResult) => {
        if (!result.destination) return;
        const newList = Array.from(sortedList);
        const [reorderedItem] = newList.splice(result.source.index, 1);
        newList.splice(result.destination.index, 0, reorderedItem);
        handleUpdate(newList as any);
    };

    const onItemUpdate = useCallback((index: number, updatedItem: RoutineVersionInputShape | RoutineVersionOutputShape) => {
        handleUpdate(updateArray(sortedList, index, updatedItem));
    }, [handleUpdate, sortedList]);

    const onItemDelete = useCallback((index: number) => {
        setIsOpenArray(isOpenArray.filter((_, i) => i !== index));
        handleUpdate([...sortedList.filter((_, i) => i !== index)]);
    }, [handleUpdate, sortedList, isOpenArray]);

    return (
        <>
            {/* Main content */}
            <DragDropContext onDragEnd={onDragEnd}>
                <ContentCollapse
                    id={`${type}-container`}
                    helpText={t(`${isInput ? "Input" : "Output"}ContainerHelp`)}
                    title={isInput ? "Inputs" : "Outputs"}
                    sxs={{
                        titleContainer: {
                            display: "flex",
                            justifyContent: "center",
                            background: palette.primary.light,
                            padding: 0.5,
                            borderBottom: `1px solid ${palette.divider}`,
                        },
                        root: {
                            color: palette.primary.contrastText,
                            border: `1px solid ${palette.divider}`,
                            borderRadius: 2,
                            padding: 0,
                            overflow: "overlay",
                        },
                        helpButton: {
                            fill: palette.mode === "light" ? palette.secondary.light : palette.secondary.dark,
                        },
                    }}
                    zIndex={zIndex}
                >
                    <Droppable droppableId="inputOutputItems">
                        {(provided) => (
                            <Stack
                                ref={provided.innerRef}
                                {...provided.droppableProps}
                                direction="column"
                                sx={{
                                    background: palette.background.paper,
                                }}>
                                {/* List of inputs. If editing, display delete icon next to each and an add button at the bottom */}
                                {sortedList.map((item, index) => (
                                    <Draggable
                                        key={item.id}
                                        draggableId={String(item.id)}
                                        index={index}
                                    >
                                        {(provided) => (
                                            <InputOutputListItem
                                                key={`${type}-item-${item.id}`}
                                                ref={provided.innerRef}
                                                dragProps={provided.draggableProps}
                                                dragHandleProps={provided.dragHandleProps}
                                                index={index}
                                                isInput={isInput}
                                                isOpen={isOpenArray.length > index && isOpenArray[index]}
                                                item={item}
                                                isEditing={isEditing}
                                                handleOpen={handleOpen}
                                                handleClose={handleClose}
                                                handleDelete={onItemDelete}
                                                handleUpdate={onItemUpdate}
                                                language={language}
                                                zIndex={zIndex}
                                            />
                                        )}
                                    </Draggable>
                                ))}
                                {/* Show add button at bottom of sortedList */}
                                {isEditing && <Button
                                    key={`add-${type}-item-${sortedList.length}`}
                                    color="secondary"
                                    onClick={() => { onItemAdd(sortedList.length, {} as any); }}
                                    variant="outlined"
                                    startIcon={<AddIcon />}
                                    sx={{
                                        margin: 1,
                                    }}
                                >{t("Add")}</Button>}
                                {provided.placeholder}
                            </Stack>
                        )}
                    </Droppable>
                </ContentCollapse>
            </DragDropContext>
        </>
    );
};
