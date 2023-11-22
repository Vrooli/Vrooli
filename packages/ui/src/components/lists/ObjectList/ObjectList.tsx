import { Bookmark, GqlModelType, isOfType, noop, OrArray, Reaction, View } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { useDimensions } from "hooks/useDimensions";
import { useObjectActions } from "hooks/useObjectActions";
import { useStableCallback } from "hooks/useStableCallback";
import { useStableObject } from "hooks/useStableObject";
import { memo, useCallback, useMemo, useState } from "react";
import { lazily } from "react-lazily";
import { useLocation } from "route";
import { ObjectAction } from "utils/actions/objectActions";
import { ListObject } from "utils/display/listTools";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { ObjectListItemProps } from "../types";

// Custom list item components
const { BookmarkListListItem } = lazily(() => import("../BookmarkListListItem/BookmarkListListItem"));
const { ChatListItem } = lazily(() => import("../ChatListItem/ChatListItem"));
const { ChatInviteListItem } = lazily(() => import("../ChatInviteListItem/ChatInviteListItem"));
const { ChatParticipantListItem } = lazily(() => import("../ChatParticipantListItem/ChatParticipantListItem"));
const { MeetingInviteListItem } = lazily(() => import("../MeetingInviteListItem/MeetingInviteListItem"));
const { MemberInviteListItem } = lazily(() => import("../MemberInviteListItem/MemberInviteListItem"));
const { MemberListItem } = lazily(() => import("../MemberListItem/MemberListItem"));
const { NotificationListItem } = lazily(() => import("../NotificationListItem/NotificationListItem"));
const { ReminderListItem } = lazily(() => import("../ReminderListItem/ReminderListItem"));
const { RunProjectListItem } = lazily(() => import("../RunProjectListItem/RunProjectListItem"));
const { RunRoutineListItem } = lazily(() => import("../RunRoutineListItem/RunRoutineListItem"));
const getListItemComponent = (objectType: `${GqlModelType}` | "CalendarEvent") => {
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

type ObjectListItemPropsForMultiple<T extends OrArray<ListObject>> = T extends ListObject ?
    ObjectListItemProps<T> :
    T extends Array<ListObject> ?
    ObjectListItemProps<T[number]> :
    never;

export type ObjectListProps<T extends OrArray<ListObject>> = Pick<ObjectListItemPropsForMultiple<T>, "canNavigate" | "hideUpdateButton" | "loading" | "onAction" | "onClick"> & {
    /** List of dummy items types to display while loading */
    dummyItems?: (GqlModelType | `${GqlModelType}`)[];
    handleToggleSelect?: (item: ListObject) => unknown,
    /** The list of item data. Objects like view and star are converted to their respective objects. */
    items?: readonly ListObject[],
    /** Hides individual list item actions and makes items selectable for bulk actions (e.g. deleting multiple items at once)  */
    isSelecting?: boolean,
    /** Each list item's key will be `${keyPrefix}-${id}` */
    keyPrefix: string,
    /** Items currently selected. Ignored if `isSelecting` is false. */
    selectedItems?: readonly ListObject[],
};

export const ObjectList = <T extends OrArray<ListObject>>({
    canNavigate,
    dummyItems,
    keyPrefix,
    handleToggleSelect,
    hideUpdateButton,
    isSelecting,
    items,
    loading,
    onAction,
    onClick,
    selectedItems,
}: ObjectListProps<T>) => {
    const { breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const stableItems = useStableObject(items);
    const stableOnClick = useStableCallback(onClick);
    const stableOnAction = useStableCallback(onAction);

    const { dimensions, ref: dimRef } = useDimensions();
    const isMobile = useMemo(() => dimensions.width <= breakpoints.values.md, [breakpoints, dimensions]);

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
        onAction,
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
                    handleToggleSelect={handleToggleSelect ?? noop}
                    hideUpdateButton={hideUpdateButton}
                    isMobile={isMobile}
                    isSelecting={isSelecting ?? false}
                    isSelected={selectedItems?.some((selected) => selected.id === curr.id) ?? false}
                    loading={false}
                    objectType={curr.__typename}
                    onAction={stableOnAction}
                    onClick={stableOnClick}
                />
            );
        });
    }, [stableItems, keyPrefix, canNavigate, handleContextMenu, handleToggleSelect, hideUpdateButton, isMobile, isSelecting, selectedItems, stableOnAction, stableOnClick]);

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
        <Box ref={dimRef}>
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
        </Box>
    );
};
