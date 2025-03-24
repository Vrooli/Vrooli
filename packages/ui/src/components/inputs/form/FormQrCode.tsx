import { Button, Divider } from "@mui/material";
import { propButtonStyle } from "./styles.js";
import { FormQrCodeProps } from "./types.js";

const dividerStyle = {
    paddingTop: "8px",
    paddingBottom: "8px",
} as const;

export function FormQrCode({
    isEditing,
    onDelete,
}: FormQrCodeProps) {
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
