// Authentication UI JavaScript
// API URL configuration - fetched from server config
let API_URL = '';

// Fetch API configuration from server
async function fetchConfig() {
    try {
        const response = await fetch('/config');
        const config = await response.json();
        API_URL = config.apiUrl;
        return config;
    } catch (error) {
        console.error('Failed to fetch config, using default:', error);
        // Fallback to default if config endpoint fails
        API_URL = `${window.location.protocol}//${window.location.hostname}:15000`;
    }
}

// Get URL parameters
const urlParams = new URLSearchParams(window.location.search);
const redirectUrl = urlParams.get('redirect');

// DOM Elements
const loginForm = document.getElementById('loginForm');
const registerForm = document.getElementById('registerForm');
const resetForm = document.getElementById('resetForm');
const tabs = document.querySelectorAll('.tab');
const successMessage = document.getElementById('successMessage');

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
            // Store token
            const storage = rememberMe ? localStorage : sessionStorage;
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
            showError('loginEmail', data.error || 'Invalid email or password');
        }
    } catch (error) {
        showError('loginEmail', 'Connection failed. Please try again.');
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
        showError('confirmPasswordError', 'Passwords do not match');
        return;
    }
    
    if (password.length < 8) {
        showError('registerPasswordError', 'Password must be at least 8 characters');
        return;
    }
    
    if (!agreeTerms) {
        showError('confirmPasswordError', 'Please agree to the terms and conditions');
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
            // Store token
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
            if (data.error?.includes('email')) {
                showError('registerEmailError', data.error);
            } else if (data.error?.includes('username')) {
                showError('registerUsernameError', data.error);
            } else {
                showError('registerEmailError', data.error || 'Registration failed');
            }
        }
    } catch (error) {
        showError('registerEmailError', 'Connection failed. Please try again.');
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
            showError('resetEmailError', data.error || 'Failed to send reset email');
        }
    } catch (error) {
        showError('resetEmailError', 'Connection failed. Please try again.');
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
    
    // Auto-hide after 5 seconds
    setTimeout(() => {
        successMessage.classList.remove('show');
    }, 5000);
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
    // Fetch config first
    await fetchConfig();
    
    // Then check auth
    checkAuth();
    
    // Focus first input
    const firstInput = document.querySelector('input:not([type="checkbox"])');
    if (firstInput) {
        firstInput.focus();
    }
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
    }
};