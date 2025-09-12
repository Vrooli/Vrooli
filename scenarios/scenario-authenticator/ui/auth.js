// Authentication UI JavaScript
// API URL configuration - fetched from server config
let API_URL = '';

// Fetch API configuration from server
async function fetchConfig() {
    try {
        const response = await fetch('/config');
        if (!response.ok) {
            throw new Error(`Config endpoint returned ${response.status}`);
        }
        const config = await response.json();
        API_URL = config.apiUrl;
        return config;
    } catch (error) {
        console.error('❌ Failed to fetch API configuration:', error);
        console.error('   This usually means the UI server is not properly configured');
        console.error('   Please ensure the scenario is running through the lifecycle system');
        
        // Show user-friendly error instead of hardcoded fallback
        document.body.innerHTML = `
            <div style="display: flex; justify-content: center; align-items: center; min-height: 100vh; font-family: system-ui;">
                <div style="text-align: center; padding: 2rem; max-width: 500px;">
                    <h2 style="color: #ef4444; margin-bottom: 1rem;">⚠️ Configuration Error</h2>
                    <p style="color: #6b7280; margin-bottom: 1rem;">Unable to connect to authentication service.</p>
                    <p style="font-size: 14px; color: #9ca3af;">
                        Please ensure the service is running through: <br>
                        <code style="background: #f3f4f6; padding: 0.2rem; border-radius: 4px;">vrooli scenario run scenario-authenticator</code>
                    </p>
                </div>
            </div>
        `;
        throw error;
    }
}

// Get URL parameters
const urlParams = new URLSearchParams(window.location.search);
const redirectUrl = urlParams.get('redirect');

// Connection monitoring
let connectionCheckInterval;

// DOM Elements
const loginForm = document.getElementById('loginForm');
const registerForm = document.getElementById('registerForm');
const resetForm = document.getElementById('resetForm');
const tabs = document.querySelectorAll('.tab');
const successMessage = document.getElementById('successMessage');
const connectionStatus = document.getElementById('connectionStatus');
const connectionText = document.getElementById('connectionText');

// Tab switching
tabs.forEach(tab => {
    tab.addEventListener('click', () => {
        const tabName = tab.dataset.tab;
        switchTab(tabName);
    });
});

function switchTab(tabName) {
    // Update active tab
    tabs.forEach(t => {
        t.classList.toggle('active', t.dataset.tab === tabName);
    });

    // Show/hide forms
    loginForm.classList.toggle('hidden', tabName !== 'login');
    registerForm.classList.toggle('hidden', tabName !== 'register');
    resetForm.classList.add('hidden');

    // Clear errors
    clearAllErrors();
}

// Password strength checker
const registerPassword = document.getElementById('registerPassword');
const passwordStrength = document.getElementById('passwordStrength');

registerPassword?.addEventListener('input', (e) => {
    const password = e.target.value;
    const strength = checkPasswordStrength(password);
    
    passwordStrength.classList.remove('weak', 'medium', 'strong');
    if (password.length > 0) {
        passwordStrength.classList.add(strength);
    }
});

function checkPasswordStrength(password) {
    if (password.length < 8) return 'weak';
    
    let strength = 0;
    if (password.match(/[a-z]/)) strength++;
    if (password.match(/[A-Z]/)) strength++;
    if (password.match(/[0-9]/)) strength++;
    if (password.match(/[^a-zA-Z0-9]/)) strength++;
    
    if (strength < 2) return 'weak';
    if (strength < 3) return 'medium';
    return 'strong';
}

// Login form handler
loginForm?.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearAllErrors();
    
    const email = document.getElementById('loginEmail').value;
    const password = document.getElementById('loginPassword').value;
    const rememberMe = document.getElementById('rememberMe').checked;
    
    const button = document.getElementById('loginButton');
    const spinner = button.querySelector('.spinner');
    const text = button.querySelector('span');
    
    // Show loading state
    button.disabled = true;
    spinner.classList.remove('hidden');
    text.textContent = 'Signing in...';
    
    try {
        const response = await fetch(`${API_URL}/api/v1/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ email, password })
        });
        
        const data = await response.json();
        
        if (response.ok && data.success) {
            // Store token - use consistent naming for dashboard compatibility
            const storage = rememberMe ? localStorage : sessionStorage;
            storage.setItem('authToken', data.token);
            storage.setItem('refreshToken', data.refresh_token);
            storage.setItem('authData', JSON.stringify({
                token: data.token,
                refreshToken: data.refresh_token,
                user: data.user,
                sessionId: Date.now().toString(36) + Math.random().toString(36).substr(2)
            }));
            
            // Also keep old keys for backward compatibility
            storage.setItem('auth_token', data.token);
            storage.setItem('refresh_token', data.refresh_token);
            storage.setItem('user', JSON.stringify(data.user));
            
            // Show success message
            showSuccess('Login successful! Redirecting...');
            
            // Redirect after short delay
            setTimeout(() => {
                if (redirectUrl) {
                    window.location.href = redirectUrl;
                } else {
                    window.location.href = '/dashboard';
                }
            }, 1000);
        } else {
            const errorMsg = data.error || 'Invalid email or password';
            showError('loginEmail', errorMsg);
            showSnackBar(errorMsg, 'error');
        }
    } catch (error) {
        const errorMsg = 'Connection failed. Please check your internet connection and try again.';
        showError('loginEmail', errorMsg);
        showSnackBar(errorMsg, 'error');
        console.error('Login error:', error);
    } finally {
        button.disabled = false;
        spinner.classList.add('hidden');
        text.textContent = 'Sign In';
    }
});

// Register form handler
registerForm?.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearAllErrors();
    
    const email = document.getElementById('registerEmail').value;
    const username = document.getElementById('registerUsername').value;
    const password = document.getElementById('registerPassword').value;
    const confirmPassword = document.getElementById('confirmPassword').value;
    const agreeTerms = document.getElementById('agreeTerms').checked;
    
    // Validate inputs
    if (password !== confirmPassword) {
        const errorMsg = 'Passwords do not match';
        showError('confirmPasswordError', errorMsg);
        showSnackBar(errorMsg, 'error');
        return;
    }
    
    if (password.length < 8) {
        const errorMsg = 'Password must be at least 8 characters';
        showError('registerPasswordError', errorMsg);
        showSnackBar(errorMsg, 'error');
        return;
    }
    
    if (!agreeTerms) {
        const errorMsg = 'Please agree to the terms and conditions';
        showError('confirmPasswordError', errorMsg);
        showSnackBar(errorMsg, 'warning');
        return;
    }
    
    const button = document.getElementById('registerButton');
    const spinner = button.querySelector('.spinner');
    const text = button.querySelector('span');
    
    // Show loading state
    button.disabled = true;
    spinner.classList.remove('hidden');
    text.textContent = 'Creating account...';
    
    try {
        const body = { email, password };
        if (username) body.username = username;
        
        const response = await fetch(`${API_URL}/api/v1/auth/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(body)
        });
        
        const data = await response.json();
        
        if (response.ok && data.success) {
            // Store token - use consistent naming for dashboard compatibility
            sessionStorage.setItem('authToken', data.token);
            sessionStorage.setItem('refreshToken', data.refresh_token);
            sessionStorage.setItem('authData', JSON.stringify({
                token: data.token,
                refreshToken: data.refresh_token,
                user: data.user,
                sessionId: Date.now().toString(36) + Math.random().toString(36).substr(2)
            }));
            
            // Also keep old keys for backward compatibility
            sessionStorage.setItem('auth_token', data.token);
            sessionStorage.setItem('refresh_token', data.refresh_token);
            sessionStorage.setItem('user', JSON.stringify(data.user));
            
            // Show success message
            showSuccess('Account created successfully! Redirecting...');
            
            // Clear form
            registerForm.reset();
            
            // Redirect after short delay
            setTimeout(() => {
                if (redirectUrl) {
                    window.location.href = redirectUrl;
                } else {
                    window.location.href = '/dashboard';
                }
            }, 1500);
        } else {
            const errorMsg = data.error || 'Registration failed';
            if (data.error?.includes('email')) {
                showError('registerEmailError', errorMsg);
            } else if (data.error?.includes('username')) {
                showError('registerUsernameError', errorMsg);
            } else {
                showError('registerEmailError', errorMsg);
            }
            showSnackBar(errorMsg, 'error');
        }
    } catch (error) {
        const errorMsg = 'Connection failed. Please check your internet connection and try again.';
        showError('registerEmailError', errorMsg);
        showSnackBar(errorMsg, 'error');
        console.error('Registration error:', error);
    } finally {
        button.disabled = false;
        spinner.classList.add('hidden');
        text.textContent = 'Create Account';
    }
});

// Reset password form handler
resetForm?.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearAllErrors();
    
    const email = document.getElementById('resetEmail').value;
    
    const button = document.getElementById('resetButton');
    const spinner = button.querySelector('.spinner');
    const text = button.querySelector('span');
    
    // Show loading state
    button.disabled = true;
    spinner.classList.remove('hidden');
    text.textContent = 'Sending...';
    
    try {
        const response = await fetch(`${API_URL}/api/v1/auth/reset-password`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ email })
        });
        
        const data = await response.json();
        
        if (response.ok && data.success) {
            showSuccess('Password reset instructions have been sent to your email.');
            resetForm.reset();
            
            // Switch back to login after delay
            setTimeout(() => {
                switchTab('login');
            }, 3000);
        } else {
            const errorMsg = data.error || 'Failed to send reset email';
            showError('resetEmailError', errorMsg);
            showSnackBar(errorMsg, 'error');
        }
    } catch (error) {
        const errorMsg = 'Connection failed. Please check your internet connection and try again.';
        showError('resetEmailError', errorMsg);
        showSnackBar(errorMsg, 'error');
        console.error('Reset password error:', error);
    } finally {
        button.disabled = false;
        spinner.classList.add('hidden');
        text.textContent = 'Send Reset Instructions';
    }
});

// Forgot password link
document.getElementById('forgotPassword')?.addEventListener('click', (e) => {
    e.preventDefault();
    tabs.forEach(t => t.classList.remove('active'));
    loginForm.classList.add('hidden');
    registerForm.classList.add('hidden');
    resetForm.classList.remove('hidden');
});

// Back to login link
document.getElementById('backToLogin')?.addEventListener('click', (e) => {
    e.preventDefault();
    switchTab('login');
});

// Helper functions
function showError(elementId, message) {
    const errorElement = document.getElementById(elementId);
    if (errorElement) {
        errorElement.textContent = message;
        errorElement.classList.add('show');
        
        // Also mark input as error
        const input = errorElement.previousElementSibling;
        if (input && input.tagName === 'INPUT') {
            input.classList.add('error');
        }
    }
}

function clearAllErrors() {
    document.querySelectorAll('.error-message').forEach(el => {
        el.textContent = '';
        el.classList.remove('show');
    });
    
    document.querySelectorAll('input.error').forEach(el => {
        el.classList.remove('error');
    });
}

function showSuccess(message) {
    successMessage.textContent = message;
    successMessage.classList.add('show');
    
    // Also show as snack bar
    showSnackBar(message, 'success');
    
    // Auto-hide after 5 seconds
    setTimeout(() => {
        successMessage.classList.remove('show');
    }, 5000);
}

// Snack Bar Functions
let snackBarTimeout;

function showSnackBar(message, type = 'error') {
    const snackbar = document.getElementById('snackbar');
    const messageSpan = document.getElementById('snackbar-message');
    const closeButton = document.getElementById('snackbar-close');
    
    if (!snackbar || !messageSpan) {
        console.warn('Snackbar elements not found');
        return;
    }
    
    // Clear any existing timeout
    if (snackBarTimeout) {
        clearTimeout(snackBarTimeout);
    }
    
    // Set message and type
    messageSpan.textContent = message;
    snackbar.className = `snackbar ${type}`;
    
    // Show snackbar
    snackbar.classList.add('show');
    
    // Auto-hide after 6 seconds
    snackBarTimeout = setTimeout(() => {
        hideSnackBar();
    }, 6000);
    
    // Close button handler
    closeButton.onclick = hideSnackBar;
}

function hideSnackBar() {
    const snackbar = document.getElementById('snackbar');
    if (snackbar) {
        snackbar.classList.remove('show');
    }
    if (snackBarTimeout) {
        clearTimeout(snackBarTimeout);
    }
}

// Check if already authenticated
async function checkAuth() {
    const token = sessionStorage.getItem('auth_token') || localStorage.getItem('auth_token');
    
    if (token) {
        try {
            const response = await fetch(`${API_URL}/api/v1/auth/validate`, {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });
            
            const data = await response.json();
            
            if (data.valid) {
                // Already authenticated, redirect
                if (redirectUrl) {
                    window.location.href = redirectUrl;
                } else {
                    window.location.href = '/dashboard';
                }
            }
        } catch (error) {
            console.error('Auth check error:', error);
        }
    }
}

// Check auth on page load
document.addEventListener('DOMContentLoaded', async () => {
    try {
        // Fetch config first
        await fetchConfig();
        
        // Start connection monitoring
        startConnectionMonitoring();
        
        // Then check auth
        checkAuth();
        
        // Focus first input
        const firstInput = document.querySelector('input:not([type="checkbox"])');
        if (firstInput) {
            firstInput.focus();
        }
    } catch (error) {
        console.error('Failed to initialize authentication UI:', error);
        // Connection monitoring will show the error state
    }
});

// Connection Status Monitoring
function updateConnectionStatus(status, message) {
    if (!connectionStatus || !connectionText) return;
    
    connectionStatus.className = `connection-status ${status}`;
    connectionText.textContent = message;
}

async function checkConnection() {
    if (!API_URL) {
        updateConnectionStatus('checking', 'Configuring...');
        return false;
    }
    
    try {
        const response = await fetch(`${API_URL}/health`, { 
            method: 'GET',
            signal: AbortSignal.timeout(5000) // 5 second timeout
        });
        
        if (response.ok) {
            const data = await response.json();
            updateConnectionStatus('connected', 'Connected');
            return true;
        } else {
            updateConnectionStatus('disconnected', `API Error (${response.status})`);
            return false;
        }
    } catch (error) {
        if (error.name === 'AbortError') {
            updateConnectionStatus('disconnected', 'Connection timeout');
        } else {
            updateConnectionStatus('disconnected', 'API unavailable');
        }
        return false;
    }
}

function startConnectionMonitoring() {
    // Initial check
    checkConnection();
    
    // Check every 30 seconds
    connectionCheckInterval = setInterval(checkConnection, 30000);
}

function stopConnectionMonitoring() {
    if (connectionCheckInterval) {
        clearInterval(connectionCheckInterval);
        connectionCheckInterval = null;
    }
}

// Connection status click handler for manual refresh
connectionStatus?.addEventListener('click', () => {
    updateConnectionStatus('checking', 'Checking...');
    checkConnection();
});

// Export for use in other scenarios
window.VrooliAuth = {
    getApiUrl: () => API_URL,
    getToken: () => sessionStorage.getItem('auth_token') || localStorage.getItem('auth_token'),
    getUser: () => {
        const userStr = sessionStorage.getItem('user') || localStorage.getItem('user');
        return userStr ? JSON.parse(userStr) : null;
    },
    logout: async () => {
        const token = window.VrooliAuth.getToken();
        if (token) {
            try {
                await fetch(`${API_URL}/api/v1/auth/logout`, {
                    method: 'POST',
                    headers: {
                        'Authorization': `Bearer ${token}`
                    }
                });
            } catch (error) {
                console.error('Logout error:', error);
            }
        }
        
        // Clear storage
        sessionStorage.clear();
        localStorage.removeItem('auth_token');
        localStorage.removeItem('refresh_token');
        localStorage.removeItem('user');
        
        // Redirect to login
        window.location.href = '/login';
    },
    validateToken: async (token) => {
        try {
            const response = await fetch(`${API_URL}/api/v1/auth/validate`, {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });
            
            const data = await response.json();
            return data.valid;
        } catch (error) {
            console.error('Token validation error:', error);
            return false;
        }
    },
    // Connection monitoring controls
    startConnectionMonitoring,
    stopConnectionMonitoring,
    checkConnection
};