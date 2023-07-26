import { Box, Stack } from "@mui/material";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { useTranslation } from "react-i18next";
import { SettingsDataViewProps } from "../types";

export const SettingsDataView = ({
    display = "page",
    onClose,
    zIndex,
}: SettingsDataViewProps) => {
    const { t } = useTranslation();

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Data")}
                zIndex={zIndex}
            />
            <Stack direction="row" mt={2}>
                <SettingsList />
                <Box m="auto">
                </Box>
            </Stack>
        </>
    );
};
