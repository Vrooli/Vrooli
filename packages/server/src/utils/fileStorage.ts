import { PutObjectCommand, S3Client } from "@aws-sdk/client-s3";
import { HttpStatus, type SessionUser, nanoid } from "@local/shared";
import http from "http"; // Import http for internal service call
import sharp from "sharp";
import { type UploadConfig } from "../endpoints/rest.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { type RequestFile } from "../types.js";
// Import FormData for multipart request
import FormData from "form-data";

// Global S3 client variable
let s3: S3Client | undefined;
const REGION = "us-east-1";
const BUCKET_NAME = "vrooli-bucket";

/**
 * Function to setup S3 client. This is needed because 
 * the environment variables are loaded from the secrets location, so they're 
 * not available at startup.
 */
export function getS3Client(): S3Client {
    if (!s3) {
        s3 = new S3Client({ region: REGION });
    }
    return s3 as S3Client;
}

/**
 * Post the image to the NSFW detection API (running as a local Docker service) and get the results.
 * @param buffer The image data.
 * @param originalFileName The original name of the file (used by the API).
 * @returns A boolean indicating if the image is NSFW.
 */
export async function checkNSFW(buffer: Buffer, originalFileName: string): Promise<boolean> {
    const NSFW_DETECTOR_TIMEOUT_MS = 10000; // 10 seconds

    return new Promise((resolve, reject) => {
        let requestHandled = false; // Flag to prevent multiple resolves/rejects

        const formData = new FormData();
        formData.append("file", buffer, originalFileName);

        const options: http.RequestOptions = {
            hostname: "nsfw-detector",
            port: 8000,
            path: "/v1/detect",
            method: "POST",
            headers: formData.getHeaders(),
        };

        const req = http.request(options, (res) => {
            if (requestHandled) return;
            let responseBody = "";
            res.on("data", (chunk) => {
                responseBody += chunk;
            });
            res.on("end", () => {
                if (requestHandled) return;
                requestHandled = true;
                clearTimeout(timeoutId); // Clear the watchdog timeout

                const HTTP_STATUS_REDIRECT_START = 300;
                if (res.statusCode && res.statusCode >= HttpStatus.Ok && res.statusCode < HTTP_STATUS_REDIRECT_START) {
                    try {
                        const result = JSON.parse(responseBody) as SafeContentAIResponse;
                        resolve(result.is_nsfw);
                    } catch (error) {
                        logger.error("Failed to parse nsfw-detector response", { trace: "checkNSFW-parse-err", error, responseBody });
                        reject(new Error("Failed to parse NSFW detection response"));
                    }
                } else {
                    logger.error("nsfw-detector service returned error status", { trace: "checkNSFW-status-err", statusCode: res.statusCode, responseBody });
                    reject(new Error(`NSFW detection service failed with status ${res.statusCode}`));
                }
            });
        });

        const timeoutId = setTimeout(() => {
            if (requestHandled) return;
            requestHandled = true;
            req.destroy();
            logger.error("NSFW detector request timed out", { trace: "checkNSFW-timeout-err" });
            reject(new Error("Request to NSFW detector timed out after " + (NSFW_DETECTOR_TIMEOUT_MS / 1000) + " seconds"));
        }, NSFW_DETECTOR_TIMEOUT_MS);

        req.on("error", (error) => {
            if (requestHandled) return;
            requestHandled = true;
            clearTimeout(timeoutId); // Clear the watchdog timeout
            logger.error("Error calling nsfw-detector service", { trace: "checkNSFW-req-err", error });
            reject(error);
        });

        formData.pipe(req);
    });
}

// heic-convert has to defer initialization because (presumably) the wasm file messes up the compiler error logs
let heicConvert;
async function getHeicConvert() {
    if (!heicConvert) {
        heicConvert = (await import("heic-convert")).default;
    }
    return heicConvert;
}

// file-type has to defer because jest tests don't like it for some reason
type FileTypeResult = {
    readonly ext: string;
    readonly mime: string;
};
let fileTypeFromBuffer: (buffer: Uint8Array | ArrayBuffer) => Promise<FileTypeResult | undefined>;
async function getFileType(buffer: Uint8Array | ArrayBuffer) {
    if (!fileTypeFromBuffer) {
        const fileTypePkg = await import("file-type");
        // Adjust import based on likely package structure
        const { fileTypeFromBuffer: func } = fileTypePkg;
        fileTypeFromBuffer = func;
    }
    return fileTypeFromBuffer(buffer);
}

/** Common configuration for profile images */
export const profileImageConfig = {
    allowedExtensions: ["png", "jpg", "jpeg", "webp", "svg", "gif", "heic", "heif"], // gif will lose animation
    imageSizes: [
        { width: 400, height: 400 },
        { width: 200, height: 200 },
        { width: 100, height: 100 },
        { width: 50, height: 50 },
    ],
};

/** Common configuration for banner images */
export const bannerImageConfig = {
    allowedExtensions: ["png", "jpg", "jpeg", "webp", "svg", "gif", "heic", "heif"], // gif will lose animation
    imageSizes: [
        { width: 1500, height: 500 },
        { width: 900, height: 300 },
        { width: 600, height: 200 },
        { width: 300, height: 100 },
    ],
};

/** Response from the safe-content-ai API */
interface SafeContentAIResponse {
    file_name: string;
    is_nsfw: boolean;
    confidence_percentage: number;
}

async function resizeImage(buffer: Buffer, width: number, height: number, format: "jpeg" | "png" | "webp" = "jpeg") {
    return await sharp(buffer)
        // Don't enlarge the image if it's already smaller than the requested size.
        .resize(width, height, { withoutEnlargement: true })
        // Replace transparent pixels with white.
        .flatten({ background: { r: 255, g: 255, b: 255 } })
        // Convert to the requested format.
        .toFormat(format)
        // Convert to a buffer.
        .toBuffer();
}

async function convertHeicToJpeg(buffer: Buffer): Promise<Buffer> {
    const convert = await getHeicConvert();
    const outputBuffer = await convert({
        buffer,  // the HEIC file buffer
        format: "JPEG", // output format
        quality: 1, // the jpeg compression quality, between 0 and 1
    });
    return Buffer.from(outputBuffer);
}

/** Supported image types/extensions */
const IMAGE_TYPES = ["jpeg", "jpg", "png", "gif", "webp", "tiff", "bmp"];

/** Uploads a file to S3 */
async function uploadFile(
    s3: S3Client,
    key: string,
    body: Buffer,
    mimetype: string,
) {
    const S3_UPLOAD_TIMEOUT_MS = 60000; // 60 seconds
    let timeoutId: NodeJS.Timeout | undefined = undefined;
    let requestHandled = false; // Prevent multiple resolutions/rejections

    const s3Promise = (async () => {
        const command = new PutObjectCommand({
            Bucket: BUCKET_NAME,
            Key: key,
            Body: body,
            ContentType: mimetype,
        });
        await s3.send(command);
        return `https://${BUCKET_NAME}.s3.${REGION}.amazonaws.com/${key}`;
    })();

    const timeoutPromise = new Promise<string>((_, reject) => {
        timeoutId = setTimeout(() => {
            if (requestHandled) return;
            requestHandled = true;
            logger.error("S3 upload operation timed out", { trace: "s3-upload-timeout", key });
            reject(new Error("S3 upload timed out after " + (S3_UPLOAD_TIMEOUT_MS / 1000) + " seconds"));
        }, S3_UPLOAD_TIMEOUT_MS);
    });

    try {
        const result = await Promise.race([s3Promise, timeoutPromise]);
        if (requestHandled) { // If timeout already handled, result might be from a completed s3Promise that raced against a fired timeout
            if (result !== undefined && !result.startsWith('https://')) { // Check if it's not the S3 URL (i.e. it was the timeout error)
                // This case should ideally not be hit if requestHandled is set correctly by timeoutPromise's reject
            } else if (result === undefined && !requestHandled) {
                // This case means s3Promise resolved with undefined, also unexpected
            }
        }
        requestHandled = true;
        if (timeoutId) clearTimeout(timeoutId);
        return result;
    } catch (error) {
        requestHandled = true;
        if (timeoutId) clearTimeout(timeoutId);
        // Log the original S3 error or the timeout error
        if (!(error instanceof Error && error.message.includes("S3 upload timed out"))) {
            logger.error("Failed to upload file to S3", { trace: "s3-upload-fail", key, mimetype, error });
        }
        throw error; // Re-throw the error after logging or timeout
    }
}

/**
 * Asynchronously processes and uploads files to an Amazon S3 bucket.
 */
export async function processAndStoreFiles<TInput>(
    files: RequestFile[],
    input: TInput,
    userData: SessionUser,
    config?: UploadConfig<TInput>,
): Promise<{ [x: string]: string[] }> {
    const s3 = getS3Client();

    // Keep track of the number of files processed for each field, so 
    // we can ensure they don't exceed the maximum defined in the config.
    const fileCounts: { [fieldName: string]: number } = {};

    // Create promise for each file
    const fileUrlsPromises = files.map(async file => {
        const urls: string[] = [];

        // Increment the count for this field
        fileCounts[file.fieldname] = (fileCounts[file.fieldname] || 0) + 1;

        // Find the correct input configuration for this file
        const fileConfig = config?.fields?.find(field => field.fieldName === file.fieldname);

        // Check if the maximum number of files for this field has been exceeded
        if (fileConfig?.maxFiles !== undefined && fileCounts[file.fieldname] > fileConfig.maxFiles) {
            throw new CustomError("0524", "MaxFilesExceeded", { file: file.originalname });
        }

        // Check if the file size exceeds the maximum allowed size
        if (fileConfig?.maxFileSize !== undefined && file.size > fileConfig.maxFileSize) {
            throw new CustomError("0525", "MaxFileSizeExceeded", { file: file.originalname });
        }

        // Get the file name
        const filenameBase = fileConfig?.fileNameBase ? fileConfig.fileNameBase(input, userData) : nanoid();

        // Find extension using the beginning of the file buffer. 
        // This is safer than using the file extension from the original file name, 
        // as malicious users can spoof the file extension.
        const fileType = await getFileType(Uint8Array.from(file.buffer));
        let extension = fileType?.ext;
        if (!extension) {
            throw new CustomError("0502", "InternalError", { file: file.originalname });
        }

        // Check if the file extension is allowed
        const allowedExtensions = fileConfig?.allowedExtensions ?? ["txt", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "heic", "heif", "png", "jpg", "jpeg", "gif", "webp", "tiff", "bmp"];
        if (!allowedExtensions.includes(extension)) {
            throw new CustomError("0508", "InternalError", { file: file.originalname });
        }

        // Find other information about the file
        let buffer = file.buffer;
        let mimetype = file.mimetype;

        // If the image is in HEIC or HEIF format, convert it to JPEG
        if (["heic", "heif"].includes(extension.toLowerCase())) {
            buffer = await convertHeicToJpeg(buffer);
            mimetype = "image/jpeg";
            extension = "jpg";
        }

        // If the file is an image, we must check for NSFW content and upload various sizes
        if (fileConfig?.imageSizes && IMAGE_TYPES.includes(extension.toLowerCase())) {
            // Check for NSFW content using the new service
            const isNsfw = await checkNSFW(buffer, file.originalname);
            if (isNsfw) {
                throw new CustomError("0526", "ActionFailed", { reason: "NSFW content detected", file: file.originalname });
            }
            // Upload image in various sizes, specified in fileConfig
            for (let i = 0; i < fileConfig.imageSizes.length; i++) {
                const { width, height } = fileConfig.imageSizes[i];
                const resizedImage = await resizeImage(buffer, width, height);

                // Construct a unique key using hash, extension, and size
                const resizedKey = `${filenameBase}_${width}x${height}.${extension}`;

                try {
                    const url = await uploadFile(s3, resizedKey, resizedImage, mimetype);
                    urls.push(url);
                } catch (error) {
                    logger.error("Failed to upload file", { trace: "0504", error, fileName: file.originalname, resizedFileName: resizedKey });
                    throw error;
                }
            }
        } else {
            const originalKey = `${filenameBase}.${extension}`;

            try {
                const url = await uploadFile(s3, originalKey, buffer, mimetype);
                urls.push(url);
            } catch (error) {
                logger.error("Failed to upload file", { trace: "0505", error, fileName: file.originalname });
                throw error;
            }
        }

        return { [file.originalname]: urls };
    });

    // Await all promises and combine into a single object
    const fileUrlsArr = await Promise.all(fileUrlsPromises);
    const fileUrls = Object.assign({}, ...fileUrlsArr);

    return fileUrls;
}
