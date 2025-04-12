import { Stack } from "@mui/material";
import { pagePaddingBottom } from "../../styles.js";

const settingsContentStackStyle = { paddingBottom: pagePaddingBottom } as const;

export function SettingsContent({ children }: { children: React.ReactNode }) {
    return (
        <Stack id="settings-page" direction="row" mt={2} sx={settingsContentStackStyle}>
            {children}
        </Stack>
    );
}
