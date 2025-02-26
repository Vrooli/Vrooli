import { Box } from "@mui/material";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsContent, SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { useTranslation } from "react-i18next";
import { ScrollBox } from "styles";
import { SettingsApiViewProps } from "../types.js";

export function SettingsApiView({
    display,
    onClose,
}: SettingsApiViewProps) {
    const { t } = useTranslation();

    return (
        <ScrollBox>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Api", { count: 1 })}
            />
            <SettingsContent>
                <SettingsList />
                <Box m="auto">
                    {/* TODO add opt-in for donating unused monthly credits, with slider 
                    for what percentage you want to donate. For now, it should be clear that it's 
                    donated back to us. Later on when we add fundraising, we can let the user pick where it goes to
                    (not sure about the legal implications here)
                    
                    TODO NOTE: Make sure that this doesn't accidentally donate additional credits that someone bought. I think this can be solved by adding the max rollover number to be the monthly credits increment. So the calculation would be if (credits > DONATION_THRESHOLD) ? (MONTHLY_CREDITS_INCREMENT * DONATION_PERCENTAGE) : 0, where the donation threshold is defaulted to 1 month (but maybe we can have another slider for this*/}
                </Box>
            </SettingsContent>
        </ScrollBox>
    );
}
