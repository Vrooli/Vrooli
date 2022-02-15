import path from 'path';
import fs from 'fs';

// How many times a file name should be checked before giving up
// ex: if 'billy.png' is taken, tries 'billy-1.png', 'billy-2.png', etc.
const MAX_FILE_NAME_ATTEMPTS = 100;
// Max size of a file buffer (how large of a file are you willing to download?)
const MAX_BUFFER_SIZE = 1000000000;
// Location of persistent storage directory
const UPLOAD_DIR = `${process.env.PROJECT_DIR}/data/uploads`;

// Replace any invalid characters from a file name
// Args:
// file - exs: 'boop.png', 'images/boop.png'
// defaultFolder - default for file's location (ex: 'images')
// Returns:
// name - name of file, excluding extension and location
// ext - extension of file
// folder - path of file
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

// Returns a filename that can be used at the specified path
// Args:
// filenname - name of file (ex: 'public/boop.png')
// defaultFolder - default for file's location (ex: 'images')
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

// Saves a file in the specified folder at the server root directory
// Returns an object containing a success boolean and the file name
// Arguments:
// stream - data stream of file
// filename - name of file, including extension and folder (ex: 'public/boop.png')
// overwrite - boolean indicating if existing files can be overwritten
// acceptedTypes - a string or array of accepted file types, in mimetype form (ex: 'application/vnd.ms-excel')
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
        console.error(error);
        return {
            success: false,
            filename: filename ?? ''
        }
    }
}

// Deletes the specified file from the specified folder
// Arguments:
// filename - name of file, including extension (ex: 'boop.png')
// folder - folder in server directory (ex: 'images')
export async function deleteFile(file: string) {
    try {
        const { name, ext, folder } = clean(file);
        fs.unlinkSync(`${UPLOAD_DIR}/${folder}/${name}${ext}`);
        return true;
    } catch (error) {
        console.error(error);
        return false;
    }
}

// Reads a list of files
export async function readFiles(files: string[]) {
    let data = [];
    for (const file of files) {
        const { name, ext, folder } = clean(file, 'public');
        const path = `${process.env.PROJECT_DIR}/data/uploads/${folder}/${name}${ext}`;
        if (fs.existsSync(path)) {
            data.push(fs.readFileSync(path, 'utf8'));
        } else {
            data.push(null);
        }
    }
    return data;
}

// Writes a list of files
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