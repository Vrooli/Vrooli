import { GqlModelType } from "@local/shared";
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
