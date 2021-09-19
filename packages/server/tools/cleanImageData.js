// 1) Delete image files no longer associated with any image rows in the database
// 2) Delete image rows in the database no longer associated with any image files
import { TABLES } from "../src/db";
import fs from 'fs';
import { deleteFile } from "../src/utils";
import pkg from '@prisma/client';
const { PrismaClient } = pkg;
const prisma = new PrismaClient()

const FOLDER = 'images';
const PATH = `${process.env.PROJECT_DIR}/assets/${FOLDER}`
console.info(`🧼 Cleaning image data in: ${PATH}...`);

if (!fs.existsSync(PATH)){
    fs.mkdirSync(PATH);
}

// Find all image files
const files = fs.readdirSync(PATH).map(f => `${FOLDER}/${f}`);
// Find all image files referenced in database
const imageData = await prisma[TABLES.Image].findMany({ 
    select: { 
        hash: true,
        files: { select: { src: true } }
    }
})
if (imageData === undefined || imageData.length === 0) console.warn('No image data found in database')
const dbFiles = imageData.map(d => {
    if (Array.isArray(d.files)) return d.files.map(f => f.src)
    return null;
}).filter(d => d !== null).flat(2);

// Delete images not referenced in database
for (const file of files) {
    if (!dbFiles.some(f => f === file)) {
        console.info(`Deleting ${file}`);
        deleteFile(file);
    }
}

// Delete database rows no longer associated with an image
for (const file of dbFiles) {
    if (!files.some(f => f === file)) {
        console.info(`Deleting image data for ${file}`);
        await prisma[TABLES.Image].deleteMany({ where: { files: { some: { src: file }}}})
    }
}

console.info('✅ Cleaning complete')
process.exit(0)