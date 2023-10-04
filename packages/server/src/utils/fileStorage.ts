import { PutObjectCommand, S3Client } from "@aws-sdk/client-s3";
import { uuid } from "@local/shared";
import { fileTypeFromBuffer } from "file-type";
import https from "https";
import sharp from "sharp";
import { UploadConfig } from "../endpoints";
import { CustomError, logger } from "../events";
import { SessionUserToken } from "../types";

// Global S3 client variable
let s3: S3Client | undefined;
const REGION = "us-east-1";
const BUCKET_NAME = "vrooli-bucket";

/**
 * Function to setup S3 client. This is needed because 
 * the environment variables are loaded from the secrets location, so they're 
 * not available at startup.
 */
const getS3Client = (): S3Client => {
    if (!s3) {
        s3 = new S3Client({ region: REGION });
    }
    return s3 as S3Client;
};
interface NSFWCheckResult {
    [key: string]: {
        drawings: number,
        hentai: number,
        neutral: number,
        porn: number,
        sexy: number,
    };
}

// heic-convert has to defer initialization because (presumably) the wasm file messes up the compiler error logs
let heicConvert;
const getHeicConvert = async () => {
    if (!heicConvert) {
        heicConvert = (await import("heic-convert")).default;
    }
    return heicConvert;
};
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

interface NSFWCheckResponse {
    predictions: NSFWCheckResult;
}

/**
 * Post the image to the NSFW detection API and get the results.
 * @param buffer The image data.
 * @param hash The hash of the image data.
 * @returns The NSFW check result.
 */
const checkNSFW = async (buffer: Buffer, hash: string): Promise<NSFWCheckResult> => {
    // Convert the image data to base64
    const base64Image = buffer.toString("base64");

    // Prepare the POST data
    const data = JSON.stringify({ images: [{ buffer: base64Image, hash }] });

    return new Promise((resolve, reject) => {
        const options = {
            hostname: "nsfwdetect.com",
            port: 443,
            path: "/",
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Content-Length": data.length,
            },
        };

        const apiRequest = https.request(options, apiRes => {
            let responseBody = "";
            apiRes.on("data", chunk => {
                responseBody += chunk;
            });
            apiRes.on("end", () => {
                try {
                    const result = JSON.parse(responseBody) as NSFWCheckResponse;
                    resolve(result.predictions);
                } catch (error) {
                    const message = "Failed to parse nsfwdetect response";
                    logger.error(message, { trace: "0507", error });
                    reject(new Error(message));
                }
            });
        });

        apiRequest.on("error", error => {
            console.error(`Error: ${error}`);
            reject(error);
        });

        apiRequest.write(data);
        apiRequest.end();
    });
};

const resizeImage = async (buffer: Buffer, width: number, height: number, format: "jpeg" | "png" | "webp" = "jpeg") => {
    return await sharp(buffer)
        // Don't enlarge the image if it's already smaller than the requested size.
        .resize(width, height, { withoutEnlargement: true })
        // Replace transparent pixels with white.
        .flatten({ background: { r: 255, g: 255, b: 255 } })
        // Convert to the requested format.
        .toFormat(format)
        // Convert to a buffer.
        .toBuffer();
};

const convertHeicToJpeg = async (buffer: Buffer): Promise<Buffer> => {
    const convert = await getHeicConvert();
    const outputBuffer = await convert({
        buffer,  // the HEIC file buffer
        format: "JPEG", // output format
        quality: 1, // the jpeg compression quality, between 0 and 1
    });
    return Buffer.from(outputBuffer);
};

/** Supported image types/extensions */
const IMAGE_TYPES = ["jpeg", "jpg", "png", "gif", "webp", "tiff", "bmp"];

/** Uploads a file to S3 */
const uploadFile = async (
    s3: S3Client,
    key: string,
    body: Buffer,
    mimetype: string,
) => {
    // Construct the PutObjectCommand with the necessary parameters.
    const command = new PutObjectCommand({
        Bucket: BUCKET_NAME,
        Key: key,
        Body: body,
        ContentType: mimetype,
    });
    // Send the command to the S3 client.
    await s3.send(command);
    // Return the URL of the uploaded file.
    return `https://${BUCKET_NAME}.s3.${REGION}.amazonaws.com/${key}`;
};

/**
 * Asynchronously processes and uploads files to an Amazon S3 bucket.
 */
export const processAndStoreFiles = async <TInput extends object | undefined>(
    files: Express.Multer.File[],
    input: TInput,
    userData: SessionUserToken,
    config?: UploadConfig<TInput>,
): Promise<{ [x: string]: string[] }> => {
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
            throw new CustomError("0524", "MaxFilesExceeded", ["en"], { file: file.filename });
        }

        // Check if the file size exceeds the maximum allowed size
        if (fileConfig?.maxFileSize !== undefined && file.size > fileConfig.maxFileSize) {
            throw new CustomError("0525", "MaxFileSizeExceeded", ["en"], { file: file.filename });
        }

        // Get the file name
        const filenameBase = fileConfig?.fileNameBase ? fileConfig.fileNameBase(input, userData) : uuid();

        // Find extension using the beginning of the file buffer. 
        // This is safer than using the file extension from the original file name, 
        // as malicious users can spoof the file extension.
        const fileType = await fileTypeFromBuffer(file.buffer);
        let extension = fileType?.ext;
        if (!extension) {
            throw new CustomError("0502", "InternalError", ["en"], { file: file.filename });
        }

        // Check if the file extension is allowed
        const allowedExtensions = fileConfig?.allowedExtensions ?? ["txt", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "heic", "heif", "png", "jpg", "jpeg", "gif", "webp", "tiff", "bmp"];
        if (!allowedExtensions.includes(extension)) {
            throw new CustomError("0508", "InternalError", ["en"], { file: file.filename });
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
            // Check for NSFW content TODO fix and add back
            // const classificationsMap = await checkNSFW(buffer, hash);
            // const isNsfw = classificationsMap[hash] ? classificationsMap[hash]["porn"] > 0.85 || classificationsMap[hash]["hentai"] > 0.85 : false;
            // if (isNsfw) {
            //     throw new CustomError("0503", "InternalError", ["en"], { file: file.filename });
            // }
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
};
