import { ListObject } from "@local/shared";
import { useCallback, useState } from "react";

type UseSelectableListReturn<T extends ListObject> = {
    isSelecting: boolean;
    handleToggleSelecting: () => void;
    handleToggleSelect: (item: ListObject) => void;
    selectedData: T[];
    setIsSelecting: (isSelecting: boolean) => void;
    setSelectedData: (selectedData: T[]) => void;
};

export function useSelectableList<T extends ListObject>(): UseSelectableListReturn<T> {
    const [isSelecting, setIsSelecting] = useState(false);
    const [selectedData, setSelectedData] = useState<T[]>([]);

    const handleToggleSelecting = useCallback(() => {
        if (isSelecting) {
            setSelectedData([]);
        }
        setIsSelecting((is) => !is);
    }, [isSelecting]);

    const handleToggleSelect = useCallback((item: ListObject) => {
        setSelectedData((items) => {
            const newItems = [...items];
            const index = newItems.findIndex((i) => i.id === item.id);
            if (index === -1) {
                newItems.push(item as T);
            } else {
                newItems.splice(index, 1);
            }
            return newItems;
        });
    }, []);

    return {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    };
}
