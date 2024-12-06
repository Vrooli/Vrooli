import { SxProps } from "@mui/material";
import { multiLineEllipsis } from "styles";

export const nodeLabel: SxProps = {
    // eslint-disable-next-line no-magic-numbers
    ...multiLineEllipsis(3),
    position: "absolute",
    textAlign: "center",
    margin: "0",
    top: "50%",
    left: "50%",
    transform: "translate(-50%, -50%)",
    width: "100%",
    lineBreak: "anywhere",
    "@media print": {
        textShadow: "none",
        color: "black",
    },
} as const;

export const routineNodeCheckboxLabel: SxProps = {
    marginLeft: "0",
};

export const routineNodeCheckboxOption: SxProps = {
    padding: "4px",
    height: "36px",
};

export function routineNodeActionStyle(isEditing: boolean): SxProps {
    return {
        display: "inline-flex",
        alignItems: "center",
        cursor: isEditing ? "pointer" : "default",
        height: "36px",
        paddingRight: "8px",
    } as const;
}

