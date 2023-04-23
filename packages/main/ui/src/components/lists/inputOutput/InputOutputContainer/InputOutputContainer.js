import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { AddIcon } from "@local/icons";
import { uuid } from "@local/uuid";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { DragDropContext, Draggable, Droppable } from "react-beautiful-dnd";
import { useTranslation } from "react-i18next";
import { updateArray } from "../../../../utils/shape/general";
import { ColorIconButton } from "../../../buttons/ColorIconButton/ColorIconButton";
import { ContentCollapse } from "../../../containers/ContentCollapse/ContentCollapse";
import { InputOutputListItem } from "../InputOutputListItem/InputOutputListItem";
const AddButton = ({ index, isInput, handleAdd }) => (_jsx(Tooltip, { placement: "top", title: "Add", children: _jsx(ColorIconButton, { id: `add-${isInput ? "input" : "output"}-item-${index}`, color: "inherit", onClick: () => handleAdd(index, {}), "aria-label": "add item", background: "#6daf72", sx: {
            zIndex: 1,
            width: "fit-content",
            margin: "5px auto !important",
            padding: "0",
            marginBottom: "16px !important",
            display: "flex",
            alignItems: "center",
            borderRadius: "100%",
        }, children: _jsx(AddIcon, { fill: 'white' }) }) }));
export const InputOutputContainer = ({ handleUpdate, isEditing, isInput, language, list, zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const sortedList = useMemo(() => {
        return list;
    }, [list]);
    const type = useMemo(() => isInput ? "input" : "output", [isInput]);
    const [isOpenArray, setIsOpenArray] = useState([]);
    useEffect(() => {
        if (sortedList.length > 0 && sortedList.length !== isOpenArray.length) {
            setIsOpenArray(sortedList.map(() => false));
        }
    }, [sortedList, isOpenArray]);
    const handleOpen = useCallback((index) => {
        setIsOpenArray(isOpenArray.map((_, i) => i === index ? true : false));
    }, [isOpenArray]);
    const handleClose = useCallback((index) => {
        setIsOpenArray(isOpenArray.map((_) => false));
    }, [isOpenArray]);
    const onItemAdd = useCallback((index, newItem) => {
        const newIsOpenArray = new Array(sortedList.length + 1).fill(false);
        newIsOpenArray[Math.min(index + 1, sortedList.length)] = true;
        setIsOpenArray(newIsOpenArray);
        const newList = [...sortedList];
        const newItemFormatted = {
            ...newItem,
            id: uuid(),
            name: newItem.name || t(`${isInput ? "Input" : "Output"}NameDefault`, { number: sortedList.length + 1 }),
            standardVersion: newItem.standardVersion || null,
            translations: newItem.translations ? newItem.translations : [{
                    id: uuid(),
                    language,
                    description: "",
                }],
        };
        if (isInput && newItem.isRequired !== true && newItem.isRequired !== false)
            newItemFormatted.isRequired = true;
        const listStart = newList.slice(0, index);
        const listEnd = newList.slice(index);
        const combined = [...listStart, newItemFormatted, ...listEnd];
        handleUpdate(combined);
    }, [sortedList, t, isInput, language, handleUpdate]);
    const onDragEnd = (result) => {
        if (!result.destination)
            return;
        const newList = Array.from(sortedList);
        const [reorderedItem] = newList.splice(result.source.index, 1);
        newList.splice(result.destination.index, 0, reorderedItem);
        handleUpdate(newList);
    };
    const onItemUpdate = useCallback((index, updatedItem) => {
        handleUpdate(updateArray(sortedList, index, updatedItem));
    }, [handleUpdate, sortedList]);
    const onItemDelete = useCallback((index) => {
        setIsOpenArray(isOpenArray.filter((_, i) => i !== index));
        handleUpdate([...sortedList.filter((_, i) => i !== index)]);
    }, [handleUpdate, sortedList, isOpenArray]);
    return (_jsx(_Fragment, { children: _jsx(DragDropContext, { onDragEnd: onDragEnd, children: _jsx(ContentCollapse, { id: `${type}-container`, helpText: t(`${isInput ? "Input" : "Output"}ContainerHelp`), title: isInput ? "Inputs" : "Outputs", sxs: {
                    titleContainer: {
                        display: "flex",
                        justifyContent: "center",
                    },
                    root: {
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                        boxShadow: 2,
                        borderRadius: 2,
                        padding: 0,
                        overflow: "overlay",
                    },
                }, children: _jsx(Droppable, { droppableId: "inputOutputItems", children: (provided) => (_jsxs(Stack, { ref: provided.innerRef, ...provided.droppableProps, direction: "column", sx: {
                            background: palette.background.default,
                        }, children: [sortedList.map((item, index) => (_jsx(Draggable, { draggableId: String(item.id), index: index, children: (provided) => (_jsx(InputOutputListItem, { ref: provided.innerRef, dragProps: provided.draggableProps, dragHandleProps: provided.dragHandleProps, index: index, isInput: isInput, isOpen: isOpenArray.length > index && isOpenArray[index], item: item, isEditing: isEditing, handleOpen: handleOpen, handleClose: handleClose, handleDelete: onItemDelete, handleUpdate: onItemUpdate, language: language, zIndex: zIndex }, `${type}-item-${item.id}`)) }, item.id))), isEditing && _jsx(AddButton, { index: sortedList.length, isInput: isInput, handleAdd: onItemAdd }, `add-${type}-item-${sortedList.length}`), provided.placeholder] })) }) }) }) }));
};
//# sourceMappingURL=InputOutputContainer.js.map