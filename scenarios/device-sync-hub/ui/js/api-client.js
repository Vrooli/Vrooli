// API client for Device Sync Hub operations

class SyncApiClient {
  constructor(apiUrl, authManager) {
    this.apiUrl = apiUrl;
    this.authManager = authManager;
  }

  // Make authenticated API request
  async request(endpoint, options = {}) {
    const url = `${this.apiUrl}${endpoint}`;
    const token = this.authManager.getToken();
    
    const requestOptions = {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': `Bearer ${token}`
      }
    };

    try {
      const response = await fetch(url, requestOptions);
      
      if (response.status === 401) {
        // Token expired or invalid
        this.authManager.clearAuth();
        throw new Error('Authentication expired');
      }
      
      return response;
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // Upload file to sync hub
  async uploadFile(file, expiryHours = 24) {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('content_type', 'file');
    formData.append('expires_in', expiryHours.toString());

    const response = await this.request('/api/v1/sync/upload', {
      method: 'POST',
      body: formData
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Upload failed' }));
      throw new Error(errorData.error || `HTTP ${response.status}`);
    }

    return await response.json();
  }

  // Upload text content to sync hub
  async uploadText(text, contentType = 'text', expiryHours = 24) {
    const response = await this.request('/api/v1/sync/upload', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        text: text,
        content_type: contentType,
        expires_in: expiryHours
      })
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Upload failed' }));
      throw new Error(errorData.error || `HTTP ${response.status}`);
    }

    return await response.json();
  }

  // Get list of sync items
  async getSyncItems() {
    const response = await this.request('/api/v1/sync/items');
    
    if (!response.ok) {
      throw new Error(`Failed to fetch sync items: HTTP ${response.status}`);
    }

    return await response.json();
  }

  // Get specific sync item
  async getSyncItem(itemId) {
    const response = await this.request(`/api/v1/sync/items/${itemId}`);
    
    if (response.status === 404) {
      throw new Error('Item not found or expired');
    }
    
    if (!response.ok) {
      throw new Error(`Failed to fetch sync item: HTTP ${response.status}`);
    }

    return await response.json();
  }

  // Download sync item
  async downloadSyncItem(itemId) {
    const response = await this.request(`/api/v1/sync/items/${itemId}/download`);
    
    if (response.status === 404) {
      throw new Error('Item not found or expired');
    }
    
    if (!response.ok) {
      throw new Error(`Download failed: HTTP ${response.status}`);
    }

    // Get filename from Content-Disposition header
    const contentDisposition = response.headers.get('Content-Disposition');
    let filename = 'download';
    
    if (contentDisposition) {
      const filenameMatch = contentDisposition.match(/filename="(.+)"/);
      if (filenameMatch) {
        filename = filenameMatch[1];
      }
    }

    const blob = await response.blob();
    
    return {
      blob: blob,
      filename: filename,
      contentType: response.headers.get('Content-Type')
    };
  }

  // Delete sync item
  async deleteSyncItem(itemId) {
    const response = await this.request(`/api/v1/sync/items/${itemId}`, {
      method: 'DELETE'
    });
    
    if (response.status === 404) {
      throw new Error('Item not found');
    }
    
    if (!response.ok) {
      throw new Error(`Delete failed: HTTP ${response.status}`);
    }

    return await response.json();
  }

  // Share clipboard content
  async shareClipboard(content, targetDevices = []) {
    const response = await this.request('/api/v1/sync/clipboard', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        content: content,
        source_device: 'web',
        target_devices: targetDevices
      })
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Clipboard share failed' }));
      throw new Error(errorData.error || `HTTP ${response.status}`);
    }

    return await response.json();
  }

  // Get device list
  async getDevices() {
    const response = await this.request('/api/v1/devices');
    
    if (!response.ok) {
      throw new Error(`Failed to fetch devices: HTTP ${response.status}`);
    }

    return await response.json();
  }

  // Register device
  async registerDevice(deviceInfo) {
    const response = await this.request('/api/v1/devices', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(deviceInfo)
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Device registration failed' }));
      throw new Error(errorData.error || `HTTP ${response.status}`);
    }

    return await response.json();
  }

  // Get server health status
  async getHealthStatus() {
    try {
      const response = await fetch(`${this.apiUrl}/health`);

      if (!response.ok) {
        throw new Error(`Health check failed: HTTP ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Health check failed:', error);
      throw error;
    }
  }

  // Get settings and statistics
  async getSettings() {
    const response = await this.request('/api/v1/sync/settings');

    if (!response.ok) {
      throw new Error(`Failed to fetch settings: HTTP ${response.status}`);
    }

    return await response.json();
  }
}

// Export for module use
window.SyncApiClient = SyncApiClient;