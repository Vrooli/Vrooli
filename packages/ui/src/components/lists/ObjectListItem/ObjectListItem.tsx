import { GqlModelType } from "@local/shared";
import { useMemo } from "react";
import { lazily } from "react-lazily";
import { ListObjectType } from "utils/display/listTools";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { ObjectListItemProps } from "../types";

// Custom list item components
const { MemberListItem } = lazily(() => import("../MemberListItem/MemberListItem"));
const { ReminderListItem } = lazily(() => import("../reminder/ReminderListItem/ReminderListItem"));
const { RunProjectListItem } = lazily(() => import("../RunProjectListItem/RunProjectListItem"));
const { RunRoutineListItem } = lazily(() => import("../RunRoutineListItem/RunRoutineListItem"));
const getListItemComponent = (objectType: `${GqlModelType}`) => {
    switch (objectType) {
        case "Member": return MemberListItem;
        case "Reminder": return ReminderListItem;
        case "RunProject": return RunProjectListItem;
        case "RunRoutine": return RunRoutineListItem;
        default: return ObjectListItemBase;
    }
};

export function ObjectListItem<T extends ListObjectType>({
    objectType,
    ...props
}: ObjectListItemProps<T>) {
    const ListItem = useMemo(() => getListItemComponent(objectType), [objectType]);
    return (
        <ListItem
            objectType={objectType}
            {...(props as any)}
        />
    );
}
