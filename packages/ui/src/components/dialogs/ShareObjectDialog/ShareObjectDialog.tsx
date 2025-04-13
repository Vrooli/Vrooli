/**
 * Dialog for sharing an object with multiple options: link sharing, QR code,
 * and direct object export.
 */
import { getObjectUrl } from "@local/shared";
import { Box, Divider, Fade, List, ListItem, ListItemButton, ListItemIcon, ListItemText, Paper, Stack, Tooltip, Typography, useTheme, Zoom } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import QRCode from "react-qr-code";
import { IconCommon } from "../../../icons/Icons.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { ObjectType } from "../../../utils/navigation/openObject.js";
import { PubSub } from "../../../utils/pubsub.js";
import { TopBar } from "../../navigation/TopBar.js";
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
    const fullUrl = useMemo(() => `${window.location.origin}${url}`, [url]);

    function copyLink() {
        navigator.clipboard.writeText(fullUrl);
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
            sxs={{
                paper: {
                    width: "min(500px, 100vw - 64px)",
                    borderRadius: 2,
                    maxHeight: "calc(100vh - 64px)",
                    display: "flex",
                    flexDirection: "column"
                }
            }}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("Share")}
                titleId={titleId}
            />

            <Box sx={{ flex: 1, overflowY: "auto", pb: 2 }}>
                {object && (
                    <Box sx={{ p: 2, pb: 0 }}>
                        <Typography variant="subtitle1" sx={{ fontWeight: "medium", mb: 1 }}>
                            {getDisplay(object).title}
                        </Typography>
                        <Typography
                            variant="body2"
                            color="text.secondary"
                            sx={{
                                mb: 1,
                                overflow: "hidden",
                                textOverflow: "ellipsis",
                                whiteSpace: "nowrap"
                            }}
                        >
                            {fullUrl}
                        </Typography>
                    </Box>
                )}

                <List sx={{ pt: 0 }}>
                    <Divider sx={{ my: 1 }} />
                    <Typography variant="overline" sx={{ px: 2, color: "text.secondary" }}>
                        {t("Share")}
                    </Typography>

                    <ListItem disablePadding>
                        <ListItemButton onClick={copyLink} sx={{ py: 1.5 }}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    fill={palette.background.textPrimary}
                                    name="Link"
                                />
                            </ListItemIcon>
                            <ListItemText
                                primary={"Copy link"}
                                secondary={"Copy URL to clipboard"}
                                primaryTypographyProps={{ fontWeight: "medium" }}
                            />
                        </ListItemButton>
                    </ListItem>

                    <ListItem disablePadding>
                        <ListItemButton onClick={shareLink} sx={{ py: 1.5 }}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    fill={palette.background.textPrimary}
                                    name="Share"
                                />
                            </ListItemIcon>
                            <ListItemText
                                primary={"Share link"}
                                secondary={"Share via platform options"}
                                primaryTypographyProps={{ fontWeight: "medium" }}
                            />
                        </ListItemButton>
                    </ListItem>

                    <Divider sx={{ my: 1 }} />
                    <Typography variant="overline" sx={{ px: 2, color: "text.secondary" }}>
                        {t("Advanced")}
                    </Typography>

                    <ListItem disablePadding>
                        <ListItemButton onClick={copyObject} sx={{ py: 1.5 }}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    fill={palette.background.textPrimary}
                                    name="Object"
                                />
                            </ListItemIcon>
                            <ListItemText
                                primary={"Copy object"}
                                secondary={"Copy JSON data to clipboard"}
                                primaryTypographyProps={{ fontWeight: "medium" }}
                            />
                        </ListItemButton>
                    </ListItem>

                    <ListItem disablePadding>
                        <ListItemButton onClick={shareObject} sx={{ py: 1.5 }}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    fill={palette.background.textPrimary}
                                    name="Download"
                                />
                            </ListItemIcon>
                            <ListItemText
                                primary={"Share object"}
                                secondary={"Share object as a file"}
                                primaryTypographyProps={{ fontWeight: "medium" }}
                            />
                        </ListItemButton>
                    </ListItem>

                    <ListItem disablePadding>
                        <ListItemButton onClick={toggleQrCode} sx={{ py: 1.5 }}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    fill={palette.background.textPrimary}
                                    name="QrCode"
                                />
                            </ListItemIcon>
                            <ListItemText
                                primary={"QR code"}
                                secondary={isQrCodeVisible ? "Hide QR code" : "Show QR code"}
                                primaryTypographyProps={{ fontWeight: "medium" }}
                            />
                        </ListItemButton>
                    </ListItem>
                </List>

                {isQrCodeVisible && (
                    <Fade in={isQrCodeVisible} timeout={300}>
                        <Stack
                            direction="column"
                            spacing={2}
                            sx={{
                                justifyContent: "center",
                                alignItems: "center",
                                p: 3,
                                pt: 1
                            }}>
                            <Paper
                                elevation={2}
                                id="qr-code-box"
                                sx={{
                                    display: "flex",
                                    width: "min(280px, 100%)",
                                    background: "white",
                                    borderRadius: 2,
                                    p: 2,
                                    boxShadow: 3,
                                }}>
                                <QRCode
                                    size={240}
                                    style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                                    value={fullUrl}
                                />
                            </Paper>

                            <Tooltip title={"Download QR code"}>
                                <Zoom in={isQrCodeVisible} style={{ transitionDelay: '150ms' }}>
                                    <Box
                                        onClick={downloadQrCode}
                                        sx={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            gap: 1,
                                            bgcolor: 'action.selected',
                                            borderRadius: 4,
                                            px: 2,
                                            py: 1,
                                            cursor: 'pointer',
                                            '&:hover': {
                                                bgcolor: 'action.hover',
                                            }
                                        }}
                                    >
                                        <IconCommon
                                            decorative
                                            fill={palette.background.textPrimary}
                                            name="Download"
                                            size={20}
                                        />
                                        <Typography variant="button">
                                            {"Download"}
                                        </Typography>
                                    </Box>
                                </Zoom>
                            </Tooltip>
                        </Stack>
                    </Fade>
                )}
            </Box>
        </LargeDialog>
    );
}
