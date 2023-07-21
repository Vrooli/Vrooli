import { PutObjectCommand, S3Client } from "@aws-sdk/client-s3";
import { uuid } from "@local/shared";
import { fileTypeFromBuffer } from "file-type";
import convert from "heic-convert";
import https from "https";
import imghash from "imghash";
import sharp from "sharp";
import { UploadConfig } from "../endpoints";
import { CustomError, logger } from "../events";

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
        s3 = new S3Client({
            region: REGION,
            credentials: {
                accessKeyId: process.env.AWS_ACCESS_KEY_ID ?? "",
                secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY ?? "",
            },
        });
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
                const result = JSON.parse(responseBody) as NSFWCheckResponse;
                // Return the predictions for the image.
                resolve(result.predictions);
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
        .resize(width, height, { withoutEnlargement: true })
        .toFormat(format)
        .toBuffer();
};

const convertHeicToJpeg = async (buffer: Buffer): Promise<Buffer> => {
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
export const processAndStoreFiles = async (
    files: Express.Multer.File[],
    config?: UploadConfig,
): Promise<{ [x: string]: string[] }> => {
    const s3 = getS3Client();

    // Create promise for each file
    const fileUrlsPromises = files.map(async file => {
        const urls: string[] = [];

        // Find extension using the beginning of the file buffer. 
        // This is safer than using the file extension from the original file name, 
        // as malicious users can spoof the file extension.
        const fileType = await fileTypeFromBuffer(file.buffer);
        let extension = fileType?.ext;
        if (!extension) {
            throw new CustomError("0502", "InternalError", ["en"], { file: file.filename });
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
        if (config?.imageSizes && IMAGE_TYPES.includes(extension.toLowerCase())) {
            // Compute hash for the original image
            //TODO if someone is able to find hash collisions, they can overwrite images (which are stored by hash). Need to investigate this.
            const hash = await imghash.hash(buffer);
            // Check for NSFW content
            // const classifications = await classifyImage(buffer);
            // const isNsfw = classifications.some((c) => ["Porn", "Hentai"].includes(c.className) && c.probability > 0.85);
            const isNsfw = await checkNSFW(buffer, hash);
            if (isNsfw) {
                throw new CustomError("0503", "InternalError", ["en"], { file: file.filename });
            }
            // Upload image in various sizes, specified in config
            for (let i = 0; i < config.imageSizes.length; i++) {
                const { width, height } = config.imageSizes[i];
                const resizedImage = await resizeImage(buffer, width, height);

                // Construct a unique key using hash, extension, and size
                const resizedKey = `${hash}_${width}x${height}.${extension}`;

                try {
                    const url = await uploadFile(s3, resizedKey, resizedImage, mimetype);
                    urls.push(url);
                } catch (error) {
                    logger.error("Failed to upload file", { trace: "0504", error, fileName: file.originalname, resizedFileName: resizedKey });
                    throw error;
                }
            }
        } else {
            const fileId = uuid(); // For non-image files, continue to use UUID to avoid collision
            const originalKey = `${fileId}.${extension}`;

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
