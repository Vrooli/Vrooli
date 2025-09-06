// Social Media Scheduler React Application
const { useState, useEffect, useRef, useCallback, useMemo } = React;

// Utility functions
const formatDate = (date) => {
    return new Intl.DateTimeFormat('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        timeZoneName: 'short'
    }).format(new Date(date));
};

const formatRelativeTime = (date) => {
    const now = new Date();
    const postDate = new Date(date);
    const diffMs = postDate - now;
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffMs < 0) {
        return 'Posted';
    } else if (diffMins < 60) {
        return `in ${diffMins} min${diffMins !== 1 ? 's' : ''}`;
    } else if (diffHours < 24) {
        return `in ${diffHours} hour${diffHours !== 1 ? 's' : ''}`;
    } else {
        return `in ${diffDays} day${diffDays !== 1 ? 's' : ''}`;
    }
};

// Platform configurations
const PLATFORMS = {
    twitter: { 
        name: 'Twitter', 
        color: '#1DA1F2', 
        icon: 'fab fa-twitter', 
        maxChars: 280,
        bgColor: '#1DA1F2',
        textColor: 'white'
    },
    instagram: { 
        name: 'Instagram', 
        color: '#E4405F', 
        icon: 'fab fa-instagram',
        maxChars: 2200,
        bgColor: 'linear-gradient(45deg, #f09433 0%,#e6683c 25%,#dc2743 50%,#cc2366 75%,#bc1888 100%)',
        textColor: 'white'
    },
    linkedin: { 
        name: 'LinkedIn', 
        color: '#0077B5', 
        icon: 'fab fa-linkedin',
        maxChars: 3000,
        bgColor: '#0077B5',
        textColor: 'white'
    },
    facebook: { 
        name: 'Facebook', 
        color: '#1877F2', 
        icon: 'fab fa-facebook',
        maxChars: 63206,
        bgColor: '#1877F2',
        textColor: 'white'
    }
};

// API service
const API = {
    baseURL: '/api/v1',
    
    async request(endpoint, options = {}) {
        const token = localStorage.getItem('auth_token');
        
        const config = {
            headers: {
                'Content-Type': 'application/json',
                ...(token && { Authorization: `Bearer ${token}` }),
                ...options.headers
            },
            ...options
        };

        try {
            const response = await fetch(`${this.baseURL}${endpoint}`, config);
            const data = await response.json();
            
            if (!data.success && response.status === 401) {
                localStorage.removeItem('auth_token');
                localStorage.removeItem('user_data');
                window.location.href = '/login';
                return null;
            }
            
            return data;
        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    },

    // Auth endpoints
    async login(email, password) {
        return this.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password })
        });
    },

    async register(userData) {
        return this.request('/auth/register', {
            method: 'POST',
            body: JSON.stringify(userData)
        });
    },

    async getCurrentUser() {
        return this.request('/auth/me');
    },

    async getPlatforms() {
        return this.request('/auth/platforms');
    },

    // Post endpoints
    async schedulePost(postData) {
        return this.request('/posts/schedule', {
            method: 'POST',
            body: JSON.stringify(postData)
        });
    },

    async getCalendarPosts(startDate, endDate, platforms = []) {
        const params = new URLSearchParams();
        if (startDate) params.append('start_date', startDate);
        if (endDate) params.append('end_date', endDate);
        platforms.forEach(platform => params.append('platforms', platform));
        
        return this.request(`/posts/calendar?${params.toString()}`);
    },

    async getPost(postId) {
        return this.request(`/posts/${postId}`);
    },

    async updatePost(postId, updates) {
        return this.request(`/posts/${postId}`, {
            method: 'PUT',
            body: JSON.stringify(updates)
        });
    },

    async deletePost(postId) {
        return this.request(`/posts/${postId}`, {
            method: 'DELETE'
        });
    },

    // Analytics endpoints
    async getAnalyticsOverview() {
        return this.request('/analytics/overview');
    },

    async getUserAccounts() {
        return this.request('/user/accounts');
    }
};

// WebSocket service for real-time updates
const useWebSocket = () => {
    const [socket, setSocket] = useState(null);
    const [connected, setConnected] = useState(false);

    useEffect(() => {
        const token = localStorage.getItem('auth_token');
        if (!token) return;

        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${wsProtocol}//${window.location.host}/ws`;

        const newSocket = new WebSocket(wsUrl);

        newSocket.onopen = () => {
            console.log('WebSocket connected');
            setConnected(true);
        };

        newSocket.onclose = () => {
            console.log('WebSocket disconnected');
            setConnected(false);
        };

        newSocket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        setSocket(newSocket);

        return () => {
            newSocket.close();
        };
    }, []);

    return { socket, connected };
};

// Authentication hook
const useAuth = () => {
    const [user, setUser] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const initAuth = async () => {
            const token = localStorage.getItem('auth_token');
            if (!token) {
                setLoading(false);
                return;
            }

            try {
                const response = await API.getCurrentUser();
                if (response.success) {
                    setUser(response.data);
                } else {
                    localStorage.removeItem('auth_token');
                    localStorage.removeItem('user_data');
                }
            } catch (error) {
                console.error('Auth initialization error:', error);
                localStorage.removeItem('auth_token');
                localStorage.removeItem('user_data');
            }
            
            setLoading(false);
        };

        initAuth();
    }, []);

    const login = async (email, password) => {
        const response = await API.login(email, password);
        if (response.success) {
            localStorage.setItem('auth_token', response.data.token);
            localStorage.setItem('user_data', JSON.stringify(response.data.user));
            setUser(response.data.user);
            return true;
        }
        return false;
    };

    const logout = () => {
        localStorage.removeItem('auth_token');
        localStorage.removeItem('user_data');
        setUser(null);
    };

    return { user, loading, login, logout };
};

// Login Component
const LoginPage = ({ onLogin }) => {
    const [email, setEmail] = useState('demo@vrooli.com');
    const [password, setPassword] = useState('demo123');
    const [isLogin, setIsLogin] = useState(true);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);
        setError('');

        try {
            const success = await onLogin(email, password);
            if (!success) {
                setError('Invalid credentials');
            }
        } catch (error) {
            setError('Login failed. Please try again.');
        }
        
        setLoading(false);
    };

    return (
        <div className="min-h-screen bg-gradient-to-br from-blue-500 via-purple-600 to-green-500 flex items-center justify-center p-4">
            <div className="bg-white rounded-lg shadow-2xl p-8 w-full max-w-md">
                <div className="text-center mb-8">
                    <div className="mx-auto w-16 h-16 bg-gradient-to-r from-blue-500 to-green-500 rounded-full flex items-center justify-center mb-4">
                        <i className="fas fa-calendar-alt text-white text-2xl"></i>
                    </div>
                    <h1 className="text-2xl font-bold text-gray-900">Social Media Scheduler</h1>
                    <p className="text-gray-600 mt-2">Your intelligent social media command center</p>
                </div>

                <form onSubmit={handleSubmit} className="space-y-6">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Email</label>
                        <input
                            type="email"
                            value={email}
                            onChange={(e) => setEmail(e.target.value)}
                            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                            required
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Password</label>
                        <input
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                            required
                        />
                    </div>

                    {error && (
                        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md text-sm">
                            {error}
                        </div>
                    )}

                    <button
                        type="submit"
                        disabled={loading}
                        className="w-full bg-gradient-to-r from-blue-500 to-green-500 text-white py-2 px-4 rounded-md hover:from-blue-600 hover:to-green-600 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                    >
                        {loading ? 'Signing In...' : 'Sign In'}
                    </button>
                </form>

                <div className="mt-6 text-center">
                    <div className="bg-blue-50 border border-blue-200 text-blue-700 px-4 py-3 rounded-md text-sm">
                        <strong>Demo Account:</strong><br />
                        Email: demo@vrooli.com<br />
                        Password: demo123
                    </div>
                </div>

                <div className="mt-6 flex justify-center space-x-6 text-gray-400">
                    <i className="fab fa-twitter text-xl"></i>
                    <i className="fab fa-instagram text-xl"></i>
                    <i className="fab fa-linkedin text-xl"></i>
                    <i className="fab fa-facebook text-xl"></i>
                </div>
            </div>
        </div>
    );
};

// Post Card Component
const PostCard = ({ post, onEdit, onDelete, onView }) => {
    const getStatusColor = (status) => {
        switch (status) {
            case 'scheduled': return 'bg-yellow-100 text-yellow-800';
            case 'posted': return 'bg-green-100 text-green-800';
            case 'failed': return 'bg-red-100 text-red-800';
            case 'posting': return 'bg-blue-100 text-blue-800';
            default: return 'bg-gray-100 text-gray-800';
        }
    };

    const truncateContent = (content, maxLength = 100) => {
        if (content.length <= maxLength) return content;
        return content.substring(0, maxLength) + '...';
    };

    return (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 hover:shadow-md transition-shadow">
            <div className="flex items-start justify-between mb-3">
                <h3 className="font-medium text-gray-900 truncate flex-1">{post.title}</h3>
                <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(post.status)}`}>
                    {post.status}
                </span>
            </div>

            <p className="text-gray-600 text-sm mb-3 line-clamp-2">
                {truncateContent(post.content)}
            </p>

            <div className="flex items-center justify-between mb-3">
                <div className="flex space-x-1">
                    {post.platforms.map(platform => (
                        <div
                            key={platform}
                            className="w-6 h-6 rounded-full flex items-center justify-center text-white text-xs"
                            style={{ backgroundColor: PLATFORMS[platform]?.color }}
                            title={PLATFORMS[platform]?.name}
                        >
                            <i className={PLATFORMS[platform]?.icon}></i>
                        </div>
                    ))}
                </div>
                <span className="text-xs text-gray-500">
                    {formatRelativeTime(post.scheduled_at)}
                </span>
            </div>

            <div className="text-xs text-gray-500 mb-3">
                {formatDate(post.scheduled_at)}
            </div>

            <div className="flex space-x-2">
                <button
                    onClick={() => onView(post)}
                    className="flex-1 bg-gray-50 text-gray-700 px-3 py-2 rounded-md text-sm hover:bg-gray-100 transition-colors"
                >
                    <i className="fas fa-eye mr-1"></i> View
                </button>
                <button
                    onClick={() => onEdit(post)}
                    className="flex-1 bg-blue-50 text-blue-700 px-3 py-2 rounded-md text-sm hover:bg-blue-100 transition-colors"
                >
                    <i className="fas fa-edit mr-1"></i> Edit
                </button>
                <button
                    onClick={() => onDelete(post.id)}
                    className="bg-red-50 text-red-700 px-3 py-2 rounded-md text-sm hover:bg-red-100 transition-colors"
                >
                    <i className="fas fa-trash"></i>
                </button>
            </div>
        </div>
    );
};

// Calendar View Component
const CalendarView = ({ posts, onDateSelect, selectedDate }) => {
    const [currentMonth, setCurrentMonth] = useState(new Date());

    const startOfMonth = new Date(currentMonth.getFullYear(), currentMonth.getMonth(), 1);
    const endOfMonth = new Date(currentMonth.getFullYear(), currentMonth.getMonth() + 1, 0);
    const startOfCalendar = new Date(startOfMonth);
    startOfCalendar.setDate(startOfCalendar.getDate() - startOfCalendar.getDay());

    const daysInCalendar = [];
    const current = new Date(startOfCalendar);
    
    for (let i = 0; i < 42; i++) { // 6 weeks * 7 days
        daysInCalendar.push(new Date(current));
        current.setDate(current.getDate() + 1);
    }

    const getPostsForDate = (date) => {
        const dateStr = date.toISOString().split('T')[0];
        return posts.filter(post => {
            const postDate = new Date(post.scheduled_at).toISOString().split('T')[0];
            return postDate === dateStr;
        });
    };

    const isToday = (date) => {
        const today = new Date();
        return date.toDateString() === today.toDateString();
    };

    const isSelectedDate = (date) => {
        if (!selectedDate) return false;
        return date.toDateString() === selectedDate.toDateString();
    };

    const isCurrentMonth = (date) => {
        return date.getMonth() === currentMonth.getMonth();
    };

    return (
        <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between mb-6">
                <h2 className="text-xl font-semibold text-gray-900">
                    {currentMonth.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
                </h2>
                <div className="flex space-x-2">
                    <button
                        onClick={() => setCurrentMonth(new Date(currentMonth.getFullYear(), currentMonth.getMonth() - 1))}
                        className="p-2 hover:bg-gray-100 rounded-md"
                    >
                        <i className="fas fa-chevron-left text-gray-500"></i>
                    </button>
                    <button
                        onClick={() => setCurrentMonth(new Date())}
                        className="px-3 py-2 text-sm bg-blue-50 text-blue-700 rounded-md hover:bg-blue-100"
                    >
                        Today
                    </button>
                    <button
                        onClick={() => setCurrentMonth(new Date(currentMonth.getFullYear(), currentMonth.getMonth() + 1))}
                        className="p-2 hover:bg-gray-100 rounded-md"
                    >
                        <i className="fas fa-chevron-right text-gray-500"></i>
                    </button>
                </div>
            </div>

            <div className="grid grid-cols-7 gap-1 mb-2">
                {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map(day => (
                    <div key={day} className="p-2 text-center text-xs font-medium text-gray-500 uppercase">
                        {day}
                    </div>
                ))}
            </div>

            <div className="grid grid-cols-7 gap-1">
                {daysInCalendar.map((date, index) => {
                    const dayPosts = getPostsForDate(date);
                    const isCurrentMonthDate = isCurrentMonth(date);
                    
                    return (
                        <div
                            key={index}
                            onClick={() => onDateSelect(date)}
                            className={`
                                min-h-[80px] p-1 border border-gray-100 cursor-pointer hover:bg-blue-50 transition-colors
                                ${isSelectedDate(date) ? 'bg-blue-100 border-blue-300' : ''}
                                ${isToday(date) ? 'ring-2 ring-blue-500' : ''}
                                ${!isCurrentMonthDate ? 'text-gray-400 bg-gray-50' : ''}
                            `}
                        >
                            <div className="text-sm font-medium mb-1">
                                {date.getDate()}
                            </div>
                            
                            {dayPosts.length > 0 && (
                                <div className="space-y-1">
                                    {dayPosts.slice(0, 2).map((post, postIndex) => (
                                        <div
                                            key={postIndex}
                                            className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded truncate"
                                            title={post.title}
                                        >
                                            {post.title}
                                        </div>
                                    ))}
                                    {dayPosts.length > 2 && (
                                        <div className="text-xs text-gray-500">
                                            +{dayPosts.length - 2} more
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                    );
                })}
            </div>
        </div>
    );
};

// Schedule Post Modal
const SchedulePostModal = ({ isOpen, onClose, onSave }) => {
    const [formData, setFormData] = useState({
        title: '',
        content: '',
        platforms: [],
        scheduled_at: '',
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        auto_optimize: true
    });
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    useEffect(() => {
        if (isOpen) {
            // Set default scheduled time to 1 hour from now
            const defaultTime = new Date();
            defaultTime.setHours(defaultTime.getHours() + 1);
            defaultTime.setMinutes(0, 0, 0); // Round to nearest hour
            
            setFormData(prev => ({
                ...prev,
                scheduled_at: defaultTime.toISOString().slice(0, 16) // Format for datetime-local input
            }));
        }
    }, [isOpen]);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);
        setError('');

        try {
            const scheduledDate = new Date(formData.scheduled_at);
            const response = await API.schedulePost({
                ...formData,
                scheduled_at: scheduledDate.toISOString()
            });

            if (response.success) {
                onSave(response.data);
                onClose();
                setFormData({
                    title: '',
                    content: '',
                    platforms: [],
                    scheduled_at: '',
                    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                    auto_optimize: true
                });
            } else {
                setError(response.error || 'Failed to schedule post');
            }
        } catch (error) {
            setError('Failed to schedule post. Please try again.');
        }
        
        setLoading(false);
    };

    const togglePlatform = (platform) => {
        setFormData(prev => ({
            ...prev,
            platforms: prev.platforms.includes(platform)
                ? prev.platforms.filter(p => p !== platform)
                : [...prev.platforms, platform]
        }));
    };

    const getCharacterCount = (platform) => {
        const maxChars = PLATFORMS[platform]?.maxChars || 280;
        const content = formData.content;
        return `${content.length}/${maxChars}`;
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
                <div className="p-6">
                    <div className="flex items-center justify-between mb-6">
                        <h2 className="text-xl font-semibold text-gray-900">Schedule New Post</h2>
                        <button
                            onClick={onClose}
                            className="p-2 hover:bg-gray-100 rounded-md"
                        >
                            <i className="fas fa-times text-gray-500"></i>
                        </button>
                    </div>

                    <form onSubmit={handleSubmit} className="space-y-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Title</label>
                            <input
                                type="text"
                                value={formData.title}
                                onChange={(e) => setFormData(prev => ({ ...prev, title: e.target.value }))}
                                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                                placeholder="Enter post title..."
                                required
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Content</label>
                            <textarea
                                value={formData.content}
                                onChange={(e) => setFormData(prev => ({ ...prev, content: e.target.value }))}
                                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 h-32"
                                placeholder="Write your post content here..."
                                required
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-3">Platforms</label>
                            <div className="grid grid-cols-2 gap-3">
                                {Object.entries(PLATFORMS).map(([key, platform]) => (
                                    <div
                                        key={key}
                                        onClick={() => togglePlatform(key)}
                                        className={`
                                            p-4 border-2 rounded-lg cursor-pointer transition-all
                                            ${formData.platforms.includes(key)
                                                ? 'border-blue-500 bg-blue-50'
                                                : 'border-gray-200 hover:border-gray-300'
                                            }
                                        `}
                                    >
                                        <div className="flex items-center space-x-3">
                                            <div
                                                className="w-8 h-8 rounded-full flex items-center justify-center text-white"
                                                style={{ backgroundColor: platform.color }}
                                            >
                                                <i className={platform.icon}></i>
                                            </div>
                                            <div>
                                                <div className="font-medium text-gray-900">{platform.name}</div>
                                                <div className="text-xs text-gray-500">
                                                    {getCharacterCount(key)}
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Scheduled Time</label>
                            <input
                                type="datetime-local"
                                value={formData.scheduled_at}
                                onChange={(e) => setFormData(prev => ({ ...prev, scheduled_at: e.target.value }))}
                                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                                required
                            />
                        </div>

                        <div className="flex items-center">
                            <input
                                type="checkbox"
                                id="auto_optimize"
                                checked={formData.auto_optimize}
                                onChange={(e) => setFormData(prev => ({ ...prev, auto_optimize: e.target.checked }))}
                                className="mr-2"
                            />
                            <label htmlFor="auto_optimize" className="text-sm text-gray-700">
                                Auto-optimize content for each platform
                            </label>
                        </div>

                        {error && (
                            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md text-sm">
                                {error}
                            </div>
                        )}

                        <div className="flex justify-end space-x-3">
                            <button
                                type="button"
                                onClick={onClose}
                                className="px-4 py-2 text-gray-700 bg-gray-200 rounded-md hover:bg-gray-300 transition-colors"
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                disabled={loading || formData.platforms.length === 0}
                                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 transition-colors"
                            >
                                {loading ? 'Scheduling...' : 'Schedule Post'}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    );
};

// Main Dashboard Component
const Dashboard = ({ user, onLogout }) => {
    const [posts, setPosts] = useState([]);
    const [loading, setLoading] = useState(true);
    const [selectedDate, setSelectedDate] = useState(null);
    const [showScheduleModal, setShowScheduleModal] = useState(false);
    const [currentView, setCurrentView] = useState('calendar'); // 'calendar' or 'list'
    const { socket, connected } = useWebSocket();

    // Load posts
    useEffect(() => {
        loadPosts();
    }, []);

    // WebSocket event handlers
    useEffect(() => {
        if (!socket) return;

        socket.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                
                switch (data.type) {
                    case 'post_scheduled':
                    case 'post_updated':
                        loadPosts(); // Refresh posts
                        break;
                    case 'post_posted':
                        loadPosts(); // Refresh posts
                        // Could add toast notification here
                        break;
                    case 'post_failed':
                        loadPosts(); // Refresh posts
                        // Could add error notification here
                        break;
                    default:
                        console.log('Received WebSocket message:', data);
                }
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };
    }, [socket]);

    const loadPosts = async () => {
        try {
            // Load posts for the current month
            const startOfMonth = new Date();
            startOfMonth.setDate(1);
            startOfMonth.setHours(0, 0, 0, 0);
            
            const endOfMonth = new Date();
            endOfMonth.setMonth(endOfMonth.getMonth() + 1);
            endOfMonth.setDate(0);
            endOfMonth.setHours(23, 59, 59, 999);

            const response = await API.getCalendarPosts(
                startOfMonth.toISOString(),
                endOfMonth.toISOString()
            );
            
            if (response.success) {
                setPosts(response.data || []);
            }
        } catch (error) {
            console.error('Error loading posts:', error);
        } finally {
            setLoading(false);
        }
    };

    const handleSchedulePost = (newPost) => {
        setPosts(prev => [newPost, ...prev]);
    };

    const handleDeletePost = async (postId) => {
        if (!confirm('Are you sure you want to delete this post?')) return;

        try {
            const response = await API.deletePost(postId);
            if (response.success) {
                setPosts(prev => prev.filter(post => post.id !== postId));
            }
        } catch (error) {
            console.error('Error deleting post:', error);
        }
    };

    const getFilteredPosts = () => {
        if (!selectedDate) return posts;
        
        const selectedDateStr = selectedDate.toISOString().split('T')[0];
        return posts.filter(post => {
            const postDate = new Date(post.scheduled_at).toISOString().split('T')[0];
            return postDate === selectedDateStr;
        });
    };

    const getStatusCounts = () => {
        const counts = { scheduled: 0, posted: 0, failed: 0 };
        posts.forEach(post => {
            if (counts.hasOwnProperty(post.status)) {
                counts[post.status]++;
            }
        });
        return counts;
    };

    const statusCounts = getStatusCounts();

    if (loading) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
                    <div className="text-gray-600">Loading your posts...</div>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50">
            {/* Header */}
            <header className="bg-white shadow-sm border-b border-gray-200">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex items-center justify-between h-16">
                        <div className="flex items-center space-x-4">
                            <div className="bg-gradient-to-r from-blue-500 to-green-500 w-8 h-8 rounded-lg flex items-center justify-center">
                                <i className="fas fa-calendar-alt text-white"></i>
                            </div>
                            <div>
                                <h1 className="text-xl font-semibold text-gray-900">Social Media Scheduler</h1>
                                <div className="flex items-center space-x-2 text-sm text-gray-500">
                                    <span>Welcome, {user.first_name || user.email}</span>
                                    {connected && (
                                        <span className="flex items-center space-x-1">
                                            <div className="w-2 h-2 bg-green-400 rounded-full"></div>
                                            <span>Live</span>
                                        </span>
                                    )}
                                </div>
                            </div>
                        </div>

                        <div className="flex items-center space-x-4">
                            <button
                                onClick={() => setShowScheduleModal(true)}
                                className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center space-x-2"
                            >
                                <i className="fas fa-plus"></i>
                                <span>Schedule Post</span>
                            </button>

                            <div className="flex items-center space-x-2">
                                <button
                                    onClick={() => setCurrentView('calendar')}
                                    className={`p-2 rounded-md transition-colors ${
                                        currentView === 'calendar' 
                                            ? 'bg-blue-100 text-blue-700' 
                                            : 'text-gray-500 hover:bg-gray-100'
                                    }`}
                                >
                                    <i className="fas fa-calendar-alt"></i>
                                </button>
                                <button
                                    onClick={() => setCurrentView('list')}
                                    className={`p-2 rounded-md transition-colors ${
                                        currentView === 'list' 
                                            ? 'bg-blue-100 text-blue-700' 
                                            : 'text-gray-500 hover:bg-gray-100'
                                    }`}
                                >
                                    <i className="fas fa-list"></i>
                                </button>
                            </div>

                            <div className="relative">
                                <button
                                    onClick={onLogout}
                                    className="p-2 text-gray-500 hover:bg-gray-100 rounded-md"
                                    title="Logout"
                                >
                                    <i className="fas fa-sign-out-alt"></i>
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </header>

            {/* Stats Cards */}
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
                    <div className="bg-white rounded-lg shadow p-6">
                        <div className="flex items-center">
                            <div className="bg-blue-100 p-3 rounded-lg">
                                <i className="fas fa-clock text-blue-600"></i>
                            </div>
                            <div className="ml-4">
                                <div className="text-2xl font-semibold text-gray-900">{statusCounts.scheduled}</div>
                                <div className="text-gray-600 text-sm">Scheduled</div>
                            </div>
                        </div>
                    </div>

                    <div className="bg-white rounded-lg shadow p-6">
                        <div className="flex items-center">
                            <div className="bg-green-100 p-3 rounded-lg">
                                <i className="fas fa-check-circle text-green-600"></i>
                            </div>
                            <div className="ml-4">
                                <div className="text-2xl font-semibold text-gray-900">{statusCounts.posted}</div>
                                <div className="text-gray-600 text-sm">Posted</div>
                            </div>
                        </div>
                    </div>

                    <div className="bg-white rounded-lg shadow p-6">
                        <div className="flex items-center">
                            <div className="bg-red-100 p-3 rounded-lg">
                                <i className="fas fa-exclamation-circle text-red-600"></i>
                            </div>
                            <div className="ml-4">
                                <div className="text-2xl font-semibold text-gray-900">{statusCounts.failed}</div>
                                <div className="text-gray-600 text-sm">Failed</div>
                            </div>
                        </div>
                    </div>

                    <div className="bg-white rounded-lg shadow p-6">
                        <div className="flex items-center">
                            <div className="bg-purple-100 p-3 rounded-lg">
                                <i className="fas fa-chart-line text-purple-600"></i>
                            </div>
                            <div className="ml-4">
                                <div className="text-2xl font-semibold text-gray-900">{posts.length}</div>
                                <div className="text-gray-600 text-sm">Total Posts</div>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Main Content */}
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    {/* Calendar/List View */}
                    <div className="lg:col-span-2">
                        {currentView === 'calendar' ? (
                            <CalendarView 
                                posts={posts}
                                onDateSelect={setSelectedDate}
                                selectedDate={selectedDate}
                            />
                        ) : (
                            <div className="bg-white rounded-lg shadow p-6">
                                <h2 className="text-xl font-semibold text-gray-900 mb-6">All Posts</h2>
                                <div className="space-y-4">
                                    {posts.length === 0 ? (
                                        <div className="text-center py-12">
                                            <i className="fas fa-calendar-plus text-gray-400 text-4xl mb-4"></i>
                                            <div className="text-gray-500">No posts scheduled yet</div>
                                            <button
                                                onClick={() => setShowScheduleModal(true)}
                                                className="mt-4 bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
                                            >
                                                Schedule Your First Post
                                            </button>
                                        </div>
                                    ) : (
                                        posts.map(post => (
                                            <PostCard
                                                key={post.id}
                                                post={post}
                                                onEdit={() => {}}
                                                onDelete={handleDeletePost}
                                                onView={() => {}}
                                            />
                                        ))
                                    )}
                                </div>
                            </div>
                        )}
                    </div>

                    {/* Sidebar */}
                    <div className="space-y-6">
                        {/* Selected Date Posts */}
                        {selectedDate && (
                            <div className="bg-white rounded-lg shadow p-6">
                                <h3 className="text-lg font-semibold text-gray-900 mb-4">
                                    {selectedDate.toLocaleDateString('en-US', { 
                                        weekday: 'long', 
                                        year: 'numeric', 
                                        month: 'long', 
                                        day: 'numeric' 
                                    })}
                                </h3>
                                <div className="space-y-3">
                                    {getFilteredPosts().length === 0 ? (
                                        <div className="text-gray-500 text-sm">No posts scheduled for this date</div>
                                    ) : (
                                        getFilteredPosts().map(post => (
                                            <PostCard
                                                key={post.id}
                                                post={post}
                                                onEdit={() => {}}
                                                onDelete={handleDeletePost}
                                                onView={() => {}}
                                            />
                                        ))
                                    )}
                                </div>
                            </div>
                        )}

                        {/* Quick Actions */}
                        <div className="bg-white rounded-lg shadow p-6">
                            <h3 className="text-lg font-semibold text-gray-900 mb-4">Quick Actions</h3>
                            <div className="space-y-2">
                                <button
                                    onClick={() => setShowScheduleModal(true)}
                                    className="w-full text-left p-3 hover:bg-gray-50 rounded-md flex items-center space-x-3"
                                >
                                    <i className="fas fa-plus text-blue-600"></i>
                                    <span>Schedule New Post</span>
                                </button>
                                <button
                                    onClick={loadPosts}
                                    className="w-full text-left p-3 hover:bg-gray-50 rounded-md flex items-center space-x-3"
                                >
                                    <i className="fas fa-sync text-green-600"></i>
                                    <span>Refresh Posts</span>
                                </button>
                            </div>
                        </div>

                        {/* Platform Status */}
                        <div className="bg-white rounded-lg shadow p-6">
                            <h3 className="text-lg font-semibold text-gray-900 mb-4">Platforms</h3>
                            <div className="space-y-3">
                                {Object.entries(PLATFORMS).map(([key, platform]) => (
                                    <div key={key} className="flex items-center justify-between">
                                        <div className="flex items-center space-x-3">
                                            <div
                                                className="w-6 h-6 rounded-full flex items-center justify-center text-white text-sm"
                                                style={{ backgroundColor: platform.color }}
                                            >
                                                <i className={platform.icon}></i>
                                            </div>
                                            <span className="text-sm text-gray-700">{platform.name}</span>
                                        </div>
                                        <span className="text-xs text-gray-500">Connected</span>
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Schedule Post Modal */}
            <SchedulePostModal
                isOpen={showScheduleModal}
                onClose={() => setShowScheduleModal(false)}
                onSave={handleSchedulePost}
            />
        </div>
    );
};

// Main App Component
const App = () => {
    const { user, loading, login, logout } = useAuth();

    if (loading) {
        return (
            <div className="min-h-screen bg-gradient-to-br from-blue-500 to-green-500 flex items-center justify-center">
                <div className="text-center text-white">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-white mx-auto mb-4"></div>
                    <div>Loading Social Media Scheduler...</div>
                </div>
            </div>
        );
    }

    return (
        <div className="App">
            {user ? (
                <Dashboard user={user} onLogout={logout} />
            ) : (
                <LoginPage onLogin={login} />
            )}
        </div>
    );
};

// Add Tailwind CSS
const link = document.createElement('link');
link.href = 'https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css';
link.rel = 'stylesheet';
document.head.appendChild(link);

// Render the app
ReactDOM.render(<App />, document.getElementById('root'));