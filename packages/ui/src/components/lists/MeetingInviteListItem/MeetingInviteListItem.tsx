import { Box, Typography, useTheme } from "@mui/material";
import { NoteIcon } from "icons";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { MeetingInviteListItemProps } from "../types";

export function MeetingInviteListItem({
    data,
    ...props
}: MeetingInviteListItemProps) {
    const { palette } = useTheme();

    return (
        <ObjectListItemBase
            {...props}
            belowTags={
                data?.message && data.message.length > 0 ?
                    <Box sx={{ display: "flex", flexDirection: "row", gap: 1, justifyContent: "flex-start", alignItems: "center" }}>
                        <NoteIcon fill={palette.background.textSecondary} width={16} height={16} />
                        <Typography variant="body2" sx={{ color: palette.background.textSecondary }}>
                            {data.message}
                        </Typography>
                    </Box> : null
            }
            data={data}
            objectType="MeetingInvite"
        />
    );
}
