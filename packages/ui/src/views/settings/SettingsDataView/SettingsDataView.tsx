import { Box, Stack } from "@mui/material";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { SettingsDataViewProps } from "../types";

export const SettingsDataView = ({
    isOpen,
    onClose,
    zIndex,
}: SettingsDataViewProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

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
