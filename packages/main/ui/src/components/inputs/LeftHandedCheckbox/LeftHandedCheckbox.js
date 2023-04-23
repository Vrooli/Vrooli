import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Checkbox, Stack, Typography } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "../../../styles";
import { getCookieIsLeftHanded } from "../../../utils/cookies";
import { PubSub } from "../../../utils/pubsub";
export function LeftHandedCheckbox() {
    const { t } = useTranslation();
    const [isLeftHanded, setIsLeftHanded] = useState(getCookieIsLeftHanded(false));
    const handleToggle = useCallback(() => {
        setIsLeftHanded(!isLeftHanded);
        PubSub.get().publishIsLeftHanded(!isLeftHanded);
    }, [isLeftHanded]);
    return (_jsxs(Stack, { direction: "row", spacing: 1, justifyContent: "center", alignItems: "center", children: [_jsx(Typography, { variant: "body1", sx: {
                    ...noSelect,
                    marginRight: "auto",
                }, children: t("LeftHandedQuestion") }), _jsx(Checkbox, { id: "leftHandedCheckbox", size: "medium", color: 'secondary', checked: isLeftHanded, onChange: handleToggle })] }));
}
//# sourceMappingURL=LeftHandedCheckbox.js.map