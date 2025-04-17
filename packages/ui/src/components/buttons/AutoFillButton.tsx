import { IconButton, Tooltip } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { IconCommon } from "../../icons/Icons.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import { AutoFillButtonProps } from "./types.js";

export function AutoFillButton({
    handleAutoFill,
    isAutoFillLoading,
}: AutoFillButtonProps) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const { credits } = useMemo(() => getCurrentUser(session), [session]);

    const handleClick = useCallback(() => {
        handleAutoFill();
        PubSub.get().publish("menu", { id: ELEMENT_IDS.RightDrawer, isOpen: true });
    }, [handleAutoFill]);

    if (!credits || BigInt(credits) <= 0) return null;
    return (
        <Tooltip title={t("AutoFill")} placement="top">
            <IconButton
                disabled={isAutoFillLoading}
                onClick={handleClick}
            >
                <IconCommon name="Magic" />
            </IconButton>
        </Tooltip>
    );
}
