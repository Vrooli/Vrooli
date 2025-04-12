import { Button, Divider } from "@mui/material";
import { propButtonStyle } from "./styles.js";
import { FormDividerProps } from "./types.js";

const displayStyle = {
    paddingTop: "0px",
    paddingBottom: "0px",
} as const;
const editingStyle = {
    paddingTop: "8px",
    paddingBottom: "8px",
} as const;

export function FormDivider({
    isEditing,
    onDelete,
}: FormDividerProps) {
    // If not selected, return a plain divider
    if (!isEditing) {
        return <Divider sx={displayStyle} />;
    }
    // Otherwise, return a divider with a delete button
    return (
        <Divider sx={editingStyle}>
            <Button variant="text" sx={propButtonStyle} onClick={onDelete}>
                Delete
            </Button>
        </Divider>
    );
}
