import { Box, IconButton, Modal, type ModalProps, styled } from "@mui/material";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";

interface ImageModalProps extends Omit<ModalProps, "zIndex"> {
    zIndex?: number;
}
const ImageModal = styled(Modal, {
    shouldForwardProp: (prop) => prop !== "zIndex",
})<ImageModalProps>(({ zIndex }) => ({
    backgroundColor: "rgba(0,0,0,0.8)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex,
}));
const ImageModalInner = styled(Box)(() => ({
    position: "relative",
    width: "80%",
    height: "80%",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
}));
const ImageButton = styled("button")(() => ({
    background: "none",
    border: "none",
    padding: 0,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    cursor: "pointer",
    maxWidth: "100%",
    maxHeight: "100%",
}));
const StyledImg = styled("img")(() => ({
    maxWidth: "100%",
    maxHeight: "100%",
    objectFit: "contain",
}));
const CloseIconButton = styled(IconButton)(() => ({
    position: "absolute",
    right: 20,
    top: 20,
    color: "white",
}));

type ImagePopupProps = {
    alt: string;
    onClose: () => unknown;
    open: boolean;
    src: string;
    zIndex?: number;
};

export function ImagePopup({
    alt,
    onClose,
    open,
    src,
    zIndex,
}: ImagePopupProps) {
    const { t } = useTranslation();

    const stopPropagation = useCallback((e: React.MouseEvent) => {
        e.stopPropagation();
    }, []);

    return (
        <ImageModal
            aria-describedby="full-size-screenshot"
            aria-labelledby="full-size-image"
            onClose={onClose}
            open={open}
            zIndex={zIndex}
        >
            <ImageModalInner onClick={onClose}>
                <ImageButton
                    onClick={stopPropagation}
                    aria-label={alt}
                >
                    <StyledImg
                        src={src}
                        alt={alt}
                    />
                </ImageButton>
                <CloseIconButton
                    aria-label={t("Close")}
                    onClick={onClose}
                >
                    <IconCommon
                        decorative
                        fill="white"
                        name="Close"
                        size={24}
                    />
                </CloseIconButton>
            </ImageModalInner>
        </ImageModal>
    );
}

interface VideoModalProps extends Omit<ModalProps, "zIndex"> {
    zIndex?: number;
}
const VideoModal = styled(Modal, {
    shouldForwardProp: (prop) => prop !== "zIndex",
})<VideoModalProps>(({ zIndex }) => ({
    backgroundColor: "rgba(0,0,0,0.8)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex,
}));
const VideoModalInner = styled(Box)(() => ({
    position: "relative",
    width: "80%",
    height: "80%",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
}));

type VideoPopupProps = {
    onClose: () => unknown;
    open: boolean;
    src: string | null;
    zIndex?: number;
};

export function VideoPopup({
    onClose,
    open,
    src,
    zIndex,
}: VideoPopupProps) {
    const { t } = useTranslation();

    return (
        <VideoModal
            aria-describedby="video-modal-description"
            aria-labelledby="video-modal"
            onClose={onClose}
            open={open}
            zIndex={zIndex}
        >
            <VideoModalInner>
                {src && (
                    <iframe
                        width="100%"
                        height="100%"
                        src={src}
                        frameBorder="0"
                        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                        allowFullScreen
                        title="Video Tour"
                    ></iframe>
                )}
                <CloseIconButton
                    aria-label={t("Close")}
                    onClick={onClose}
                >
                    <IconCommon
                        decorative
                        fill="white"
                        name="Close"
                        size={24}
                    />
                </CloseIconButton>
            </VideoModalInner>
        </VideoModal>
    );
}

const ThumbnailContainer = styled(Box)(({ theme }) => ({
    position: "relative",
    width: "100%",
    maxHeight: 200,
    overflow: "hidden",
    borderRadius: theme.shape.borderRadius,
    cursor: "pointer",
    marginTop: theme.spacing(2),
    aspectRatio: "16/9",
}));
const ImageWrapper = styled(Box)(() => ({
    position: "relative",
    width: "100%",
    height: "100%",
}));
const ThumbnailImage = styled("img")(() => ({
    width: "100%",
    height: "100%",
    objectFit: "cover",
    display: "block",
}));
const Overlay = styled(Box)(({ theme }) => ({
    position: "absolute",
    inset: 0,
    backgroundColor: "rgba(0, 0, 0, 0.3)",
    opacity: 0,
    transition: theme.transitions.create("opacity"),
    "$ThumbnailContainer:hover &": {
        opacity: 1,
    },
}));
const PlayButtonWrapper = styled(Box)(() => ({
    position: "absolute",
    inset: 0,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
}));
const PlayIconBox = styled(Box)(({ theme }) => ({
    position: "absolute",
    top: "50%",
    left: "50%",
    transform: "translate(-50%, -50%)",
    display: "flex",
    padding: theme.spacing(1),
    background: "#bfbfbf88",
    borderRadius: "100%",
    zIndex: 10,
    "&:hover": {
        transform: "translate(-50%, -50%) scale(1.1)",
        background: "#bfbfbfcc",
    },
    transition: "all 0.2s ease",
}));

type VideoThumbnailProps = {
    onClick: () => unknown;
    src: string;
};

export function VideoThumbnail({
    onClick,
    src,
}: VideoThumbnailProps) {
    // Extract video ID from YouTube URL
    function getYouTubeId(url: string) {
        const match = url.match(/(?:youtu\.be\/|youtube\.com\/(?:embed\/|v\/|watch\?v=|watch\?.+&v=))([^&?]+)/);
        return match ? match[1] : null;
    }

    const videoId = getYouTubeId(src);
    const thumbnailUrl = videoId ? `https://img.youtube.com/vi/${videoId}/maxresdefault.jpg` : "";

    return (
        <ThumbnailContainer
            aria-label={"Play video"}
            onClick={onClick}
        >
            <ImageWrapper>
                <ThumbnailImage
                    src={thumbnailUrl}
                    alt="Video thumbnail"
                />
                <Overlay />
                <PlayButtonWrapper>
                    <PlayIconBox>
                        <IconCommon
                            decorative
                            fill="white"
                            name="Play"
                            size={40}
                        />
                    </PlayIconBox>
                </PlayButtonWrapper>
            </ImageWrapper>
        </ThumbnailContainer>
    );
}
