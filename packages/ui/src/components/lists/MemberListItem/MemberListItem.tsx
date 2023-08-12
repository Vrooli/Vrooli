import { useTheme } from "@mui/material";
import { block } from "million/react";
import { useMemo } from "react";
import { ListObject } from "utils/display/listTools";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { RoleList } from "../RoleList/RoleList";
import { smallHorizontalScrollbar } from "../styles";
import { MemberListItemProps, ObjectListItemProps } from "../types";

export const MemberListItem = block(({
    data,
    onClick,
    ...props
}: MemberListItemProps) => {
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
            objectType="Member"
            onClick={onClick as ObjectListItemProps<ListObject>["onClick"]}
        />
    );
});
