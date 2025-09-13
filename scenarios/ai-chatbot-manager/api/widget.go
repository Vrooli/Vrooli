package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

// ServeWidgetJS serves the widget JavaScript file
func (s *Server) ServeWidgetJS(w http.ResponseWriter, r *http.Request) {
	// Read the widget.js file
	widgetPath := filepath.Join("static", "widget.js")
	content, err := ioutil.ReadFile(widgetPath)
	if err != nil {
		s.logger.Printf("Failed to read widget.js: %v", err)
		http.Error(w, "Widget script not found", http.StatusNotFound)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Header().Set("Access-Control-Allow-Origin", "*")     // Allow cross-origin requests
	w.Write(content)
}

// GenerateWidgetEmbedCode generates embeddable widget code for a chatbot
func (s *Server) GenerateWidgetEmbedCode(chatbotID string, widgetConfig map[string]interface{}) string {
	// Generate comprehensive widget embed code
	configJSON, _ := json.Marshal(widgetConfig)

	// Get the API URL from config (falls back to placeholder for production)
	apiURL := s.config.APIBaseURL
	if apiURL == "" {
		// For local development, construct from port
		if s.config.APIPort != "" {
			apiURL = "http://localhost:" + s.config.APIPort
		} else {
			apiURL = "YOUR_API_URL" // Placeholder for production
		}
	}

	// Generate a cleaner embed code that loads the external widget script
	return fmt.Sprintf(`<!-- AI Chatbot Manager Widget -->
<!-- 
	Integration Instructions:
	1. Replace YOUR_API_URL with your actual API server URL
	2. The widget will automatically appear in the bottom-right corner
	3. Customize appearance using the config object
-->

<!-- Step 1: Configure your API URL -->
<script>
	window.CHATBOT_API_URL = '%s'; // IMPORTANT: Change this to your API server URL
</script>

<!-- Step 2: Load the widget library -->
<script src="%s/api/v1/widget.js"></script>

<!-- Step 3: Initialize the widget -->
<script>
	// Initialize the chatbot widget when DOM is ready
	(function() {
		function initWidget() {
			var widget = window.AIChatbotWidget({
				chatbotId: '%s',
				config: %s,
				apiUrl: window.CHATBOT_API_URL
			});
			
			// Store widget instance for programmatic control
			window.chatbotWidget = widget;
		}
		
		if (document.readyState === 'loading') {
			document.addEventListener('DOMContentLoaded', initWidget);
		} else {
			initWidget();
		}
	})();
</script>

<!-- 
	Optional: Programmatic Control
	After initialization, you can control the widget:
	
	window.chatbotWidget.open();     // Open chat window
	window.chatbotWidget.close();    // Close chat window
	window.chatbotWidget.sendMessage('Hello!'); // Send a message
	window.chatbotWidget.destroy();  // Remove widget from page
-->`, apiURL, apiURL, chatbotID, string(configJSON))
}