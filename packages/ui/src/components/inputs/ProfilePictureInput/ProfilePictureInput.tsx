import { Avatar, Box, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { BotIcon, DeleteIcon, EditIcon, OrganizationIcon, UserIcon } from "icons";
import { useEffect, useMemo, useState } from "react";
import { useDropzone } from "react-dropzone";
import { placeholderColor } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { ProfilePictureInputProps } from "../types";

export const ProfilePictureInput = ({
    name,
    onChange,
    profile,
    zIndex,
}: ProfilePictureInputProps) => {
    const { palette } = useTheme();

    const profileColors = useMemo(() => placeholderColor(), []);
    const [files, setFiles] = useState<any[]>([]);
    const defaultImg = "/broken-image.jpg";
    const [profileImg, setProfileImg] = useState(defaultImg);

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
            if (acceptedFiles.length <= 0) {
                console.error("No files were uploaded");
                return;
            }

            const previews = await Promise.all(
                acceptedFiles.map(file =>
                    handleImagePreview(file).then(preview =>
                        Object.assign(file, { preview }),
                    ),
                ),
            );

            setFiles(previews);
            onChange(acceptedFiles[0]);
        },
    });

    // Update profile image after file has been uploaded
    useEffect(() => {
        if (files.length > 0) {
            setProfileImg(files[0].preview);
        }
    }, [files]);

    const removeImage = (event: any) => {
        event.stopPropagation();
        setProfileImg(defaultImg);
        setFiles([]);
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
                    src={profileImg}
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
                {profileImg !== defaultImg && (
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
