'use strict';

/**
 * Shared file system utilities for completeness scoring
 * Provides common file reading, directory scanning, and validation operations
 * @module scenarios/lib/utils/file-utils
 */

const fs = require('node:fs');
const path = require('node:path');
const { EXCLUDED_DIRS, CODE_EXTENSIONS } = require('../constants');

/**
 * Read and parse a JSON file with unified error handling
 * @param {string} filePath - Absolute path to JSON file
 * @param {object} options - Optional configuration
 * @param {boolean} options.silent - Suppress warnings (default: false)
 * @param {*} options.defaultValue - Return this if file doesn't exist (default: null)
 * @returns {object|null} Parsed JSON object or defaultValue
 */
function readJsonFile(filePath, options = {}) {
  const { silent = false, defaultValue = null } = options;

  if (!fs.existsSync(filePath)) {
    if (!silent) {
      console.warn(`File not found: ${filePath}`);
    }
    return defaultValue;
  }

  try {
    const content = fs.readFileSync(filePath, 'utf8');
    return JSON.parse(content);
  } catch (error) {
    if (!silent) {
      console.warn(`Unable to parse ${filePath}: ${error.message}`);
    }
    return defaultValue;
  }
}

/**
 * Read a text file with error handling
 * @param {string} filePath - Absolute path to file
 * @param {object} options - Optional configuration
 * @param {boolean} options.silent - Suppress warnings (default: false)
 * @param {string} options.defaultValue - Return this if file doesn't exist (default: null)
 * @returns {string|null} File content or defaultValue
 */
function readTextFile(filePath, options = {}) {
  const { silent = false, defaultValue = null } = options;

  if (!fs.existsSync(filePath)) {
    if (!silent) {
      console.warn(`File not found: ${filePath}`);
    }
    return defaultValue;
  }

  try {
    return fs.readFileSync(filePath, 'utf8');
  } catch (error) {
    if (!silent) {
      console.warn(`Unable to read ${filePath}: ${error.message}`);
    }
    return defaultValue;
  }
}

/**
 * Count files recursively matching given extensions
 * @param {string} dir - Directory to scan
 * @param {array} extensions - Array of extensions to match (e.g., ['.tsx', '.ts'])
 * @param {array} excludedDirs - Directories to skip (default: EXCLUDED_DIRS)
 * @returns {number} Count of matching files
 */
function countFilesRecursive(dir, extensions, excludedDirs = EXCLUDED_DIRS) {
  if (!fs.existsSync(dir)) {
    return 0;
  }

  let count = 0;

  try {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);

      if (entry.isDirectory()) {
        if (!excludedDirs.includes(entry.name)) {
          count += countFilesRecursive(fullPath, extensions, excludedDirs);
        }
      } else if (entry.isFile()) {
        const ext = path.extname(entry.name);
        if (extensions.includes(ext)) {
          count++;
        }
      }
    }
  } catch (error) {
    // Silently ignore read errors for individual directories
  }

  return count;
}

/**
 * Get total lines of code in directory
 * @param {string} dir - Directory to scan
 * @param {array} extensions - File extensions to count (default: CODE_EXTENSIONS)
 * @param {array} excludedDirs - Directories to skip (default: EXCLUDED_DIRS)
 * @returns {number} Total lines of code
 */
function getTotalLOC(dir, extensions = CODE_EXTENSIONS, excludedDirs = EXCLUDED_DIRS) {
  if (!fs.existsSync(dir)) {
    return 0;
  }

  let totalLines = 0;

  const countLinesInFile = (filePath) => {
    try {
      const content = fs.readFileSync(filePath, 'utf8');
      return content.split('\n').length;
    } catch (error) {
      return 0;
    }
  };

  const scanDir = (directory) => {
    try {
      const entries = fs.readdirSync(directory, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(directory, entry.name);

        if (entry.isDirectory()) {
          if (!excludedDirs.includes(entry.name)) {
            scanDir(fullPath);
          }
        } else if (entry.isFile()) {
          const ext = path.extname(entry.name);
          if (extensions.includes(ext)) {
            totalLines += countLinesInFile(fullPath);
          }
        }
      }
    } catch (error) {
      // Silently ignore read errors
    }
  };

  scanDir(dir);
  return totalLines;
}

/**
 * Find first matching file from a list of candidates
 * @param {string} baseDir - Base directory to search from
 * @param {array} candidates - Array of relative paths to try
 * @returns {string|null} Absolute path to first existing file, or null
 */
function findFirstFile(baseDir, candidates) {
  for (const candidate of candidates) {
    const fullPath = path.join(baseDir, candidate);
    if (fs.existsSync(fullPath)) {
      return fullPath;
    }
  }
  return null;
}

/**
 * Find all files matching specific names in a directory and its subdirectories
 * @param {string} dir - Directory to search
 * @param {array} fileNames - File names to match
 * @param {array} subdirs - Specific subdirectories to check (if provided)
 * @returns {array} Array of absolute paths to matching files
 */
function findFilesByName(dir, fileNames, subdirs = null) {
  const files = [];

  // Check root directory first
  for (const fileName of fileNames) {
    const filePath = path.join(dir, fileName);
    if (fs.existsSync(filePath)) {
      files.push(filePath);
    }
  }

  // Check specific subdirectories if provided
  if (subdirs) {
    for (const subdir of subdirs) {
      const subdirPath = path.join(dir, subdir);
      if (fs.existsSync(subdirPath)) {
        // Check for target files in subdirectory
        for (const fileName of fileNames) {
          const filePath = path.join(subdirPath, fileName);
          if (fs.existsSync(filePath)) {
            files.push(filePath);
          }
        }
      }
    }
  }

  return files;
}

/**
 * Scan directory recursively and collect file information
 * Optimized for single-pass collection of multiple metrics
 * @param {string} dir - Directory to scan
 * @param {object} options - Scan options
 * @param {array} options.extensions - File extensions to include
 * @param {array} options.excludedDirs - Directories to skip
 * @param {boolean} options.collectContent - Whether to read file contents
 * @param {boolean} options.countLines - Whether to count lines per file
 * @returns {object} Scan results with files array and aggregated metrics
 */
function scanDirectoryRecursive(dir, options = {}) {
  const {
    extensions = CODE_EXTENSIONS,
    excludedDirs = EXCLUDED_DIRS,
    collectContent = false,
    countLines = false
  } = options;

  const results = {
    files: [],
    totalFiles: 0,
    totalLines: 0
  };

  if (!fs.existsSync(dir)) {
    return results;
  }

  const scan = (directory, relativePath = '') => {
    try {
      const entries = fs.readdirSync(directory, { withFileTypes: true });

      for (const entry of entries) {
        const fullPath = path.join(directory, entry.name);
        const relPath = path.join(relativePath, entry.name);

        if (entry.isDirectory()) {
          if (!excludedDirs.includes(entry.name)) {
            scan(fullPath, relPath);
          }
        } else if (entry.isFile()) {
          const ext = path.extname(entry.name);
          if (extensions.includes(ext)) {
            const fileInfo = {
              path: fullPath,
              relativePath: relPath,
              name: entry.name,
              extension: ext
            };

            if (collectContent) {
              try {
                fileInfo.content = fs.readFileSync(fullPath, 'utf8');
                if (countLines && fileInfo.content) {
                  fileInfo.lines = fileInfo.content.split('\n').length;
                  results.totalLines += fileInfo.lines;
                }
              } catch (error) {
                fileInfo.content = null;
                fileInfo.lines = 0;
              }
            } else if (countLines) {
              try {
                const content = fs.readFileSync(fullPath, 'utf8');
                fileInfo.lines = content.split('\n').length;
                results.totalLines += fileInfo.lines;
              } catch (error) {
                fileInfo.lines = 0;
              }
            }

            results.files.push(fileInfo);
            results.totalFiles++;
          }
        }
      }
    } catch (error) {
      // Silently ignore read errors for individual directories
    }
  };

  scan(dir);
  return results;
}

/**
 * Check if a file exists and is readable
 * @param {string} filePath - Path to check
 * @returns {boolean} True if file exists and is readable
 */
function isReadableFile(filePath) {
  try {
    fs.accessSync(filePath, fs.constants.R_OK);
    return fs.statSync(filePath).isFile();
  } catch (error) {
    return false;
  }
}

/**
 * Check if a directory exists and is readable
 * @param {string} dirPath - Path to check
 * @returns {boolean} True if directory exists and is readable
 */
function isReadableDirectory(dirPath) {
  try {
    fs.accessSync(dirPath, fs.constants.R_OK);
    return fs.statSync(dirPath).isDirectory();
  } catch (error) {
    return false;
  }
}

module.exports = {
  readJsonFile,
  readTextFile,
  countFilesRecursive,
  getTotalLOC,
  findFirstFile,
  findFilesByName,
  scanDirectoryRecursive,
  isReadableFile,
  isReadableDirectory
};
