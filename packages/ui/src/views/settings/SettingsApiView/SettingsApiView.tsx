import { Box, Stack } from "@mui/material";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { useTranslation } from "react-i18next";
import { pagePaddingBottom } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { SettingsApiViewProps } from "../types";

export const SettingsApiView = ({
    isOpen,
    onClose,
}: SettingsApiViewProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Api", { count: 1 })}
            />
            <Stack direction="row" mt={2} sx={{ paddingBottom: pagePaddingBottom }}>
                <SettingsList />
                <Box m="auto">
                </Box>
            </Stack>
        </>
    );
};
