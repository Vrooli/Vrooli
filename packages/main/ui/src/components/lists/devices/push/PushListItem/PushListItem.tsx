import { DeleteIcon } from "@local/shared";
import { IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback } from "react";
import { multiLineEllipsis } from "../../../../../styles";
import { PushListItemProps } from "../types";

const Status = {
    NotVerified: "#a71c2d", // Red
    Verified: "#19972b", // Green
};

//  TODO copied from emaillistitem. need to rewrite
export function PushListItem({
    handleDelete,
    handleUpdate,
    index,
    data,
}: PushListItemProps) {
    const { palette } = useTheme();

    const onDelete = useCallback(() => {
        handleDelete(data);
    }, [data, handleDelete]);

    return (
        <ListItem
            disablePadding
            sx={{
                display: "flex",
                padding: 1,
            }}
        >
            {/* Left informational column */}
            <Stack direction="column" spacing={1} pl={2} sx={{ marginRight: "auto" }}>
                <ListItemText
                    primary={data.name ?? data.id}
                    sx={{ ...multiLineEllipsis(1) }}
                />
                {/* Verified indicator */}
                {/* <Box sx={{
                    borderRadius: 1,
                    border: `2px solid ${data.verified ? Status.Verified : Status.NotVerified}`,
                    color: data.verified ? Status.Verified : Status.NotVerified,
                    height: 'fit-content',
                    fontWeight: 'bold',
                    marginTop: 'auto',
                    marginBottom: 'auto',
                    textAlign: 'center',
                    padding: 0.25,
                    width: 'fit-content',
                }}>
                    {data.verified ? "Verified" : "Not Verified"}
                </Box> */}
            </Stack>
            {/* Right action buttons */}
            <Stack direction="row" spacing={1}>
                <Tooltip title="Delete Email">
                    <IconButton
                        onClick={onDelete}
                    >
                        <DeleteIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
            </Stack>
        </ListItem>
    );
}
