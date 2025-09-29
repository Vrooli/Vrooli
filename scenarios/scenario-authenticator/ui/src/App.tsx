import { initIframeBridgeChild } from '@vrooli/iframe-bridge';
import {
  AlertTriangle,
  CheckCircle2,
  Eye,
  EyeOff,
  Info,
  Loader2,
  ShieldCheck,
  X as XIcon,
} from 'lucide-react';
import {
  FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

type Tab = 'login' | 'register' | 'reset';
type SnackbarType = 'success' | 'warning' | 'error' | 'info';
type ConnectionState = 'checking' | 'connected' | 'disconnected';

interface SnackbarState {
  message: string;
  type: SnackbarType;
  visible: boolean;
}

interface ConnectionStatusState {
  state: ConnectionState;
  message: string;
}

type PasswordField = 'loginPassword' | 'registerPassword' | 'confirmPassword';

declare global {
  interface Window {
    VrooliAuth?: {
      getApiUrl: () => string;
      getToken: () => string | null;
      getUser: () => unknown;
      logout: () => Promise<void>;
      validateToken: (token: string) => Promise<boolean>;
      startConnectionMonitoring: () => void;
      stopConnectionMonitoring: () => void;
      checkConnection: () => Promise<boolean>;
    };
    __scenarioAuthenticatorBridgeInitialized?: boolean;
  }
}

const PASSWORD_MIN_LENGTH = 8;

function evaluatePasswordStrength(password: string): '' | 'weak' | 'medium' | 'strong' {
  if (!password) return '';
  if (password.length < PASSWORD_MIN_LENGTH) return 'weak';

  let strength = 0;
  if (/[a-z]/.test(password)) strength += 1;
  if (/[A-Z]/.test(password)) strength += 1;
  if (/[0-9]/.test(password)) strength += 1;
  if (/[^a-zA-Z0-9]/.test(password)) strength += 1;

  if (strength < 2) return 'weak';
  if (strength < 3) return 'medium';
  return 'strong';
}

function persistAuthTokens(
  token: string,
  refreshToken: string,
  user: unknown,
  remember: boolean
) {
  const storage = remember ? window.localStorage : window.sessionStorage;
  const authData = {
    token,
    refreshToken,
    user,
    sessionId: `${Date.now().toString(36)}${Math.random().toString(36).slice(2)}`,
  };

  storage.setItem('authToken', token);
  storage.setItem('refreshToken', refreshToken);
  storage.setItem('authData', JSON.stringify(authData));

  // Backward compatibility keys
  storage.setItem('auth_token', token);
  storage.setItem('refresh_token', refreshToken);
  storage.setItem('user', JSON.stringify(user));
}

function clearStoredAuth() {
  window.sessionStorage.clear();
  window.localStorage.removeItem('auth_token');
  window.localStorage.removeItem('refresh_token');
  window.localStorage.removeItem('user');
  window.localStorage.removeItem('authToken');
  window.localStorage.removeItem('refreshToken');
  window.localStorage.removeItem('authData');
}

function getStoredToken() {
  return (
    window.sessionStorage.getItem('auth_token') ||
    window.localStorage.getItem('auth_token') ||
    window.sessionStorage.getItem('authToken') ||
    window.localStorage.getItem('authToken')
  );
}

function getStoredUser(): unknown {
  const raw =
    window.sessionStorage.getItem('user') ||
    window.localStorage.getItem('user');
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch (error) {
    console.warn('Unable to parse stored user payload', error);
    return null;
  }
}

const redirectAfterAuth = (redirectUrl: string | null) => {
  window.setTimeout(() => {
    window.location.href = redirectUrl ?? '/dashboard';
  }, 1000);
};

export default function App() {
  const redirectUrl = useMemo(() => {
    const params = new URLSearchParams(window.location.search);
    return params.get('redirect');
  }, []);

  const [activeTab, setActiveTab] = useState<Tab>('login');
  const [apiUrl, setApiUrl] = useState('');
  const [configError, setConfigError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatusState>(
    {
      state: 'checking',
      message: 'Checking connection...',
    }
  );
  const [snackbar, setSnackbar] = useState<SnackbarState>({
    message: '',
    type: 'info',
    visible: false,
  });
  const [passwordVisibility, setPasswordVisibility] = useState<
    Record<PasswordField, boolean>
  >({
    loginPassword: false,
    registerPassword: false,
    confirmPassword: false,
  });

  const [loginState, setLoginState] = useState({
    email: '',
    password: '',
    remember: false,
    loading: false,
  });

  const [registerState, setRegisterState] = useState({
    email: '',
    username: '',
    password: '',
    confirmPassword: '',
    agree: false,
    loading: false,
  });

  const [resetState, setResetState] = useState({
    email: '',
    loading: false,
  });

  const successTimeoutRef = useRef<number>();
  const snackbarTimeoutRef = useRef<number>();
  const connectionIntervalRef = useRef<number>();

  const clearSuccessMessage = useCallback(() => {
    if (successTimeoutRef.current) {
      window.clearTimeout(successTimeoutRef.current);
    }
    setSuccessMessage('');
  }, []);

  const hideSnackbar = useCallback(() => {
    if (snackbarTimeoutRef.current) {
      window.clearTimeout(snackbarTimeoutRef.current);
    }
    setSnackbar((prev) => ({ ...prev, visible: false }));
  }, []);

  const showSnackbar = useCallback(
    (message: string, type: SnackbarType = 'error', durationMs = 6000) => {
      if (snackbarTimeoutRef.current) {
        window.clearTimeout(snackbarTimeoutRef.current);
      }
      setSnackbar({ message, type, visible: true });
      snackbarTimeoutRef.current = window.setTimeout(() => {
        setSnackbar((prev) => ({ ...prev, visible: false }));
      }, durationMs);
    },
    []
  );

  const showSuccess = useCallback(
    (message: string) => {
      clearSuccessMessage();
      setSuccessMessage(message);
      showSnackbar(message, 'success', 5000);
      successTimeoutRef.current = window.setTimeout(() => {
        setSuccessMessage('');
      }, 5000);
    },
    [clearSuccessMessage, showSnackbar]
  );

  const clearErrors = useCallback(() => {
    setFieldErrors({});
  }, []);

  const setError = useCallback((key: string, message: string) => {
    setFieldErrors((prev) => ({ ...prev, [key]: message }));
  }, []);

  const togglePasswordVisibility = useCallback((field: PasswordField) => {
    setPasswordVisibility((prev) => ({ ...prev, [field]: !prev[field] }));
  }, []);

  const checkConnection = useCallback(async (): Promise<boolean> => {
    if (!apiUrl) {
      setConnectionStatus({
        state: 'checking',
        message: 'Configuring...',
      });
      return false;
    }

    setConnectionStatus({ state: 'checking', message: 'Checking connection...' });

    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), 5000);

    try {
      const response = await fetch(`${apiUrl}/health`, {
        method: 'GET',
        signal: controller.signal,
      });

      if (response.ok) {
        setConnectionStatus({ state: 'connected', message: 'Connected' });
        return true;
      }

      setConnectionStatus({
        state: 'disconnected',
        message: `API error (${response.status})`,
      });
      return false;
    } catch (error) {
      const message =
        error instanceof DOMException && error.name === 'AbortError'
          ? 'Connection timeout'
          : 'API unavailable';
      setConnectionStatus({ state: 'disconnected', message });
      return false;
    } finally {
      window.clearTimeout(timeout);
    }
  }, [apiUrl]);

  const stopConnectionMonitoring = useCallback(() => {
    if (connectionIntervalRef.current) {
      window.clearInterval(connectionIntervalRef.current);
      connectionIntervalRef.current = undefined;
    }
  }, []);

  const startConnectionMonitoring = useCallback(() => {
    if (!apiUrl) return;

    stopConnectionMonitoring();
    void checkConnection();
    connectionIntervalRef.current = window.setInterval(() => {
      void checkConnection();
    }, 30000);
  }, [apiUrl, checkConnection, stopConnectionMonitoring]);

  useEffect(() => {
    if (apiUrl) {
      startConnectionMonitoring();
    }

    return () => {
      stopConnectionMonitoring();
    };
  }, [apiUrl, startConnectionMonitoring, stopConnectionMonitoring]);

  useEffect(() => {
    return () => {
      clearSuccessMessage();
      hideSnackbar();
      stopConnectionMonitoring();
    };
  }, [clearSuccessMessage, hideSnackbar, stopConnectionMonitoring]);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (
      window.parent !== window &&
      !window.__scenarioAuthenticatorBridgeInitialized
    ) {
      let parentOrigin: string | undefined;
      try {
        if (document.referrer) {
          parentOrigin = new URL(document.referrer).origin;
        }
      } catch (error) {
        console.warn(
          '[ScenarioAuthenticator] Unable to parse parent origin for iframe bridge',
          error
        );
      }

      initIframeBridgeChild({ parentOrigin, appId: 'scenario-authenticator' });
      window.__scenarioAuthenticatorBridgeInitialized = true;
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    const loadConfig = async () => {
      try {
        const response = await fetch('/config');
        if (!response.ok) {
          throw new Error(`Config endpoint returned ${response.status}`);
        }

        const config = await response.json();
        if (!cancelled) {
          setApiUrl(config.apiUrl);
          setConfigError(null);
        }
      } catch (error) {
        console.error('Failed to fetch UI configuration', error);

        const fallback = import.meta.env.VITE_API_URL as string | undefined;
        if (fallback && !cancelled) {
          setApiUrl(fallback);
          setConfigError(null);
          return;
        }

        if (!cancelled) {
          setConfigError(
            'Unable to connect to authentication service. Start the scenario through the lifecycle system so the UI can discover the API port.'
          );
          setConnectionStatus({
            state: 'disconnected',
            message: 'Configuration error',
          });
        }
      }
    };

    void loadConfig();

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const token = getStoredToken();
    if (!token || !apiUrl) {
      return;
    }

    const validate = async () => {
      try {
        const response = await fetch(`${apiUrl}/api/v1/auth/validate`, {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });
        const data = await response.json();
        if (data.valid) {
          redirectAfterAuth(redirectUrl);
        }
      } catch (error) {
        console.error('Token validation error', error);
      }
    };

    void validate();
  }, [apiUrl, redirectUrl]);

  useEffect(() => {
    window.VrooliAuth = {
      getApiUrl: () => apiUrl,
      getToken: getStoredToken,
      getUser: getStoredUser,
      logout: async () => {
        const token = getStoredToken();
        if (token && apiUrl) {
          try {
            await fetch(`${apiUrl}/api/v1/auth/logout`, {
              method: 'POST',
              headers: { Authorization: `Bearer ${token}` },
            });
          } catch (error) {
            console.error('Logout error', error);
          }
        }
        clearStoredAuth();
        window.location.href = '/login';
      },
      validateToken: async (token: string) => {
        if (!apiUrl) return false;
        try {
          const response = await fetch(`${apiUrl}/api/v1/auth/validate`, {
            headers: { Authorization: `Bearer ${token}` },
          });
          const data = await response.json();
          return Boolean(data.valid);
        } catch (error) {
          console.error('Token validation error', error);
          return false;
        }
      },
      startConnectionMonitoring: () => startConnectionMonitoring(),
      stopConnectionMonitoring: () => stopConnectionMonitoring(),
      checkConnection: () => checkConnection(),
    };

    return () => {
      delete window.VrooliAuth;
    };
  }, [apiUrl, checkConnection, startConnectionMonitoring, stopConnectionMonitoring]);

  const handleTabChange = useCallback(
    (tab: Tab) => {
      setActiveTab(tab);
      clearErrors();
      clearSuccessMessage();
      if (tab === 'login') {
        setResetState((prev) => ({ ...prev, email: '' }));
      }
    },
    [clearErrors, clearSuccessMessage]
  );

  const handleLogin = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      clearErrors();

      if (!apiUrl) {
        showSnackbar('Authentication service not configured yet.', 'warning');
        return;
      }

      setLoginState((prev) => ({ ...prev, loading: true }));

      try {
        const response = await fetch(`${apiUrl}/api/v1/auth/login`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            email: loginState.email,
            password: loginState.password,
          }),
        });

        const data = await response.json();

        if (response.ok && data.success) {
          persistAuthTokens(
            data.token,
            data.refresh_token,
            data.user,
            loginState.remember
          );
          showSuccess('Login successful! Redirecting...');
          redirectAfterAuth(redirectUrl);
        } else {
          const message = data.error || 'Invalid email or password';
          setError('loginEmail', message);
          setError('loginPassword', message);
          showSnackbar(message, 'error');
        }
      } catch (error) {
        console.error('Login error', error);
        const message =
          'Connection failed. Please check your network and try again.';
        setError('loginEmail', message);
        setError('loginPassword', message);
        showSnackbar(message, 'error');
      } finally {
        setLoginState((prev) => ({ ...prev, loading: false }));
      }
    },
    [apiUrl, clearErrors, loginState, redirectUrl, setError, showSnackbar, showSuccess]
  );

  const handleRegister = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      clearErrors();

      if (!apiUrl) {
        showSnackbar('Authentication service not configured yet.', 'warning');
        return;
      }

      if (registerState.password !== registerState.confirmPassword) {
        const message = 'Passwords do not match';
        setError('confirmPassword', message);
        showSnackbar(message, 'error');
        return;
      }

      if (registerState.password.length < PASSWORD_MIN_LENGTH) {
        const message = `Password must be at least ${PASSWORD_MIN_LENGTH} characters`;
        setError('registerPassword', message);
        showSnackbar(message, 'error');
        return;
      }

      if (!registerState.agree) {
        const message = 'Please agree to the terms and conditions';
        setError('agree', message);
        showSnackbar(message, 'warning');
        return;
      }

      setRegisterState((prev) => ({ ...prev, loading: true }));

      try {
        const body: Record<string, string> = {
          email: registerState.email,
          password: registerState.password,
        };
        if (registerState.username) {
          body.username = registerState.username;
        }

        const response = await fetch(`${apiUrl}/api/v1/auth/register`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        });

        const data = await response.json();

        if (response.ok && data.success) {
          persistAuthTokens(data.token, data.refresh_token, data.user, false);
          showSuccess('Account created successfully! Redirecting...');
          setRegisterState({
            email: '',
            username: '',
            password: '',
            confirmPassword: '',
            agree: false,
            loading: false,
          });
          redirectAfterAuth(redirectUrl);
        } else {
          const message = data.error || 'Registration failed';
          if (data.error?.includes('email')) {
            setError('registerEmail', message);
          } else if (data.error?.includes('username')) {
            setError('registerUsername', message);
          } else {
            setError('registerEmail', message);
          }
          showSnackbar(message, 'error');
        }
      } catch (error) {
        console.error('Registration error', error);
        const message =
          'Connection failed. Please check your network and try again.';
        setError('registerEmail', message);
        showSnackbar(message, 'error');
      } finally {
        setRegisterState((prev) => ({ ...prev, loading: false }));
      }
    },
    [
      apiUrl,
      clearErrors,
      redirectUrl,
      registerState,
      setError,
      showSnackbar,
      showSuccess,
    ]
  );

  const handleResetPassword = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      clearErrors();

      if (!apiUrl) {
        showSnackbar('Authentication service not configured yet.', 'warning');
        return;
      }

      setResetState((prev) => ({ ...prev, loading: true }));

      try {
        const response = await fetch(`${apiUrl}/api/v1/auth/reset-password`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: resetState.email }),
        });

        const data = await response.json();

        if (response.ok && data.success) {
          showSuccess('Password reset instructions have been sent to your email.');
          setResetState({ email: '', loading: false });
          window.setTimeout(() => handleTabChange('login'), 3000);
        } else {
          const message = data.error || 'Failed to send reset email';
          setError('resetEmail', message);
          showSnackbar(message, 'error');
          setResetState((prev) => ({ ...prev, loading: false }));
        }
      } catch (error) {
        console.error('Reset password error', error);
        const message =
          'Connection failed. Please check your network and try again.';
        setError('resetEmail', message);
        showSnackbar(message, 'error');
        setResetState((prev) => ({ ...prev, loading: false }));
      }
    },
    [apiUrl, clearErrors, handleTabChange, resetState.email, setError, showSnackbar, showSuccess]
  );

  const passwordStrength = useMemo(
    () => evaluatePasswordStrength(registerState.password),
    [registerState.password]
  );

  const renderPasswordToggle = (field: PasswordField) => {
    const isVisible = passwordVisibility[field];
    const Icon = isVisible ? EyeOff : Eye;

    return (
      <button
        type="button"
        className="password-toggle-button"
        onClick={() => togglePasswordVisibility(field)}
        aria-label={`${isVisible ? 'Hide' : 'Show'} password`}
        aria-pressed={isVisible}
      >
        <Icon size={16} />
      </button>
    );
  };

  return (
    <div className="page">
      <div
        className={`connection-status connection-status--${connectionStatus.state}`}
        role="status"
        tabIndex={0}
        onClick={() => {
          void checkConnection();
        }}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            void checkConnection();
          }
        }}
        title="Click to refresh connection"
      >
        <span className="connection-status__icon">
          {connectionStatus.state === 'checking' && (
            <Loader2 size={16} className="icon-rotate" />
          )}
          {connectionStatus.state === 'connected' && (
            <CheckCircle2 size={16} />
          )}
          {connectionStatus.state === 'disconnected' && (
            <AlertTriangle size={16} />
          )}
        </span>
        <span>{connectionStatus.message}</span>
      </div>

      {configError ? (
        <div className="auth-wrapper">
          <div className="config-error" role="alert">
            <div className="config-error__icon">
              <AlertTriangle size={32} />
            </div>
            <h2>Configuration Error</h2>
            <p>{configError}</p>
            <code>vrooli scenario run scenario-authenticator</code>
          </div>
        </div>
      ) : (
        <div className="auth-wrapper">
          <div className="auth-card">
            <div className="logo">
              <h1>Vrooli</h1>
              <p>Secure Authentication Service</p>
            </div>

            {successMessage && (
              <div className="success-banner" role="status">
                <ShieldCheck size={18} />
                <span>{successMessage}</span>
              </div>
            )}

            <div className="tabs" role="tablist">
              <button
                type="button"
                className={`tab-button ${
                  activeTab === 'login' ? 'is-active' : ''
                }`}
                role="tab"
                aria-selected={activeTab === 'login'}
                onClick={() => handleTabChange('login')}
              >
                Sign In
              </button>
              <button
                type="button"
                className={`tab-button ${
                  activeTab === 'register' ? 'is-active' : ''
                }`}
                role="tab"
                aria-selected={activeTab === 'register'}
                onClick={() => handleTabChange('register')}
              >
                Sign Up
              </button>
            </div>

            {activeTab === 'login' && (
              <form onSubmit={handleLogin} aria-label="Sign in form">
                <div className="form-group">
                  <label htmlFor="loginEmail">Email Address</label>
                  <input
                    id="loginEmail"
                    type="email"
                    autoComplete="email"
                    value={loginState.email}
                    onChange={(event) =>
                      setLoginState((prev) => ({
                        ...prev,
                        email: event.target.value,
                      }))
                    }
                    className={fieldErrors.loginEmail ? 'input-error' : ''}
                    required
                  />
                  {fieldErrors.loginEmail && (
                    <div className="error-text" role="alert">
                      {fieldErrors.loginEmail}
                    </div>
                  )}
                </div>

                <div className="form-group">
                  <label htmlFor="loginPassword">Password</label>
                  <div className="password-field">
                    <input
                      id="loginPassword"
                      type={passwordVisibility.loginPassword ? 'text' : 'password'}
                      autoComplete="current-password"
                      value={loginState.password}
                      onChange={(event) =>
                        setLoginState((prev) => ({
                          ...prev,
                          password: event.target.value,
                        }))
                      }
                      className={fieldErrors.loginPassword ? 'input-error' : ''}
                      required
                    />
                    {renderPasswordToggle('loginPassword')}
                  </div>
                  {fieldErrors.loginPassword && (
                    <div className="error-text" role="alert">
                      {fieldErrors.loginPassword}
                    </div>
                  )}
                </div>

                <div className="checkbox-group">
                  <input
                    id="rememberMe"
                    type="checkbox"
                    checked={loginState.remember}
                    onChange={(event) =>
                      setLoginState((prev) => ({
                        ...prev,
                        remember: event.target.checked,
                      }))
                    }
                  />
                  <label htmlFor="rememberMe">Remember me for 7 days</label>
                </div>

                <button
                  className="btn btn-primary"
                  type="submit"
                  disabled={loginState.loading}
                >
                  {loginState.loading && (
                    <Loader2 size={16} className="icon-rotate" />
                  )}
                  <span>{loginState.loading ? 'Signing in...' : 'Sign In'}</span>
                </button>

                <div className="form-footer">
                  <button
                    type="button"
                    className="link-button"
                    onClick={() => handleTabChange('reset')}
                  >
                    Forgot your password?
                  </button>
                </div>
              </form>
            )}

            {activeTab === 'register' && (
              <form onSubmit={handleRegister} aria-label="Sign up form">
                <div className="form-group">
                  <label htmlFor="registerEmail">Email Address</label>
                  <input
                    id="registerEmail"
                    type="email"
                    autoComplete="email"
                    value={registerState.email}
                    onChange={(event) =>
                      setRegisterState((prev) => ({
                        ...prev,
                        email: event.target.value,
                      }))
                    }
                    className={fieldErrors.registerEmail ? 'input-error' : ''}
                    required
                  />
                  {fieldErrors.registerEmail && (
                    <div className="error-text" role="alert">
                      {fieldErrors.registerEmail}
                    </div>
                  )}
                </div>

                <div className="form-group">
                  <label htmlFor="registerUsername">Username (Optional)</label>
                  <input
                    id="registerUsername"
                    type="text"
                    autoComplete="username"
                    value={registerState.username}
                    onChange={(event) =>
                      setRegisterState((prev) => ({
                        ...prev,
                        username: event.target.value,
                      }))
                    }
                    className={fieldErrors.registerUsername ? 'input-error' : ''}
                  />
                  {fieldErrors.registerUsername && (
                    <div className="error-text" role="alert">
                      {fieldErrors.registerUsername}
                    </div>
                  )}
                </div>

                <div className="form-group">
                  <label htmlFor="registerPassword">Password</label>
                  <div className="password-field">
                    <input
                      id="registerPassword"
                      type={passwordVisibility.registerPassword ? 'text' : 'password'}
                      autoComplete="new-password"
                      value={registerState.password}
                      onChange={(event) =>
                        setRegisterState((prev) => ({
                          ...prev,
                          password: event.target.value,
                        }))
                      }
                      className={fieldErrors.registerPassword ? 'input-error' : ''}
                      required
                    />
                    {renderPasswordToggle('registerPassword')}
                  </div>
                  <div className="password-strength">
                    <div
                      className={`password-strength-bar ${passwordStrength}`}
                    />
                  </div>
                  {fieldErrors.registerPassword && (
                    <div className="error-text" role="alert">
                      {fieldErrors.registerPassword}
                    </div>
                  )}
                </div>

                <div className="form-group">
                  <label htmlFor="confirmPassword">Confirm Password</label>
                  <div className="password-field">
                    <input
                      id="confirmPassword"
                      type={passwordVisibility.confirmPassword ? 'text' : 'password'}
                      autoComplete="new-password"
                      value={registerState.confirmPassword}
                      onChange={(event) =>
                        setRegisterState((prev) => ({
                          ...prev,
                          confirmPassword: event.target.value,
                        }))
                      }
                      className={fieldErrors.confirmPassword ? 'input-error' : ''}
                      required
                    />
                    {renderPasswordToggle('confirmPassword')}
                  </div>
                  {fieldErrors.confirmPassword && (
                    <div className="error-text" role="alert">
                      {fieldErrors.confirmPassword}
                    </div>
                  )}
                </div>

                <div className="checkbox-group">
                  <input
                    id="agreeTerms"
                    type="checkbox"
                    checked={registerState.agree}
                    onChange={(event) =>
                      setRegisterState((prev) => ({
                        ...prev,
                        agree: event.target.checked,
                      }))
                    }
                    aria-invalid={Boolean(fieldErrors.agree)}
                    required
                  />
                  <label htmlFor="agreeTerms">I agree to the Terms and Conditions</label>
                </div>
                {fieldErrors.agree && (
                  <div className="error-text" role="alert">
                    {fieldErrors.agree}
                  </div>
                )}

                <button
                  className="btn btn-primary"
                  type="submit"
                  disabled={registerState.loading}
                >
                  {registerState.loading && (
                    <Loader2 size={16} className="icon-rotate" />
                  )}
                  <span>
                    {registerState.loading ? 'Creating account...' : 'Create Account'}
                  </span>
                </button>
              </form>
            )}

            {activeTab === 'reset' && (
              <form onSubmit={handleResetPassword} aria-label="Password reset form">
                <p className="form-helper-text">
                  Enter your email address and we will send you instructions to reset
                  your password.
                </p>

                <div className="form-group">
                  <label htmlFor="resetEmail">Email Address</label>
                  <input
                    id="resetEmail"
                    type="email"
                    autoComplete="email"
                    value={resetState.email}
                    onChange={(event) =>
                      setResetState((prev) => ({
                        ...prev,
                        email: event.target.value,
                      }))
                    }
                    className={fieldErrors.resetEmail ? 'input-error' : ''}
                    required
                  />
                  {fieldErrors.resetEmail && (
                    <div className="error-text" role="alert">
                      {fieldErrors.resetEmail}
                    </div>
                  )}
                </div>

                <button
                  className="btn btn-primary"
                  type="submit"
                  disabled={resetState.loading}
                >
                  {resetState.loading && (
                    <Loader2 size={16} className="icon-rotate" />
                  )}
                  <span>
                    {resetState.loading
                      ? 'Sending instructions...'
                      : 'Send Reset Instructions'}
                  </span>
                </button>

                <div className="form-footer">
                  <button
                    type="button"
                    className="link-button"
                    onClick={() => handleTabChange('login')}
                  >
                    Back to Sign In
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}

      <div
        className={`snackbar snackbar--${snackbar.type} ${
          snackbar.visible ? 'snackbar--visible' : ''
        }`}
        role="status"
        aria-live="polite"
      >
        <span className="snackbar__icon" aria-hidden="true">
          {snackbar.type === 'success' && <CheckCircle2 size={18} />}
          {snackbar.type === 'warning' && <AlertTriangle size={18} />}
          {snackbar.type === 'error' && <AlertTriangle size={18} />}
          {snackbar.type === 'info' && <Info size={18} />}
        </span>
        <span className="snackbar__message">{snackbar.message}</span>
        <button
          className="snackbar__close"
          type="button"
          onClick={hideSnackbar}
          aria-label="Dismiss notification"
        >
          <XIcon size={16} />
        </button>
      </div>
    </div>
  );
}
