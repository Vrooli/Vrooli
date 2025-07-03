import { styled } from "@mui/material";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import type { ElementBuildOuterBoxProps, ToolbarBoxProps } from "./FormView.types.js";

export const ElementBuildOuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isSelected",
})<ElementBuildOuterBoxProps>(({ isSelected, theme }) => ({
    display: "flex",
    background: "inherit",
    border: isSelected ? `4px solid ${theme.palette.secondary.main}` : "none",
    borderRadius: "8px",
    overflow: "overlay",
    height: "100%",
    alignItems: "stretch",
}));

export const ElementButton = styled("div")(() => ({
    padding: 0,
    width: "100%",
    overflow: "hidden",
    background: "none",
    color: "inherit",
    border: "none",
    textAlign: "left",
    cursor: "pointer",
}));

export const DragBox = styled(Box)(({ theme }) => ({
    cursor: "pointer",
    display: "flex",
    flexDirection: "column",
    padding: "4px",
    background: theme.palette.secondary.main,
}));

export const ToolbarBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "formElementsCount",
})<ToolbarBoxProps>(({ formElementsCount, theme }) => ({
    display: "flex",
    justifyContent: "space-around",
    margin: "-4px",
    padding: "4px",
    background: theme.palette.primary.main,
    color: theme.palette.secondary.contrastText,
    borderBottomLeftRadius: formElementsCount === 0 ? "8px" : "0px",
    borderBottomRightRadius: formElementsCount === 0 ? "8px" : "0px",
}));

export const FormHelperText = styled(Typography)(({ theme }) => ({
    textAlign: "center",
    color: theme.palette.text.secondary,
    padding: "20px",
}));

export const ElementRunOuterBox = styled(Box)(() => ({
    padding: 0,
    width: "100%",
    overflow: "hidden",
    background: "none",
    color: "inherit",
    border: "none",
    textAlign: "left",
}));

// Style constants
export const dragIconStyle = { marginTop: "auto", marginBottom: "auto" } as const;
export const toolbarLargeButtonStyle = { cursor: "pointer" } as const;
export const popoverAnchorOrigin = { vertical: "bottom", horizontal: "center" } as const;
export const formDividerStyle = { marginBottom: 2 } as const;
export const sectionsStackStyle = { width: "100%" } as const;
export const formViewDividerStyle = { paddingTop: 2 } as const;
