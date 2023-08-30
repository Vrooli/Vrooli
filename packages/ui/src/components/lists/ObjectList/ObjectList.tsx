import { Bookmark, GqlModelType, isOfType, Reaction, View } from "@local/shared";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { useObjectActions } from "hooks/useObjectActions";
import { useStableCallback } from "hooks/useStableCallback";
import { useStableObject } from "hooks/useStableObject";
import { memo, useCallback, useMemo, useState } from "react";
import { lazily } from "react-lazily";
import { useLocation } from "route";
import { NavigableObject } from "types";
import { ObjectAction } from "utils/actions/objectActions";
import { ListObject } from "utils/display/listTools";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { ActionsType, ListActions, ObjectListItemProps } from "../types";

// Custom list item components
const { ChatListItem } = lazily(() => import("../ChatListItem/ChatListItem"));
const { MemberListItem } = lazily(() => import("../MemberListItem/MemberListItem"));
const { NotificationListItem } = lazily(() => import("../NotificationListItem/NotificationListItem"));
const { ReminderListItem } = lazily(() => import("../ReminderListItem/ReminderListItem"));
const { RunProjectListItem } = lazily(() => import("../RunProjectListItem/RunProjectListItem"));
const { RunRoutineListItem } = lazily(() => import("../RunRoutineListItem/RunRoutineListItem"));
const getListItemComponent = (objectType: `${GqlModelType}` | "CalendarEvent") => {
    switch (objectType) {
        case "Chat": return ChatListItem;
        case "Member": return MemberListItem;
        case "Notification": return NotificationListItem;
        case "Reminder": return ReminderListItem;
        case "RunProject": return RunProjectListItem;
        case "RunRoutine": return RunRoutineListItem;
        default: return ObjectListItemBase;
    }
};

function ObjectListItem<T extends ListObject>({
    objectType,
    ...props
}: ObjectListItemProps<T>) {
    const ListItem = useMemo<(props: any) => JSX.Element>(() => getListItemComponent(objectType), [objectType]);
    return (
        <ListItem
            objectType={objectType}
            {...props}
        />
    );
}

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
} & (T extends keyof ListActions ? ActionsType<ListActions[T & keyof ListActions]> : object);

export const ObjectList = <T extends keyof ListActions | undefined>({
    canNavigate,
    dummyItems,
    keyPrefix,
    hideUpdateButton,
    items,
    loading,
    onClick,
}: ObjectListProps<T>) => {
    const [, setLocation] = useLocation();
    const stableItems = useStableObject(items);
    const stableOnClick = useStableCallback(onClick);

    // Handle context menu
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const [object, setObject] = useState<ListObject | null>(null);
    const handleContextMenu = useCallback((target: EventTarget, object: ListObject | null) => {
        if (!object) return;
        setAnchorEl(target as HTMLElement);
        setObject(object);
    }, []);
    const closeContextMenu = useCallback(() => {
        setAnchorEl(null);
        // Don't remove object, since dialogs opened from context menu may need it
    }, []);
    const actionData = useObjectActions({
        canNavigate,
        object,
        objectType: object?.__typename ?? "Routine" as GqlModelType,
        setLocation,
        setObject,
    });

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
                    handleContextMenu={handleContextMenu}
                    hideUpdateButton={hideUpdateButton}
                    loading={false}
                    objectType={curr.__typename}
                    onClick={stableOnClick}
                />
            );
        });
    }, [stableItems, canNavigate, handleContextMenu, hideUpdateButton, stableOnClick, keyPrefix]);

    // Generate dummy items
    const dummyListItems = useMemo(() => {
        if (loading && dummyItems) {
            return dummyItems.map((dummy, index) => (
                <MemoizedObjectListItem
                    key={`${keyPrefix}-dummy-${index}`}
                    data={null}
                    handleContextMenu={handleContextMenu}
                    hideUpdateButton={hideUpdateButton}
                    loading={true}
                    objectType={dummy}
                />
            ));
        }
        return [];
    }, [loading, dummyItems, keyPrefix, handleContextMenu, hideUpdateButton]);

    return (
        <>
            {/* Context menus */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={anchorEl}
                exclude={[ObjectAction.Comment, ObjectAction.FindInPage]} // Find in page only relevant when viewing object - not in list. And shouldn't really comment without viewing full page
                object={object}
                onClose={closeContextMenu}
            />
            {/* Actual results */}
            {realItems}
            {/* Placeholders while loading more data */}
            {dummyListItems}
        </>
    );
};
