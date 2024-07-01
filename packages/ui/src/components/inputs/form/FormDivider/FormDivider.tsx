import { Button, Divider } from "@mui/material";
import { propButtonStyle } from "../styles";
import { FormDividerProps } from "../types";

const dividerStyle = {
    paddingTop: "8px",
    paddingBottom: "8px",
} as const;

export function FormDivider({
    isEditing,
    onDelete,
}: FormDividerProps) {
    // If not selected, return a plain divider
    if (!isEditing) {
        return <Divider sx={dividerStyle} />;
    }
    // Otherwise, return a divider with a delete button
    return (
        <Divider sx={dividerStyle}>
            <Button variant="text" sx={propButtonStyle} onClick={onDelete}>
                Delete
            </Button>
        </Divider>
    );
}
