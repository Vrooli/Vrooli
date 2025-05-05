import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { MemberListItemProps } from "../types.js";

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
