/**
 * Utilities for estimating and validating payload sizes before submission
 * to prevent 413 Request Entity Too Large errors
 */

/**
 * Estimate the size of a JSON payload in bytes
 * Accounts for JSON serialization overhead
 */
export function estimateJsonSize(obj: unknown): number {
  try {
    const json = JSON.stringify(obj);
    // UTF-8 encoding: most chars are 1 byte, some are 2-4 bytes
    // Use TextEncoder for accurate byte count
    if (typeof TextEncoder !== 'undefined') {
      return new TextEncoder().encode(json).length;
    }
    // Fallback: estimate assuming mostly ASCII with some overhead
    return json.length * 1.1;
  } catch {
    return 0;
  }
}

/**
 * Estimate base64 string size from its encoded length
 * Base64 adds ~33% overhead over raw binary
 */
export function estimateBase64Size(base64String: string): number {
  if (!base64String) return 0;

  // Remove data URL prefix if present
  const base64Data = base64String.includes(',')
    ? base64String.split(',')[1]
    : base64String;

  // Each base64 character represents 6 bits
  // 4 base64 chars = 3 bytes
  const cleanedLength = base64Data.replace(/=/g, '').length;
  return Math.ceil((cleanedLength * 3) / 4);
}

/**
 * Format bytes as human-readable string
 */
export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['Bytes', 'KiB', 'MiB', 'GiB'];

  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
}

/**
 * Check if payload size is within limits
 */
export function validatePayloadSize(
  estimatedBytes: number,
  maxBytes: number = 10 * 1024 * 1024, // 10 MiB default
  warningThreshold: number = 0.8, // 80%
): {
  ok: boolean;
  warning: boolean;
  estimatedSize: string;
  maxSize: string;
  percentUsed: number;
  message: string | null;
} {
  const percentUsed = estimatedBytes / maxBytes;
  const estimatedSize = formatBytes(estimatedBytes);
  const maxSize = formatBytes(maxBytes);

  if (estimatedBytes > maxBytes) {
    return {
      ok: false,
      warning: false,
      estimatedSize,
      maxSize,
      percentUsed: Math.min(percentUsed, 1),
      message: `Payload too large: ${estimatedSize} exceeds limit of ${maxSize}. Remove some captures or reduce screenshot quality.`,
    };
  }

  if (percentUsed >= warningThreshold) {
    return {
      ok: true,
      warning: true,
      estimatedSize,
      maxSize,
      percentUsed,
      message: `Payload size (${estimatedSize}) is approaching the ${maxSize} limit. Consider removing captures to ensure delivery.`,
    };
  }

  return {
    ok: true,
    warning: false,
    estimatedSize,
    maxSize,
    percentUsed,
    message: null,
  };
}

/**
 * Compress a base64 image by reducing quality
 * Returns a promise that resolves to the compressed base64 string
 */
export async function compressBase64Image(
  base64String: string,
  maxWidth: number = 1920,
  maxHeight: number = 1080,
  quality: number = 0.85,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const img = new Image();

    img.onload = () => {
      // Calculate new dimensions maintaining aspect ratio
      let { width, height } = img;

      if (width > maxWidth) {
        height = Math.round((height * maxWidth) / width);
        width = maxWidth;
      }

      if (height > maxHeight) {
        width = Math.round((width * maxHeight) / height);
        height = maxHeight;
      }

      // Create canvas and draw resized image
      const canvas = document.createElement('canvas');
      canvas.width = width;
      canvas.height = height;

      const ctx = canvas.getContext('2d');
      if (!ctx) {
        reject(new Error('Failed to get canvas context'));
        return;
      }

      ctx.drawImage(img, 0, 0, width, height);

      // Convert to compressed base64
      try {
        const compressed = canvas.toDataURL('image/jpeg', quality);
        resolve(compressed);
      } catch (error) {
        reject(error);
      }
    };

    img.onerror = () => {
      reject(new Error('Failed to load image for compression'));
    };

    img.src = base64String;
  });
}

/**
 * Estimate the total size of an issue report payload
 */
export interface PayloadSizeBreakdown {
  message: number;
  captures: number;
  logs: number;
  consoleLogs: number;
  networkRequests: number;
  healthChecks: number;
  metadata: number;
  total: number;
}

export function estimateReportPayloadSize(payload: unknown): PayloadSizeBreakdown {
  const breakdown: PayloadSizeBreakdown = {
    message: 0,
    captures: 0,
    logs: 0,
    consoleLogs: 0,
    networkRequests: 0,
    healthChecks: 0,
    metadata: 0,
    total: 0,
  };

  // Type guard for payload structure
  if (typeof payload !== 'object' || payload === null) {
    return breakdown;
  }

  const payloadObj = payload as Record<string, unknown>;

  // Message text
  if (typeof payloadObj.message === 'string') {
    breakdown.message = estimateJsonSize(payloadObj.message);
  }

  // Captures (primarily base64 images)
  if (Array.isArray(payloadObj.captures)) {
    breakdown.captures = payloadObj.captures.reduce((sum: number, capture: unknown) => {
      if (typeof capture === 'object' && capture !== null) {
        const captureObj = capture as Record<string, unknown>;
        const imageSize = typeof captureObj.data === 'string' ? estimateBase64Size(captureObj.data) : 0;
        const metadataSize = estimateJsonSize({ ...captureObj, data: '' });
        return sum + imageSize + metadataSize;
      }
      return sum;
    }, 0);
  }

  // Logs
  if (Array.isArray(payloadObj.logs)) {
    breakdown.logs = estimateJsonSize(payloadObj.logs);
  }

  // Console logs
  if (Array.isArray(payloadObj.consoleLogs)) {
    breakdown.consoleLogs = estimateJsonSize(payloadObj.consoleLogs);
  }

  // Network requests
  if (Array.isArray(payloadObj.networkRequests)) {
    breakdown.networkRequests = estimateJsonSize(payloadObj.networkRequests);
  }

  // Health checks
  if (Array.isArray(payloadObj.healthChecks)) {
    breakdown.healthChecks = estimateJsonSize(payloadObj.healthChecks);
  }

  // Remaining metadata
  const payloadWithoutData = { ...payloadObj };
  delete payloadWithoutData.captures;
  delete payloadWithoutData.logs;
  delete payloadWithoutData.consoleLogs;
  delete payloadWithoutData.networkRequests;
  delete payloadWithoutData.healthChecks;
  delete payloadWithoutData.message;

  breakdown.metadata = estimateJsonSize(payloadWithoutData);

  // Total
  breakdown.total =
    breakdown.message +
    breakdown.captures +
    breakdown.logs +
    breakdown.consoleLogs +
    breakdown.networkRequests +
    breakdown.healthChecks +
    breakdown.metadata;

  return breakdown;
}
