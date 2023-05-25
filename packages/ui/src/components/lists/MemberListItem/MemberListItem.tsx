import { useTheme } from "@mui/material";
import { useMemo } from "react";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { RoleList } from "../RoleList/RoleList";
import { smallHorizontalScrollbar } from "../styles";
import { MemberListItemProps } from "../types";

export function MemberListItem({
    data,
    ...props
}: MemberListItemProps) {
    const { palette } = useTheme();

    const roles = useMemo(() => data?.roles ?? [], [data?.roles]);

    return (
        <ObjectListItemBase
            {...props}
            belowTags={
                roles.length > 0 ?
                    <RoleList
                        roles={roles}
                        sx={{ ...smallHorizontalScrollbar(palette) }}
                    /> : null
            }
            data={data}
            objectType="RunProject"
        />
    );
}
