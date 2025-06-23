import { alpha } from "@mui/material/styles";
import Box from "@mui/material/Box";
import { IconButton } from "../../buttons/IconButton.js";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useDropzone } from "react-dropzone";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../../contexts/session.js";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { ProfilePictureInputAvatar } from "../../../styles.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { ELEMENT_IDS, Z_INDEX } from "../../../utils/consts.js";
import { extractImageUrl } from "../../../utils/display/imageTools.js";
import { placeholderColor } from "../../../utils/display/listTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { type ProfilePictureInputProps } from "../types.js";

// What size image to display
const BANNER_TARGET_SIZE = 1000;
const PROFILE_TARGET_SIZE = 100;

// Alpha values for gradient colors
const GRADIENT_START_ALPHA = 0.6;
const GRADIENT_END_ALPHA = 0.8;

/**
 * Processes an image file and returns an object URL for it
 */
async function handleImagePreview(file: File): Promise<string | undefined> {
    // Extract extension from file name or path
    const ext = (file.name ?? (file as { path?: string }).path ?? "").split(".").pop()?.toLowerCase() ?? "";
    // .heic and .heif files are not supported by browsers, 
    // so we need to convert them to JPEGs (thanks, Apple)
    if (ext === "heic" || ext === "heif") {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.FullPageSpinner, data: { show: true } });
        // Dynamic import of heic2any
        const heic2any = (await import("../../../utils/heic/heic2any.js")).default;

        // Convert HEIC/HEIF file to JPEG Blob
        try {
            const outputBlob = await heic2any({
                blob: file,  // Use the original file object
                toType: "image/jpeg",
                quality: 0.7, // adjust quality as needed
            }) as Blob;
            // Return as object URL
            return URL.createObjectURL(outputBlob);
        } catch (error) {
            console.error(error);
            PubSub.get().publish("alertDialog", {
                messageKey: "HeicConvertNotLoaded",
                buttons: [{ labelKey: "Ok" }],
            });
        } finally {
            PubSub.get().publish("menu", { id: ELEMENT_IDS.FullPageSpinner, data: { show: false } });
        }
    }
    // If not a HEIC/HEIF file, proceed as normal
    else {
        // But if it's a GIF, notify the user that it will be converted to a static image
        if (ext === "gif") {
            PubSub.get().publish("snack", { messageKey: "GifWillBeStatic", severity: "Warning" });
        }
        return URL.createObjectURL(file);
    }
    return undefined;
}

export function ProfilePictureInput({
    name,
    onBannerImageChange,
    onProfileImageChange,
    profile,
}: ProfilePictureInputProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const session = useContext(SessionContext);

    const [bannerImageUrl, setBannerImageUrl] = useState(extractImageUrl(profile?.bannerImage, profile?.updatedAt, BANNER_TARGET_SIZE));
    const [profileImageUrl, setProfileImageUrl] = useState(extractImageUrl(profile?.profileImage, profile?.updatedAt, PROFILE_TARGET_SIZE));
    useEffect(() => {
        setBannerImageUrl(extractImageUrl(profile?.bannerImage, profile?.updatedAt, BANNER_TARGET_SIZE));
        setProfileImageUrl(extractImageUrl(profile?.profileImage, profile?.updatedAt, PROFILE_TARGET_SIZE));
    }, [profile]);

    // Colorful placeholder if no image, or white if there is an image (in case there's transparency)
    const profileColors = useMemo<[string, string]>(() =>
        profileImageUrl ? ["#fff", "#fff"] : placeholderColor(getCurrentUser(session).id),
        [profileImageUrl, session]);

    // Get the banner colors - we'll use alpha() to make them duller
    const [baseColor1, baseColor2] = useMemo(() =>
        placeholderColor(getCurrentUser(session).id),
        [session]);

    const { getRootProps: getBannerRootProps, getInputProps: getBannerInputProps } = useDropzone({
        accept: ["image/*", ".heic", ".heif"],
        maxFiles: 1,
        onDrop: async acceptedFiles => {
            // Ignore if no files were uploaded
            if (acceptedFiles.length <= 0) {
                console.error("No files were uploaded");
                return;
            }
            // Process first file, and ignore any others
            handleImagePreview(acceptedFiles[0]).then(preview => {
                Object.assign(acceptedFiles[0], { preview });
                onBannerImageChange(acceptedFiles[0]);
                setBannerImageUrl(preview);
            });
        },
    });
    const { getRootProps: getProfileRootProps, getInputProps: getProfileInputProps } = useDropzone({
        accept: ["image/*", ".heic", ".heif"],
        maxFiles: 1,
        onDrop: async acceptedFiles => {
            // Ignore if no files were uploaded
            if (acceptedFiles.length <= 0) {
                console.error("No files were uploaded");
                return;
            }
            // Process first file, and ignore any others
            handleImagePreview(acceptedFiles[0]).then(preview => {
                Object.assign(acceptedFiles[0], { preview });
                onProfileImageChange(acceptedFiles[0]);
                setProfileImageUrl(preview);
            });
        },
    });

    const removeBannerImage = useCallback(function removeBannerImageCallback(event: React.MouseEvent) {
        event.stopPropagation();
        setBannerImageUrl(undefined);
        onBannerImageChange(null);
    }, [onBannerImageChange]);
    const removeProfileImage = useCallback(function removeProfileImageCallback(event: React.MouseEvent) {
        event.stopPropagation();
        setProfileImageUrl(undefined);
        onProfileImageChange(null);
    }, [onProfileImageChange]);

    /** Fallback icon displayed when profile image is not available */
    const profileIconInfo = useMemo(() => {
        if (!profile) return { name: "Edit", type: "Common" } as const;
        if (profile.__typename === "Team") return { name: "Team", type: "Common" } as const;
        if (profile.isBot) return { name: "Bot", type: "Common" } as const;
        return { name: "Profile", type: "Common" } as const;
    }, [profile]);

    // Extract styles as constants to fix linter errors
    const bannerStyle = useMemo(() => {
        if (bannerImageUrl) {
            return {
                backgroundColor: "#fff",
                backgroundImage: `url(${bannerImageUrl})`,
                backgroundSize: "cover",
                backgroundPosition: "center",
            };
        }

        // Use MUI's alpha function to create duller colors - it's basically opacity
        return {
            background: `linear-gradient(135deg, ${alpha(baseColor1, GRADIENT_START_ALPHA)} 0%, ${alpha(baseColor2, GRADIENT_END_ALPHA)} 100%)`,
            color: "white",
            "& svg": {
                fill: "white",
            },
        };
    }, [bannerImageUrl, baseColor1, baseColor2]);

    // Define the Banner component's style separately
    const bannerBoxProps = {
        width: "100%",
        height: "200px",
        borderRadius: 1,
        boxShadow: 2,
        sx: bannerStyle,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        cursor: "pointer",
    };

    const editButtonStyle = useMemo(() => ({
        background: palette.secondary.main,
    }), [palette.secondary.main]);

    const deleteButtonStyle = useMemo(() => ({
        background: palette.error.main,
    }), [palette.error.main]);

    const profileBoxStyle = useMemo(() => ({
        // Left-aligned position for profile image
        left: "5%",
        transform: "none",
    }), []);

    const placeholderTextStyle = useMemo(() => ({
        fontSize: "14px",
        fontWeight: "medium",
        opacity: 0.9,
        pointerEvents: "none",
    }), []);

    return (
        <Box
            marginBottom="56px"
            marginTop={2}
            position="relative"
            width="100%"
        >
            {/* Banner image */}
            <Box {...getBannerRootProps()}>
                <input name={name ?? "banner"} {...getBannerInputProps()} />
                <Box
                    {...bannerBoxProps}
                >
                    {/* Add a helper icon and text when no banner image is set */}
                    {!bannerImageUrl && (
                        <Stack
                            direction="column"
                            alignItems="center"
                            justifyContent="center"
                            spacing={2}
                            height="100%"
                            width="100%"
                        >
                            <IconCommon
                                name="Image"
                                fill={baseColor2}
                                opacity={0.7}
                                size={48}
                            />
                            <Box sx={placeholderTextStyle}>
                                {t("ClickToUploadBanner")}
                            </Box>
                        </Stack>
                    )}
                </Box>
                <Stack
                    direction="row"
                    spacing={0.5}
                    position="absolute"
                    top="-16px"
                    right="0px"
                    zIndex={Z_INDEX.PageElement + 1}
                >
                    <IconButton
                        variant="transparent"
                        size="md"
                        aria-label={t("Edit")}
                        style={editButtonStyle}
                    >
                        <IconCommon
                            decorative
                            fill={palette.secondary.contrastText}
                            name="Edit"
                            size={24}
                        />
                    </IconButton>
                    {bannerImageUrl !== undefined && (
                        <IconButton
                            variant="transparent"
                            size="md"
                            aria-label={t("Delete")}
                            onClick={removeBannerImage}
                            style={deleteButtonStyle}
                        >
                            <IconCommon
                                fill={palette.secondary.contrastText}
                                name="Delete"
                                size={24}
                            />
                        </IconButton>
                    )}
                </Stack>
            </Box>
            {/* Profile image */}
            <Box
                position="absolute"
                bottom="-60px"
                sx={profileBoxStyle}
                {...getProfileRootProps()}
            >
                <input name={name ?? "picture"} {...getProfileInputProps()} />
                <ProfilePictureInputAvatar
                    isBot={profile?.isBot ?? false}
                    profileColors={profileColors}
                    src={profileImageUrl}
                >
                    <Icon
                        decorative
                        info={profileIconInfo}
                    />
                </ProfilePictureInputAvatar>
                <Stack
                    direction="row"
                    position="absolute"
                    right={profileImageUrl !== undefined ? "-56px" : "-8px"}
                    spacing={0.5}
                    top="-16px"
                    zIndex={Z_INDEX.PageElement + 1}
                >
                    <IconButton
                        variant="transparent"
                        size="md"
                        aria-label={t("Edit")}
                        style={editButtonStyle}
                    >
                        <IconCommon
                            decorative
                            fill={palette.secondary.contrastText}
                            name="Edit"
                            size={24}
                        />
                    </IconButton>
                    {profileImageUrl !== undefined && (
                        <IconButton
                            variant="transparent"
                            size="md"
                            aria-label={t("Delete")}
                            onClick={removeProfileImage}
                            style={deleteButtonStyle}
                        >
                            <IconCommon
                                fill={palette.secondary.contrastText}
                                name="Delete"
                                size={24}
                            />
                        </IconButton>
                    )}
                </Stack>
            </Box>
        </Box>
    );
}
