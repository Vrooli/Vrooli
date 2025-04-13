import { Bookmark, ListObject, ModelType, OrArray, Reaction, View, isOfType, noop } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { lazily } from "react-lazily";
import { UsePressEvent } from "../../../hooks/gestures.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useDimensions } from "../../../hooks/useDimensions.js";
import { useObjectContextMenu } from "../../../hooks/useObjectContextMenu.js";
import { useStableCallback } from "../../../hooks/useStableCallback.js";
import { useStableObject } from "../../../hooks/useStableObject.js";
import { useLocation } from "../../../route/router.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { ObjectActionMenu } from "../../dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { ObjectListItemProps } from "../types.js";

// Custom list item components
const { BookmarkListListItem } = lazily(() => import("../BookmarkListListItem/BookmarkListListItem.js"));
const { ChatListItem } = lazily(() => import("../ChatListItem/ChatListItem.js"));
const { ChatInviteListItem } = lazily(() => import("../ChatInviteListItem/ChatInviteListItem.js"));
const { ChatParticipantListItem } = lazily(() => import("../ChatParticipantListItem/ChatParticipantListItem.js"));
const { MeetingInviteListItem } = lazily(() => import("../MeetingInviteListItem/MeetingInviteListItem.js"));
const { MemberInviteListItem } = lazily(() => import("../MemberInviteListItem/MemberInviteListItem.js"));
const { MemberListItem } = lazily(() => import("../MemberListItem/MemberListItem.js"));
const { NotificationListItem } = lazily(() => import("../NotificationListItem/NotificationListItem.js"));
const { ReminderListItem } = lazily(() => import("../ReminderListItem/ReminderListItem.js"));
const { RunProjectListItem } = lazily(() => import("../RunProjectListItem/RunProjectListItem.js"));
const { RunRoutineListItem } = lazily(() => import("../RunRoutineListItem/RunRoutineListItem.js"));
function getListItemComponent(objectType: `${ModelType}` | "CalendarEvent") {
    switch (objectType) {
        case "BookmarkList": return BookmarkListListItem;
        case "Chat": return ChatListItem;
        case "ChatInvite": return ChatInviteListItem;
        case "ChatParticipant": return ChatParticipantListItem;
        case "MeetingInvite": return MeetingInviteListItem;
        case "MemberInvite": return MemberInviteListItem;
        case "Member": return MemberListItem;
        case "Notification": return NotificationListItem;
        case "Reminder": return ReminderListItem;
        case "RunProject": return RunProjectListItem;
        case "RunRoutine": return RunRoutineListItem;
        default: return ObjectListItemBase;
    }
}

export function ObjectListItem<T extends ListObject>({
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

type ObjectListItemPropsForMultiple<T extends OrArray<ListObject>> = T extends ListObject ?
    ObjectListItemProps<T> :
    T extends Array<ListObject> ?
    ObjectListItemProps<T[number]> :
    never;

const KEYBOARD_KEYS = {
    DOWN: 'ArrowDown',
    UP: 'ArrowUp',
    ENTER: 'Enter',
    ESCAPE: 'Escape',
    HOME: 'Home',
    END: 'End',
};

export type ObjectListProps<T extends OrArray<ListObject>> = Pick<ObjectListItemPropsForMultiple<T>, "canNavigate" | "hideUpdateButton" | "loading" | "onAction" | "onClick"> & {
    /** List of dummy items types to display while loading */
    dummyItems?: (ModelType | `${ModelType}`)[];
    /** Enable keyboard navigation between list items */
    enableKeyboardNavigation?: boolean;
    handleToggleSelect?: (item: ListObject, event?: UsePressEvent) => unknown,
    /** True if list can be reordered (e.g. resource list) */
    isListReorderable?: boolean;
    /** The list of item data. Objects like view and star are converted to their respective objects. */
    items?: readonly ListObject[],
    /** Hides individual list item actions and makes items selectable for bulk actions (e.g. deleting multiple items at once)  */
    isSelecting?: boolean,
    /** Each list item's key will be `${keyPrefix}-${id}` */
    keyPrefix: string,
    /** Items currently selected. Ignored if `isSelecting` is false. */
    selectedItems?: readonly ListObject[],
};

const contextActionsExcluded = [ObjectAction.Comment, ObjectAction.FindInPage]; // Find in page only relevant when viewing object - not in list. And shouldn't really comment without viewing full page

export function ObjectList<T extends OrArray<ListObject>>({
    canNavigate,
    dummyItems,
    enableKeyboardNavigation,
    keyPrefix,
    handleToggleSelect,
    hideUpdateButton,
    isListReorderable,
    isSelecting,
    items,
    loading,
    onAction,
    onClick,
    selectedItems,
}: ObjectListProps<T>) {
    const { breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const stableItems = useStableObject(items);
    const stableOnAction = useStableCallback(onAction);
    const [focusedIndex, setFocusedIndex] = useState<number>(-1);
    const listRef = useRef<HTMLDivElement>(null);

    const { dimensions, ref: dimRef } = useDimensions();
    const isMobile = useMemo(() => dimensions.width <= breakpoints.values.md, [breakpoints, dimensions]);

    const contextData = useObjectContextMenu();
    const actionData = useObjectActions({
        canNavigate,
        isListReorderable,
        objectType: contextData.object?.__typename as ModelType | undefined,
        onAction,
        setLocation,
        ...contextData,
    });

    // Reset focused index when items change
    useEffect(() => {
        setFocusedIndex(-1);
    }, [stableItems]);

    /**
     * Handle keyboard navigation within the list
     */
    const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
        if (!enableKeyboardNavigation || !stableItems?.length) return;

        const visibleItems = stableItems.length;
        let newIndex = focusedIndex;

        switch (event.key) {
            case KEYBOARD_KEYS.DOWN:
                newIndex = Math.min(focusedIndex + 1, visibleItems - 1);
                event.preventDefault();
                break;
            case KEYBOARD_KEYS.UP:
                newIndex = Math.max(focusedIndex - 1, 0);
                event.preventDefault();
                break;
            case KEYBOARD_KEYS.HOME:
                newIndex = 0;
                event.preventDefault();
                break;
            case KEYBOARD_KEYS.END:
                newIndex = visibleItems - 1;
                event.preventDefault();
                break;
            case KEYBOARD_KEYS.ENTER:
                if (focusedIndex >= 0 && focusedIndex < visibleItems) {
                    // Trigger the same action as clicking
                    const item = stableItems[focusedIndex];
                    if (isSelecting && typeof handleToggleSelect === "function") {
                        handleToggleSelect(item);
                    } else if (typeof onClick === "function") {
                        onClick(item);
                    } else if (canNavigate) {
                        const shouldContinue = typeof canNavigate === "function" ? canNavigate(item) : true;
                        if (shouldContinue !== false) {
                            setLocation(getObjectUrl(item));
                        }
                    }
                    event.preventDefault();
                }
                break;
            case KEYBOARD_KEYS.ESCAPE:
                // Return focus to search bar
                const searchBar = document.querySelector(`[id$="-search-bar"] input`) as HTMLElement;
                if (searchBar) {
                    searchBar.focus();
                    setFocusedIndex(-1);
                    event.preventDefault();
                }
                break;
            default:
                return; // Exit without preventing default for other keys
        }

        // Update focus if index changed
        if (newIndex !== focusedIndex) {
            setFocusedIndex(newIndex);
            // Focus the element
            const itemToFocus = listRef.current?.querySelector(`[data-index="${newIndex}"]`) as HTMLElement;
            if (itemToFocus) {
                itemToFocus.focus();
            }
        }
    }, [enableKeyboardNavigation, stableItems, focusedIndex, isSelecting, handleToggleSelect, onClick, canNavigate, setLocation]);

    // Generate real list items
    const realItems = useMemo(() => {
        const usedKeys = new Set<string>();
        return stableItems?.map((item, index) => {
            let curr = item;
            if (isOfType(curr, "Bookmark", "Reaction", "View")) {
                curr = (curr as Partial<Bookmark | Reaction | View>).to as ListObject;
            }
            if (!curr) {
                console.error("[ObjectList] Item is not defined", item);
                return null;
            }
            // Checks to prevent duplicate keys
            if (!curr.id) {
                console.error("[ObjectList] Item is missing an id", curr);
                return null;
            }
            const key = `${keyPrefix}-${curr.id}`;
            if (usedKeys.has(key)) {
                console.error("[ObjectList] Duplicate key", key);
                return null;
            }
            usedKeys.add(key);
            return (
                <MemoizedObjectListItem
                    key={key}
                    canNavigate={canNavigate}
                    data={curr as ListObject}
                    handleContextMenu={contextData.handleContextMenu}
                    handleToggleSelect={handleToggleSelect ?? noop}
                    hideUpdateButton={hideUpdateButton}
                    isMobile={isMobile}
                    isSelecting={isSelecting ?? false}
                    isSelected={selectedItems?.some((selected) => selected.id === curr.id) ?? false}
                    loading={false}
                    objectType={curr.__typename}
                    onAction={stableOnAction}
                    onClick={onClick}
                    tabIndex={enableKeyboardNavigation ? 0 : -1}
                    dataIndex={index}
                />
            );
        });
    }, [stableItems, keyPrefix, canNavigate, contextData.handleContextMenu, handleToggleSelect, hideUpdateButton, isMobile, isSelecting, selectedItems, stableOnAction, onClick, enableKeyboardNavigation]);

    // Generate dummy items
    const dummyListItems = useMemo(() => {
        if (loading && dummyItems) {
            return dummyItems.map((dummy, index) => (
                <MemoizedObjectListItem
                    key={`${keyPrefix}-dummy-${index}`}
                    data={null}
                    handleContextMenu={noop}
                    handleToggleSelect={noop}
                    hideUpdateButton={hideUpdateButton}
                    isMobile={isMobile}
                    isSelecting={isSelecting ?? false}
                    isSelected={false}
                    loading={true}
                    objectType={dummy}
                    onAction={noop}
                />
            ));
        }
        return [];
    }, [loading, dummyItems, keyPrefix, hideUpdateButton, isMobile, isSelecting]);

    return (
        <Box
            ref={(el) => {
                // We can't combine refs directly because dimRef is a RefObject, not a callback ref
                if (el) {
                    // Update the dimension ref's current property if element exists
                    if (dimRef.current !== el) {
                        // @ts-ignore - This is fine as we just need to assign the element
                        dimRef.current = el;
                    }
                    // Update our list ref
                    listRef.current = el;
                }
            }}
            width="100%"
            role={enableKeyboardNavigation ? "list" : undefined}
            onKeyDown={handleKeyDown}
            tabIndex={-1}
        >
            {/* Context menus */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={contextData.anchorEl}
                exclude={contextActionsExcluded}
                object={contextData.object}
                onClose={contextData.closeContextMenu}
            />
            {/* Actual results */}
            {realItems}
            {/* Placeholders while loading more data */}
            {dummyListItems}
        </Box>
    );
}
