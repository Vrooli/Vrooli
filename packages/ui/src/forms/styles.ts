import { Theme } from "@mui/material";

export const formPaper = {
    margin: "16px",
    background: "transparent",
    boxShadow: "none",
} as const;

export const formSubmit = {
    margin: "16px auto",
} as const;

export const formNavLink = {
    color: (t: Theme) => t.palette.mode === "light" ? t.palette.secondary.dark : t.palette.background.textPrimary,
    display: "flex",
    alignItems: "center",
    cursor: "pointer",
} as const;
