import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { CopyIcon, EllipsisIcon, EmailIcon, LinkedInIcon, TwitterIcon } from "@local/icons";
import { Box, Stack, Tooltip, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import QRCode from "react-qr-code";
import { getDeviceInfo } from "../../../utils/display/device";
import usePress from "../../../utils/hooks/usePress";
import { PubSub } from "../../../utils/pubsub";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
import { TopBar } from "../../navigation/TopBar/TopBar";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const inviteLink = `https://vrooli.com${LINKS.Start}`;
const postTitle = "Vrooli - Visual Work Routines";
const postText = `The future of work in a decentralized world. ${inviteLink}`;
const emailUrl = `mailto:?subject=${encodeURIComponent(postTitle)}&body=${encodeURIComponent(postText)}`;
const twitterUrl = `https://twitter.com/intent/tweet?text=${encodeURIComponent(postText)}`;
const linkedInUrl = `https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(inviteLink)}&title=${encodeURIComponent(postTitle)}&summary=${encodeURIComponent(postText)}`;
const buttonProps = (palette) => ({
    height: "48px",
    width: "48px",
});
const titleId = "share-site-dialog-title";
export const ShareSiteDialog = ({ open, onClose, zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const openLink = (link) => window.open(link, "_blank", "noopener,noreferrer");
    const copyInviteLink = () => {
        navigator.clipboard.writeText(inviteLink);
        PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
    };
    const shareNative = () => {
        navigator.share({
            title: postTitle,
            text: postText,
            url: inviteLink,
        });
    };
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
    return (_jsxs(LargeDialog, { id: "share-site-dialog", isOpen: open, onClose: onClose, titleId: titleId, zIndex: zIndex, children: [_jsx(TopBar, { display: "dialog", onClose: onClose, titleData: { titleId, titleKey: "SpreadTheWord" } }), _jsxs(Box, { sx: { padding: 2 }, children: [_jsxs(Stack, { direction: "row", spacing: 1, mb: 2, display: "flex", justifyContent: "center", alignItems: "center", children: [_jsx(Tooltip, { title: t("CopyLink"), children: _jsx(ColorIconButton, { onClick: copyInviteLink, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(CopyIcon, { fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: t("ShareByEmail"), children: _jsx(ColorIconButton, { href: emailUrl, onClick: (e) => { e.preventDefault(); openLink(emailUrl); }, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(EmailIcon, { fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: t("TweetIt"), children: _jsx(ColorIconButton, { href: twitterUrl, onClick: (e) => { e.preventDefault(); openLink(twitterUrl); }, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(TwitterIcon, { fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: t("LinkedInPost"), children: _jsx(ColorIconButton, { href: linkedInUrl, onClick: (e) => { e.preventDefault(); openLink(linkedInUrl); }, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(LinkedInIcon, { fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: t("Other"), children: _jsx(ColorIconButton, { onClick: shareNative, background: palette.secondary.main, sx: buttonProps(palette), children: _jsx(EllipsisIcon, { fill: palette.secondary.contrastText }) }) })] }), _jsx(Box, { id: "qr-code-box", ...pressEvents, sx: {
                            width: "210px",
                            height: "210px",
                            background: palette.secondary.main,
                            borderRadius: 1,
                            padding: 0.5,
                            marginLeft: "auto",
                            marginRight: "auto",
                        }, children: _jsx(QRCode, { size: 200, style: { height: "auto", maxWidth: "100%", width: "100%" }, value: "https://vrooli.com" }) })] })] }));
};
//# sourceMappingURL=ShareSiteDialog.js.map