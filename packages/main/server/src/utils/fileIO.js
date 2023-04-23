import path from "path";
import fs from "fs";
import { logger } from "../events/logger";
const MAX_FILE_NAME_ATTEMPTS = 100;
const MAX_BUFFER_SIZE = 1000000000;
const UPLOAD_DIR = `${process.env.PROJECT_DIR}/data/uploads`;
export function clean(file, defaultFolder) {
    const pathRegex = /([^a-z0-9 \.\-\_\/]+)/gi;
    const cleanPath = file.replace(pathRegex, "");
    const folder = path.dirname(cleanPath)?.replace(".", "") || defaultFolder?.replace(pathRegex, "")?.replace(".", "");
    if (!cleanPath || cleanPath.length === 0)
        return {};
    if (!cleanPath.includes("."))
        return { folder: folder ?? defaultFolder };
    const { name, ext } = path.parse(path.basename(cleanPath));
    return { name, ext, folder: folder ?? defaultFolder };
}
export async function findFileName(file, defaultFolder) {
    const { name, ext, folder } = clean(file, defaultFolder);
    if (!fs.existsSync(`${UPLOAD_DIR}/${folder}/${name}${ext}`))
        return { name, ext, folder };
    let curr = 0;
    while (curr < MAX_FILE_NAME_ATTEMPTS) {
        const currName = `${name}-${curr}${ext}`;
        if (!fs.existsSync(`${UPLOAD_DIR}/${folder}/${currName}`))
            return { name: `${currName}`, ext, folder };
        curr++;
    }
    return {};
}
export async function saveFile(stream, filename, mimetype, overwrite, acceptedTypes) {
    try {
        const { name, ext, folder } = await (overwrite ? clean(filename, "public") : findFileName(filename));
        if (name === null)
            throw Error("Could not create a valid file name");
        if (acceptedTypes) {
            if (Array.isArray(acceptedTypes) && !acceptedTypes.some(type => mimetype.startsWith(type) || ext === type)) {
                throw Error("File type not accepted");
            }
        }
        await stream.pipe(fs.createWriteStream(`${UPLOAD_DIR}/${folder}/${name}${ext}`));
        return {
            success: true,
            filename: `${folder}/${name}${ext}`,
        };
    }
    catch (error) {
        logger.error("Failed to save file.", { trace: "0008", error });
        return {
            success: false,
            filename: filename ?? "",
        };
    }
}
export async function deleteFile(file) {
    try {
        const { name, ext, folder } = clean(file);
        fs.unlinkSync(`${UPLOAD_DIR}/${folder}/${name}${ext}`);
        return true;
    }
    catch (error) {
        logger.error("Failed to delete file", { trace: "0009", error });
        return false;
    }
}
export async function readFiles(files) {
    const data = [];
    for (const file of files) {
        const { name, ext, folder } = clean(file, "public");
        const path = `${UPLOAD_DIR}/${folder}/${name}${ext}`;
        if (fs.existsSync(path)) {
            data.push(fs.readFileSync(path, "utf8"));
        }
        else {
            data.push(null);
        }
    }
    return data;
}
export async function saveFiles(files, overwrite = true, acceptedTypes = []) {
    const data = [];
    for (const file of files) {
        const { createReadStream, filename, mimetype } = await file;
        const stream = createReadStream();
        const { success, filename: finalFilename } = await saveFile(stream, filename, mimetype, overwrite, acceptedTypes);
        data.push(success ? finalFilename : null);
    }
    return data;
}
export async function appendToFile(file, data) {
    try {
        const { name, ext, folder } = clean(file, "public");
        fs.appendFileSync(`${UPLOAD_DIR}/${folder}/${name}${ext}`, data);
        return true;
    }
    catch (error) {
        logger.error("Failed to append to file", { trace: "0010", error });
        return false;
    }
}
//# sourceMappingURL=fileIO.js.map