import { SxProps } from "@mui/material";
import { CSSProperties } from "@mui/styles";
import { multiLineEllipsis, textShadow } from "styles";

export const nodeLabel: SxProps = {
    ...multiLineEllipsis(3),
    ...textShadow,
    position: "absolute",
    textAlign: "center",
    margin: "0",
    top: "50%",
    left: "50%",
    transform: "translate(-50%, -50%)",
    width: "100%",
    lineBreak: "anywhere",
} as CSSProperties;

export const routineNodeCheckboxLabel: SxProps = {
    marginLeft: "0",
};

export const routineNodeCheckboxOption: SxProps = {
    padding: "4px",
    height: "36px",
};

export const routineNodeActionStyle = (isEditing: boolean): SxProps => ({
    display: "inline-flex",
    alignItems: "center",
    cursor: isEditing ? "pointer" : "default",
    height: "36px",
    paddingRight: "8px",
});
