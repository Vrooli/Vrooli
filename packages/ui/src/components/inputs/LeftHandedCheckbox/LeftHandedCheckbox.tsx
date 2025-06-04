import { Box, Checkbox, Typography } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "../../../styles.js";
import { type SxType } from "../../../types.js";
import { getCookie } from "../../../utils/localStorage.js";
import { PubSub } from "../../../utils/pubsub.js";

type LeftHandedCheckboxProps = {
    sx?: SxType;
};

/**
 * Updates the font size of the entire app
 */
export function LeftHandedCheckbox({
    sx,
}: LeftHandedCheckboxProps) {
    const { t } = useTranslation();

    const [isLeftHanded, setIsLeftHanded] = useState<boolean>(getCookie("IsLeftHanded"));

    const handleToggle = useCallback(() => {
        setIsLeftHanded(!isLeftHanded);
        PubSub.get().publish("isLeftHanded", !isLeftHanded);
    }, [isLeftHanded]);

    return (
        <Box
            sx={{
                display: "flex",
                flexDirection: "row",
                gap: 1,
                justifyContent: "center",
                alignItems: "center",
                ...sx,
            }}
        >
            <Typography variant="body1" sx={noSelect}>
                {t("LeftHandedQuestion")}
            </Typography>
            <Checkbox
                id="leftHandedCheckbox"
                size="medium"
                color='secondary'
                checked={isLeftHanded}
                onChange={handleToggle}
            />
        </Box>
    );
}
