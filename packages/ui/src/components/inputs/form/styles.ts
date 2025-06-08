import Box from "@mui/material/Box";
import { styled } from "@mui/material";

export const propButtonStyle = { textDecoration: "underline", textTransform: "none" } as const;

export function propButtonWithSectionStyle(isSectionOpen: boolean) {
    return {
        ...propButtonStyle,
        color: isSectionOpen ? "primary.main" : undefined,
    };
}

export const FormSettingsButtonRow = styled(Box)(() => ({
    display: "flex",
    flexDirection: "row",
}));

export const FormSettingsSection = styled(Box)(() => ({
    marginTop: "10px",
    padding: "8px",
    display: "flex",
    flexDirection: "column",
    gap: "8px",
}));
