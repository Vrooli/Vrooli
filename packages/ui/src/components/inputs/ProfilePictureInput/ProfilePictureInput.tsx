import { Avatar, Box, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { BotIcon, DeleteIcon, EditIcon, OrganizationIcon, UserIcon } from "icons";
import { useEffect, useMemo, useState } from "react";
import { useDropzone } from "react-dropzone";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { ProfilePictureInputProps } from "../types";

// What size image to display
const TARGET_SIZE = 100;

export const ProfilePictureInput = ({
    name,
    onChange,
    profile,
    zIndex,
}: ProfilePictureInputProps) => {
    const { palette } = useTheme();

    const profileColors = useMemo(() => placeholderColor(), []);
    const [imgUrl, setImgUrl] = useState(extractImageUrl(profile?.profileImage, profile?.updated_at, TARGET_SIZE));
    useEffect(() => {
        setImgUrl(extractImageUrl(profile?.profileImage, profile?.updated_at, TARGET_SIZE));
    }, [profile]);

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
        } else {
            // If not a HEIC/HEIF file, proceed as normal
            return URL.createObjectURL(file);
        }
    };

    const { getRootProps, getInputProps } = useDropzone({
        accept: ["image/*", ".heic", ".heif"],
        maxFiles: 1,
        onDrop: async acceptedFiles => {
            console.log("ondrop", acceptedFiles);
            // Ignore if no files were uploaded
            if (acceptedFiles.length <= 0) {
                console.error("No files were uploaded");
                return;
            }
            // Process first file, and ignore any others
            handleImagePreview(acceptedFiles[0]).then(preview => {
                console.log("got preview", preview);
                Object.assign(acceptedFiles[0], { preview });
                onChange(acceptedFiles[0]);
                setImgUrl(preview);
            });
        },
    });

    const removeImage = (event: any) => {
        event.stopPropagation();
        setImgUrl(undefined);
        onChange(null);
    };

    /** Fallback icon displayed when image is not available */
    const Icon = useMemo(() => {
        if (!profile) return EditIcon;
        if (profile.__typename === "Organization") return OrganizationIcon;
        if (profile.isBot) return BotIcon;
        return UserIcon;
    }, [profile]);

    return (
        <Box display="flex" justifyContent="center">
            <Box sx={{
                padding: 2,
                position: "relative",
                display: "inline-block",
            }} {...getRootProps()}>
                <input name={name ?? "picture"} {...getInputProps()} />
                <Avatar
                    src={imgUrl}
                    sx={{
                        backgroundColor: profileColors[0],
                        color: profileColors[1],
                        boxShadow: 4,
                        width: "min(100px, 25vw)",
                        height: "min(100px, 25vw)",
                        fontSize: "min(50px, 10vw)",
                        cursor: "pointer",
                    }}
                >
                    <Icon
                        width="75%"
                        height="75%"
                    />
                </Avatar>
                <ColorIconButton
                    background={palette.secondary.main}
                    sx={{
                        position: "absolute",
                        top: 0,
                        right: 0,
                        zIndex: zIndex + 1,
                    }}
                >
                    <EditIcon width="32px" height="32px" fill={palette.secondary.contrastText} />
                </ColorIconButton>
                {imgUrl !== undefined && (
                    <ColorIconButton
                        background={palette.error.main}
                        sx={{
                            position: "absolute",
                            bottom: 0,
                            right: 0,
                            zIndex: zIndex + 1,
                        }}
                        onClick={removeImage}
                    >
                        <DeleteIcon width="32px" height="32px" fill={palette.secondary.contrastText} />
                    </ColorIconButton>
                )}
            </Box>
        </Box>
    );
};
