import path from 'path';
import fs from 'fs';
import { genErrorCode, logger, LogLevel } from '../logger';

// How many times a file name should be checked before giving up
// ex: if 'billy.png' is taken, tries 'billy-1.png', 'billy-2.png', etc.
const MAX_FILE_NAME_ATTEMPTS = 100;
// Max size of a file buffer (how large of a file are you willing to download?)
const MAX_BUFFER_SIZE = 1000000000;
// Location of persistent storage directory
const UPLOAD_DIR = `${process.env.PROJECT_DIR}/data/uploads`;

/**
 * Replaces any invalid characters from a file name
 * @param file Name of file (e.g. 'boop.png', 'images/boop.png')
 * @param defaultFolder Default for file's location (e.g. 'images')
 * @returns Object of shape:
 * - name - name of file, excluding extension and location
 * - ext - extension of file
 * - folder - path of file
 */
export function clean(file: string, defaultFolder?: string): {
    name?: string,
    ext?: string,
    folder?: string,
} {
    const pathRegex = /([^a-z0-9 \.\-\_\/]+)/gi;
    // First, remove any invalid characters
    const cleanPath = file.replace(pathRegex, '');
    const folder = path.dirname(cleanPath)?.replace('.', '') || defaultFolder?.replace(pathRegex, '')?.replace('.', '');
    if (!cleanPath || cleanPath.length === 0) return { };
    // If a directory was passed in, instead of a file
    if (!cleanPath.includes('.')) return { folder: folder ?? defaultFolder };
    const { name, ext } = path.parse(path.basename(cleanPath));
    return { name, ext, folder: folder ?? defaultFolder };
}

/**
 * Finds a filename that can be used at the specified path
 * @param file The preferred file name
 * @param defaultFolder Directory the file will be in, if not already part of file name
 * @returns The preferred file name, or the name with the lowest available number appended to it
 * (starting from 0)
 */
export async function findFileName(file: string, defaultFolder?: string): Promise<{
    name?: string,
    ext?: string,
    folder?: string,
}> {
    const { name, ext, folder } = clean(file, defaultFolder);
    // If file name is available, no need to append a number
    if (!fs.existsSync(`${UPLOAD_DIR}/${folder}/${name}${ext}`)) return { name, ext, folder };
    // If file name was not available, start appending a number until one works
    let curr = 0;
    while (curr < MAX_FILE_NAME_ATTEMPTS) {
        let currName = `${name}-${curr}${ext}`;
        if (!fs.existsSync(`${UPLOAD_DIR}/${folder}/${currName}`)) return { name: `${currName}`, ext: ext, folder: folder };
        curr++;
    }
    // If no valid name found after max tries, return null
    return { };
}

/**
 * Saves a file in the specified folder, that folder being located in UPLOAD_DIR
 * @param stream Data stream of file
 * @param filename Name of file, including extension and folder (ex: 'public/boop.png')
 * @param mimetype Mime type of file (e.g. 'image/png', 'application/vnd.ms-excel')
 * @param overwrite Boolean indicating if existing files can be overwritten
 * @param acceptedTypes String or array of accepted file types, in mimetype form (e.g. 'image/png', 'application/vnd.ms-excel')
 * @returns An object containing:
 * - success - True if successful
 * - filename - Name of file that was saved (since naming conflicts might mean that a number was appended)
 */
export async function saveFile(stream: any, filename: string, mimetype: any, overwrite?: boolean, acceptedTypes?: string[]) {
    try {
        const { name, ext, folder } = await (overwrite ? clean(filename, 'public') : findFileName(filename));
        if (name === null) throw Error('Could not create a valid file name');
        if (acceptedTypes) {
            if (Array.isArray(acceptedTypes) && !acceptedTypes.some(type => mimetype.startsWith(type) || ext === type)) {
                throw Error('File type not accepted');
            }
        }
        // Download the file
        await stream.pipe(fs.createWriteStream(`${UPLOAD_DIR}/${folder}/${name}${ext}`));
        return {
            success: true,
            filename: `${folder}/${name}${ext}`
        }
    } catch (error) {
        logger.log(LogLevel.error, 'Failed to save file.', { code: genErrorCode('0008'), error });
        return {
            success: false,
            filename: filename ?? ''
        }
    }
}

/**
 * Deletes the specified file from the specified folder (in UPLOAD_DIR)
 * @param file Name of file, including extension (ex: 'boop.png')
 * @returns True if successful
 */
export async function deleteFile(file: string) {
    try {
        const { name, ext, folder } = clean(file);
        fs.unlinkSync(`${UPLOAD_DIR}/${folder}/${name}${ext}`);
        return true;
    } catch (error) {
        logger.log(LogLevel.error, 'Failed to delete file', { code: genErrorCode('0009'), error });
        return false;
    }
}

/**
 * Reads all lines from each file in an array of filenames
 * @param files Array of filenames
 * @returns Array of data from each file
 */
export async function readFiles(files: string[]) {
    let data = [];
    for (const file of files) {
        const { name, ext, folder } = clean(file, 'public');
        const path = `${UPLOAD_DIR}/${folder}/${name}${ext}`;
        if (fs.existsSync(path)) {
            data.push(fs.readFileSync(path, 'utf8'));
        } else {
            data.push(null);
        }
    }
    return data;
}

/**
 * Saves a list of files
 * @param files Array of filenames, including extension (ex: 'boop.png')
 * @param overwrite Boolean indicating if existing files should be overwritten
 * @param acceptedTypes String or array of accepted file types, in mimetype form (e.g. 'image/png', 'application/vnd.ms-excel')
 * @returns Array of each filename saved, or null if unsuccessful
 */
export async function saveFiles(files: any, overwrite: boolean = true, acceptedTypes: string[] = []) {
    let data = [];
    for (const file of files) {
        const { createReadStream, filename, mimetype } = await file;
        const stream = createReadStream();
        const { success, filename: finalFilename } = await saveFile(stream, filename, mimetype, overwrite, acceptedTypes);
        data.push(success ? finalFilename : null);
    }
    return data;
}

/**
 * Appends data to the end of a file. Useful for writing to a log file
 * @param file Name of file, including extension (ex: 'boop.txt')
 * @returns True if successful
 */
export async function appendToFile(file: string, data: string) {
    try {
        const { name, ext, folder } = clean(file, 'public');
        fs.appendFileSync(`${UPLOAD_DIR}/${folder}/${name}${ext}`, data);
        return true;
    } catch (error) {
        logger.log(LogLevel.error, 'Failed to append to file', { code: genErrorCode('00010'), error });
        return false;
    }
}