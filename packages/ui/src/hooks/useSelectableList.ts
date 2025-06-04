import { type ListObject } from "@local/shared";
import { useCallback, useRef, useState } from "react";
import { type UsePressEvent } from "./gestures.js";

type UseSelectableListReturn<T extends ListObject> = {
    isSelecting: boolean;
    handleToggleSelecting: () => unknown;
    handleToggleSelect: (item: ListObject, event?: UsePressEvent) => unknown;
    selectedData: T[];
    setIsSelecting: (isSelecting: boolean) => unknown;
    setSelectedData: (selectedData: T[]) => unknown;
};

export function useSelectableList<T extends ListObject>(items: T[]): UseSelectableListReturn<T> {
    const [isSelecting, setIsSelecting] = useState(false);
    const [selectedData, setSelectedData] = useState<T[]>([]);
    // Track last selected index to support shift-clicking
    const lastSelectedIndex = useRef<number | null>(null);

    const handleToggleSelecting = useCallback(function handleToggleSelectingCallback() {
        if (isSelecting) {
            setSelectedData([]);
            lastSelectedIndex.current = null;
        }
        setIsSelecting((is) => !is);
    }, [isSelecting]);

    const handleToggleSelect = useCallback(function handleToggleSelectCallback(item: ListObject, event?: UsePressEvent) {
        const index = items.findIndex(i => i.id === item.id);
        if (index === -1) return;

        if (event?.shiftKey && lastSelectedIndex.current !== null) { // Check if shift key is pressed
            const start = Math.min(lastSelectedIndex.current, index);
            const end = Math.max(lastSelectedIndex.current, index);

            const itemsToSelect = items.slice(start, end + 1);
            setSelectedData(prevSelected => {
                const newSelected = [...prevSelected];
                itemsToSelect.forEach(i => {
                    if (!newSelected.some(s => s.id === i.id)) {
                        newSelected.push(i as T);
                    }
                });
                return newSelected;
            });
        } else {
            setSelectedData((prevItems) => {
                const newItems = [...prevItems];
                const selectedIndex = newItems.findIndex((i) => i.id === item.id);
                if (selectedIndex === -1) {
                    newItems.push(item as T);
                } else {
                    newItems.splice(selectedIndex, 1);
                }
                return newItems;
            });
            lastSelectedIndex.current = index;
        }
    }, [items]);

    return {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    };
}
