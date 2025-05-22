import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { type MemberListItemProps } from "../types.js";

export function MemberListItem({
    data,
    ...props
}: MemberListItemProps) {
    return (
        <ObjectListItemBase
            {...props}
            data={data}
            objectType="Member"
        />
    );
}
