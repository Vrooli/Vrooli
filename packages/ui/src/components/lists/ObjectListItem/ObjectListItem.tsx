import { GqlModelType } from "@local/shared";
import { block } from "million/react";
import { useMemo } from "react";
import { lazily } from "react-lazily";
import { ListObject } from "utils/display/listTools";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { ObjectListItemProps } from "../types";

// Custom list item components
const { ChatListItem } = lazily(() => import("../ChatListItem/ChatListItem"));
const { MemberListItem } = lazily(() => import("../MemberListItem/MemberListItem"));
const { NotificationListItem } = lazily(() => import("../NotificationListItem/NotificationListItem"));
const { ReminderListItem } = lazily(() => import("../reminder/ReminderListItem/ReminderListItem"));
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

export const ObjectListItem = block(({
    objectType,
    ...props
}: ObjectListItemProps<ListObject>) => {
    const ListItem = useMemo<(props: ObjectListItemProps<ListObject>) => JSX.Element>(() => getListItemComponent(objectType) as ((props: ObjectListItemProps<ListObject>) => JSX.Element), [objectType]);
    return (
        <ListItem
            objectType={objectType}
            {...props}
        />
    );
}) as <T extends ListObject>(props: ObjectListItemProps<T>) => JSX.Element;
