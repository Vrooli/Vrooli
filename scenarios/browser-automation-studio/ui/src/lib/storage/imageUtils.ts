/**
 * Image utility functions for asset storage
 *
 * Handles validation, thumbnail generation, and dimension extraction.
 */

import {
  ALLOWED_MIME_TYPES,
  MAX_FILE_SIZE,
  MAX_DIMENSION,
  THUMBNAIL_SIZE,
  AssetValidationError,
  type AllowedMimeType,
} from './types';

/**
 * Check if a MIME type is allowed
 */
export function isAllowedMimeType(mimeType: string): mimeType is AllowedMimeType {
  return (ALLOWED_MIME_TYPES as readonly string[]).includes(mimeType);
}

/**
 * Validate a file before upload
 * @throws {AssetValidationError} if validation fails
 */
export async function validateFile(file: File): Promise<void> {
  // Check MIME type
  if (!isAllowedMimeType(file.type)) {
    throw new AssetValidationError(
      `Invalid file type "${file.type}". Allowed types: PNG, JPEG, WebP`,
      'INVALID_TYPE',
    );
  }

  // Check file size
  if (file.size > MAX_FILE_SIZE) {
    const sizeMB = (file.size / (1024 * 1024)).toFixed(1);
    const maxMB = (MAX_FILE_SIZE / (1024 * 1024)).toFixed(0);
    throw new AssetValidationError(
      `File size ${sizeMB}MB exceeds maximum ${maxMB}MB`,
      'FILE_TOO_LARGE',
    );
  }

  // Validate it's actually an image by loading it
  try {
    const dimensions = await getImageDimensions(file);

    // Check dimensions
    if (dimensions.width > MAX_DIMENSION || dimensions.height > MAX_DIMENSION) {
      throw new AssetValidationError(
        `Image dimensions ${dimensions.width}x${dimensions.height} exceed maximum ${MAX_DIMENSION}x${MAX_DIMENSION}`,
        'DIMENSION_TOO_LARGE',
      );
    }
  } catch (e) {
    if (e instanceof AssetValidationError) throw e;
    throw new AssetValidationError('File is not a valid image', 'INVALID_IMAGE');
  }
}

/**
 * Get dimensions of an image file
 */
export function getImageDimensions(file: File): Promise<{ width: number; height: number }> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);

    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve({ width: img.naturalWidth, height: img.naturalHeight });
    };

    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Failed to load image'));
    };

    img.src = url;
  });
}

/**
 * Generate a thumbnail for an image file
 * @param file - Source image file
 * @param maxSize - Maximum dimension for thumbnail (default: THUMBNAIL_SIZE)
 * @returns Base64 data URL of the thumbnail (JPEG)
 */
export async function generateThumbnail(file: File, maxSize: number = THUMBNAIL_SIZE): Promise<string> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);

    img.onload = () => {
      URL.revokeObjectURL(url);

      // Calculate thumbnail dimensions preserving aspect ratio
      let { naturalWidth: width, naturalHeight: height } = img;
      if (width > height) {
        if (width > maxSize) {
          height = Math.round((height * maxSize) / width);
          width = maxSize;
        }
      } else {
        if (height > maxSize) {
          width = Math.round((width * maxSize) / height);
          height = maxSize;
        }
      }

      // Draw to canvas
      const canvas = document.createElement('canvas');
      canvas.width = width;
      canvas.height = height;

      const ctx = canvas.getContext('2d');
      if (!ctx) {
        reject(new Error('Failed to get canvas context'));
        return;
      }

      // Use high-quality image smoothing
      ctx.imageSmoothingEnabled = true;
      ctx.imageSmoothingQuality = 'high';

      ctx.drawImage(img, 0, 0, width, height);

      // Export as JPEG data URL (smaller than PNG for thumbnails)
      const dataUrl = canvas.toDataURL('image/jpeg', 0.85);
      resolve(dataUrl);
    };

    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Failed to load image for thumbnail'));
    };

    img.src = url;
  });
}

/**
 * Resize an image if it exceeds maximum dimensions
 * @param file - Source image file
 * @returns Original file if within limits, or resized Blob
 */
export async function resizeIfNeeded(file: File): Promise<Blob> {
  const dimensions = await getImageDimensions(file);

  // If within limits, return original
  if (dimensions.width <= MAX_DIMENSION && dimensions.height <= MAX_DIMENSION) {
    return file;
  }

  // Calculate new dimensions
  let { width, height } = dimensions;
  if (width > height) {
    height = Math.round((height * MAX_DIMENSION) / width);
    width = MAX_DIMENSION;
  } else {
    width = Math.round((width * MAX_DIMENSION) / height);
    height = MAX_DIMENSION;
  }

  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);

    img.onload = () => {
      URL.revokeObjectURL(url);

      const canvas = document.createElement('canvas');
      canvas.width = width;
      canvas.height = height;

      const ctx = canvas.getContext('2d');
      if (!ctx) {
        reject(new Error('Failed to get canvas context'));
        return;
      }

      ctx.imageSmoothingEnabled = true;
      ctx.imageSmoothingQuality = 'high';
      ctx.drawImage(img, 0, 0, width, height);

      // Export in original format (or PNG for best quality)
      const mimeType = file.type === 'image/jpeg' ? 'image/jpeg' : 'image/png';
      const quality = mimeType === 'image/jpeg' ? 0.92 : undefined;

      canvas.toBlob(
        (blob) => {
          if (blob) {
            resolve(blob);
          } else {
            reject(new Error('Failed to create resized image'));
          }
        },
        mimeType,
        quality,
      );
    };

    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Failed to load image for resize'));
    };

    img.src = url;
  });
}

/**
 * Extract file extension from filename
 */
export function getFileExtension(filename: string): string {
  const parts = filename.split('.');
  return parts.length > 1 ? parts[parts.length - 1].toLowerCase() : 'png';
}

/**
 * Generate a clean filename from user input
 */
export function sanitizeFilename(name: string): string {
  return name
    .replace(/[^a-zA-Z0-9._-]/g, '_')
    .replace(/_+/g, '_')
    .substring(0, 100);
}
