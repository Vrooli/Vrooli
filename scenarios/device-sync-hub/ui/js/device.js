// Device management module for Device Sync Hub

class DeviceManager {
  constructor() {
    this.deviceId = null;
    this.deviceInfo = null;
    this.initDevice();
  }

  // Initialize device information
  initDevice() {
    this.deviceId = this.getOrCreateDeviceId();
    this.deviceInfo = this.collectDeviceInfo();
  }

  // Get or create a unique device ID
  getOrCreateDeviceId() {
    let deviceId = localStorage.getItem('device_id');
    if (!deviceId) {
      deviceId = 'web-' + Math.random().toString(36).substr(2, 9);
      localStorage.setItem('device_id', deviceId);
    }
    return deviceId;
  }

  // Get the current device ID
  getDeviceId() {
    return this.deviceId;
  }

  // Collect device information for registration
  collectDeviceInfo() {
    const info = {
      platform: this.getPlatform(),
      browser: this.getBrowserInfo(),
      screen: `${screen.width}x${screen.height}`,
      language: navigator.language,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      userAgent: navigator.userAgent,
      capabilities: this.getCapabilities()
    };
    
    return info;
  }

  // Get device information
  getDeviceInfo() {
    return this.deviceInfo;
  }

  // Detect platform
  getPlatform() {
    const userAgent = navigator.userAgent.toLowerCase();
    
    if (/android/.test(userAgent)) return 'android';
    if (/iphone|ipad|ipod/.test(userAgent)) return 'ios';
    if (/windows/.test(userAgent)) return 'windows';
    if (/macintosh|mac os x/.test(userAgent)) return 'macos';
    if (/linux/.test(userAgent)) return 'linux';
    
    return 'web';
  }

  // Get browser information
  getBrowserInfo() {
    const userAgent = navigator.userAgent;
    
    if (userAgent.includes('Chrome') && !userAgent.includes('Edg')) return 'chrome';
    if (userAgent.includes('Firefox')) return 'firefox';
    if (userAgent.includes('Safari') && !userAgent.includes('Chrome')) return 'safari';
    if (userAgent.includes('Edg')) return 'edge';
    if (userAgent.includes('Opera')) return 'opera';
    
    return 'unknown';
  }

  // Get device capabilities
  getCapabilities() {
    const capabilities = ['websocket', 'localStorage'];
    
    // Check for file API support
    if (window.File && window.FileReader && window.FileList && window.Blob) {
      capabilities.push('fileAPI');
    }
    
    // Check for clipboard API support
    if (navigator.clipboard && navigator.clipboard.readText) {
      capabilities.push('clipboardRead');
    }
    
    if (navigator.clipboard && navigator.clipboard.writeText) {
      capabilities.push('clipboardWrite');
    }
    
    // Check for drag and drop support
    if ('draggable' in document.createElement('div')) {
      capabilities.push('dragDrop');
    }
    
    // Check for notification support
    if ('Notification' in window) {
      capabilities.push('notifications');
    }
    
    // Check for service worker support
    if ('serviceWorker' in navigator) {
      capabilities.push('serviceWorker');
    }
    
    // Check for camera/media support
    if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia) {
      capabilities.push('camera');
    }
    
    return capabilities;
  }

  // Check if device supports a specific capability
  hasCapability(capability) {
    return this.deviceInfo.capabilities.includes(capability);
  }

  // Get a user-friendly device name
  getDisplayName() {
    const platform = this.deviceInfo.platform;
    const browser = this.deviceInfo.browser;
    
    let name = `${browser} on ${platform}`;
    
    // Make it more user-friendly
    if (platform === 'android') name = `${browser} on Android`;
    else if (platform === 'ios') name = `${browser} on iOS`;
    else if (platform === 'windows') name = `${browser} on Windows`;
    else if (platform === 'macos') name = `${browser} on macOS`;
    else if (platform === 'linux') name = `${browser} on Linux`;
    else name = `${browser} Browser`;
    
    return name;
  }

  // Get device type for categorization
  getDeviceType() {
    const platform = this.deviceInfo.platform;
    
    if (platform === 'android' || platform === 'ios') {
      return 'mobile';
    } else {
      return 'desktop';
    }
  }

  // Check if this is a mobile device
  isMobile() {
    return this.getDeviceType() === 'mobile';
  }

  // Get screen size category
  getScreenCategory() {
    const width = screen.width;
    
    if (width < 480) return 'small';
    if (width < 768) return 'medium';
    if (width < 1024) return 'large';
    return 'xlarge';
  }

  // Register device with the server
  async registerDevice(apiUrl, authToken) {
    try {
      const response = await fetch(`${apiUrl}/api/v1/devices`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authToken}`
        },
        body: JSON.stringify({
          name: this.getDisplayName(),
          type: this.getDeviceType(),
          platform: this.deviceInfo.platform,
          capabilities: this.deviceInfo.capabilities
        })
      });

      if (response.ok) {
        const result = await response.json();
        console.log('Device registered:', result);
        return result;
      } else {
        console.error('Device registration failed:', response.status);
        return null;
      }
    } catch (error) {
      console.error('Device registration error:', error);
      return null;
    }
  }
}

// Export for module use
window.DeviceManager = DeviceManager;