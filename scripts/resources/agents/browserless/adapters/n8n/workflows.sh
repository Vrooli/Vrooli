#!/usr/bin/env bash

#######################################
# N8N Integration for Browserless
# 
# This module provides functionality for executing N8N workflows
# via browser automation using the browserless service.
#
# Dependencies:
#   - browserless::is_healthy from core.sh  
#   - browserless::ensure_test_output_dir from api.sh
#   - log:: functions from common logging
#   - jq for JSON processing
#######################################

# The browserless API functionality needs to be available
# We source from the browserless lib directory
WORKFLOWS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BROWSERLESS_DIR="$(dirname "$(dirname "$WORKFLOWS_DIR")")"
source "${BROWSERLESS_DIR}/lib/common.sh"

#######################################
# Execute n8n workflow via browser automation with session persistence
# Arguments:
#   $1 - Workflow ID (required)
#   $2 - N8N base URL (optional, default: http://localhost:5678)
#   $3 - Timeout in milliseconds (optional, default: 60000)
#   $4 - Input data (optional, JSON string or @file or env var)
#   $5 - Use persistent session (optional, true/false, default: true)
#   $6 - Session ID for persistence (optional, auto-generated if not provided)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::execute_n8n_workflow() {
    local workflow_id="${1:?Workflow ID required}"
    local n8n_url="${2:-http://localhost:5678}"
    local timeout="${3:-60000}"
    local input_data="${4:-}"
    local use_persistent_session="${5:-true}"
    local session_id="${6:-n8n_workflow_session_$(date +%s)}"
    
    log::header "🚀 Executing N8N Workflow via Browser Automation"
    
    if ! browserless::is_healthy; then
        log::error "${MSG_NOT_HEALTHY}"
        return 1
    fi
    
    # Build workflow URL
    local workflow_url
    if [[ "$workflow_id" == http* ]]; then
        workflow_url="$workflow_id"
    else
        workflow_url="${n8n_url}/workflow/${workflow_id}"
    fi
    
    # Process input data
    local processed_input=""
    local input_description="None"
    
    if [[ -n "$input_data" ]]; then
        # Check if input_data starts with @ (file reference)
        if [[ "$input_data" == @* ]]; then
            local input_file="${input_data#@}"
            if [[ -f "$input_file" ]]; then
                processed_input=$(cat "$input_file")
                input_description="From file: $input_file"
            else
                log::error "Input file not found: $input_file"
                return 1
            fi
        else
            # Direct JSON input or environment variable
            if [[ "$input_data" == "\$"* ]]; then
                # Environment variable reference
                local env_var="${input_data#\$}"
                processed_input="${!env_var}"
                input_description="From environment variable: $env_var"
            else
                # Direct JSON input
                processed_input="$input_data"
                input_description="Direct JSON input"
            fi
        fi
        
        # Check for WORKFLOW_INPUT environment variable as fallback
        if [[ -z "$processed_input" && -n "${WORKFLOW_INPUT:-}" ]]; then
            processed_input="$WORKFLOW_INPUT"
            input_description="From WORKFLOW_INPUT environment variable"
        fi
        
        # Validate JSON if input provided
        if [[ -n "$processed_input" ]]; then
            if ! echo "$processed_input" | jq . >/dev/null 2>&1; then
                log::error "Invalid JSON input data"
                log::debug "Input: $processed_input"
                return 1
            fi
        fi
    fi
    
    log::info "Workflow ID: $workflow_id"
    log::info "Workflow URL: $workflow_url"
    log::info "Timeout: ${timeout}ms"
    log::info "Input Data: $input_description"
    log::info "Persistent Session: $use_persistent_session"
    log::info "Session ID: $session_id"
    if [[ -n "$processed_input" ]]; then
        log::debug "Input JSON: $(echo "$processed_input" | jq -c . 2>/dev/null || echo "$processed_input")"
    fi
    echo
    
    # Generate function code for workflow execution with enhanced monitoring
    local function_code
    read -r -d '' function_code << 'EOF' || true
export default async ({ page }) => {
  const executionData = {
    workflowId: '%WORKFLOW_ID%',
    workflowUrl: '%WORKFLOW_URL%',
    sessionId: '%SESSION_ID%',
    usePersistentSession: %USE_PERSISTENT_SESSION%,
    consoleLogs: [],
    pageErrors: [],
    networkErrors: [],
    executionStatus: {
      started: false,
      completed: false,
      failed: false,
      running: false
    },
    screenshots: [],
    finalState: null,
    startTime: new Date().toISOString(),
    success: false,
    enhancedMonitoring: true
  };
  
  const takeScreenshot = async (label, highQuality = false) => {
    try {
      const buffer = await page.screenshot({ 
        fullPage: false, 
        quality: highQuality ? 90 : 70,
        type: highQuality ? 'jpeg' : 'png'
      });
      
      // Store screenshot with base64 data temporarily (will be processed by bash later)
      const timestamp = new Date().toISOString();
      const base64 = buffer.toString('base64');
      
      // Store base64 temporarily - bash script will extract and save to file
      executionData.screenshots.push({
        label,
        timestamp,
        data: base64,  // This will be removed by bash processing
        size: buffer.length,
        type: highQuality ? 'jpeg' : 'png',
        highQuality
      });
      // Emit a marker for bash to optionally extract
      console.log(`__BL_SCREENSHOT__ ${label} ${timestamp} ${buffer.length}`);
      
      console.log(`Screenshot captured: ${label} (${buffer.length} bytes)`);
    } catch (e) {
      console.log('Screenshot failed:', e.message);
    }
  };
  
  const sleep = (ms) => new Promise(r => setTimeout(r, ms));
  
    const waitForStableUI = async (totalMs = 5000, stepMs = 400, requiredStable = 2) => {
      const deadline = Date.now() + totalMs;
      let prev = null;
      let stableCount = 0;
      while (Date.now() < deadline) {
        const sig = await page.evaluate(() => {
          const btn = document.querySelector('[data-test-id="execute-workflow-button"]');
          const dropdown = document.querySelector('.action-dropdown-container button');
          const nodes = document.querySelectorAll('*').length;
          const bodyLen = document.body ? document.body.innerText.length : 0;
          return { btn: !!btn, dropdown: !!dropdown, nodes, bodyLen };
        }).catch(() => null);
        if (!sig) { await sleep(stepMs); continue; }
        const key = JSON.stringify(sig);
        if (key === prev) {
          stableCount += 1;
          if (stableCount >= requiredStable) return true;
        } else {
          prev = key;
          stableCount = 0;
        }
        await sleep(stepMs);
      }
      return false;
    };

    // Selectors profile embedded at runtime from Bash (configurable)
    const SELECTORS = %SELECTORS%;

    // Wait for a selector to be visible and scroll into view
    const waitForSelectorVisible = async (selector, timeout = 10000) => {
      try {
        await page.waitForSelector(selector, { timeout, visible: true });
        await page.evaluate((sel) => {
          const el = document.querySelector(sel);
          if (el) el.scrollIntoView({ block: 'center', inline: 'center' });
        }, selector);
        return true;
      } catch (e) {
        return false;
      }
    };
      
  // Function to detect and parse snack messages
  const detectSnackMessages = async () => {
    const snackMessages = [];
    
    try {
      const notifications = await page.$$('.el-notification');
      
      for (const notification of notifications) {
        try {
          const messageData = await page.evaluate(el => {
            const titleEl = el.querySelector('.el-notification__title');
            const contentEl = el.querySelector('.el-notification__content p');
            const iconEl = el.querySelector('.el-notification__icon');
            
            let type = 'info';
            if (iconEl) {
              if (iconEl.classList.contains('el-notification--error')) {
                type = 'error';
              } else if (iconEl.classList.contains('el-notification--warning')) {
                type = 'warning';
              } else if (iconEl.classList.contains('el-notification--success')) {
                type = 'success';
              }
            }
            
            return {
              title: titleEl ? titleEl.textContent.trim() : '',
              content: contentEl ? contentEl.textContent.trim() : '',
              type: type,
              timestamp: new Date().toISOString()
            };
          }, notification);
          
          if (messageData.title || messageData.content) {
            snackMessages.push(messageData);
            console.log(`Detected snack message [${messageData.type.toUpperCase()}]: ${messageData.title} - ${messageData.content}`);
          }
        } catch (err) {
          console.log('Error parsing notification:', err.message);
        }
      }
    } catch (err) {
      console.log('Error detecting snack messages:', err.message);
    }
    
    return snackMessages;
  };
  
  // Function to clear all existing snack messages
  const clearSnackMessages = async () => {
    console.log('Clearing existing snack messages...');
    let clearedCount = 0;
    
    try {
      // Find all close buttons in notifications
      const closeButtons = await page.$$('.el-notification__closeBtn');
      
      for (const closeBtn of closeButtons) {
        try {
          await closeBtn.click();
          clearedCount++;
          await new Promise(resolve => setTimeout(resolve, 200)); // Brief delay between clicks
        } catch (err) {
          console.log('Error clicking close button:', err.message);
        }
      }
      
      // Also try clicking notification areas to dismiss them
      const notifications = await page.$$('.el-notification');
      for (const notification of notifications) {
        try {
          // Check if notification is still visible
          const isVisible = await page.evaluate(el => {
            const rect = el.getBoundingClientRect();
            return rect.width > 0 && rect.height > 0;
          }, notification);
          
          if (isVisible) {
            // Try pressing Escape key to dismiss
            await page.keyboard.press('Escape');
            await new Promise(resolve => setTimeout(resolve, 200));
          }
        } catch (err) {
          // Ignore errors, notification might have been removed
        }
      }
      
      console.log(`Cleared ${clearedCount} snack messages`);
      
      // Wait a moment for notifications to fully disappear
      await new Promise(resolve => setTimeout(resolve, 500));
      
    } catch (err) {
      console.log('Error clearing snack messages:', err.message);
    }
    
    return clearedCount;
  };
  
  const safeHas = async (selector, timeout = 1500, attempts = 3) => {
    for (let i = 0; i < attempts; i++) {
      try {
        const el = await Promise.race([
          page.$(selector),
          sleep(timeout).then(() => null)
        ]);
        return el !== null;
      } catch (e) {
        if (String(e.message || '').includes('detached Frame')) {
          await sleep(300);
          continue;
        }
        throw e;
      }
    }
    return false;
  };
  
  const safeWaitForSelector = async (selector, timeout = 2000, attempts = 3) => {
    for (let i = 0; i < attempts; i++) {
      try {
        await page.waitForSelector(selector, { timeout });
        return true;
      } catch (e) {
        if (String(e.message || '').includes('detached Frame')) {
          await sleep(300);
          continue;
        }
        return false;
      }
    }
    return false;
  };
  
  // Enhanced execution state monitoring
  const checkExecutionState = async () => {
    try {
      const executionInfo = await page.evaluate(() => {
        // Check various UI indicators for execution state
        const executeButton = document.querySelector('[data-test-id="execute-workflow-button"]');
        const cancelButton = document.querySelector('[data-test-id="cancel-execution-button"]');
        const spinner = document.querySelector('.el-loading-spinner, .spinner, [class*="loading"], [class*="spinner"]');
        const successIndicator = document.querySelector('[class*="success"], [data-test-id*="success"]');
        const errorIndicator = document.querySelector('[class*="error"], [data-test-id*="error"]');
        const executionCount = document.querySelector('[data-test-id="execution-count"]');
        
        // Check for execution status text
        const statusElements = Array.from(document.querySelectorAll('*')).filter(el => {
          const text = el.textContent || '';
          return /execution.*complete|execution.*finished|execution.*success|execution.*fail|execution.*error|workflow.*complete|workflow.*finished/i.test(text);
        });
        
        // Check for workflow result panels or output displays
        const resultPanels = document.querySelectorAll('[data-test-id*="execution-result"], [data-test-id*="output"], .execution-data, .node-output, .workflow-result');
        
        return {
          executeButtonExists: !!executeButton,
          executeButtonText: executeButton ? executeButton.textContent : null,
          executeButtonDisabled: executeButton ? executeButton.disabled : null,
          cancelButtonExists: !!cancelButton,
          spinnerExists: !!spinner,
          successIndicatorExists: !!successIndicator,
          errorIndicatorExists: !!errorIndicator,
          executionCountText: executionCount ? executionCount.textContent : null,
          statusElementsCount: statusElements.length,
          statusTexts: statusElements.slice(0, 3).map(el => el.textContent.trim().substring(0, 100)),
          resultPanelsCount: resultPanels.length,
          timestamp: new Date().toISOString()
        };
      });
      
      return executionInfo;
    } catch (e) {
      console.log('Error checking execution state:', e.message);
      return null;
    }
  };
  
  try {
    // Set up console log capture
    page.on('console', msg => {
      executionData.consoleLogs.push({
        type: msg.type(),
        text: msg.text(),
        timestamp: new Date().toISOString()
      });
    });
    
    // Set up page error capture
    page.on('pageerror', err => {
      executionData.pageErrors.push({
        message: err.message,
        stack: err.stack,
        timestamp: new Date().toISOString()
      });
    });
    
    // Set up network error capture
    page.on('requestfailed', req => {
      executionData.networkErrors.push({
        url: req.url(),
        method: req.method(),
        failure: req.failure()?.errorText || 'Unknown error',
        timestamp: new Date().toISOString()
      });
    });
    
    console.log('Setting page default timeouts...');
    try { page.setDefaultNavigationTimeout(%TIMEOUT%); } catch (e) { console.log('setDefaultNavigationTimeout failed:', e.message); }
    try { page.setDefaultTimeout(%TIMEOUT%); } catch (e) { console.log('setDefaultTimeout failed:', e.message); }
    console.log('Navigating to workflow:', '%WORKFLOW_URL%');
    
    // Navigate to workflow with better error handling
    try {
      await page.goto('%WORKFLOW_URL%', {
        waitUntil: 'domcontentloaded',
        timeout: %TIMEOUT%
      });
      await safeWaitForSelector('body', 5000);
      await takeScreenshot('after-goto-workflow');
    } catch (navError) {
      console.log('Navigation error:', navError.message);
      await takeScreenshot('goto-error');
      // Continue anyway, might be a redirect to auth
    }
    
    // Wait a moment for any redirects and console errors to appear
    await sleep(3000);
    await takeScreenshot('post-redirect-wait');
    
    // Enhanced authentication detection and handling
    const currentUrl = page.url();
    console.log('Current URL after navigation:', currentUrl);
    
    // Check for authentication requirements using multiple patterns
    console.log('Checking authentication requirements...');
    
    // URL-based checks
    const urlCheck = currentUrl.includes('/signin') || currentUrl.includes('/login') || currentUrl.includes('/auth');
    
    // Console error-based checks (check for 401 Unauthorized errors)
    let has401Error = false;
    for (const log of executionData.consoleLogs) {
      if (log.type === 'error' && log.text.includes('401') && log.text.includes('Unauthorized')) {
        has401Error = true;
        console.log('Found 401 Unauthorized error in console logs');
        break;
      }
    }
    
    // Element-based checks with retry handling to mitigate detached frames
    let emailFieldCheck = false;
    try {
      emailFieldCheck = await page.evaluate(() => !!document.querySelector('input[type="email"], [data-test-id="user-email"]'));
    } catch (e) { emailFieldCheck = false; }
    let inputEmailCheck = emailFieldCheck;
    let signinClassCheck = false;
    let loginFormCheck = false;
    let passwordFieldCheck = false;
    let executeButtonMissing = false;
    
    try {
      signinClassCheck = await page.evaluate(() => !!document.querySelector('.signin'));
    } catch (e) { signinClassCheck = false; }
    
    try {
      loginFormCheck = await page.evaluate(() => !!document.querySelector('.login-form'));
    } catch (e) { loginFormCheck = false; }
    
    try {
      passwordFieldCheck = await page.evaluate(() => !!document.querySelector('input[type="password"]'));
    } catch (e) { passwordFieldCheck = false; }
    
    try {
      executeButtonMissing = await page.evaluate(() => !document.querySelector('[data-test-id="execute-workflow-button"]'));
    } catch (e) { executeButtonMissing = false; }
    
    console.log('Auth detection results:');
    console.log('  URL contains auth path:', urlCheck);
    console.log('  Has 401 error:', has401Error);
    console.log('  Has email test-id field:', emailFieldCheck);
    console.log('  Has email input:', inputEmailCheck);
    console.log('  Has signin class:', signinClassCheck);
    console.log('  Has login form:', loginFormCheck);
    console.log('  Has password field:', passwordFieldCheck);
    console.log('  Execute button missing:', executeButtonMissing);
    
    let needsAuth = urlCheck || has401Error || emailFieldCheck || inputEmailCheck || 
                     signinClassCheck || loginFormCheck || passwordFieldCheck || executeButtonMissing;
    console.log('Authentication needed:', needsAuth);
    
    if (needsAuth) {
      console.log('Authentication required - attempting login');
      
      // If we detected 401 but not on login page, navigate to login
      if (has401Error && !urlCheck) {
        console.log('401 error detected but not on login page, navigating to signin');
        const loginUrl = '%WORKFLOW_URL%'.replace('/workflow/', '/signin');
        console.log('Login URL:', loginUrl);
        
        try {
          await page.goto(loginUrl, {
            waitUntil: 'networkidle2',
            timeout: 15000
          });
          console.log('Successfully navigated to login page');
          await takeScreenshot('login-page');
        } catch (navError) {
          console.log('Failed to navigate to login page:', navError.message);
          // Try alternative login URL
          const altLoginUrl = '%WORKFLOW_URL%'.replace('/workflow/', '/login');
          try {
            await page.goto(altLoginUrl, {
              waitUntil: 'networkidle2', 
              timeout: 15000
            });
            console.log('Successfully navigated to alternative login page');
            await takeScreenshot('alt-login-page');
          } catch (altNavError) {
            console.log('Failed to navigate to alternative login page:', altNavError.message);
            await takeScreenshot('login-nav-failed');
            throw new Error('Cannot access login page');
          }
        }
        
        // Wait for login page to load
        await new Promise(resolve => setTimeout(resolve, 2000));
        await takeScreenshot('login-page-waited');
      }
      
      // Get web UI credentials from Vrooli's secrets system
      const email = '%N8N_EMAIL%';
      const password = '%N8N_PASSWORD%';
      
      console.log('Checking web authentication credentials');
      console.log('Email/Password available:', email && password && email !== '%N8N_EMAIL_PLACEHOLDER%' && password !== '%N8N_PASSWORD_PLACEHOLDER%');
      
      if (email && password && email !== '%N8N_EMAIL_PLACEHOLDER%' && password !== '%N8N_PASSWORD_PLACEHOLDER%') {
        console.log('Using web UI authentication with credentials from Vrooli secrets');
        console.log('Email:', email);
        console.log('Password length:', password.length);
      } else {
        console.log('No web credentials available - attempting unauthenticated access');
        console.log('(This is normal for development n8n instances)');
        needsAuth = false; // Let it try without auth first
      }
      
      // Proceed with email/password UI login if we have valid credentials
      if (email && password && email !== '%N8N_EMAIL_PLACEHOLDER%' && password !== '%N8N_PASSWORD_PLACEHOLDER%' && email.length > 0 && password.length > 0) {
        // Try multiple selectors for email/username field
        const emailSelectors = [
          '[data-test-id="user-email"]',
          'input[type="email"]',
          'input[name="email"]',
          'input[placeholder*="email" i]',
          'input[placeholder*="username" i]',
          '#email',
          '#username'
        ];
        
        const passwordSelectors = [
          '[data-test-id="user-password"]',
          'input[type="password"]',
          'input[name="password"]',
          '#password'
        ];
        
        let emailField = null;
        let passwordField = null;
        
        // Find email field
        // Prefer evaluation to avoid frame detaches
        for (const selector of emailSelectors) {
          const found = await page.evaluate((sel) => !!document.querySelector(sel), selector).catch(() => false);
          if (found) {
            emailField = selector;
            console.log(`Found email field: ${selector}`);
            break;
          }
        }
        
        // Find password field
        for (const selector of passwordSelectors) {
          const found = await page.evaluate((sel) => !!document.querySelector(sel), selector).catch(() => false);
          if (found) {
            passwordField = selector;
            console.log(`Found password field: ${selector}`);
            break;
          }
        }
        
        if (emailField && passwordField) {
          // Clear and fill login form
          // Fill via evaluate to reduce selector race
          await page.evaluate(({ emailField, email }) => {
            const el = document.querySelector(emailField);
            if (el) { (el).value = ''; (el).dispatchEvent(new Event('input', { bubbles: true })); (el).focus(); }
          }, { emailField, email });
          await page.type(emailField, email);
          await takeScreenshot('email-filled');
          await page.evaluate(({ passwordField }) => {
            const el = document.querySelector(passwordField);
            if (el) { (el).value = ''; (el).dispatchEvent(new Event('input', { bubbles: true })); (el).focus(); }
          }, { passwordField });
          await page.type(passwordField, password);
          await takeScreenshot('password-filled');
          
          // Small settle to avoid frame detaches
          await sleep(300);
          
          // Submit form - try button first, then Enter key
          try {
            const submitSelectors = [
              'button[type="submit"]',
              'button[data-test-id="signin-button"]',
              '.signin-button',
              '.login-button'
            ];
            
            let submitted = false;
            for (const selector of submitSelectors) {
              try {
                await page.waitForSelector(selector, { timeout: 1000 });
                await page.click(selector);
                await takeScreenshot(`clicked-submit-${selector}`);
                console.log(`Clicked submit button: ${selector}`);
                submitted = true;
                break;
              } catch (e) {
                // Continue to next selector
              }
            }
            
            if (!submitted) {
              // Fallback to Enter key
              console.log('No submit button found, using Enter key');
              await page.focus(passwordField);
              await page.keyboard.press('Enter');
            }
          } catch (submitError) {
            console.log('Submit button not found, using Enter key');
            await page.focus(passwordField);
            await page.keyboard.press('Enter');
          }
          
          // Wait for login to complete with error handling
          await new Promise(resolve => setTimeout(resolve, 3000));
          await takeScreenshot('post-login-wait');
          // Prefer waiting for app content rather than navigation event (SPA)
          try {
            await page.waitForFunction(() => !!document.querySelector('[data-test-id="execute-workflow-button"], .action-dropdown-container button'), { timeout: 20000 });
            console.log('Detected app UI after login');
          } catch (navError) {
            console.log('UI not detected after login, will continue');
          }
          
          await sleep(500);
          await takeScreenshot('after-login-navigation');
          console.log('Login completed, new URL:', page.url());
          
          // Stabilize UI after login (short)
          console.log('Waiting for stable UI...');
          await waitForStableUI(2000, 400, 2);
          await takeScreenshot('after-login-stable');
          
          // Check if login was successful
          const afterLoginUrl = page.url();
          const stillNeedsAuth = afterLoginUrl.includes('/signin') || 
                               afterLoginUrl.includes('/login') ||
                               afterLoginUrl.includes('/auth') ||
                               await page.evaluate(() => !!document.querySelector('[data-test-id="user-email"]')).catch(() => false);
          
          if (stillNeedsAuth) {
            throw new Error('Login failed - still on authentication page');
          }
          
          console.log('Authentication successful');
          
          // Navigate back to workflow after successful authentication
          const currentUrlAfterAuth = page.url();
          if (!currentUrlAfterAuth.includes('/workflow/')) {
            console.log('Navigating back to workflow after authentication');
            try {
              await page.goto('%WORKFLOW_URL%', {
                waitUntil: 'domcontentloaded',
                timeout: 15000
              });
              await takeScreenshot('workflow-after-auth');
              console.log('Successfully returned to workflow page');
            } catch (returnNavError) {
              console.log('Warning: Failed to return to workflow page:', returnNavError.message);
              await takeScreenshot('return-to-workflow-failed');
              // Continue anyway, might still work
            }
          }
          
        } else {
          throw new Error('Login form fields not found');
        }
      } else {
        const authMethods = [];
        if (email && email !== '%N8N_EMAIL_PLACEHOLDER%') authMethods.push('email/password');
        if (authMethods.length === 0) authMethods.push('none available');
        
        throw new Error(`Authentication required but failed. Available methods: ${authMethods.join(', ')}. Consider configuring N8N_EMAIL and N8N_PASSWORD in Vrooli secrets.`);
      }
    }
    
    // Wait for page to load after potential authentication
    await new Promise(resolve => setTimeout(resolve, 3000));
    
    await takeScreenshot('before-find-execute');
    console.log('Looking for execute button...');
    
      // Enhanced execute button logic with trigger selection
  const candidates = SELECTORS && SELECTORS.executeButton ? SELECTORS.executeButton : [
      '[data-test-id="execute-workflow-button"]',
      'button[aria-label="Execute workflow"]',
      'button[title*="Execute"]',
      'button[title*="Run"]',
      '.execute-button',
      '.run-button'
    ];

    let executeButton = null;
    for (const selector of candidates) {
      console.log(`Trying execute selector: ${selector}`);
      const visible = await waitForSelectorVisible(selector, 20000);
      if (visible) {
        executeButton = selector;
        console.log(`Found execute button visible: ${selector}`);
        break;
      } else {
        console.log(`Execute selector ${selector} not visible yet`);
      }
    }

    if (!executeButton) {
      await takeScreenshot('no-execute-button');
      throw new Error('No execute button found');
    }

    await takeScreenshot('execute-button-found');
    
    // Check if there's a dropdown for trigger selection - try profile candidates
    const dropdownCandidates = SELECTORS && SELECTORS.executeDropdown ? SELECTORS.executeDropdown : ['[data-test-id="execute-workflow-button"] ._chevron_p52lz_142', '.action-dropdown-container button', '._chevron_p52lz_142'];
    let opened = false;
    for (const dd of dropdownCandidates) {
      const visible = await waitForSelectorVisible(dd, 8000);
      if (visible) {
        console.log('Opening trigger selection dropdown via:', dd);
        try { await page.click(dd); } catch (e) {
          await page.evaluate((sel) => { const el = document.querySelector(sel); if (el) el.dispatchEvent(new MouseEvent('click', { bubbles: true })); }, dd);
        }
        opened = true;
        break;
      }
    }
    
          if (opened) {
        console.log('Dropdown opened, searching for Manual Trigger...');
        await new Promise(resolve => setTimeout(resolve, 500));
        await takeScreenshot('dropdown-opened');
      
      // Wait for dropdown menu to appear
      await new Promise(resolve => setTimeout(resolve, 1000));
      await takeScreenshot('dropdown-waited');
      
      // Look for Manual Trigger option using profile candidates and text fallback
      let manualTriggerFound = false;
      // Try XPaths/CSS from profile
      if (SELECTORS && SELECTORS.triggerMenuItems) {
        for (const sel of SELECTORS.triggerMenuItems) {
          try {
            // If it's an XPath
            if (sel.startsWith('//')) {
              const found = await page.evaluate((xpath) => {
                const snap = document.evaluate(xpath, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null);
                const el = snap.singleNodeValue;
                if (el) { el.dispatchEvent(new MouseEvent('click', { bubbles: true })); return true; }
                return false;
              }, sel);
              if (found) { manualTriggerFound = true; break; }
            } else {
              const visible = await waitForSelectorVisible(sel, 2000);
              if (visible) { await page.click(sel); manualTriggerFound = true; break; }
            }
          } catch { /* continue */ }
        }
      }
      // Fallback by text scan
      if (!manualTriggerFound) {
        manualTriggerFound = await page.evaluate(() => {
          const popovers = Array.from(document.querySelectorAll('[role="menu"], .el-dropdown-menu, .el-popper'));
          for (const pop of popovers) {
            const items = Array.from(pop.querySelectorAll('*'));
            const match = items.find(el => /manual trigger|manual/i.test(el.textContent || ''));
            if (match) { match.dispatchEvent(new MouseEvent('click', { bubbles: true })); return true; }
          }
          return false;
        }).catch(() => false);
      }
      
      if (!manualTriggerFound) {
        // Secondary scan within any open popover containers
        manualTriggerFound = await page.evaluate(() => {
          const popovers = Array.from(document.querySelectorAll('[role="menu"], .el-dropdown-menu, .el-popper'));
          for (const pop of popovers) {
            const items = Array.from(pop.querySelectorAll('*'));
            const match = items.find(el => /manual trigger|manual/i.test(el.textContent || ''));
            if (match) {
              match.dispatchEvent(new MouseEvent('click', { bubbles: true }));
              return true;
            }
          }
          return false;
        }).catch(() => false);
      }
      
      if (!manualTriggerFound) {
        console.log('Manual trigger option not found in dropdown, proceeding with default');
        await takeScreenshot('manual-trigger-not-found');
        // Close dropdown by clicking elsewhere
        await page.click('body');
      } else {
        console.log('Manual trigger selected successfully');
        await takeScreenshot('manual-trigger-selected');
        // Wait for dropdown to close and button to update
        await new Promise(resolve => setTimeout(resolve, 1000));
        // Verify main execute shows Manual selected
        try {
          await page.waitForFunction(() => {
            const btn = document.querySelector('[data-test-id="execute-workflow-button"]');
            return btn && /manual/i.test(btn.textContent || '');
          }, { timeout: 4000 });
          console.log('Verified: Execute button shows Manual selected');
          await takeScreenshot('manual-verified');
        } catch (e) {
          console.log('Warning: Execute button did not update to Manual in time');
          await takeScreenshot('manual-verify-failed');
        }
      }
    }
    
    // Clear any existing snack messages before execution
    console.log('Clearing any existing snack messages before workflow execution...');
    const clearedBefore = await clearSnackMessages();
    console.log(`Cleared ${clearedBefore} existing snack messages`);
    await takeScreenshot('after-clear-snacks');
    
      console.log('Clicking execute button...');
  await takeScreenshot('before-click-execute');
  // Ensure in view and click with retries
  await waitForSelectorVisible(executeButton, 20000);
  let clicked = false;
  for (let i = 0; i < 3 && !clicked; i++) {
    try {
      await page.click(executeButton, { delay: 20 });
      clicked = true;
    } catch (e) {
      console.log('Click failed, retrying...', e.message);
      await sleep(500 + i * 500);
    }
  }
  if (!clicked) {
    // Fallback to dispatching event
    await page.evaluate((sel) => {
      const el = document.querySelector(sel);
      if (el) el.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    }, executeButton);
  }
    await takeScreenshot('after-click-execute');
  
  // Mark execution as started
  executionData.executionStatus.started = true;
  executionData.executionStatus.running = true;
  executionData.triggeredAt = new Date().toISOString();
    
    // Additional fallback: trigger run via keyboard shortcut (Ctrl/Cmd + Enter)
    try {
      await page.keyboard.down('Control');
      await page.keyboard.press('Enter');
      await page.keyboard.up('Control');
      console.log('Fallback: triggered Ctrl+Enter');
      await takeScreenshot('after-hotkey-execute');
    } catch (e) { /* ignore */ }
    
    // If input data is provided, try to fill it in
    const inputData = '%INPUT_DATA%';
    if (inputData && inputData !== '%INPUT_DATA%') {
      console.log('Input data provided, looking for input form...');
      
      try {
        // Wait a moment for any input dialog to appear
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // Look for common input field selectors in n8n
        const inputSelectors = [
          'textarea[data-test-id="workflow-input-data"]',
          'textarea[placeholder*="JSON"]',
          'textarea[placeholder*="input"]',
          '.cm-editor textarea',
          '.monaco-editor textarea',
          'textarea:not([readonly])',
          'input[type="text"]:not([readonly])'
        ];
        
        let inputFound = false;
        for (const selector of inputSelectors) {
          try {
            await page.waitForSelector(selector, { timeout: 2000 });
            console.log(`Found input field with selector: ${selector}`);
            
            // Clear existing content and input new data
            await page.evaluate((sel) => {
              const element = document.querySelector(sel);
              if (element) {
                element.value = '';
                element.focus();
              }
            }, selector);
            
            await page.type(selector, inputData, { delay: 50 });
            console.log('Input data entered successfully');
            inputFound = true;
            break;
          } catch (e) {
            // Continue to next selector
          }
        }
        
        if (!inputFound) {
          console.log('No input field found, proceeding without input data');
        }
        
        // Look for and click any "Execute" or "Run" button that might appear
        try {
          const executeSelectors = [
            'button[data-test-id="execute-button"]',
            '.execute-button'
          ];
          
          for (const selector of executeSelectors) {
            try {
              await page.waitForSelector(selector, { timeout: 1000 });
              await page.click(selector);
              console.log(`Clicked additional execute button: ${selector}`);
              break;
            } catch (e) {
              // Continue to next selector
            }
          }
        } catch (e) {
          // No additional execute button found
        }
        
      } catch (error) {
        console.log('Input handling failed, continuing with execution:', error.message);
      }
    }
    
    await takeScreenshot('before-monitoring');
    console.log('Workflow execution triggered, starting enhanced monitoring...');
    
    // ENHANCED EXECUTION MONITORING - Wait for actual completion via snack messages or UI state
    let pollCount = 0;
    const pollInterval = 3000; // 3 seconds
    const maxPolls = Math.max(10, Math.floor(%TIMEOUT% / pollInterval));
    let sawSnackSuccess = false;
    let sawSnackError = false;
    
    console.log(`Starting enhanced monitoring: max ${maxPolls} polls every ${pollInterval}ms`);
    
    while (pollCount < maxPolls) {
      await new Promise(resolve => setTimeout(resolve, pollInterval));
      pollCount++;
      
      console.log(`Enhanced monitoring poll ${pollCount}/${maxPolls}...`);
      
      // Check execution state through UI inspection
      const executionInfo = await checkExecutionState();
      if (executionInfo) {
        console.log(`Poll ${pollCount} - Execute button: ${executionInfo.executeButtonExists ? 'exists' : 'missing'}`);
        console.log(`Poll ${pollCount} - Cancel button: ${executionInfo.cancelButtonExists ? 'exists' : 'missing'}`);
        console.log(`Poll ${pollCount} - Spinner: ${executionInfo.spinnerExists ? 'exists' : 'missing'}`);
        console.log(`Poll ${pollCount} - Success indicator: ${executionInfo.successIndicatorExists ? 'exists' : 'missing'}`);
        console.log(`Poll ${pollCount} - Error indicator: ${executionInfo.errorIndicatorExists ? 'exists' : 'missing'}`);
        console.log(`Poll ${pollCount} - Result panels: ${executionInfo.resultPanelsCount}`);
        
        // PRIMARY: Check for snack messages (most reliable indicator)
        const currentSnackMessages = await detectSnackMessages();
        if (currentSnackMessages.length > 0) {
          console.log(`Poll ${pollCount}: Found ${currentSnackMessages.length} snack message(s)`);
          
          if (!executionData.snackMessages) executionData.snackMessages = [];
          
          for (const newMsg of currentSnackMessages) {
            const isDuplicate = executionData.snackMessages.some(existing => 
              existing.title === newMsg.title && existing.content === newMsg.content && existing.type === newMsg.type
            );
            if (!isDuplicate) {
              executionData.snackMessages.push(newMsg);
              console.log(`New snack message recorded: [${newMsg.type.toUpperCase()}] ${newMsg.title} - ${newMsg.content}`);
            }
            if (newMsg.type === 'success') sawSnackSuccess = true;
            if (newMsg.type === 'error') sawSnackError = true;
          }
        }
        
        // Check for completion indicators
        let executionComplete = false;
        let executionFailed = false;
        
        // 1. Look for success/error indicators in UI
        if (executionInfo.successIndicatorExists) {
          executionData.executionStatus.completed = true;
          executionData.executionStatus.running = false;
          executionComplete = true;
          console.log(`Poll ${pollCount}: Detected SUCCESS via UI indicator`);
        } else if (executionInfo.errorIndicatorExists) {
          executionData.executionStatus.failed = true;
          executionData.executionStatus.running = false;
          executionFailed = true;
          executionComplete = true;
          console.log(`Poll ${pollCount}: Detected FAILURE via UI indicator`);
        }
        
        // 2. Check for status text patterns
        if (!executionComplete && executionInfo.statusTexts && executionInfo.statusTexts.length > 0) {
          for (const statusText of executionInfo.statusTexts) {
            const text = statusText.toLowerCase();
            if (text.includes('execution completed') || 
                text.includes('execution finished') || 
                text.includes('workflow completed') ||
                text.includes('execution successful')) {
              executionData.executionStatus.completed = true;
              executionData.executionStatus.running = false;
              executionComplete = true;
              console.log(`Poll ${pollCount}: Detected SUCCESS via status text: "${statusText}"`);
              break;
            } else if (text.includes('execution failed') || 
                       text.includes('execution error') || 
                       text.includes('workflow failed') ||
                       text.includes('execution cancelled')) {
              executionData.executionStatus.failed = true;
              executionData.executionStatus.running = false;
              executionFailed = true;
              executionComplete = true;
              console.log(`Poll ${pollCount}: Detected FAILURE via status text: "${statusText}"`);
              break;
            }
          }
        }
        
        // 3. Check button states (if execute button is re-enabled, execution likely complete)
        if (!executionComplete && !executionInfo.cancelButtonExists && executionInfo.executeButtonExists && !executionInfo.spinnerExists) {
          // Execution might be complete if cancel button disappeared and execute button is back
          if (pollCount >= 2) { // Wait at least 2 polls to avoid false positives
            console.log(`Poll ${pollCount}: Detected potential completion via button state change`);
            // Take screenshot to verify completion visually
            await takeScreenshot(`completion-check-poll-${pollCount}`, true);
            
            // Additional check for result data
            if (executionInfo.resultPanelsCount > 0) {
              executionData.executionStatus.completed = true;
              executionData.executionStatus.running = false;
              executionComplete = true;
              console.log(`Poll ${pollCount}: Confirmed SUCCESS via result panels`);
            }
          }
        }
        
        // 4. Check console logs for execution messages
        if (!executionComplete) {
          const recentLogs = executionData.consoleLogs.slice(-10); // Check last 10 logs
          for (const log of recentLogs) {
            const text = log.text.toLowerCase();
            if (text.includes('workflow execution finished') ||
                text.includes('execution completed') ||
                text.includes('execution successful') ||
                text.includes('finished executing')) {
              executionData.executionStatus.completed = true;
              executionData.executionStatus.running = false;
              executionComplete = true;
              console.log(`Poll ${pollCount}: Detected SUCCESS via console log: "${log.text}"`);
              break;
            } else if (text.includes('workflow execution failed') ||
                       text.includes('execution error') ||
                       text.includes('execution failed')) {
              executionData.executionStatus.failed = true;
              executionData.executionStatus.running = false;
              executionFailed = true;
              executionComplete = true;
              console.log(`Poll ${pollCount}: Detected FAILURE via console log: "${log.text}"`);
              break;
            }
          }
        }
        
        // If we detected snack success/error, treat as completion immediately
        if (!executionComplete && (sawSnackSuccess || sawSnackError)) {
          executionComplete = true;
          executionData.executionStatus.running = false;
          executionData.executionStatus.completed = sawSnackSuccess && !sawSnackError;
          executionData.executionStatus.failed = sawSnackError;
          console.log(`Poll ${pollCount}: Completion via snack (${sawSnackSuccess ? 'success' : ''}${sawSnackError ? 'error' : ''})`);
        }

        if (executionComplete) {
          console.log(`Execution detected complete after ${pollCount} polls`);
          break;
        }
      }
      
      // Take periodic screenshots during execution
      if (pollCount % 2 === 0) { // Every other poll
        await takeScreenshot(`monitoring-poll-${pollCount}`);
      }
    }
    
    // Capture final state with high-quality screenshot
    await takeScreenshot('final-execution-state', true);
    
    // Final snack message check - most important for determining actual status
    console.log('Performing final snack message detection...');
    const finalSnackMessages = await detectSnackMessages();
    if (finalSnackMessages.length > 0) {
      console.log(`Final check found ${finalSnackMessages.length} snack message(s)`);
      
      if (!executionData.snackMessages) {
        executionData.snackMessages = [];
      }
      
      // Add any new final snack messages
      for (const newMsg of finalSnackMessages) {
        const isDuplicate = executionData.snackMessages.some(existing => 
          existing.title === newMsg.title && 
          existing.content === newMsg.content && 
          existing.type === newMsg.type
        );
        if (!isDuplicate) {
          executionData.snackMessages.push(newMsg);
          console.log(`Final snack message recorded: [${newMsg.type.toUpperCase()}] ${newMsg.title} - ${newMsg.content}`);
        }
      }
    }
    
    // Capture final execution state details
    const finalExecutionInfo = await checkExecutionState();
    if (finalExecutionInfo) {
      executionData.finalState = finalExecutionInfo;
      console.log('Final execution state captured');
    }
    
    // Final console log check
    const finalLogs = executionData.consoleLogs.slice(-5);
    console.log('Final console logs check:');
    for (const log of finalLogs) {
      console.log(`  [${log.type}] ${log.text}`);
    }
    
    executionData.endTime = new Date().toISOString();
    executionData.pollsCompleted = pollCount;
    
    // Compute accurate total duration since trigger
    try {
      const t0 = new Date(executionData.triggeredAt || executionData.startTime).getTime();
      const t1 = new Date(executionData.endTime).getTime();
      executionData.totalDurationMs = Math.max(0, t1 - t0);
      console.log(`⏱️ Total time from trigger to end: ${executionData.totalDurationMs} ms`);
    } catch {}
    
    // Determine success from snack messages only; if none, treat as still running/unknown
    if (executionData.snackMessages && executionData.snackMessages.length > 0) {
      console.log('=== SNACK MESSAGE ANALYSIS ===');
      console.log(`Found ${executionData.snackMessages.length} snack message(s) during execution:`);
      
      let hasErrors = false;
      let hasSuccess = false;
      
      for (const msg of executionData.snackMessages) {
        console.log(`  [${msg.type.toUpperCase()}] ${msg.title}: ${msg.content}`);
        if (msg.type === 'error') { hasErrors = true; }
        if (msg.type === 'success') { hasSuccess = true; }
      }
      
      if (hasErrors) {
        executionData.success = false;
        executionData.executionStatus.failed = true;
        executionData.executionStatus.running = false;
        console.log('❌ Workflow execution failed (error snack messages detected)');
      } else if (hasSuccess) {
        executionData.success = true;
        executionData.executionStatus.completed = true;
        executionData.executionStatus.running = false;
        console.log('🎉 Workflow execution completed successfully (success snack messages detected)');
      } else {
        executionData.success = false;
        executionData.executionStatus.completed = false;
        executionData.executionStatus.running = true;
        console.log('ℹ️ Snack messages present but neither success nor error; treating as still running');
      }
    } else {
      // No snack messages at all: treat as still running/unknown
      executionData.success = false;
      executionData.executionStatus.completed = false;
      executionData.executionStatus.running = true;
      console.log('=== NO SNACK MESSAGES DETECTED ===');
      console.log('No snack messages found - treating as still running (will require further investigation)');
    }
    
    return {
      data: executionData,
      type: 'application/json'
    };
    
  } catch (err) {
    await takeScreenshot('execution-error', true);
    
    executionData.error = {
      message: err.message,
      stack: err.stack,
      timestamp: new Date().toISOString()
    };
    executionData.endTime = new Date().toISOString();
    executionData.success = false;
    executionData.executionStatus.failed = true;
    executionData.executionStatus.running = false;
    
    return {
      data: executionData,
      type: 'application/json'
    };
  }
}
EOF
    
    # Replace placeholders in function code  
    function_code="${function_code//%WORKFLOW_ID%/$workflow_id}"
    function_code="${function_code//%WORKFLOW_URL%/$workflow_url}"
    function_code="${function_code//%TIMEOUT%/$timeout}"
    function_code="${function_code//%SESSION_ID%/$session_id}"
    function_code="${function_code//%USE_PERSISTENT_SESSION%/$use_persistent_session}"
    
    # Source Vrooli's secrets management system
    local scripts_dir="${VROOLI_PROJECT_ROOT:-$(dirname "$(dirname "$(dirname "$(dirname "$(dirname "${BASH_SOURCE[0]}")")")")}")}/scripts"
    if [[ -f "${scripts_dir}/lib/service/secrets.sh" ]]; then
        source "${scripts_dir}/lib/service/secrets.sh"
    fi
    
    # Get N8N web UI credentials from Vrooli's unified secrets system
    local n8n_email="${N8N_EMAIL:-}"
    local n8n_password="${N8N_PASSWORD:-}"
    
    # Try to get credentials from Vrooli secrets system
    if command -v secrets::resolve >/dev/null 2>&1; then
        if [[ -z "$n8n_email" ]]; then
            n8n_email=$(secrets::resolve "N8N_EMAIL" 2>/dev/null || echo "")
        fi
        if [[ -z "$n8n_password" ]]; then
            n8n_password=$(secrets::resolve "N8N_PASSWORD" 2>/dev/null || echo "")
        fi
    fi
    
    # Log credential status
    if [[ -n "$n8n_email" && -n "$n8n_password" ]]; then
        log::info "🔐 Found N8N web credentials from Vrooli secrets system"
        log::info "🔐 Email: $n8n_email | Password: [${#n8n_password} characters]"
    else
        log::info "ℹ️  No N8N web credentials found - will attempt without authentication"
        log::info "    (This is normal for development n8n instances)"
    fi
    
    # Inject credentials into browserless function
    function_code=$(echo "$function_code" | sed "s|%N8N_EMAIL%|$n8n_email|g")
    function_code=$(echo "$function_code" | sed "s|%N8N_PASSWORD%|$n8n_password|g")
    
    # Build selector profile (env > file > defaults)
    local selectors_default='{"executeButton":["[data-test-id=\"execute-workflow-button\"]","button[aria-label=\"Execute workflow\"]","button[title*=\"Execute\"]","button[title*=\"Run\"]",".execute-button",".run-button"]}'
    local selectors_raw=""
    if [[ -n "${N8N_SELECTORS:-}" ]]; then
        selectors_raw="$N8N_SELECTORS"
    elif [[ -n "${N8N_SELECTORS_FILE:-}" && -f "$N8N_SELECTORS_FILE" ]]; then
        selectors_raw=$(cat "$N8N_SELECTORS_FILE" 2>/dev/null || echo "")
    elif [[ -f "$(dirname "${BASH_SOURCE[0]}")/selectors.json" ]]; then
        selectors_raw=$(cat "$(dirname "${BASH_SOURCE[0]}")/selectors.json" 2>/dev/null || echo "")
    else
        selectors_raw="$selectors_default"
    fi
    # Validate JSON and collapse to oneliner
    if echo "$selectors_raw" | jq . >/dev/null 2>&1; then
        selectors_raw=$(echo "$selectors_raw" | jq -c . 2>/dev/null || echo "$selectors_default")
    else
        log::info "Using default selectors profile (invalid custom JSON)"
        selectors_raw="$selectors_default"
    fi
    # Inject into code (no quotes; insert as object literal)
    function_code="${function_code//%SELECTORS%/$selectors_raw}"
    
    # Escape input data for JavaScript insertion
    local escaped_input=""
    if [[ -n "$processed_input" ]]; then
        # Escape quotes and newlines for safe JavaScript insertion
        escaped_input=$(echo "$processed_input" | sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n' | sed 's/\\n$//')
    fi
    function_code="${function_code//%INPUT_DATA%/$escaped_input}"
    
    # Ensure test output directory exists
    browserless::ensure_test_output_dir
    
    local temp_file="/tmp/browserless_workflow_exec_$$"
    # Sanitize workflow_id for filename use (replace special characters with underscores)
    local safe_workflow_id
    safe_workflow_id=$(echo "$workflow_id" | sed 's/[^a-zA-Z0-9._-]/_/g')
    # Create a per-run artifacts dir
    local run_dir="${BROWSERLESS_TEST_OUTPUT_DIR}/n8n/${safe_workflow_id}_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$run_dir" 2>/dev/null || true
    local output_file="${run_dir}/result.json"
    local function_file="${run_dir}/function.js"
    
    log::info "Executing workflow automation..."
    
    # Persist session launch options
    local launch_json=""
    if [[ "$use_persistent_session" == "true" ]]; then
        local user_data_dir="/tmp/browserless-sessions/${session_id}"
        launch_json="{\"args\":[\"--user-data-dir=${user_data_dir}\"]}"
        log::info "Using persistent session: $session_id"
        log::info "User data directory: $user_data_dir"
    fi
    
    # Save function code for repro/debug
    echo "$function_code" > "$function_file"
    
    # Execute via shared Browserless runner
    local response
    response=$(browserless::run_function "$function_code" "$timeout" "$use_persistent_session" "$session_id" "$launch_json")
    local curl_exit_code=$?
    
    if [[ $curl_exit_code -ne 0 ]]; then
        echo
        log::error "❌ Failed to execute workflow automation (curl exit code: $curl_exit_code)"
        if [[ -n "$response" ]]; then
            echo
            log::info "🔍 Curl Response/Error:"
            echo "----------------------------------------"
            echo "$response"
            echo "----------------------------------------"
        fi
        log::info "💡 Troubleshooting steps:"
        log::info "  1. Check browserless status: resource-browserless status"
        log::info "  2. Test browserless health: curl -s $BROWSERLESS_BASE_URL/pressure"
        log::info "  3. Verify n8n is accessible: curl -s $n8n_url/healthz"
        return 1
    fi
    
    # Save and parse response
    echo "$response" > "$temp_file"
    
    # Try to parse the response
    if command -v jq &> /dev/null && jq . "$temp_file" &> /dev/null; then
        # Valid JSON response
        local success=$(jq -r '.success // false' "$temp_file")
        local workflow_started=$(jq -r '.executionStatus.started // false' "$temp_file")
        local workflow_running=$(jq -r '.executionStatus.running // false' "$temp_file")
        local workflow_completed=$(jq -r '.executionStatus.completed // false' "$temp_file")
        local workflow_failed=$(jq -r '.executionStatus.failed // false' "$temp_file")
        local console_logs_count=$(jq -r '.consoleLogs | length' "$temp_file" 2>/dev/null || echo "0")
        local errors_count=$(jq -r '.pageErrors | length' "$temp_file" 2>/dev/null || echo "0")
        local network_errors_count=$(jq -r '.networkErrors | length' "$temp_file" 2>/dev/null || echo "0")
        local polls_completed=$(jq -r '.pollsCompleted // 0' "$temp_file")
        local enhanced_monitoring=$(jq -r '.enhancedMonitoring // false' "$temp_file")
        local session_id_used=$(jq -r '.sessionId // "unknown"' "$temp_file")
        local snack_messages_count=$(jq -r '.data.snackMessages | length' "$temp_file" 2>/dev/null || echo "0")
        
        # Move to final output location
        mv "$temp_file" "$output_file"
        
        # Save screenshots emitted in JSON into files in artifacts dir
        local ss_count
        ss_count=$(jq -r '.screenshots | length' "$output_file" 2>/dev/null || echo "0")
        if [[ "$ss_count" != "0" ]]; then
            for i in $(seq 0 $((ss_count-1))); do
                local label
                local type
                local data
                label=$(jq -r ".screenshots[$i].label" "$output_file")
                type=$(jq -r ".screenshots[$i].type" "$output_file")
                data=$(jq -r ".screenshots[$i].data" "$output_file")
                if [[ -n "$data" && "$data" != "null" ]]; then
                    echo "$data" | base64 -d > "$run_dir/${i}_${label}.${type}" 2>/dev/null || true
                fi
            done
        fi
        
        echo
        log::success "✓ Enhanced workflow automation completed"
        log::info "Results saved to: $output_file"
        log::info "Artifacts directory: $run_dir"
        echo
        
        # Display enhanced summary
        log::info "📊 Enhanced Execution Summary:"
        log::info "  Automation Success: $success"
        log::info "  Enhanced Monitoring: $enhanced_monitoring"
        log::info "  Session ID: $session_id_used"
        log::info "  Persistent Session: $use_persistent_session"
        log::info "  Workflow Started: $workflow_started"
        log::info "  Workflow Running: $workflow_running"
        log::info "  Workflow Completed: $workflow_completed"
        log::info "  Workflow Failed: $workflow_failed"
        log::info "  Console Logs: $console_logs_count"
        log::info "  Page Errors: $errors_count"
        log::info "  Network Errors: $network_errors_count"
        log::info "  Monitoring Polls: $polls_completed"
        log::info "  Snack Messages: $snack_messages_count"
        log::info "  Input Data Used: $input_description"
        
        # Show snack messages (most important feedback)
        if [[ "$snack_messages_count" -gt 0 ]]; then
            echo
            log::info "📢 Snack Messages (n8n notifications):"
                         jq -r '.data.snackMessages[]? | "  [\(.type | ascii_upcase)] \(.title): \(.content)"' "$output_file" 2>/dev/null || true
        else
            echo
            log::info "📢 No snack messages detected (likely successful execution)"
        fi
        
        # Show recent console logs
        if [[ "$console_logs_count" -gt 0 ]]; then
            echo
            log::info "📋 Recent Console Logs:"
            jq -r '.consoleLogs[-5:][] | "  [\(.type | ascii_upcase)] \(.text)"' "$output_file" 2>/dev/null || true
            
            # Check for error patterns in console logs
            local error_log_count
            error_log_count=$(jq -r '.consoleLogs[] | select(.type == "error") | .text' "$output_file" 2>/dev/null | wc -l || echo "0")
            local warn_log_count
            warn_log_count=$(jq -r '.consoleLogs[] | select(.type == "warning") | .text' "$output_file" 2>/dev/null | wc -l || echo "0")
            
            if [[ "$error_log_count" -gt 0 ]] || [[ "$warn_log_count" -gt 0 ]]; then
                echo
                log::warn "⚠️  Detected $error_log_count errors and $warn_log_count warnings in console logs"
                log::info "Run the following to see detailed logs:"
                log::info "  jq '.consoleLogs[] | select(.type == \"error\" or .type == \"warning\")' '$output_file'"
            fi
        fi
        
        # Show errors if any
        if [[ "$errors_count" -gt 0 ]]; then
            echo
            log::warn "⚠️  Page Errors:"
            jq -r '.pageErrors[] | "  \(.message)"' "$output_file" 2>/dev/null || true
        fi
        
        # Show final execution state if available
        local final_state_exists=$(jq -r '.finalState != null' "$output_file" 2>/dev/null || echo "false")
        if [[ "$final_state_exists" == "true" ]]; then
            echo
            log::info "🎯 Final Execution State:"
            local execute_btn_exists=$(jq -r '.finalState.executeButtonExists // false' "$output_file")
            local result_panels=$(jq -r '.finalState.resultPanelsCount // 0' "$output_file")
            local success_indicator=$(jq -r '.finalState.successIndicatorExists // false' "$output_file")
            local error_indicator=$(jq -r '.finalState.errorIndicatorExists // false' "$output_file")
            log::info "  Execute Button Present: $execute_btn_exists"
            log::info "  Result Panels: $result_panels"
            log::info "  Success Indicator: $success_indicator"
            log::info "  Error Indicator: $error_indicator"
        fi
        
        # Process and save screenshots if available
        # Check both possible locations for screenshots (wrapped and unwrapped)
        local screenshot_count=$(jq -r '.data.screenshots | length' "$output_file" 2>/dev/null || echo "0")
        if [[ "$screenshot_count" == "0" || "$screenshot_count" == "null" ]]; then
            screenshot_count=$(jq -r '.screenshots | length' "$output_file" 2>/dev/null || echo "0")
        fi
        
        if [[ "$screenshot_count" -gt 0 ]]; then
            echo
            log::info "📷 Processing $screenshot_count screenshots..."
            
            # Extract and save each screenshot, then update JSON with file paths
            local temp_json=$(mktemp)
            cp "$output_file" "$temp_json"
            local safe_workflow_id=$(echo "$workflow_id" | sed 's/[^a-zA-Z0-9._-]/_/g')
            local timestamp=$(date +%Y%m%d_%H%M%S)
            
            # Create array to store updated screenshots
            local updated_screenshots="[]"
            local screenshot_index=0
            
            # Determine screenshot path in JSON (wrapped or unwrapped)
            local screenshot_path=".data.screenshots"
            if ! jq -e '.data.screenshots' "$output_file" >/dev/null 2>&1; then
                screenshot_path=".screenshots"
            fi
            
            # Process each screenshot
            while IFS=$'\t' read -r label b64 type; do
                if [[ -n "$b64" && "$b64" != "null" ]]; then
                    local filename="screenshot_${safe_workflow_id}_${label}_${timestamp}_${screenshot_index}.${type}"
                    local filepath="${BROWSERLESS_TEST_OUTPUT_DIR}/${filename}"
                    
                    if echo "$b64" | base64 -d > "$filepath" 2>/dev/null; then
                        log::info "  ✅ Saved: $filepath"
                        
                        # Add screenshot metadata to updated array (without base64 data)
                        local current_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
                        local file_size=$(stat -c%s "$filepath" 2>/dev/null || echo 0)
                        updated_screenshots=$(echo "$updated_screenshots" | jq --arg label "$label" \
                                                                               --arg filepath "$filepath" \
                                                                               --arg filename "$filename" \
                                                                               --arg type "$type" \
                                                                               --arg timestamp "$current_timestamp" \
                                                                               --argjson size "$file_size" \
                                                                               '. += [{
                                                                                   label: $label,
                                                                                   timestamp: $timestamp,
                                                                                   filename: $filename,
                                                                                   filePath: $filepath,
                                                                                   size: $size,
                                                                                   type: $type,
                                                                                   highQuality: true
                                                                               }]')
                    else
                        log::warn "  ❌ Failed to save: $filename"
                    fi
                fi
                screenshot_index=$((screenshot_index + 1))
            done < <(jq -r "${screenshot_path}[]? | [.label // \"unknown\", .data // \"\", .type // \"png\"] | @tsv" "$output_file" 2>/dev/null || true)
            
            # Update the JSON file with new screenshot data (removing base64)
            if [[ "$screenshot_path" == ".data.screenshots" ]]; then
                jq --argjson screenshots "$updated_screenshots" '.data.screenshots = $screenshots' "$temp_json" > "$output_file"
            else
                jq --argjson screenshots "$updated_screenshots" '.screenshots = $screenshots' "$temp_json" > "$output_file"
            fi
            rm -f "$temp_json"
            
            log::info "📷 Screenshot processing completed - JSON updated with file paths"
        fi
        
        # IMPROVED: Determine overall success based on snack messages first, then fallback to other indicators
        if [[ "$snack_messages_count" -gt 0 ]]; then
                         local has_error_snacks=$(jq -r '.data.snackMessages[]? | select(.type == "error") | .type' "$output_file" 2>/dev/null | wc -l || echo "0")
             local has_success_snacks=$(jq -r '.data.snackMessages[]? | select(.type == "success") | .type' "$output_file" 2>/dev/null | wc -l || echo "0")
            
            if [[ "$has_error_snacks" -gt 0 ]]; then
                log::error "❌ Workflow execution failed (error notifications detected)"
                return 1
            elif [[ "$has_success_snacks" -gt 0 ]]; then
                log::success "🎉 Workflow execution completed successfully (success notifications detected)!"
                return 0
            else
                log::info "ℹ️  Workflow execution completed (info/warning notifications only)"
                return 0
            fi
        else
            # Fallback to traditional completion detection
            if [[ "$success" == "true" ]]; then
                if [[ "$workflow_completed" == "true" ]]; then
                    log::success "🎉 Workflow execution completed successfully (no error notifications)!"
                    return 0
                elif [[ "$workflow_failed" == "true" ]]; then
                    log::warn "⚠️  Workflow execution failed"
                    return 1
                elif [[ "$workflow_started" == "true" ]]; then
                    log::warn "⚠️  Workflow started but completion status unclear"
                    return 1
                else
                    log::warn "⚠️  Workflow execution status unclear"
                    return 1
                fi
            else
                log::error "❌ Browser automation failed"
                return 1
            fi
        fi
    else
        # Invalid JSON or non-JSON response - provide detailed error information
        mv "$temp_file" "$output_file"
        echo
        log::error "❌ Invalid response from enhanced workflow automation"
        log::info "Full response saved to: $output_file"
        
        # Always show response preview for debugging
        echo
        log::info "🔍 Response Preview (first 1000 characters):"
        echo "----------------------------------------"
        head -c 1000 "$output_file" 2>/dev/null || echo "Cannot read response file"
        echo
        echo "----------------------------------------"
        
        # Check if it's an HTTP error response
        if head -c 50 "$output_file" 2>/dev/null | grep -qi "error\|exception\|failed\|timeout"; then
            echo
            log::warn "⚠️  This appears to be an error response from browserless"
            log::info "Common causes:"
            log::info "  • N8N authentication required (workflow redirects to login)"
            log::info "  • Workflow ID '$workflow_id' not found"  
            log::info "  • N8N service not accessible at $n8n_url"
            log::info "  • Browser timeout during execution"
            log::info "  • Workflow execution errors"
        fi
        
        # Check if response looks like HTML (authentication redirect)
        if head -c 50 "$output_file" 2>/dev/null | grep -qi "<html\|<!doctype"; then
            echo
            log::warn "🔐 Response appears to be HTML - likely an authentication redirect"
            log::info "Try using the enhanced authentication workflow:"
            log::info "  resource-n8n list-workflows | grep auth"
        fi
        
        return 1
    fi
}

# Export the function to make it available to other scripts
export -f browserless::execute_n8n_workflow 