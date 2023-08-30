/**
 * Dialog for sharing an object
 */
import { Box, IconButton, Palette, Stack, Tooltip, useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import usePress from "hooks/usePress";
import { CopyIcon, EllipsisIcon, EmailIcon, LinkedInIcon, TwitterIcon } from "icons";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import QRCode from "react-qr-code";
import { getDeviceInfo } from "utils/display/device";
import { getObjectUrl, ObjectType } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { ShareObjectDialogProps } from "../types";

// Title for social media posts
const postTitle: { [key in ObjectType]?: string } = {
    "Comment": "Check out this comment on Vrooli",
    "Organization": "Check out this organization on Vrooli",
    "Project": "Check out this project on Vrooli",
    "Routine": "Check out this routine on Vrooli",
    "Standard": "Check out this standard on Vrooli",
    "User": "Check out this user on Vrooli",
};

const buttonProps = (palette: Palette) => ({
    background: palette.secondary.main,
    height: "48px",
    width: "48px",
});

const openLink = (link: string) => window.open(link, "_blank", "noopener,noreferrer");

const titleId = "share-object-dialog-title";

export const ShareObjectDialog = ({
    object,
    open,
    onClose,
}: ShareObjectDialogProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const title = useMemo(() => object && object.__typename in postTitle ? postTitle[object.__typename] : "Check out this object on Vrooli", [object]);
    const url = useMemo(() => object ? getObjectUrl(object) : window.location.href.split("?")[0].split("#")[0], [object]);
    console.log("shareobjectdialog", url, object);

    const emailUrl = useMemo(() => `mailto:?subject=${encodeURIComponent(title)}&body=${encodeURIComponent(url)}`, [title, url]);
    const twitterUrl = useMemo(() => `https://twitter.com/intent/tweet?text=${encodeURIComponent(url)}`, [url]);
    const linkedInUrl = useMemo(() => `https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(url)}&title=${encodeURIComponent(title)}&summary=${encodeURIComponent(url)}`, [title, url]);

    const copyLink = () => {
        console.log("copying link", url);
        navigator.clipboard.writeText(`${window.location.origin}${url}`);
        PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
    };

    /**
    * When QR code is long-pressed in standalone mode (i.e. app is downloaded), open copy/save photo dialog
    */
    const handleQRCodeLongPress = () => {
        const { isStandalone } = getDeviceInfo();
        if (!isStandalone) return;
        // Find image using parent element's ID
        const qrCode = document.getElementById("qr-code-box")?.firstChild as HTMLImageElement;
        if (!qrCode) return;
        // Create file
        const file = new File([qrCode.src], "qr-code.png", { type: "image/png" });
        // Open save dialog
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


    return (
        <LargeDialog
            id="share-object-dialog"
            isOpen={open}
            onClose={onClose}
            titleId={titleId}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("Share")}
                titleId={titleId}
            />
            <Stack direction="column" spacing={2} p={2} sx={{ justifyContent: "center", alignItems: "center" }}>
                <Box
                    id="qr-code-box"
                    {...pressEvents}
                    sx={{
                        width: "210px",
                        height: "210px",
                        background: "white",
                        borderRadius: 1,
                        padding: 0.5,
                        marginLeft: "auto",
                        marginRight: "auto",
                    }}>
                    <QRCode
                        size={200}
                        style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                        value={`${window.location.origin}${url}`}
                    />
                </Box>
                <Stack direction="row" spacing={1} mb={2} display="flex" justifyContent="center" alignItems="center">
                    <Tooltip title={t("CopyLink")}>
                        <IconButton
                            onClick={copyLink}
                            sx={buttonProps(palette)}
                        >
                            <CopyIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("ShareByEmail")}>
                        <IconButton
                            href={emailUrl}
                            onClick={(e) => { e.preventDefault(); openLink(emailUrl); }}
                            sx={buttonProps(palette)}
                        >
                            <EmailIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("TweetIt")}>
                        <IconButton
                            href={twitterUrl}
                            onClick={(e) => { e.preventDefault(); openLink(twitterUrl); }}
                            sx={buttonProps(palette)}
                        >
                            <TwitterIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("LinkedInPost")}>
                        <IconButton
                            href={linkedInUrl}
                            onClick={(e) => { e.preventDefault(); openLink(linkedInUrl); }}
                            sx={buttonProps(palette)}
                        >
                            <LinkedInIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>
                    {navigator.share && <Tooltip title={t("Other")}>
                        <IconButton
                            onClick={() => { navigator.share({ title, url }); }}
                            sx={buttonProps(palette)}
                        >
                            <EllipsisIcon fill={palette.secondary.contrastText} />
                        </IconButton>
                    </Tooltip>}
                </Stack>
            </Stack>
        </LargeDialog>
    );
};
