const fs = require('fs');
const path = require('path');

const rootDir = __dirname;
const distDir = path.join(rootDir, 'dist');
const filesToCopy = ['index.html', 'styles.css', 'app.js'];
const directoriesToCopy = ['vendor'];

function emptyDir(target) {
  fs.rmSync(target, { recursive: true, force: true });
  fs.mkdirSync(target, { recursive: true });
}

function copyFile(relativePath) {
  const source = path.join(rootDir, relativePath);
  if (!fs.existsSync(source)) {
    return;
  }
  const destination = path.join(distDir, relativePath);
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  fs.copyFileSync(source, destination);
}

function copyDirectory(relativePath) {
  const source = path.join(rootDir, relativePath);
  if (!fs.existsSync(source)) {
    return;
  }
  const destination = path.join(distDir, relativePath);
  fs.mkdirSync(destination, { recursive: true });
  for (const entry of fs.readdirSync(source, { withFileTypes: true })) {
    const entrySource = path.join(source, entry.name);
    const entryDestination = path.join(destination, entry.name);
    if (entry.isDirectory()) {
      copyDirectory(path.join(relativePath, entry.name));
    } else if (entry.isFile()) {
      fs.copyFileSync(entrySource, entryDestination);
    }
  }
}

function writeBuildMetadata() {
  const metadataPath = path.join(distDir, 'build.json');
  const metadata = {
    generated_at: new Date().toISOString()
  };
  fs.writeFileSync(metadataPath, JSON.stringify(metadata, null, 2));
}

emptyDir(distDir);
filesToCopy.forEach(copyFile);
directoriesToCopy.forEach(copyDirectory);
writeBuildMetadata();

console.log(`[date-night-planner-ui] Built static bundle in ${distDir}`);
