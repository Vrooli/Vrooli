import { PutObjectCommand, S3Client } from "@aws-sdk/client-s3";
import { uuid } from "@local/shared";
// import * as nsfwjs from "nsfwjs";
import sharp from "sharp";

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

/**
 * Asynchronously processes and uploads files to an Amazon S3 bucket.
 */
export const processAndStoreFiles = async (files: Express.Multer.File[]): Promise<{ [x: string]: string }> => {
    const s3 = getS3Client();
    const fileUrls = {};
    for (const file of files) {
        // Generate a unique identifier for the file
        const fileId = uuid();

        // You can also add a file extension if needed, but be careful not to trust the original name of the file,
        // as it could be used for an injection attack. You should determine the file type yourself from the file content.
        const key = `${fileId}`;

        try {
            const command = new PutObjectCommand({
                Bucket: BUCKET_NAME,
                Key: key,
                Body: file.buffer, // assuming the file's data is stored in a Buffer
                ContentType: file.mimetype,
            });
            await s3.send(command);

            // Add the file's URL to the result object
            fileUrls[file.originalname] = `https://${BUCKET_NAME}.s3.${REGION}.amazonaws.com/${key}`;

        } catch (error) {
            console.error(`Failed to upload file ${file.originalname}:`, error);
            throw error;
        }
    }

    return fileUrls;
};
