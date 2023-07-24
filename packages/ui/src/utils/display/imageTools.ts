/**
 * Finds the best image URL to use for the given field and target size.
 * @param imageData The image field
 * @param lastUpdated The last updated timestamp of the source object. Used for cache busting.
 * @param size Target size of the image.
 * @returns The best image URL to use for the given field and target size, or undefined if no URL is found.
 */
export const extractImageUrl = (
    imageData: string | File | null | undefined,
    lastUpdated: string | null | undefined,
    targetSize: number,
): string | undefined => {
    // Make sure the image data is a string
    if (!imageData || typeof imageData !== "string") return undefined;
    // Image data can either be a URL or a stringified JSON object containing the filename base and available sizes
    if (imageData.startsWith("http")) {
        return imageData;
    }
    try {
        const { file, sizes } = JSON.parse(imageData) as { file: string; sizes: string[] };

        // Find the size that is closest to the targetSize. 
        // NOTE: This only considers image width
        const closestSize = sizes.reduce((prev, curr) => {
            const prevSize = Number(prev.split("x")[0]);
            const currSize = Number(curr.split("x")[0]);
            return (Math.abs(currSize - targetSize) < Math.abs(prevSize - targetSize) ? curr : prev);
        });

        // Replace wildcard with appropriate size, and add cache busting query param
        return `${file.replace("_*", `_${closestSize}`)}?v=${lastUpdated}`;
    } catch (error) {
        console.error("Could not parse image data", imageData);
    }
    return undefined;
};
