import { useMemo } from "react";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { RoleList } from "../RoleList/RoleList";
import { MemberListItemProps } from "../types";

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
