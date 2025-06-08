import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import type { ChipProps } from "@mui/material";
import { styled } from "@mui/material";
import { ProfileAvatar } from "../../../styles.js";

export function commonIconProps() {
    return {
        width: "24px",
        height: "24px",
        fill: "white",
    } as const;
}

export const RelationshipAvatar = styled(ProfileAvatar)(() => ({
    width: 24,
    height: 24,
    fontSize: 12,
}));

/** Chip shown for relationship item when not editing */
export const RelationshipChip = styled(Chip)<ChipProps>(({ icon, onClick, theme }) => ({
    background: `${theme.palette.primary.light}CC`,
    // eslint-disable-next-line no-magic-numbers
    borderRadius: theme.spacing(4),
    color: theme.palette.primary.contrastText,
    fill: theme.palette.primary.contrastText,
    height: "36px",
    paddingLeft: icon ? theme.spacing(1) : 0,
    pointerEvents: typeof onClick === "function" ? "auto" : "none",
    cursor: typeof onClick === "function" ? "pointer" : "default",
    boxShadow: typeof onClick === "function" ? theme.shadows[2] : "none",
    "& .MuiChip-icon": {
        marginLeft: 0,
        marginRight: 0,
    },
    "& .MuiChip-label": {
        // eslint-disable-next-line no-magic-numbers
        paddingLeft: icon ? theme.spacing(0.5) : theme.spacing(1),
        paddingRight: theme.spacing(1),
    },
    "&:hover": {
        // eslint-disable-next-line no-magic-numbers
        filter: `brightness(${theme.palette.mode === "light" ? 1.05 : 0.95})`,
        background: theme.palette.primary.light,
    },
}));

/** Button shown for relationship item when editing */
export const RelationshipButton = styled(Button)(({ theme }) => ({
    borderColor: theme.palette.primary.light,
    // eslint-disable-next-line no-magic-numbers
    borderRadius: theme.spacing(4),
    color: theme.palette.primary.light,
    height: "36px",
    minWidth: "fit-content",
}));

