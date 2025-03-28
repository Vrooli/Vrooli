import { Tooltip, useTheme } from "@mui/material";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts.js";
import { IconCommon } from "../../icons/Icons.js";
import { SideActionsButton } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { AutoFillButtonProps } from "./types.js";

export function AutoFillButton({
    handleAutoFill,
    isAutoFillLoading,
}: AutoFillButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const { credits } = useMemo(() => getCurrentUser(session), [session]);

    if (!credits || BigInt(credits) <= 0) return null;
    return (
        <Tooltip title={t("AutoFill")} placement="top">
            <SideActionsButton
                aria-label={t("AutoFill")}
                disabled={isAutoFillLoading}
                onClick={handleAutoFill}
            >
                <IconCommon
                    decorative
                    fill={palette.secondary.contrastText}
                    name="Magic"
                />
            </SideActionsButton>
        </Tooltip>
    );
}
