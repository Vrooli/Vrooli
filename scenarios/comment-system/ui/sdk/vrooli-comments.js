/**
 * Vrooli Comments SDK
 * Universal JavaScript widget for integrating comments into any Vrooli scenario
 * Version: 1.0.0
 */

(function(global) {
    'use strict';

    // SDK Configuration
    const DEFAULT_CONFIG = {
        apiUrl: 'http://localhost:8080',
        theme: 'default',
        allowAnonymous: false,
        showReplies: true,
        maxDepth: 5,
        sortBy: 'newest', // newest, oldest, threaded
        pageSize: 20,
        authToken: null,
        enableMarkdown: true,
        enableRichMedia: false,
        autoRefresh: false,
        refreshInterval: 30000, // 30 seconds
        placeholder: 'Write your comment...',
        loginUrl: null, // URL to redirect for login
        onCommentAdded: null,
        onCommentUpdated: null,
        onCommentDeleted: null,
        onError: null
    };

    // Comment widget class
    class VrooliComments {
        constructor(config = {}) {
            this.config = { ...DEFAULT_CONFIG, ...config };
            this.instances = new Map();
            this.templates = this.createTemplates();
            this.currentUser = null;
            
            // Auto-initialize on DOM ready
            if (document.readyState === 'loading') {
                document.addEventListener('DOMContentLoaded', () => this.autoInitialize());
            } else {
                this.autoInitialize();
            }
        }

        // Auto-initialize widgets with data attributes
        autoInitialize() {
            const widgets = document.querySelectorAll('[data-vrooli-comments]');
            widgets.forEach(widget => {
                const scenarioId = widget.dataset.vrooliComments;
                if (scenarioId && !widget.dataset.initialized) {
                    this.init({
                        scenarioId: scenarioId,
                        container: widget,
                        ...this.parseDataAttributes(widget)
                    });
                    widget.dataset.initialized = 'true';
                }
            });
        }

        // Parse data attributes for configuration
        parseDataAttributes(element) {
            const config = {};
            const dataset = element.dataset;
            
            if (dataset.theme) config.theme = dataset.theme;
            if (dataset.sortBy) config.sortBy = dataset.sortBy;
            if (dataset.pageSize) config.pageSize = parseInt(dataset.pageSize);
            if (dataset.maxDepth) config.maxDepth = parseInt(dataset.maxDepth);
            if (dataset.allowAnonymous !== undefined) config.allowAnonymous = dataset.allowAnonymous === 'true';
            if (dataset.showReplies !== undefined) config.showReplies = dataset.showReplies === 'true';
            if (dataset.enableMarkdown !== undefined) config.enableMarkdown = dataset.enableMarkdown === 'true';
            
            return config;
        }

        // Initialize a comment widget
        init(options = {}) {
            const config = { ...this.config, ...options };
            
            if (!config.scenarioId) {
                this.handleError('scenarioId is required', config);
                return null;
            }

            if (!config.container) {
                this.handleError('container is required', config);
                return null;
            }

            // Get container element
            let container;
            if (typeof config.container === 'string') {
                container = document.querySelector(config.container);
            } else {
                container = config.container;
            }

            if (!container) {
                this.handleError('Container element not found', config);
                return null;
            }

            // Create widget instance
            const widget = new CommentWidget(config, container, this);
            this.instances.set(container, widget);

            return widget;
        }

        // Update configuration for all instances
        configure(newConfig) {
            this.config = { ...this.config, ...newConfig };
            this.instances.forEach(widget => {
                widget.updateConfig(newConfig);
            });
        }

        // Set authentication token
        setAuth(token) {
            this.config.authToken = token;
            this.instances.forEach(widget => {
                widget.setAuth(token);
            });
        }

        // Refresh all comment widgets
        refresh() {
            this.instances.forEach(widget => {
                widget.refresh();
            });
        }

        // Get widget instance for container
        getInstance(container) {
            if (typeof container === 'string') {
                container = document.querySelector(container);
            }
            return this.instances.get(container);
        }

        // Handle errors
        handleError(error, config) {
            console.error('[VrooliComments]', error);
            if (config && config.onError) {
                config.onError(error);
            }
        }

        // Create HTML templates
        createTemplates() {
            return {
                widget: `
                    <div class="vrooli-comments-widget">
                        <div class="vrooli-comments-header">
                            <h3 class="comments-title">Comments</h3>
                            <div class="comments-controls">
                                <select class="sort-select">
                                    <option value="newest">Newest First</option>
                                    <option value="oldest">Oldest First</option>
                                    <option value="threaded">Threaded</option>
                                </select>
                                <button class="refresh-btn" title="Refresh comments">🔄</button>
                            </div>
                        </div>
                        <div class="vrooli-comments-composer"></div>
                        <div class="vrooli-comments-list"></div>
                        <div class="vrooli-comments-pagination"></div>
                    </div>
                `,
                
                composer: `
                    <div class="comment-composer">
                        <div class="composer-form">
                            <textarea class="comment-input" placeholder="{{placeholder}}" rows="3"></textarea>
                            <div class="composer-actions">
                                <div class="composer-options">
                                    <label class="markdown-toggle">
                                        <input type="checkbox" checked> Markdown
                                    </label>
                                </div>
                                <div class="composer-buttons">
                                    <button class="btn-cancel" style="display: none;">Cancel</button>
                                    <button class="btn-submit" disabled>Post Comment</button>
                                </div>
                            </div>
                        </div>
                        <div class="auth-prompt" style="display: none;">
                            <p>Please <a href="#" class="login-link">log in</a> to post comments.</p>
                        </div>
                    </div>
                `,
                
                comment: `
                    <div class="comment" data-comment-id="{{id}}">
                        <div class="comment-avatar">
                            <div class="avatar-placeholder">{{authorInitial}}</div>
                        </div>
                        <div class="comment-content">
                            <div class="comment-header">
                                <span class="comment-author">{{authorName}}</span>
                                <span class="comment-date">{{date}}</span>
                                {{#isOwner}}
                                <div class="comment-actions">
                                    <button class="btn-edit" title="Edit">✏️</button>
                                    <button class="btn-delete" title="Delete">🗑️</button>
                                </div>
                                {{/isOwner}}
                            </div>
                            <div class="comment-body">{{content}}</div>
                            <div class="comment-footer">
                                <button class="btn-reply">Reply</button>
                                <span class="reply-count">{{replyCount}} replies</span>
                            </div>
                            <div class="comment-replies"></div>
                        </div>
                    </div>
                `,
                
                loading: `
                    <div class="loading-spinner">
                        <div class="spinner"></div>
                        <span>Loading comments...</span>
                    </div>
                `,
                
                error: `
                    <div class="error-message">
                        <span class="error-icon">⚠️</span>
                        <span class="error-text">{{message}}</span>
                        <button class="retry-btn">Retry</button>
                    </div>
                `,

                empty: `
                    <div class="empty-state">
                        <div class="empty-icon">💬</div>
                        <h4>No comments yet</h4>
                        <p>Be the first to share your thoughts!</p>
                    </div>
                `
            };
        }
    }

    // Individual comment widget instance
    class CommentWidget {
        constructor(config, container, sdk) {
            this.config = config;
            this.container = container;
            this.sdk = sdk;
            this.comments = [];
            this.currentPage = 0;
            this.totalCount = 0;
            this.loading = false;
            
            this.render();
            this.bindEvents();
            this.loadComments();

            // Auto-refresh if enabled
            if (this.config.autoRefresh) {
                this.startAutoRefresh();
            }
        }

        render() {
            // Apply theme class
            this.container.classList.add('vrooli-comments', `theme-${this.config.theme}`);
            
            // Render widget HTML
            this.container.innerHTML = this.sdk.templates.widget;
            
            // Get element references
            this.elements = {
                header: this.container.querySelector('.vrooli-comments-header'),
                composer: this.container.querySelector('.vrooli-comments-composer'),
                list: this.container.querySelector('.vrooli-comments-list'),
                pagination: this.container.querySelector('.vrooli-comments-pagination'),
                sortSelect: this.container.querySelector('.sort-select'),
                refreshBtn: this.container.querySelector('.refresh-btn')
            };

            // Set initial sort value
            this.elements.sortSelect.value = this.config.sortBy;

            // Render composer
            this.renderComposer();
        }

        renderComposer() {
            if (!this.config.allowAnonymous && !this.config.authToken) {
                // Show auth prompt
                let composerHTML = this.sdk.templates.composer;
                composerHTML = composerHTML.replace('{{placeholder}}', this.config.placeholder);
                this.elements.composer.innerHTML = composerHTML;
                
                this.elements.composer.querySelector('.composer-form').style.display = 'none';
                this.elements.composer.querySelector('.auth-prompt').style.display = 'block';
                
                // Set login link
                const loginLink = this.elements.composer.querySelector('.login-link');
                if (this.config.loginUrl) {
                    loginLink.href = this.config.loginUrl;
                } else {
                    loginLink.addEventListener('click', (e) => {
                        e.preventDefault();
                        alert('Please configure loginUrl in your VrooliComments settings');
                    });
                }
            } else {
                // Show composer form
                let composerHTML = this.sdk.templates.composer;
                composerHTML = composerHTML.replace('{{placeholder}}', this.config.placeholder);
                this.elements.composer.innerHTML = composerHTML;
                
                // Bind composer events
                this.bindComposerEvents();
            }
        }

        bindEvents() {
            // Sort change
            this.elements.sortSelect.addEventListener('change', (e) => {
                this.config.sortBy = e.target.value;
                this.currentPage = 0;
                this.loadComments();
            });

            // Refresh button
            this.elements.refreshBtn.addEventListener('click', () => {
                this.refresh();
            });
        }

        bindComposerEvents() {
            const textarea = this.elements.composer.querySelector('.comment-input');
            const submitBtn = this.elements.composer.querySelector('.btn-submit');
            const markdownToggle = this.elements.composer.querySelector('.markdown-toggle input');

            // Enable/disable submit button based on content
            textarea.addEventListener('input', () => {
                const hasContent = textarea.value.trim().length > 0;
                submitBtn.disabled = !hasContent;
            });

            // Submit comment
            submitBtn.addEventListener('click', () => {
                this.submitComment(textarea.value, markdownToggle.checked);
            });

            // Handle Enter key (with Shift+Enter for new line)
            textarea.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    if (!submitBtn.disabled) {
                        this.submitComment(textarea.value, markdownToggle.checked);
                    }
                }
            });
        }

        async loadComments() {
            this.setLoading(true);

            try {
                const url = new URL(`${this.config.apiUrl}/api/v1/comments/${this.config.scenarioId}`);
                url.searchParams.set('limit', this.config.pageSize.toString());
                url.searchParams.set('offset', (this.currentPage * this.config.pageSize).toString());
                url.searchParams.set('sort', this.config.sortBy);

                const response = await fetch(url.toString());
                
                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                const data = await response.json();
                this.comments = data.comments || [];
                this.totalCount = data.total_count || 0;

                this.renderComments();
                this.renderPagination();

            } catch (error) {
                this.renderError(error.message);
            } finally {
                this.setLoading(false);
            }
        }

        renderComments() {
            if (this.comments.length === 0) {
                this.elements.list.innerHTML = this.sdk.templates.empty;
                return;
            }

            const commentsHTML = this.comments.map(comment => {
                return this.renderComment(comment);
            }).join('');

            this.elements.list.innerHTML = commentsHTML;
            this.bindCommentEvents();
        }

        renderComment(comment, depth = 0) {
            let html = this.sdk.templates.comment;
            
            // Replace template variables
            html = html.replace(/{{id}}/g, comment.id);
            html = html.replace(/{{authorName}}/g, comment.author_name || 'Anonymous');
            html = html.replace(/{{authorInitial}}/g, (comment.author_name || 'A')[0].toUpperCase());
            html = html.replace(/{{date}}/g, this.formatDate(comment.created_at));
            html = html.replace(/{{content}}/g, this.formatContent(comment));
            html = html.replace(/{{replyCount}}/g, comment.reply_count || 0);

            // Handle owner-only actions
            const isOwner = this.isCommentOwner(comment);
            if (isOwner) {
                html = html.replace(/{{#isOwner}}(.*?){{\/isOwner}}/gs, '$1');
            } else {
                html = html.replace(/{{#isOwner}}.*?{{\/isOwner}}/gs, '');
            }

            // Add depth styling
            if (depth > 0) {
                html = `<div class="comment-reply" style="margin-left: ${Math.min(depth * 20, 100)}px;">${html}</div>`;
            }

            return html;
        }

        bindCommentEvents() {
            // Reply buttons
            this.elements.list.querySelectorAll('.btn-reply').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const commentEl = e.target.closest('.comment');
                    const commentId = commentEl.dataset.commentId;
                    this.showReplyForm(commentId, commentEl);
                });
            });

            // Edit buttons
            this.elements.list.querySelectorAll('.btn-edit').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const commentEl = e.target.closest('.comment');
                    const commentId = commentEl.dataset.commentId;
                    this.showEditForm(commentId, commentEl);
                });
            });

            // Delete buttons
            this.elements.list.querySelectorAll('.btn-delete').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const commentEl = e.target.closest('.comment');
                    const commentId = commentEl.dataset.commentId;
                    this.deleteComment(commentId);
                });
            });
        }

        async submitComment(content, isMarkdown, parentId = null) {
            if (!content.trim()) return;

            try {
                const payload = {
                    content: content.trim(),
                    content_type: isMarkdown ? 'markdown' : 'plaintext',
                    author_token: this.config.authToken
                };

                if (parentId) {
                    payload.parent_id = parentId;
                }

                const response = await fetch(`${this.config.apiUrl}/api/v1/comments/${this.config.scenarioId}`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(payload)
                });

                if (!response.ok) {
                    throw new Error(`Failed to post comment: ${response.statusText}`);
                }

                const result = await response.json();
                
                if (result.success) {
                    // Clear form
                    const textarea = this.elements.composer.querySelector('.comment-input');
                    if (textarea) {
                        textarea.value = '';
                    }

                    // Reload comments
                    this.refresh();

                    // Callback
                    if (this.config.onCommentAdded) {
                        this.config.onCommentAdded(result.comment);
                    }
                }

            } catch (error) {
                this.sdk.handleError(error.message, this.config);
            }
        }

        formatContent(comment) {
            if (comment.rendered_content) {
                return comment.rendered_content;
            }
            
            if (comment.content_type === 'markdown' && this.config.enableMarkdown) {
                // Simple markdown rendering (in a real implementation, use a proper markdown library)
                return this.simpleMarkdown(comment.content);
            }
            
            return this.escapeHtml(comment.content);
        }

        simpleMarkdown(text) {
            // Very basic markdown rendering
            return text
                .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
                .replace(/\*(.*?)\*/g, '<em>$1</em>')
                .replace(/`(.*?)`/g, '<code>$1</code>')
                .replace(/\n/g, '<br>');
        }

        escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML.replace(/\n/g, '<br>');
        }

        formatDate(dateString) {
            const date = new Date(dateString);
            const now = new Date();
            const diffMs = now - date;
            const diffMins = Math.floor(diffMs / 60000);
            const diffHours = Math.floor(diffMs / 3600000);
            const diffDays = Math.floor(diffMs / 86400000);

            if (diffMins < 1) return 'just now';
            if (diffMins < 60) return `${diffMins}m ago`;
            if (diffHours < 24) return `${diffHours}h ago`;
            if (diffDays < 7) return `${diffDays}d ago`;
            
            return date.toLocaleDateString();
        }

        isCommentOwner(comment) {
            // In a real implementation, this would check against current user
            return false; // Placeholder
        }

        renderPagination() {
            if (this.totalCount <= this.config.pageSize) {
                this.elements.pagination.innerHTML = '';
                return;
            }

            const totalPages = Math.ceil(this.totalCount / this.config.pageSize);
            const currentPage = this.currentPage;

            let paginationHTML = '<div class="pagination">';
            
            // Previous button
            if (currentPage > 0) {
                paginationHTML += '<button class="page-btn" data-page="prev">Previous</button>';
            }
            
            // Page info
            paginationHTML += `<span class="page-info">Page ${currentPage + 1} of ${totalPages}</span>`;
            
            // Next button
            if (currentPage < totalPages - 1) {
                paginationHTML += '<button class="page-btn" data-page="next">Next</button>';
            }
            
            paginationHTML += '</div>';
            
            this.elements.pagination.innerHTML = paginationHTML;

            // Bind pagination events
            this.elements.pagination.querySelectorAll('.page-btn').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const action = e.target.dataset.page;
                    if (action === 'prev') {
                        this.currentPage = Math.max(0, this.currentPage - 1);
                    } else if (action === 'next') {
                        this.currentPage = Math.min(totalPages - 1, this.currentPage + 1);
                    }
                    this.loadComments();
                });
            });
        }

        renderError(message) {
            let errorHTML = this.sdk.templates.error;
            errorHTML = errorHTML.replace('{{message}}', message);
            this.elements.list.innerHTML = errorHTML;

            // Bind retry button
            const retryBtn = this.elements.list.querySelector('.retry-btn');
            if (retryBtn) {
                retryBtn.addEventListener('click', () => {
                    this.loadComments();
                });
            }
        }

        setLoading(loading) {
            this.loading = loading;
            
            if (loading) {
                this.elements.list.innerHTML = this.sdk.templates.loading;
            }
        }

        refresh() {
            this.currentPage = 0;
            this.loadComments();
        }

        updateConfig(newConfig) {
            this.config = { ...this.config, ...newConfig };
            // Re-render if necessary
        }

        setAuth(token) {
            this.config.authToken = token;
            this.renderComposer();
        }

        startAutoRefresh() {
            this.autoRefreshInterval = setInterval(() => {
                this.loadComments();
            }, this.config.refreshInterval);
        }

        stopAutoRefresh() {
            if (this.autoRefreshInterval) {
                clearInterval(this.autoRefreshInterval);
                this.autoRefreshInterval = null;
            }
        }

        destroy() {
            this.stopAutoRefresh();
            this.container.innerHTML = '';
            this.container.classList.remove('vrooli-comments');
        }
    }

    // CSS injection
    const CSS = `
        .vrooli-comments {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            font-size: 14px;
            line-height: 1.5;
            color: #333;
            background: #fff;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            overflow: hidden;
        }

        .vrooli-comments-widget {
            min-height: 200px;
        }

        .vrooli-comments-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 16px 20px;
            border-bottom: 1px solid #eee;
            background: #fafafa;
        }

        .comments-title {
            margin: 0;
            font-size: 16px;
            font-weight: 600;
        }

        .comments-controls {
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .sort-select {
            padding: 6px 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 12px;
        }

        .refresh-btn {
            background: none;
            border: none;
            cursor: pointer;
            font-size: 14px;
            padding: 6px;
            border-radius: 4px;
            transition: background-color 0.2s;
        }

        .refresh-btn:hover {
            background: #eee;
        }

        .comment-composer {
            padding: 20px;
            border-bottom: 1px solid #eee;
        }

        .composer-form textarea {
            width: 100%;
            min-height: 80px;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 6px;
            resize: vertical;
            font-family: inherit;
            font-size: 14px;
        }

        .composer-actions {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-top: 10px;
        }

        .composer-buttons {
            display: flex;
            gap: 10px;
        }

        .composer-buttons button {
            padding: 8px 16px;
            border: none;
            border-radius: 4px;
            font-size: 14px;
            cursor: pointer;
            transition: all 0.2s;
        }

        .btn-submit {
            background: #007bff;
            color: white;
        }

        .btn-submit:disabled {
            background: #ccc;
            cursor: not-allowed;
        }

        .btn-submit:not(:disabled):hover {
            background: #0056b3;
        }

        .comment {
            display: flex;
            padding: 16px 20px;
            border-bottom: 1px solid #f0f0f0;
        }

        .comment:last-child {
            border-bottom: none;
        }

        .comment-avatar {
            margin-right: 12px;
        }

        .avatar-placeholder {
            width: 32px;
            height: 32px;
            border-radius: 50%;
            background: #007bff;
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 600;
            font-size: 14px;
        }

        .comment-content {
            flex: 1;
        }

        .comment-header {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 6px;
        }

        .comment-author {
            font-weight: 600;
            color: #333;
        }

        .comment-date {
            font-size: 12px;
            color: #666;
        }

        .comment-actions {
            display: flex;
            gap: 5px;
            margin-left: auto;
        }

        .comment-actions button {
            background: none;
            border: none;
            cursor: pointer;
            font-size: 12px;
            padding: 4px;
            border-radius: 3px;
            transition: background-color 0.2s;
        }

        .comment-actions button:hover {
            background: #f0f0f0;
        }

        .comment-body {
            margin-bottom: 8px;
            line-height: 1.5;
        }

        .comment-footer {
            display: flex;
            align-items: center;
            gap: 15px;
        }

        .btn-reply {
            background: none;
            border: none;
            color: #007bff;
            cursor: pointer;
            font-size: 12px;
            font-weight: 500;
        }

        .btn-reply:hover {
            text-decoration: underline;
        }

        .reply-count {
            font-size: 12px;
            color: #666;
        }

        .loading-spinner {
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 40px;
            gap: 10px;
        }

        .spinner {
            width: 20px;
            height: 20px;
            border: 2px solid #f3f3f3;
            border-top: 2px solid #007bff;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        .error-message {
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 40px;
            gap: 10px;
            color: #dc3545;
        }

        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #666;
        }

        .empty-icon {
            font-size: 48px;
            margin-bottom: 16px;
            opacity: 0.5;
        }

        .empty-state h4 {
            margin: 0 0 8px 0;
            font-size: 18px;
            font-weight: 500;
        }

        .pagination {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 15px;
            padding: 16px;
            border-top: 1px solid #eee;
        }

        .page-btn {
            padding: 6px 12px;
            border: 1px solid #ddd;
            background: white;
            border-radius: 4px;
            cursor: pointer;
            font-size: 12px;
        }

        .page-btn:hover {
            background: #f8f9fa;
        }

        .page-info {
            font-size: 12px;
            color: #666;
        }

        .auth-prompt {
            text-align: center;
            padding: 20px;
            color: #666;
        }

        .auth-prompt a {
            color: #007bff;
            text-decoration: none;
        }

        .auth-prompt a:hover {
            text-decoration: underline;
        }

        /* Dark theme */
        .vrooli-comments.theme-dark {
            background: #2d3748;
            color: #e2e8f0;
        }

        .vrooli-comments.theme-dark .vrooli-comments-header {
            background: #4a5568;
            border-bottom-color: #4a5568;
        }

        .vrooli-comments.theme-dark .comment {
            border-bottom-color: #4a5568;
        }

        .vrooli-comments.theme-dark .comment-composer {
            border-bottom-color: #4a5568;
        }

        .vrooli-comments.theme-dark textarea,
        .vrooli-comments.theme-dark .sort-select {
            background: #4a5568;
            border-color: #4a5568;
            color: #e2e8f0;
        }
    `;

    // Inject CSS
    function injectCSS() {
        if (!document.querySelector('#vrooli-comments-css')) {
            const style = document.createElement('style');
            style.id = 'vrooli-comments-css';
            style.textContent = CSS;
            document.head.appendChild(style);
        }
    }

    // Initialize CSS injection
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', injectCSS);
    } else {
        injectCSS();
    }

    // Create global instance
    const vrooliComments = new VrooliComments();

    // Export API
    const API = {
        init: (options) => vrooliComments.init(options),
        configure: (config) => vrooliComments.configure(config),
        setAuth: (token) => vrooliComments.setAuth(token),
        refresh: () => vrooliComments.refresh(),
        getInstance: (container) => vrooliComments.getInstance(container),
        
        // Utility methods
        addComment: (scenarioId, content) => {
            // Helper method to add comment programmatically
            const instances = Array.from(vrooliComments.instances.values())
                .filter(widget => widget.config.scenarioId === scenarioId);
            
            if (instances.length > 0) {
                instances[0].submitComment(content, true);
            }
        }
    };

    // Export for different environments
    if (typeof module !== 'undefined' && module.exports) {
        module.exports = API;
    } else if (typeof define === 'function' && define.amd) {
        define(function() { return API; });
    } else {
        global.VrooliComments = API;
    }

})(typeof window !== 'undefined' ? window : this);