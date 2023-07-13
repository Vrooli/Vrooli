import { PutObjectCommand, S3Client } from "@aws-sdk/client-s3";
import { uuid } from "@local/shared";

// Global S3 client variable
let s3: S3Client | undefined;
const REGION = "us-east-1";
const BUCKET_NAME = "vrooli-bucket";

/**
 * Function to setup S3 client. This is needed because 
 * the environment variables are loaded from the secrets location, so they're 
 * not available at startup.
 */
const initS3Client = () => {
    if (!s3) {
        s3 = new S3Client({
            region: REGION,
            credentials: {
                accessKeyId: process.env.AWS_ACCESS_KEY_ID ?? "",
                secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY ?? "",
            },
        });
    }
};

/**
 * Asynchronously processes and uploads files to an Amazon S3 bucket.
 */
export const processAndStoreFiles = async (files: any[]): Promise<{ [x: string]: string }> => {
    initS3Client();
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
            await s3!.send(command);

            // Add the file's URL to the result object
            fileUrls[file.originalname] = `https://your-bucket-name.s3.your-region.amazonaws.com/${key}`;

        } catch (error) {
            console.error(`Failed to upload file ${file.originalname}:`, error);
            throw error;
        }
    }

    return fileUrls;
};
