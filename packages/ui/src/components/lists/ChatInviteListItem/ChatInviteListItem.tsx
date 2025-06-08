import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material";
import { IconCommon } from "../../../icons/Icons.js";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { type ChatInviteListItemProps } from "../types.js";

export function ChatInviteListItem({
    data,
    ...props
}: ChatInviteListItemProps) {
    const { palette } = useTheme();

    return (
        <ObjectListItemBase
            {...props}
            belowTags={
                data?.message && data.message.length > 0 ?
                    <Box
                        display="flex"
                        flexDirection="row"
                        gap={1}
                        justifyContent="flex-start"
                        alignItems="center"
                    >
                        <IconCommon
                            decorative
                            fill="background.textSecondary"
                            name="Note"
                            size={16}
                        />
                        <Typography variant="body2" sx={{ color: palette.background.textSecondary }}>
                            {data.message}
                        </Typography>
                    </Box> : null
            }
            data={data}
            objectType="ChatInvite"
        />
    );
}
