import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { CopyIcon, EllipsisIcon, EmailIcon, LinkedInIcon, TwitterIcon } from "@local/icons";
import { Box, Stack, Tooltip, useTheme } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import QRCode from "react-qr-code";
import { getDeviceInfo } from "../../../utils/display/device";
import usePress from "../../../utils/hooks/usePress";
import { getObjectUrl } from "../../../utils/navigation/openObject";
import { PubSub } from "../../../utils/pubsub";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
import { TopBar } from "../../navigation/TopBar/TopBar";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const postTitle = {
    "Comment": "Check out this comment on Vrooli",
    "Organization": "Check out this organization on Vrooli",
    "Project": "Check out this project on Vrooli",
    "Routine": "Check out this routine on Vrooli",
    "Standard": "Check out this standard on Vrooli",
    "User": "Check out this user on Vrooli",
};
const buttonProps = (palette) => ({
    height: "48px",
    width: "48px",
});
const openLink = (link) => window.open(link, "_blank", "noopener,noreferrer");
const titleId = "share-object-dialog-title";
export const ShareObjectDialog = ({ object, open, onClose, zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const title = useMemo(() => object && object.__typename in postTitle ? postTitle[object.__typename] : "Check out this object on Vrooli", [object]);
    const url = useMemo(() => object ? getObjectUrl(object) : window.location.href.split("?")[0].split("#")[0], [object]);
    const emailUrl = useMemo(() => `mailto:?subject=${encodeURIComponent(title)}&body=${encodeURIComponent(url)}`, [title, url]);
    const twitterUrl = useMemo(() => `https://twitter.com/intent/tweet?text=${encodeURIComponent(url)}`, [url]);
    const linkedInUrl = useMemo(() => `https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(url)}&title=${encodeURIComponent(title)}&summary=${encodeURIComponent(url)}`, [title, url]);
    const copyLink = () => {
        navigator.clipboard.writeText(url);
        PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
    };
    const shareNative = () => { navigator.share({ title, url }); };
    const handleQRCodeLongPress = () => {
        const { isStandalone } = getDeviceInfo();
        if (!isStandalone)
            return;
        const qrCode = document.getElementById("qr-code-box")?.firstChild;
        if (!qrCode)
            return;
        const file = new File([qrCode.src], "qr-code.png", { type: "image/png" });
        const url = URL.createObjectURL(file);
        const a = document.createElement("a");
        a.href = url;
        a.download = "qr-code.png";
        a.click();
        URL.revokeObjectURL(url);
    };
    const pressEvents = usePress({
        onLongPress: handleQRCodeLongPress,
        onClick: handleQRCodeLongPress,
        onRightClick: handleQRCodeLongPress,
    });
    return (_jsxs(LargeDialog, { id: "share-object-dialog", isOpen: open, onClose: onClose, titleId: titleId, zIndex: zIndex, children: [_jsx(TopBar, { display: "dialog", onClose: onClose, titleData: { titleId, titleKey: "Share" } }), _jsxs(Box, { sx: { padding: 2 }, children: [_jsxs(Stack, { direction: "row", spacing: 1, mb: 2, display: "flex", justifyContent: "center", alignItems: "center", children: [_jsx(Tooltip, { title: t("CopyLink"), children: _jsx(ColorIconButton, { onClick: copyLink, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(CopyIcon, { fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: t("ShareByEmail"), children: _jsx(ColorIconButton, { href: emailUrl, onClick: (e) => { e.preventDefault(); openLink(emailUrl); }, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(EmailIcon, { fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: t("TweetIt"), children: _jsx(ColorIconButton, { href: twitterUrl, onClick: (e) => { e.preventDefault(); openLink(twitterUrl); }, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(TwitterIcon, { fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: t("LinkedInPost"), children: _jsx(ColorIconButton, { href: linkedInUrl, onClick: (e) => { e.preventDefault(); openLink(linkedInUrl); }, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(LinkedInIcon, { fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: t("Other"), children: _jsx(ColorIconButton, { onClick: shareNative, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(EllipsisIcon, { fill: palette.secondary.contrastText }) }) })] }), _jsx(Box, { id: "qr-code-box", ...pressEvents, sx: {
                            width: "210px",
                            height: "210px",
                            background: palette.secondary.main,
                            borderRadius: 1,
                            padding: 0.5,
                            marginLeft: "auto",
                            marginRight: "auto",
                        }, children: _jsx(QRCode, { size: 200, style: { height: "auto", maxWidth: "100%", width: "100%" }, value: window.location.href }) })] })] }));
};
//# sourceMappingURL=ShareObjectDialog.js.map