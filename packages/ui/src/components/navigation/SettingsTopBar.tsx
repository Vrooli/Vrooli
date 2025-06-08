import Stack from "@mui/material/Stack";
import { pagePaddingBottom } from "../../styles.js";

const settingsContentStackStyle = {
    paddingBottom: pagePaddingBottom,
    display: "flex",
    flex: 1,
    minHeight: 0,
} as const;

export function SettingsContent({ children }: { children: React.ReactNode }) {
    return (
        <Stack id="settings-page" direction="row" mt={2} sx={settingsContentStackStyle}>
            {children}
        </Stack>
    );
}
