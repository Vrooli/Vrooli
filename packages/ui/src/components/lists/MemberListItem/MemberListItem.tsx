import { useTheme } from "@mui/material";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { RoleList } from "../RoleList/RoleList";
import { smallHorizontalScrollbar } from "../styles";
import { MemberListItemProps } from "../types";

export function MemberListItem({
    data,
    ...props
}: MemberListItemProps) {
    const { palette } = useTheme();

    return (
        <ObjectListItemBase
            {...props}
            belowTags={
                data?.roles?.length > 0 ?
                    <RoleList
                        roles={data.roles}
                        sx={{ ...smallHorizontalScrollbar(palette) }}
                    /> : null
            }
            data={data}
            objectType="RunProject"
        />
    );
}
