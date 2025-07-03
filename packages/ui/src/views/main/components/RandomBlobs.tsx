/* eslint-disable no-magic-numbers */
import Box, { type BoxProps } from "@mui/material/Box";
import { styled } from "@mui/material/styles";
import { useEffect, useState } from "react";
import Blob1 from "../../../assets/img/blob1.svg";
import Blob2 from "../../../assets/img/blob2.svg";

// Blob properties
/** Blur factor for odd-numbered blobs */
const BLUR_DIVISOR_ODD = 16;
/** Blur factor for even-numbered blobs */
const BLUR_DIVISOR_EVEN = 50;
const ODD_SIZE_BASE = 0.4;
const EVEN_SIZE_BASE = 0.6;
const RANDOM_SIZE_RANGE = 0.3;
const POSITION_OFFSET_MIN = 0.1;
const POSITION_OFFSET_RANGE = 0.8;
const COOL_HUE_MIN = 120; // Green
const COOL_HUE_MAX = 300; // Purple

type BlobInfo = {
    id: number;
    imgSrc: string;
    top: number;
    left: number;
    width: number;
    height: number;
    hueRotate: number;
};

type Dimensions = {
    width: number;
    height: number;
};

function calculateBlobPosition(percent: number, dimension: number, offset: number) {
    return (percent * dimension) - (offset / 2);
}

function calculateBlobSize(percent: number, { width, height }: Dimensions) {
    return percent * Math.min(width, height);
}

function calculateBlobBlur(widthPercent: number, heightPercent: number, { width, height }: Dimensions, id: number) {
    return Math.floor((widthPercent + heightPercent) * (width + height) / (id % 2 ? BLUR_DIVISOR_ODD : BLUR_DIVISOR_EVEN));
}

interface BlobBoxProps extends BoxProps {
    blob: BlobInfo;
    viewport: Dimensions;
}

const BlobBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "blob" && prop !== "viewport",
})<BlobBoxProps>(({ blob, viewport }) => {
    const width = calculateBlobSize(blob.width, viewport);
    const height = calculateBlobSize(blob.height, viewport);
    const top = calculateBlobPosition(blob.top, viewport.height, height);
    const left = calculateBlobPosition(blob.left, viewport.width, width);

    return {
        position: "fixed",
        pointerEvents: "none",
        top: top + "px",
        left: left + "px",
        width: width + "px",
        height: height + "px",
        opacity: 0.5,
        filter: `hue-rotate(${blob.hueRotate}deg) blur(${calculateBlobBlur(blob.width, blob.height, viewport, blob.id)}px)`,
        zIndex: 0,
    } as const;
});

const BlobImage = styled(Box)(() => ({
    maxWidth: "100%",
    maxHeight: "100%",
}));

export function RandomBlobs({ numberOfBlobs }: { numberOfBlobs: number }) {
    const [viewport, setViewport] = useState<Dimensions>({ width: window.innerWidth, height: window.innerHeight });
    const [blobs, setBlobs] = useState<BlobInfo[]>([]);

    useEffect(function generateBlobsEffect() {
        function generateInitialBlobs() {
            const newBlobs: BlobInfo[] = [];
            for (let i = 0; i < numberOfBlobs; i++) {
                const isOdd = i % 2 !== 0;
                const sizeBase = isOdd ? ODD_SIZE_BASE : EVEN_SIZE_BASE;
                const width = Math.random() * RANDOM_SIZE_RANGE + sizeBase;
                const height = Math.random() * RANDOM_SIZE_RANGE + sizeBase;
                const top = Math.random() * POSITION_OFFSET_RANGE + POSITION_OFFSET_MIN;
                const left = Math.random() * POSITION_OFFSET_RANGE + POSITION_OFFSET_MIN;

                newBlobs.push({
                    id: i,
                    imgSrc: isOdd ? Blob1 : Blob2,
                    top,
                    left,
                    width,
                    height,
                    hueRotate: COOL_HUE_MIN + Math.random() * (COOL_HUE_MAX - COOL_HUE_MIN),
                });
            }
            return newBlobs;
        }

        setBlobs(generateInitialBlobs());
    }, [numberOfBlobs]);

    useEffect(function resizeEffect() {
        function handleResize() {
            setViewport({ width: window.innerWidth, height: window.innerHeight });
        }

        window.addEventListener("resize", handleResize);
        return () => window.removeEventListener("resize", handleResize);
    }, []);

    return (
        <>
            {blobs.map(blob => (
                <BlobBox key={blob.id} blob={blob} viewport={viewport}>
                    <BlobImage
                        component="img"
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore
                        src={blob.imgSrc}
                        alt={`Blob ${blob.id}`}
                    />
                </BlobBox>
            ))}
        </>
    );
}
