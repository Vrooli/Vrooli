import { Bookmark, GqlModelType, isOfType, Reaction, View } from "@local/shared";
import { memo, useMemo } from "react";
import { NavigableObject } from "types";
import { ListObject } from "utils/display/listTools";
import { useStableCallback } from "utils/hooks/useStableCallback";
import { useStableObject } from "utils/hooks/useStableObject";
import { ObjectListItem } from "../ObjectListItem/ObjectListItem";
import { ActionsType, ListActions, ObjectListItemProps } from "../types";

const MemoizedObjectListItem = memo<ObjectListItemProps<ListObject>>(ObjectListItem, (prevProps, nextProps) => {
    // Add custom comparison if needed. For now, a shallow comparison will suffice.
    return JSON.stringify(prevProps) === JSON.stringify(nextProps);
});

export type ObjectListProps<T extends keyof ListActions | undefined> = {
    /**
     * Callback triggered before the list item is selected (for viewing, editing, adding a comment, etc.). 
     * If the callback returns false, the list item will not be selected.
     */
    canNavigate?: (item: NavigableObject) => boolean | void,
    /** List of dummy items types to display while loading */
    dummyItems?: (GqlModelType | `${GqlModelType}`)[];
    /** True if update button should be hidden */
    hideUpdateButton?: boolean,
    /** The list of item data. Objects like view and star are converted to their respective objects. */
    items?: readonly ListObject[],
    /** Each list item's key will be `${keyPrefix}-${id}` */
    keyPrefix: string,
    /** Whether the list is loading */
    loading: boolean,
    onClick?: (item: NavigableObject) => void,
    zIndex: number,
} & (T extends keyof ListActions ? ActionsType<ListActions[T & keyof ListActions]> : object);

export const ObjectList = <T extends keyof ListActions | undefined>({
    canNavigate,
    dummyItems,
    keyPrefix,
    hideUpdateButton,
    items,
    loading,
    onClick,
    zIndex,
}: ObjectListProps<T>) => {
    const stableItems = useStableObject(items);
    const stableOnClick = useStableCallback(onClick);

    // Generate real list items
    const realItems = useMemo(() => {
        return stableItems?.map((item, index) => {
            let curr = item;
            if (isOfType(curr, "Bookmark", "Reaction", "View")) {
                curr = (curr as Partial<Bookmark | Reaction | View>).to as ListObject;
            }
            return (
                <MemoizedObjectListItem
                    key={`${keyPrefix}-${curr.id}`}
                    canNavigate={canNavigate}
                    data={curr as ListObject}
                    hideUpdateButton={hideUpdateButton}
                    loading={false}
                    objectType={curr.__typename}
                    onClick={stableOnClick}
                    zIndex={zIndex}
                />
            );
        });
    }, [stableItems, canNavigate, hideUpdateButton, stableOnClick, keyPrefix, zIndex]);

    // Generate dummy items
    const dummyListItems = useMemo(() => {
        if (loading && dummyItems) {
            return dummyItems.map((dummy, index) => (
                <MemoizedObjectListItem
                    key={`${keyPrefix}-dummy-${index}`}
                    data={null}
                    hideUpdateButton={hideUpdateButton}
                    loading={true}
                    objectType={dummy}
                    zIndex={zIndex}
                />
            ));
        }
        return [];
    }, [loading, dummyItems, keyPrefix, hideUpdateButton, zIndex]);

    return (
        <>
            {realItems}
            {dummyListItems}
        </>
    );
};
