import { Avatar, Box, Stack, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { BotIcon, DeleteIcon, EditIcon, OrganizationIcon, UserIcon } from "icons";
import { useEffect, useMemo, useState } from "react";
import { useDropzone } from "react-dropzone";
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
const handleImagePreview = async (file: any) => {
    // Extract extension from file name or path
    const ext = (file.name ? file.name.split(".").pop() : file.path.split(".").pop()).toLowerCase();
    // .heic and .heif files are not supported by browsers, 
    // so we need to convert them to JPEGs (thanks, Apple)
    if (ext === "heic" || ext === "heif") {
        PubSub.get().publishLoading(true);
        // Dynamic import of heic2any
        const heic2any = (await import("heic2any")).default;

        // Convert HEIC/HEIF file to JPEG Blob
        const outputBlob = await heic2any({
            blob: file,  // Use the original file object
            toType: "image/jpeg",
            quality: 0.7, // adjust quality as needed
        }) as Blob;
        PubSub.get().publishLoading(false);
        // Return as object URL
        return URL.createObjectURL(outputBlob);
    }
    // If not a HEIC/HEIF file, proceed as normal
    else {
        // But if it's a GIF, notify the user that it will be converted to a static image
        if (ext === "gif") {
            PubSub.get().publishSnack({ messageKey: "GifWillBeStatic", severity: "Warning" });
        }
        return URL.createObjectURL(file);
    }
};

export const ProfilePictureInput = ({
    name,
    onBannerImageChange,
    onProfileImageChange,
    profile,
    zIndex,
}: ProfilePictureInputProps) => {
    const { palette } = useTheme();

    const [bannerImageUrl, setBannerImageUrl] = useState(extractImageUrl(profile?.bannerImage, profile?.updated_at, BANNER_TARGET_SIZE));
    const [profileImageUrl, setProfileImageUrl] = useState(extractImageUrl(profile?.profileImage, profile?.updated_at, PROFILE_TARGET_SIZE));
    useEffect(() => {
        setBannerImageUrl(extractImageUrl(profile?.bannerImage, profile?.updated_at, BANNER_TARGET_SIZE));
        setProfileImageUrl(extractImageUrl(profile?.profileImage, profile?.updated_at, PROFILE_TARGET_SIZE));
    }, [profile]);
    // Colorful placeholder if no image, or white if there is an image (in case there's transparency)
    const profileColors = useMemo(() => profileImageUrl ? ["#fff", "#fff"] : placeholderColor(), [profileImageUrl]);

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
            console.log("profile image dropped", acceptedFiles);
            // Ignore if no files were uploaded
            if (acceptedFiles.length <= 0) {
                console.error("No files were uploaded");
                return;
            }
            // Process first file, and ignore any others
            handleImagePreview(acceptedFiles[0]).then(preview => {
                console.log("preview", preview);
                Object.assign(acceptedFiles[0], { preview });
                onProfileImageChange(acceptedFiles[0]);
                setProfileImageUrl(preview);
            });
        },
    });

    const removeBannerImage = (event: any) => {
        event.stopPropagation();
        setBannerImageUrl(undefined);
        onBannerImageChange(null);
    };
    const removeProfileImage = (event: any) => {
        event.stopPropagation();
        setProfileImageUrl(undefined);
        onProfileImageChange(null);
    };

    /** Fallback icon displayed when profile image is not available */
    const ProfileIcon = useMemo(() => {
        if (!profile) return EditIcon;
        if (profile.__typename === "Organization") return OrganizationIcon;
        if (profile.isBot) return BotIcon;
        return UserIcon;
    }, [profile]);

    return (
        <Box position="relative" sx={{ marginBottom: "56px" }}>
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
                    <ColorIconButton background={palette.secondary.main}>
                        <EditIcon width="24px" height="24px" fill={palette.secondary.contrastText} />
                    </ColorIconButton>
                    {bannerImageUrl !== undefined && (
                        <ColorIconButton
                            background={palette.error.main}
                            onClick={removeBannerImage}
                        >
                            <DeleteIcon width="24px" height="24px" fill={palette.secondary.contrastText} />
                        </ColorIconButton>
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
                <Avatar
                    src={profileImageUrl}
                    sx={{
                        backgroundColor: profileColors[0],
                        color: profileColors[1],
                        boxShadow: 4,
                        width: "100px",
                        height: "100px",
                        cursor: "pointer",
                        // Bots show up as squares, to distinguish them from users
                        ...(profile?.isBot ? { borderRadius: "8px" } : {}),
                    }}
                >
                    <ProfileIcon
                        width="75%"
                        height="75%"
                    />
                </Avatar>
                <Stack direction="row" spacing={0.5} sx={{
                    position: "absolute",
                    top: "-16px",
                    right: profileImageUrl !== undefined ? "-56px" : "-8px",
                    zIndex: zIndex + 1,
                }}>
                    <ColorIconButton background={palette.secondary.main}>
                        <EditIcon width="24px" height="24px" fill={palette.secondary.contrastText} />
                    </ColorIconButton>
                    {profileImageUrl !== undefined && (
                        <ColorIconButton
                            background={palette.error.main}
                            onClick={removeProfileImage}
                        >
                            <DeleteIcon width="24px" height="24px" fill={palette.secondary.contrastText} />
                        </ColorIconButton>
                    )}
                </Stack>
            </Box>
        </Box>
    );
};
