import { SxProps, Theme } from "@mui/material";

export const formPaper: SxProps = {
    margin: "16px",
    background: "transparent",
    boxShadow: "none",
};

export const formSubmit: SxProps = {
    margin: "16px auto",
};

export const formNavLink: SxProps<Theme> = {
    color: (t) => t.palette.mode === "light" ? t.palette.secondary.dark : t.palette.background.textPrimary,
    display: "flex",
    alignItems: "center",
    cursor: "pointer",
};
