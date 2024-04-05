import { Checkbox, Stack, Typography } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "styles";
import { getCookie } from "utils/cookies";
import { PubSub } from "utils/pubsub";

/**
 * Updates the font size of the entire app
 */
export function LeftHandedCheckbox() {
    const { t } = useTranslation();

    const [isLeftHanded, setIsLeftHanded] = useState<boolean>(getCookie("IsLeftHanded"));

    const handleToggle = useCallback(() => {
        setIsLeftHanded(!isLeftHanded);
        PubSub.get().publish("isLeftHanded", !isLeftHanded);
    }, [isLeftHanded]);

    return (
        <Stack direction="row" spacing={1} justifyContent="center" alignItems="center">
            <Typography variant="body1" sx={{
                ...noSelect,
                marginRight: "auto",
            }}>
                {t("LeftHandedQuestion")}
            </Typography>
            <Checkbox
                id="leftHandedCheckbox"
                size="medium"
                color='secondary'
                checked={isLeftHanded}
                onChange={handleToggle}
            />
        </Stack>
    );
}
