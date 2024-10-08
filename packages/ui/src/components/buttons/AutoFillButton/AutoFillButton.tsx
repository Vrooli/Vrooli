import { Tooltip, useTheme } from "@mui/material";
import { SessionContext } from "contexts";
import { MagicIcon } from "icons";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SideActionsButton } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { AutoFillButtonProps } from "../types";

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
                <MagicIcon fill={palette.secondary.contrastText} />
            </SideActionsButton>
        </Tooltip>
    );
}
