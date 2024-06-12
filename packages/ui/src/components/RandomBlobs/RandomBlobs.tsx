import { Box } from "@mui/material";
import Blob1 from "assets/img/blob1.svg";
import Blob2 from "assets/img/blob2.svg";
import { useEffect, useState } from "react";

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

export const RandomBlobs = ({ numberOfBlobs }) => {
    const [viewport, setViewport] = useState<Dimensions>({ width: window.innerWidth, height: window.innerHeight });
    const [blobs, setBlobs] = useState<BlobInfo[]>([]);

    useEffect(() => {
        const generateInitialBlobs = () => {
            const newBlobs: BlobInfo[] = [];
            for (let i = 0; i < numberOfBlobs; i++) {
                const isOdd = i % 2 !== 0;
                const width = Math.random() * 0.3 + (i % 2 ? 0.4 : 0.6);
                const height = Math.random() * 0.3 + (i % 2 ? 0.4 : 0.6);
                newBlobs.push({
                    id: i,
                    imgSrc: isOdd ? Blob1 : Blob2,
                    top: Math.random() * 0.8 + 0.1,
                    left: Math.random() * 0.8 + 0.1,
                    width,
                    height,
                    hueRotate: Math.random() * 360,
                });
            }
            console.log("got blob info", newBlobs);
            return newBlobs;
        };

        setBlobs(generateInitialBlobs());
    }, [numberOfBlobs]);

    useEffect(() => {
        const handleResize = () => {
            setViewport({ width: window.innerWidth, height: window.innerHeight });
        };

        window.addEventListener("resize", handleResize);
        return () => window.removeEventListener("resize", handleResize);
    }, []);

    const calculatePosition = (percent: number, dimension: number, offset: number) => (percent * dimension) - (offset / 2);
    const calculateSize = (percent: number, { width, height }: Dimensions) => percent * Math.min(width, height);
    const calculateBlur = (widthPercent: number, heightPercent: number, { width, height }: Dimensions, id: number) =>
        Math.floor((widthPercent + heightPercent) * (width + height) / (id % 2 ? 16 : 50));

    return (
        <>
            {blobs.map(blob => {
                const width = calculateSize(blob.width, viewport);
                const height = calculateSize(blob.height, viewport);
                const top = calculatePosition(blob.top, viewport.height, height);
                const left = calculatePosition(blob.left, viewport.width, width);
                return (
                    <Box
                        key={blob.id}
                        sx={{
                            position: "fixed",
                            pointerEvents: "none",
                            top: top + "px",
                            left: left + "px",
                            width: width + "px",
                            height: height + "px",
                            opacity: 0.5,
                            filter: `hue-rotate(${blob.hueRotate}deg) blur(${calculateBlur(blob.width, blob.height, viewport, blob.id)}px)`,
                            zIndex: 0,
                        }}
                    >
                        <Box
                            component="img"
                            src={blob.imgSrc}
                            alt={`Blob ${blob.id}`}
                            sx={{
                                maxWidth: "100%",
                                maxHeight: "100%",
                            }}
                        />
                    </Box>
                );
            })}
        </>
    );
};
