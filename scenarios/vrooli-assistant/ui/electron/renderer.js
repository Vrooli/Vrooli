// Renderer process script for overlay UI

let currentScreenshot = null;
let currentContext = null;

// Initialize on load
window.addEventListener('DOMContentLoaded', async () => {
  // Load settings
  const settings = await window.electronAPI.getSettings();
  document.getElementById('agentType').value = settings.defaultAgentType || 'claude-code';
  
  // Get initial context
  await updateContext();
  
  // Set up keyboard shortcuts
  setupKeyboardShortcuts();
  
  // Focus description field
  document.getElementById('description').focus();
});

// Capture screenshot
async function captureScreenshot() {
  try {
    // Update UI
    document.getElementById('screenshotStatus').textContent = 'Capturing...';
    
    // Capture screenshot
    const screenshotPath = await window.electronAPI.captureScreenshot();
    currentScreenshot = screenshotPath;
    
    // Update preview (simplified - just show success)
    document.getElementById('screenshotPreview').innerHTML = '✓';
    document.getElementById('screenshotPreview').style.background = 'rgba(16, 185, 129, 0.1)';
    document.getElementById('screenshotStatus').textContent = 'Screenshot captured';
    
    // Update context after screenshot
    await updateContext();
    
  } catch (error) {
    console.error('Screenshot failed:', error);
    document.getElementById('screenshotStatus').textContent = 'Capture failed';
    document.getElementById('screenshotPreview').innerHTML = '❌';
  }
}

// Update context
async function updateContext() {
  try {
    currentContext = await window.electronAPI.getContext();
    
    // Update scenario display
    if (currentContext.scenario && currentContext.scenario !== 'unknown') {
      document.getElementById('scenarioName').textContent = `Scenario: ${currentContext.scenario}`;
    } else if (currentContext.url && currentContext.url !== 'unknown') {
      // Extract domain from URL
      const urlMatch = currentContext.url.match(/localhost:(\d+)/);
      if (urlMatch) {
        document.getElementById('scenarioName').textContent = `Port: ${urlMatch[1]}`;
      } else {
        document.getElementById('scenarioName').textContent = 'Context captured';
      }
    }
  } catch (error) {
    console.error('Failed to get context:', error);
  }
}

// Submit issue
async function submitIssue() {
  const description = document.getElementById('description').value.trim();
  const agentType = document.getElementById('agentType').value;
  
  if (!description) {
    document.getElementById('description').style.borderColor = 'rgba(239, 68, 68, 0.5)';
    document.getElementById('description').focus();
    return;
  }
  
  // Show loading
  document.getElementById('loading').style.display = 'block';
  document.getElementById('submitBtn').disabled = true;
  
  try {
    // Prepare issue data
    const issueData = {
      description,
      screenshot: currentScreenshot,
      context: currentContext,
      scenario: currentContext?.scenario || 'unknown',
      url: currentContext?.url || 'unknown'
    };
    
    // Submit issue
    const response = await window.electronAPI.submitIssue(issueData);
    
    // Spawn agent if requested
    if (agentType !== 'none') {
      await window.electronAPI.spawnAgent(response, agentType);
    }
    
    // Show success
    document.getElementById('loading').style.display = 'none';
    document.getElementById('successMessage').style.display = 'block';
    
    // Auto-hide after success
    setTimeout(() => {
      window.electronAPI.hideOverlay();
      resetForm();
    }, 2000);
    
  } catch (error) {
    console.error('Submit failed:', error);
    document.getElementById('loading').style.display = 'none';
    document.getElementById('submitBtn').disabled = false;
    
    // Show error (simplified)
    document.getElementById('screenshotStatus').textContent = 'Submission failed';
    document.getElementById('screenshotStatus').style.color = '#ef4444';
  }
}

// Reset form
function resetForm() {
  document.getElementById('description').value = '';
  document.getElementById('description').style.borderColor = '';
  currentScreenshot = null;
  currentContext = null;
  document.getElementById('screenshotPreview').innerHTML = '📸';
  document.getElementById('screenshotPreview').style.background = '';
  document.getElementById('screenshotStatus').textContent = 'No screenshot captured';
  document.getElementById('scenarioName').textContent = 'Press button to capture';
  document.getElementById('successMessage').style.display = 'none';
  document.getElementById('submitBtn').disabled = false;
}

// Hide overlay
function hideOverlay() {
  window.electronAPI.hideOverlay();
  resetForm();
}

// Open history
function openHistory() {
  // This would open a separate window - simplified for now
  console.log('Opening history...');
}

// Open settings
function openSettings() {
  // This would open a separate window - simplified for now
  console.log('Opening settings...');
}

// Set up keyboard shortcuts
function setupKeyboardShortcuts() {
  document.addEventListener('keydown', (e) => {
    // Escape to close
    if (e.key === 'Escape') {
      hideOverlay();
    }
    
    // Cmd/Ctrl + Enter to submit
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      submitIssue();
    }
    
    // Cmd/Ctrl + S for screenshot
    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault();
      captureScreenshot();
    }
  });
}

// Auto-resize textarea
const textarea = document.getElementById('description');
if (textarea) {
  textarea.addEventListener('input', () => {
    // Reset border color on input
    textarea.style.borderColor = '';
  });
}