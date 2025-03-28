import { Box, IconButton, Stack, useTheme } from "@mui/material";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useDropzone } from "react-dropzone";
import { useTranslation } from "react-i18next";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { ProfilePictureInputAvatar } from "../../../styles.js";
import { ELEMENT_IDS, Z_INDEX } from "../../../utils/consts.js";
import { extractImageUrl } from "../../../utils/display/imageTools.js";
import { placeholderColor } from "../../../utils/display/listTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ProfilePictureInputProps } from "../types.js";

// What size image to display
const BANNER_TARGET_SIZE = 1000;
const PROFILE_TARGET_SIZE = 100;

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

    const [bannerImageUrl, setBannerImageUrl] = useState(extractImageUrl(profile?.bannerImage, profile?.updated_at, BANNER_TARGET_SIZE));
    const [profileImageUrl, setProfileImageUrl] = useState(extractImageUrl(profile?.profileImage, profile?.updated_at, PROFILE_TARGET_SIZE));
    useEffect(() => {
        setBannerImageUrl(extractImageUrl(profile?.bannerImage, profile?.updated_at, BANNER_TARGET_SIZE));
        setProfileImageUrl(extractImageUrl(profile?.profileImage, profile?.updated_at, PROFILE_TARGET_SIZE));
    }, [profile]);
    // Colorful placeholder if no image, or white if there is an image (in case there's transparency)
    const profileColors = useMemo<[string, string]>(() => profileImageUrl ? ["#fff", "#fff"] : placeholderColor(), [profileImageUrl]);

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
        return { name: "User", type: "Common" } as const;
    }, [profile]);

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
                    width="100%"
                    height="200px"
                    borderRadius={1}
                    boxShadow={2}
                    sx={{
                        backgroundColor: bannerImageUrl ? "#fff" : (palette.mode === "light" ? "#b2b3b3" : "#303030"),
                        backgroundImage: bannerImageUrl ? `url(${bannerImageUrl})` : undefined,
                        backgroundSize: "cover",
                        backgroundPosition: "center",
                    }}
                />
                <Stack
                    direction="row"
                    spacing={0.5}
                    position="absolute"
                    top="-16px"
                    right="0px"
                    zIndex={Z_INDEX.PageElement + 1}
                >
                    <IconButton
                        aria-label={t("Edit")}
                        sx={{ background: palette.secondary.main }}
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
                            aria-label={t("Delete")}
                            onClick={removeBannerImage}
                            sx={{ background: palette.error.main }}
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
                left="50%"
                bottom="-25px"
                sx={{ transform: "translateX(-50%)" }}
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
                        aria-label={t("Edit")}
                        sx={{ background: palette.secondary.main }}
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
                            aria-label={t("Delete")}
                            onClick={removeProfileImage}
                            sx={{ background: palette.error.main }}
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
