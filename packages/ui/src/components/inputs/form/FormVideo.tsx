import Button from "@mui/material/Button";
import Divider from "@mui/material/Divider";
import { propButtonStyle } from "./styles.js";
import { type FormVideoProps } from "./types.js";

const dividerStyle = {
    paddingTop: "8px",
    paddingBottom: "8px",
} as const;

// TODO implement FormImage, FormTip, FormQrCode and FormVideo. Use them to replace tutorial steps content.
export function FormVideo({
    isEditing,
    onDelete,
}: FormVideoProps) {
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
