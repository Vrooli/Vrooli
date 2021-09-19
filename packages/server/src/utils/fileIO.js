import path from 'path';
import fs from 'fs';
import { IMAGE_SIZE, IMAGE_EXTENSION } from '@local/shared';
import probe from 'probe-image-size';
import convert from 'heic-convert';
import sharp from 'sharp';
import imghash from 'imghash';
import { TABLES } from '../db';
import pkg from '@prisma/client';
const { PrismaClient } = pkg;

const prisma = new PrismaClient()

// How many times a file name should be checked before giving up
// ex: if 'billy.png' is taken, tries 'billy-1.png', 'billy-2.png', etc.
const MAX_FILE_NAME_ATTEMPTS = 100;
// Max size of a file buffer (how large of a file are you willing to download?)
const MAX_BUFFER_SIZE = 1000000000;
// Location of assets directory
const ASSET_DIR = `${process.env.PROJECT_DIR}/assets`;

// Replace any invalid characters from a file name
// Args:
// file - exs: 'boop.png', 'images/boop.png'
// defaultFolder - default for file's location (ex: 'images')
// Returns:
// name - name of file, excluding extension and location
// ext - extension of file
// folder - path of file
export function clean(file, defaultFolder) {
    const pathRegex = /([^a-z0-9 \.\-\_\/]+)/gi;
    // First, remove any invalid characters
    const cleanPath = file.replace(pathRegex, '');
    const folder = path.dirname(cleanPath)?.replace('.', '') || defaultFolder?.replace(pathRegex, '')?.replace('.', '');
    if (!cleanPath || cleanPath.length === 0) return { name: null, ext: null, folder: null };
    // If a directory was passed in, instead of a file
    if (!cleanPath.includes('.')) return { name: null, ext: null, folder: folder ?? defaultFolder };
    const { name, ext } = path.parse(path.basename(cleanPath));
    return { name, ext, folder: folder ?? defaultFolder };
}

// Returns a filename that can be used at the specified path
// Args:
// filenname - name of file (ex: 'public/boop.png')
// defaultFolder - default for file's location (ex: 'images')
export async function findFileName(file, defaultFolder) {
    const { name, ext, folder } = clean(file, defaultFolder);
    // If file name is available, no need to append a number
    if (!fs.existsSync(`${ASSET_DIR}/${folder}/${name}${ext}`)) return { name, ext, folder };
    // If file name was not available, start appending a number until one works
    let curr = 0;
    while (curr < MAX_FILE_NAME_ATTEMPTS) {
        let currName = `${name}-${curr}${ext}`;
        if (!fs.existsSync(`${ASSET_DIR}/${folder}/${currName}`)) return { name: `${currName}`, ext: ext, folder: folder };
        curr++;
    }
    // If no valid name found after max tries, return null
    return null;
}

// Convert a file stream into a buffer
function streamToBuffer(stream, numBytes) {
    if (numBytes === null || numBytes === undefined) numBytes = MAX_BUFFER_SIZE;
    return new Promise((resolve, reject) => {
        let _buf = []

        stream.on('data', chunk => {
            _buf.push(chunk);
            if (_buf.length >= numBytes) stream.destroy();
        })
        stream.on('end', () => resolve(Buffer.concat(_buf)))
        stream.on('error', err => reject(err))

    })
}

// Returns all image sizes smaller or equal to the image size
function resizeOptions(width, height) {
    let sizes = {};
    for (const [key, value] of Object.entries(IMAGE_SIZE)) {
        if (width >= value || height >= value) sizes[key] = value;
    }
    return sizes;
}

// Saves a file in the specified folder at the server root directory
// Returns an object containing a success boolean and the file name
// Arguments:
// stream - data stream of file
// filename - name of file, including extension and folder (ex: 'public/boop.png')
// overwrite - boolean indicating if existing files can be overwritten
// acceptedTypes - a string or array of accepted file types, in mimetype form (ex: 'application/vnd.ms-excel')
export async function saveFile(stream, filename, mimetype, overwrite, acceptedTypes) {
    try {
        const { name, ext, folder } = await (overwrite ? clean(filename, 'public') : findFileName(filename));
        if (name === null) throw Error('Could not create a valid file name');
        if (acceptedTypes) {
            if (Array.isArray(acceptedTypes) && !acceptedTypes.some(type => mimetype.startsWith(type) || ext === type)) {
                throw Error('File type not accepted');
            }
            if (typeof acceptedTypes === "string" && !(mimetype.startsWith(type) || ext === type)) {
                throw Error('File type not accepted');
            }
        }
        // Download the file
        await stream.pipe(fs.createWriteStream(`${ASSET_DIR}/${folder}/${name}${ext}`));
        return {
            success: true,
            filename: `${folder}/${name}${ext}`
        }
    } catch (error) {
        console.error(error);
        return {
            success: false,
            filename: filename ?? ''
        }
    }
}

// Saves an image file and its resizes in the specified folder at the server root directory
// Returns: {
//      success
//      src
//      hash
//}
// Arguments:
// upload - image upload
// alt - alt text for image
// description - image description
// errorOnDuplicate - If image previously updated, throw error
export async function saveImage({ file, alt, description, labels, errorOnDuplicate = false }) {
    try {
        // Destructure data. Each file upload is a promise
        const { createReadStream, filename, mimetype } = await file;
        // Make sure that the file is actually an image
        if (!mimetype.startsWith('image/')) throw Error('Invalid mimetype')
        // Make sure image type is supported
        let { ext: extCheck } = path.parse(filename)
        if (Object.values(IMAGE_EXTENSION).indexOf(extCheck.toLowerCase()) <= 0) throw Error('Image type not supported')
        // Create a read stream
        const stream = createReadStream();
        const { name, ext, folder } = await findFileName(filename, 'images')
        if (name === null) throw Error('Could not create a valid file name');
        // Determine image dimensions
        let image_buffer = await streamToBuffer(stream);
        const dimensions = probe.sync(image_buffer);
        console.log('GOT DIMENSIONS', dimensions);
        // If image is .heic or .heif, convert to jpg. Thanks, Apple
        if (['.heic', '.heif'].includes(extCheck.toLowerCase())) {
            console.log('converting image buffer')
            image_buffer = await convert({
                buffer: image_buffer, // the HEIC file buffer
                format: 'JPEG',      // output format
                quality: 1           // the jpeg compression quality, between 0 and 1
            });
            extCheck = 'jpg'
        }
        // Determine image hash
        const hash = await imghash.hash(image_buffer);
        console.log('IMAGE HASH', hash)
        // Check if hash already exists (image previously uploaded)
        const previously_uploaded = await prisma[TABLES.Image].findUnique({ where: { hash } });
        console.log('previously uploaded', previously_uploaded);
        if (previously_uploaded && errorOnDuplicate) throw Error('File has already been uploaded');
        // Download the original image, and store metadata in database
        const full_size_filename = `${folder}/${name}-XXL${extCheck}`;
        console.log('name', full_size_filename);
        await sharp(image_buffer).toFile(`${ASSET_DIR}/${full_size_filename}`);
        const imageData = { hash, alt, description };
        await prisma[TABLES.Image].upsert({
            where: { hash },
            create: imageData,
            update: imageData
        })
        await prisma[TABLES.ImageFile].deleteMany({ where: { hash } });
        await prisma[TABLES.ImageFile].create({
            data: {
                hash,
                src: full_size_filename,
                width: dimensions.width,
                height: dimensions.height
            }
        })
        if (Array.isArray(labels)) {
            await prisma[TABLES.ImageLabels].deleteMany({ where: { hash } });
            for (let i = 0; i < labels.length; i++) {
                await prisma[TABLES.ImageLabels].create({
                    data: {
                        hash,
                        label: labels[i],
                        index: i
                    }
                })
            }
        }
        // Find resize options
        const sizes = resizeOptions(dimensions.width, dimensions.height);
        for (const [key, value] of Object.entries(sizes)) {
            // XXL reserved for original image
            if (key === 'XXL') continue;
            // Use largest dimension for resize
            const sizing_dimension = dimensions.width > dimensions.height ? 'width' : 'height';
            const resize_filename = `${folder}/${name}-${key}${extCheck}`;
            const { width, height } = await sharp(image_buffer)
                .resize({ [sizing_dimension]: value })
                .toFile(`${ASSET_DIR}/${resize_filename}`);
            await prisma[TABLES.ImageFile].create({
                data: {
                    hash,
                    src: resize_filename,
                    width: width,
                    height: height
                }
            });
        }
        return {
            success: true,
            src: full_size_filename,
            hash: hash
        }
    } catch (error) {
        console.error(error);
        return {
            success: false,
            src: null,
            hash: null,
        }
    }
}

// Deletes the specified file from the specified folder
// Arguments:
// filename - name of file, including extension (ex: 'boop.png')
// folder - folder in server directory (ex: 'images')
export async function deleteFile(file) {
    try {
        const { name, ext, folder } = clean(file);
        fs.unlinkSync(`${ASSET_DIR}/${folder}/${name}${ext}`);
        return true;
    } catch (error) {
        console.error(error);
        return false;
    }
}

// Deletes an image and all resizes, using its hash
export async function deleteImage(hash) {
    // Find all files associated with image
    const imageData = await prisma[TABLES.Image].findUnique({
        where: { hash },
        select: { files: { select: { src: true } } }
    });
    if (!imageData) return false;
    // Delete database information for image
    await prisma[TABLES.Image].delete({ where: { hash } });
    // Delete image files
    let success = true;
    if (Array.isArray(imageData.files)) {
        for (const file of imageData.files) {
            if (!await deleteFile(file.src)) success = false;
        }
    }
    return success;
}

// Reads a list of files
export async function readFiles(files) {
    let data = [];
    for (const file of files) {
        const { name, ext, folder } = clean(file, 'public');
        const path = `${process.env.PROJECT_DIR}/assets/${folder}/${name}${ext}`;
        if (fs.existsSync(path)) {
            data.push(fs.readFileSync(path, 'utf8'));
        } else {
            data.push(null);
        }
    }
    return data;
}

// Writes a list of files
export async function saveFiles(files, overwrite = true, acceptedTypes) {
    let data = [];
    for (const file of files) {
        const { createReadStream, filename, mimetype } = await file;
        const stream = createReadStream();
        const { success, filename: finalFilename } = await saveFile(stream, filename, mimetype, overwrite, acceptedTypes);
        data.push(success ? finalFilename : null);
    }
    return data;
}