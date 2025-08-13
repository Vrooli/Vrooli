/**
 * Content Script for {{APP_NAME}} Extension
 * 
 * This script runs in the context of web pages.
 * It can read and modify the DOM, but runs in an isolated environment.
 * 
 * AGENT NOTE: This file is OPTIONAL. Only include if the app needs to:
 * - Extract data from web pages
 * - Modify page appearance
 * - Inject UI elements into pages
 * - Monitor user interactions with pages
 * 
 * To enable, uncomment the content_scripts section in manifest.json
 */

// ============================================
// Configuration
// ============================================

const CONFIG = {
    APP_NAME: '{{APP_NAME}}',
    HIGHLIGHT_COLOR: '#3b82f6',
    OVERLAY_Z_INDEX: 999999,
};

// ============================================
// State Management
// ============================================

let isActive = false;
let extractedData: any[] = [];

// ============================================
// Message Handling (Communication with Background)
// ============================================

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    console.log('Content script received message:', message);
    
    switch (message.action) {
        case 'ACTIVATE':
            activate();
            sendResponse({ success: true });
            break;
            
        case 'DEACTIVATE':
            deactivate();
            sendResponse({ success: true });
            break;
            
        case 'EXTRACT_DATA':
            const data = extractPageData();
            sendResponse({ success: true, data });
            break;
            
        case 'HIGHLIGHT_ELEMENTS':
            highlightElements(message.selector);
            sendResponse({ success: true });
            break;
            
        case 'INJECT_UI':
            injectOverlay();
            sendResponse({ success: true });
            break;
            
        case 'GET_SELECTION':
            const selection = window.getSelection()?.toString();
            sendResponse({ success: true, selection });
            break;
            
        default:
            sendResponse({ success: false, error: 'Unknown action' });
    }
    
    return true; // Keep message channel open for async response
});

// ============================================
// Core Functionality
// ============================================

function activate() {
    if (isActive) return;
    isActive = true;
    
    console.log(`${CONFIG.APP_NAME} content script activated`);
    
    // Add page listeners
    document.addEventListener('click', handleClick, true);
    document.addEventListener('mouseenter', handleHover, true);
    document.addEventListener('contextmenu', handleRightClick, true);
    
    // Add visual indicator that extension is active
    addActiveIndicator();
}

function deactivate() {
    if (!isActive) return;
    isActive = false;
    
    console.log(`${CONFIG.APP_NAME} content script deactivated`);
    
    // Remove listeners
    document.removeEventListener('click', handleClick, true);
    document.removeEventListener('mouseenter', handleHover, true);
    document.removeEventListener('contextmenu', handleRightClick, true);
    
    // Clean up UI elements
    removeActiveIndicator();
    removeHighlights();
    removeOverlay();
}

// ============================================
// Data Extraction
// ============================================

function extractPageData() {
    const data = {
        url: window.location.href,
        title: document.title,
        description: getMetaContent('description'),
        keywords: getMetaContent('keywords'),
        author: getMetaContent('author'),
        publishedTime: getMetaContent('article:published_time'),
        modifiedTime: getMetaContent('article:modified_time'),
        
        // Open Graph data
        ogTitle: getMetaContent('og:title'),
        ogDescription: getMetaContent('og:description'),
        ogImage: getMetaContent('og:image'),
        ogType: getMetaContent('og:type'),
        
        // Twitter Card data
        twitterTitle: getMetaContent('twitter:title'),
        twitterDescription: getMetaContent('twitter:description'),
        twitterImage: getMetaContent('twitter:image'),
        
        // Structured data
        jsonLd: extractJsonLd(),
        
        // Custom extraction based on your app's needs
        emails: extractEmails(),
        phoneNumbers: extractPhoneNumbers(),
        prices: extractPrices(),
        
        // Page statistics
        wordCount: document.body?.innerText.split(/\\s+/).length || 0,
        imageCount: document.images.length,
        linkCount: document.links.length,
        
        // Selected text if any
        selectedText: window.getSelection()?.toString(),
    };
    
    return data;
}

function getMetaContent(name: string): string | null {
    const element = document.querySelector(
        `meta[name="${name}"], meta[property="${name}"]`
    ) as HTMLMetaElement;
    return element?.content || null;
}

function extractJsonLd() {
    const scripts = document.querySelectorAll('script[type="application/ld+json"]');
    const data: any[] = [];
    
    scripts.forEach(script => {
        try {
            const json = JSON.parse(script.textContent || '{}');
            data.push(json);
        } catch (e) {
            console.error('Failed to parse JSON-LD:', e);
        }
    });
    
    return data;
}

function extractEmails(): string[] {
    const emailRegex = /\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b/g;
    const text = document.body?.innerText || '';
    const matches = text.match(emailRegex) || [];
    return [...new Set(matches)]; // Remove duplicates
}

function extractPhoneNumbers(): string[] {
    const phoneRegex = /\\b\\d{3}[-.]?\\d{3}[-.]?\\d{4}\\b/g;
    const text = document.body?.innerText || '';
    const matches = text.match(phoneRegex) || [];
    return [...new Set(matches)];
}

function extractPrices(): string[] {
    const priceRegex = /\\$\\d+(?:,\\d{3})*(?:\\.\\d{2})?/g;
    const text = document.body?.innerText || '';
    const matches = text.match(priceRegex) || [];
    return [...new Set(matches)];
}

// ============================================
// Element Highlighting
// ============================================

function highlightElements(selector: string) {
    try {
        removeHighlights(); // Clear existing highlights
        
        const elements = document.querySelectorAll(selector);
        elements.forEach(element => {
            const el = element as HTMLElement;
            el.dataset.originalBorder = el.style.border;
            el.style.border = `2px solid ${CONFIG.HIGHLIGHT_COLOR}`;
            el.style.boxShadow = `0 0 10px ${CONFIG.HIGHLIGHT_COLOR}40`;
            el.classList.add('{{app-name}}-highlighted');
        });
        
        console.log(`Highlighted ${elements.length} elements`);
    } catch (error) {
        console.error('Failed to highlight elements:', error);
    }
}

function removeHighlights() {
    const highlighted = document.querySelectorAll('.{{app-name}}-highlighted');
    highlighted.forEach(element => {
        const el = element as HTMLElement;
        el.style.border = el.dataset.originalBorder || '';
        el.style.boxShadow = '';
        delete el.dataset.originalBorder;
        el.classList.remove('{{app-name}}-highlighted');
    });
}

// ============================================
// UI Injection
// ============================================

function injectOverlay() {
    if (document.getElementById('{{app-name}}-overlay')) return;
    
    const overlay = document.createElement('div');
    overlay.id = '{{app-name}}-overlay';
    overlay.innerHTML = `
        <div class="{{app-name}}-panel">
            <header>
                <h3>{{APP_NAME}}</h3>
                <button class="close-btn">&times;</button>
            </header>
            <main>
                <p>Page URL: ${window.location.href}</p>
                <p>Extracted ${extractedData.length} items</p>
                <button class="action-btn">{{PRIMARY_ACTION}}</button>
            </main>
        </div>
    `;
    
    // Add styles
    const style = document.createElement('style');
    style.textContent = `
        #{{app-name}}-overlay {
            position: fixed;
            top: 20px;
            right: 20px;
            width: 300px;
            background: white;
            border: 1px solid #ddd;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
            z-index: ${CONFIG.OVERLAY_Z_INDEX};
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
        
        .{{app-name}}-panel header {
            padding: 12px 16px;
            background: #f5f5f5;
            border-bottom: 1px solid #ddd;
            border-radius: 8px 8px 0 0;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        
        .{{app-name}}-panel h3 {
            margin: 0;
            font-size: 14px;
            font-weight: 600;
        }
        
        .{{app-name}}-panel .close-btn {
            background: none;
            border: none;
            font-size: 20px;
            cursor: pointer;
            padding: 0;
            width: 24px;
            height: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .{{app-name}}-panel main {
            padding: 16px;
        }
        
        .{{app-name}}-panel p {
            margin: 0 0 8px 0;
            font-size: 13px;
            color: #333;
        }
        
        .{{app-name}}-panel .action-btn {
            width: 100%;
            padding: 8px;
            background: ${CONFIG.HIGHLIGHT_COLOR};
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 13px;
            font-weight: 500;
            margin-top: 8px;
        }
        
        .{{app-name}}-panel .action-btn:hover {
            opacity: 0.9;
        }
    `;
    
    document.head.appendChild(style);
    document.body.appendChild(overlay);
    
    // Add event listeners
    overlay.querySelector('.close-btn')?.addEventListener('click', removeOverlay);
    overlay.querySelector('.action-btn')?.addEventListener('click', handlePrimaryAction);
}

function removeOverlay() {
    const overlay = document.getElementById('{{app-name}}-overlay');
    overlay?.remove();
}

function addActiveIndicator() {
    if (document.getElementById('{{app-name}}-indicator')) return;
    
    const indicator = document.createElement('div');
    indicator.id = '{{app-name}}-indicator';
    indicator.textContent = '{{APP_NAME}} Active';
    
    const style = document.createElement('style');
    style.textContent = `
        #{{app-name}}-indicator {
            position: fixed;
            bottom: 20px;
            left: 20px;
            padding: 8px 12px;
            background: ${CONFIG.HIGHLIGHT_COLOR};
            color: white;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 500;
            z-index: ${CONFIG.OVERLAY_Z_INDEX};
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
    `;
    
    document.head.appendChild(style);
    document.body.appendChild(indicator);
}

function removeActiveIndicator() {
    const indicator = document.getElementById('{{app-name}}-indicator');
    indicator?.remove();
}

// ============================================
// Event Handlers
// ============================================

function handleClick(event: MouseEvent) {
    if (!isActive) return;
    
    const target = event.target as HTMLElement;
    
    // Skip if clicking on our own UI elements
    if (target.closest('#{{app-name}}-overlay')) return;
    
    // Example: Capture clicked element's text
    if (event.altKey) {
        event.preventDefault();
        event.stopPropagation();
        
        const text = target.innerText || target.textContent || '';
        if (text) {
            extractedData.push({
                type: 'click',
                text: text.substring(0, 100),
                tagName: target.tagName,
                timestamp: Date.now(),
            });
            
            // Send to background
            chrome.runtime.sendMessage({
                type: 'ELEMENT_CAPTURED',
                data: extractedData[extractedData.length - 1],
            });
            
            // Visual feedback
            target.style.backgroundColor = `${CONFIG.HIGHLIGHT_COLOR}20`;
            setTimeout(() => {
                target.style.backgroundColor = '';
            }, 500);
        }
    }
}

function handleHover(event: MouseEvent) {
    if (!isActive) return;
    
    // Example: Show tooltip on hover
    // Implement hover behavior based on app needs
}

function handleRightClick(event: MouseEvent) {
    if (!isActive) return;
    
    // Store context for context menu actions
    const target = event.target as HTMLElement;
    chrome.storage.local.set({
        contextMenuTarget: {
            text: target.innerText?.substring(0, 100),
            href: (target as HTMLAnchorElement).href || null,
            src: (target as HTMLImageElement).src || null,
        },
    });
}

async function handlePrimaryAction() {
    const data = extractPageData();
    
    // Send to background for processing
    chrome.runtime.sendMessage({
        type: '{{PRIMARY_ACTION_MESSAGE}}',
        data: data,
    }, (response) => {
        if (response?.success) {
            // Show success feedback
            showNotification('{{PRIMARY_ACTION}} completed!');
        } else {
            showNotification('Action failed: ' + (response?.error || 'Unknown error'));
        }
    });
}

// ============================================
// Utility Functions
// ============================================

function showNotification(message: string) {
    const notification = document.createElement('div');
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        left: 50%;
        transform: translateX(-50%);
        padding: 12px 20px;
        background: #333;
        color: white;
        border-radius: 4px;
        font-size: 14px;
        z-index: ${CONFIG.OVERLAY_Z_INDEX + 1};
        animation: slideDown 0.3s ease;
    `;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.remove();
    }, 3000);
}

// ============================================
// Mutation Observer (Monitor DOM changes)
// ============================================

const observer = new MutationObserver((mutations) => {
    if (!isActive) return;
    
    // Example: Detect when new content is added
    mutations.forEach(mutation => {
        if (mutation.type === 'childList' && mutation.addedNodes.length > 0) {
            // Process new content
            console.log('New content detected');
        }
    });
});

// Start observing when needed
function startObserving() {
    observer.observe(document.body, {
        childList: true,
        subtree: true,
        attributes: false,
    });
}

function stopObserving() {
    observer.disconnect();
}

// ============================================
// Initialization
// ============================================

// Check if should auto-activate based on URL or settings
chrome.storage.sync.get(['autoActivate', 'activeDomains'], (data) => {
    const currentDomain = window.location.hostname;
    
    if (data.autoActivate || data.activeDomains?.includes(currentDomain)) {
        activate();
    }
});

// Log that content script is loaded
console.log(`${CONFIG.APP_NAME} content script loaded on ${window.location.href}`);