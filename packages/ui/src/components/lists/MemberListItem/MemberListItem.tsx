import { useMemo } from "react";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { RoleList } from "../RoleList/RoleList.js";
import { MemberListItemProps } from "../types.js";

export function MemberListItem({
    data,
    ...props
}: MemberListItemProps) {
    const roles = useMemo(() => data?.roles ?? [], [data?.roles]);

    return (
        <ObjectListItemBase
            {...props}
            belowTags={
                roles.length > 0 ?
                    <RoleList
                        roles={roles}
                    /> : null
            }
            data={data}
            objectType="Member"
        />
    );
}
