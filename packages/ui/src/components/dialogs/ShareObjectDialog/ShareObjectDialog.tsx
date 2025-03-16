/**
 * Dialog for sharing an object
 */
import { getObjectUrl } from "@local/shared";
import { Box, List, ListItem, ListItemIcon, ListItemText, Stack, useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { DownloadIcon, EmailIcon, LinkIcon, ObjectIcon, QrCodeIcon, ShareIcon } from "icons/common.js";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import QRCode from "react-qr-code";
import { getDisplay } from "utils/display/listTools.js";
import { ObjectType } from "utils/navigation/openObject.js";
import { PubSub } from "utils/pubsub.js";
import { LargeDialog } from "../LargeDialog/LargeDialog.js";
import { ShareObjectDialogProps } from "../types.js";

// Title for social media posts
const postTitle: { [key in ObjectType]?: string } = {
    "Comment": "Check out this comment on Vrooli",
    "Project": "Check out this project on Vrooli",
    "Routine": "Check out this routine on Vrooli",
    "Standard": "Check out this standard on Vrooli",
    "Team": "Check out this team on Vrooli",
    "User": "Check out this user on Vrooli",
};

export function sanitizeFilename(filename: string) {
    const invalidChars = /[<>:"/\\|?*]/g;
    // eslint-disable-next-line no-control-regex
    const controlChars = /[\x00-\x1f\x80-\x9f]/g;
    const reservedWords = /^(con|prn|aux|nul|com[0-9]|lpt[0-9])(\..*)?$/i;

    // Replace invalid characters with underscore
    let sanitized = filename.replace(invalidChars, "_");

    // Remove control characters
    sanitized = sanitized.replace(controlChars, "");

    // Avoid using reserved words
    if (reservedWords.test(sanitized)) {
        sanitized = "_" + sanitized;
    }

    return sanitized;
}

function canBrowserShare(data: ShareData) {
    if (!navigator.share || !navigator.canShare) {
        return false;
    }
    return navigator.canShare(data);
}

function omitFields(obj: unknown, ...fieldsToOmit: string[]): unknown {
    // Base case: if it's not an object, return it as is
    if (obj === null || typeof obj !== "object") {
        return obj;
    }
    // If it's an array, map over it and recursively omit
    if (Array.isArray(obj)) {
        return obj.map(item => omitFields(item, ...fieldsToOmit));
    }
    // If it's an object, construct a new object without the specified fields
    const result: { [key: string]: any } = {};
    for (const key in obj) {
        if (!fieldsToOmit.includes(key)) {
            result[key] = omitFields(obj[key], ...fieldsToOmit);
        }
    }
    return result;
}
// When sharing, we'll remove any fields specific to you, or related to ownership
const omittedFields = ["you", "createdBy", "createdById"];

function prepareObjectForShare(object: any): any {
    return omitFields(object, ...omittedFields);
}

const titleId = "share-object-dialog-title";

export function ShareObjectDialog({
    object,
    open,
    onClose,
}: ShareObjectDialogProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const title = useMemo(() => object && object.__typename in postTitle ? postTitle[object.__typename] : "Check out this object on Vrooli", [object]);
    const url = useMemo(() => object ? getObjectUrl(object) : window.location.href.split("?")[0].split("#")[0], [object]);

    function copyLink() {
        navigator.clipboard.writeText(`${window.location.origin}${url}`);
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }

    function copyObject() {
        navigator.clipboard.writeText(JSON.stringify(prepareObjectForShare(object), null, 2));
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }

    async function shareLink() {
        navigator.share({ title, url });
    }

    async function shareObject() {
        if (!object) return;
        try {
            const jsonString = JSON.stringify(prepareObjectForShare(object), null, 2);  // Pretty-printed
            // Create txt file (JSON is not currently supported in Chromium, despite canBrowserShare saying it is. See https://developer.mozilla.org/en-US/docs/Web/API/Navigator/share#shareable_file_types)
            const file = new File([jsonString], `${sanitizeFilename(getDisplay(object).title)}-json.txt`, { type: "text/plain" });
            const shareData = {
                title: `${getDisplay(object).title} - Vrooli`,
                files: [file],
            };
            if (canBrowserShare(shareData)) {
                console.log("sharing txt");
                let success = false;
                await navigator.share(shareData).then(() => {
                    success = true;
                }).catch((err) => {
                    console.error(`The file could not be shared: ${err}`);
                });
                if (success) {
                    return;
                }
            }
            console.error("Could not share file");
        } catch (err) {
            console.error(`The file could not be shared: ${err}`);
        }
    }

    const [isQrCodeVisible, setIsQrCodeVisible] = useState(false);
    function toggleQrCode() {
        setIsQrCodeVisible(!isQrCodeVisible);
    }
    async function downloadQrCode() {
        const qrCode = document.getElementById("qr-code-box")?.firstChild as SVGSVGElement;
        if (!qrCode || !object) return;

        // Convert SVG to Data URL
        const svgData = new XMLSerializer().serializeToString(qrCode);
        const svgBlob = new Blob([svgData], { type: "image/svg+xml;charset=utf-8" });
        const svgUrl = URL.createObjectURL(svgBlob);

        // Load SVG into image
        const img = new Image();
        img.onload = function onLoadHandler() {
            // Create canvas
            const canvas = document.createElement("canvas");
            canvas.width = img.width;
            canvas.height = img.height;
            const ctx = canvas.getContext("2d");
            ctx?.drawImage(img, 0, 0);

            // Get PNG data URL
            const pngDataUrl = canvas.toDataURL("image/png");

            // Download PNG
            const a = document.createElement("a");
            a.href = pngDataUrl;
            a.download = `${sanitizeFilename(getDisplay(object).title)}-qr-code.png`;
            a.click();

            // Cleanup
            URL.revokeObjectURL(svgUrl);
        };

        img.src = svgUrl;
    }


    return (
        <LargeDialog
            id="share-object-dialog"
            isOpen={open}
            onClose={onClose}
            titleId={titleId}
            sxs={{ paper: { width: "min(500px, 100vw - 64px)" } }}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("Share")}
                titleId={titleId}
            />
            <List>
                <ListItem
                    button
                    onClick={() => { }}
                >
                    <ListItemIcon>
                        <EmailIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={"Message..."} />
                </ListItem>
                <ListItem
                    button
                    onClick={copyLink}
                >
                    <ListItemIcon>
                        <LinkIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={"Copy link"} />
                </ListItem>
                <ListItem
                    button
                    onClick={shareLink}
                >
                    <ListItemIcon>
                        <ShareIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={"Share link"} />
                </ListItem>
                <ListItem
                    button
                    onClick={copyObject}
                >
                    <ListItemIcon>
                        <ObjectIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={"Copy object"} />
                </ListItem>
                <ListItem
                    button
                    onClick={shareObject}
                >
                    <ListItemIcon>
                        <DownloadIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={"Share object"} />
                </ListItem>
                <ListItem
                    button
                    onClick={toggleQrCode}
                >
                    <ListItemIcon>
                        <QrCodeIcon fill={palette.background.textPrimary} />
                    </ListItemIcon>
                    <ListItemText primary={"QR code"} />
                </ListItem>
            </List>
            {isQrCodeVisible && <Stack
                direction="column"
                sx={{ justifyContent: "center", alignItems: "center" }}>
                <Box
                    id="qr-code-box"
                    sx={{
                        display: "flex",
                        width: "min(250px, 100%)",
                        background: "white",
                        borderRadius: 1,
                        padding: 0.5,
                    }}>
                    <QRCode
                        size={200}
                        style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                        value={`${window.location.origin}${url}`}
                    />
                </Box>
                <List>
                    <ListItem
                        button
                        onClick={downloadQrCode}
                    >
                        <ListItemIcon>
                            <DownloadIcon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={"Download QR code"} />
                    </ListItem>
                </List>
            </Stack>}
        </LargeDialog>
    );
}
