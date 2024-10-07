import { Box, IconButton, Stack, useTheme } from "@mui/material";
import { useZIndex } from "hooks/useZIndex";
import { BotIcon, DeleteIcon, EditIcon, TeamIcon, UserIcon } from "icons";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useDropzone } from "react-dropzone";
import { ProfilePictureInputAvatar } from "styles";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { ProfilePictureInputProps } from "../types";

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
        PubSub.get().publish("loading", true);
        // Dynamic import of heic2any
        const heic2any = (await import("utils/heic/heic2any")).default;

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
            PubSub.get().publish("loading", false);
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
    const { palette } = useTheme();
    const zIndex = useZIndex();

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
    const ProfileIcon = useMemo(() => {
        if (!profile) return EditIcon;
        if (profile.__typename === "Team") return TeamIcon;
        if (profile.isBot) return BotIcon;
        return UserIcon;
    }, [profile]);

    return (
        <Box position="relative" sx={{ marginBottom: "56px", marginTop: 2, width: "100%" }}>
            {/* Banner image */}
            <Box {...getBannerRootProps()}>
                <input name={name ?? "banner"} {...getBannerInputProps()} />
                <Box
                    sx={{
                        width: "100%",
                        height: "200px",
                        backgroundColor: bannerImageUrl ? "#fff" : (palette.mode === "light" ? "#b2b3b3" : "#303030"),
                        backgroundImage: bannerImageUrl ? `url(${bannerImageUrl})` : undefined,
                        backgroundSize: "cover",
                        backgroundPosition: "center",
                        borderRadius: 1,
                        boxShadow: 2,
                    }}
                />
                <Stack direction="row" spacing={0.5} sx={{
                    position: "absolute",
                    top: "-16px",
                    right: "0px",
                    zIndex: zIndex + 1,
                }}>
                    <IconButton sx={{ background: palette.secondary.main }}>
                        <EditIcon width="24px" height="24px" fill={palette.secondary.contrastText} />
                    </IconButton>
                    {bannerImageUrl !== undefined && (
                        <IconButton
                            onClick={removeBannerImage}
                            sx={{ background: palette.error.main }}
                        >
                            <DeleteIcon width="24px" height="24px" fill={palette.secondary.contrastText} />
                        </IconButton>
                    )}
                </Stack>
            </Box>
            {/* Profile image */}
            <Box sx={{
                position: "absolute",
                left: "50%",
                transform: "translateX(-50%)",
                bottom: "-25px",
            }} {...getProfileRootProps()}>
                <input name={name ?? "picture"} {...getProfileInputProps()} />
                <ProfilePictureInputAvatar
                    isBot={profile?.isBot ?? false}
                    profileColors={profileColors}
                    src={profileImageUrl}
                >
                    <ProfileIcon
                        width="75%"
                        height="75%"
                    />
                </ProfilePictureInputAvatar>
                <Stack direction="row" spacing={0.5} sx={{
                    position: "absolute",
                    top: "-16px",
                    right: profileImageUrl !== undefined ? "-56px" : "-8px",
                    zIndex: zIndex + 1,
                }}>
                    <IconButton sx={{ background: palette.secondary.main }}>
                        <EditIcon width="24px" height="24px" fill={palette.secondary.contrastText} />
                    </IconButton>
                    {profileImageUrl !== undefined && (
                        <IconButton
                            onClick={removeProfileImage}
                            sx={{ background: palette.error.main }}
                        >
                            <DeleteIcon width="24px" height="24px" fill={palette.secondary.contrastText} />
                        </IconButton>
                    )}
                </Stack>
            </Box>
        </Box>
    );
}
