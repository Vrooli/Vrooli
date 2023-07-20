import { PutObjectCommand, S3Client } from "@aws-sdk/client-s3";
import { uuid } from "@local/shared";
// import * as nsfwjs from "nsfwjs";
import sharp from "sharp";
import { UploadConfig } from "../endpoints";
import { logger } from "../events";

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

// NSFW model
let nsfwModel: any;//nsfwjs.NSFWJS | undefined;

/**
 * Loads the NSFW model
 */
const getNSFWModel = async (): Promise<any> => {//Promise<nsfwjs.NSFWJS> => {
    // if (!nsfwModel) {
    //     nsfwModel = await nsfwjs.load();
    // }
    return nsfwModel as any;
};

const resizeImage = async (buffer: Buffer, width: number, height: number, format: "jpeg" | "png" | "webp" = "jpeg") => {
    return await sharp(buffer)
        .resize(width, height, { withoutEnlargement: true })
        .toFormat(format)
        .toBuffer();
};

const convertHeicToJpeg = async (buffer: Buffer): Promise<Buffer> => {
    // return await convert({
    //     buffer,
    //     format: "JPEG",
    //     quality: 1,
    // }) as any;
    return [] as any;
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
    config: UploadConfig,
): Promise<{ [x: string]: string[] }> => {
    const s3 = getS3Client();
    const fileUrls = {};

    for (const file of files) {
        const extension = file.originalname.split(".").pop();
        const urls: string[] = [];

        if (config.imageSizes && IMAGE_TYPES.includes(extension.toLowerCase())) {
            // Compute hash for the original image
            const hash = await imghash.hash(file.buffer);
            for (let i = 0; i < config.imageSizes.length; i++) {
                const { width, height } = config.imageSizes[i];
                const resizedImage = await resizeImage(file.buffer, width, height);

                // Construct a unique key using hash, extension, and size
                const resizedKey = `${hash}_${width}x${height}.${extension}`;

                try {
                    const url = await uploadFile(s3, resizedKey, resizedImage, file.mimetype);
                    urls.push(url);
                } catch (error) {
                    logger.error("Failed to upload file", { trace: "0502", error, fileName: file.originalname });
                    throw error;
                }
            }
        } else {
            const fileId = uuid(); // For non-image files, continue to use UUID to avoid collision
            const originalKey = `${fileId}.${extension}`;

            try {
                const url = await uploadFile(s3, originalKey, file.buffer, file.mimetype);
                urls.push(url);
            } catch (error) {
                logger.error("Failed to upload file", { trace: "0502", error, fileName: file.originalname });
                throw error;
            }
        }

        fileUrls[file.originalname] = urls;
    }

    return fileUrls;
};
