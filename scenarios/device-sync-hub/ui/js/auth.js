// Authentication module for Device Sync Hub

class AuthManager {
  constructor(apiConfig) {
    this.apiUrl = apiConfig.apiUrl || '';
    this.authServiceUrl = apiConfig.authUrl;
    this.authApiBase = this.apiUrl ? `${this.apiUrl}/api/v1/auth` : `${this.authServiceUrl}/api/v1/auth`;
    this.authToken = localStorage.getItem('auth_token');
    this.user = null;
  }

  // Check authentication status
  async checkAuth() {
    if (!this.authToken) {
      return false;
    }

    try {
      const response = await fetch(`${this.authApiBase}/validate`, {
        headers: {
          'Authorization': `Bearer ${this.authToken}`
        }
      });

      const result = await response.json();
      
      if (result.valid) {
        this.user = {
          id: result.user_id,
          email: result.email,
          roles: result.roles || []
        };
        return true;
      } else {
        this.clearAuth();
        return false;
      }
    } catch (error) {
      console.error('Auth check failed:', error);
      return false;
    }
  }

  // Login with email and password
  async login(email, password) {
    try {
      const response = await fetch(`${this.authApiBase}/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ email, password })
      });

      const result = await response.json();
      
      if (response.ok) {
        this.authToken = result.token;
        localStorage.setItem('auth_token', this.authToken);
        this.user = {
          id: result.user_id,
          email: result.email,
          roles: result.roles || []
        };
        return { success: true, user: this.user };
      } else {
        return { success: false, error: result.error || 'Login failed' };
      }
    } catch (error) {
      console.error('Login error:', error);
      return { success: false, error: 'Connection failed. Please try again.' };
    }
  }

  // Logout and clear stored authentication
  logout() {
    this.clearAuth();
  }

  // Clear authentication data
  clearAuth() {
    localStorage.removeItem('auth_token');
    this.authToken = null;
    this.user = null;
  }

  // Get current authentication token
  getToken() {
    return this.authToken;
  }

  // Get current user
  getUser() {
    return this.user;
  }

  // Check if user is authenticated
  isAuthenticated() {
    return !!this.authToken && !!this.user;
  }
}

// Export for module use
window.AuthManager = AuthManager;
