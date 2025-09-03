# App Monitor - Product Requirements Document

## Overview
VROOLI App Monitor is a comprehensive monitoring dashboard for managing and overseeing all running scenarios within the Vrooli ecosystem. The system provides real-time monitoring, control capabilities, and performance insights through a Matrix-themed cyberpunk interface.

## Core Features

### 1. Real-time Application Monitoring
- Live status tracking for all running applications/scenarios
- Performance metrics (CPU, Memory, Network, Disk usage)
- Health check monitoring with automated alerts
- Container-level monitoring through Docker integration

### 2. Application Control Interface
- Start/Stop/Restart application controls
- Quick batch operations (restart all, stop all)
- Individual application detail views with comprehensive information
- Real-time log streaming and filtering

### 3. Resource Management
- Monitor status of core resources (PostgreSQL, Redis, n8n, Node-RED, Ollama)
- Visual resource health indicators
- Integration status tracking

### 4. Interactive Terminal
- Built-in command interface for system operations
- Command history and auto-completion
- Direct integration with app control functions

### 5. Performance Analytics
- Real-time charts and metrics visualization
- Historical performance data
- System-wide health monitoring dashboard

## UI Design Specifications

### Design Theme: Matrix Cyberpunk
The application employs a distinctive Matrix-inspired cyberpunk aesthetic that creates an immersive monitoring experience.

#### Color Palette
- **Primary Green**: `#00ff41` (Matrix green) - Used for primary text, borders, and accents
- **Dark Green**: `#008f11` - Used for secondary elements and hover states
- **Light Green**: `#39ff14` - Used for highlighting and active states
- **Cyan**: `#00ffff` - Used for section headers and special elements
- **Background**: `#0a0a0a` - Primary dark background
- **Card Background**: `#0d1117` - Secondary background for content areas
- **Border**: `#1a472a` - Green-tinted borders
- **Red**: `#ff0040` - Error states and critical alerts
- **Yellow**: `#ffb000` - Warning states
- **Blue**: `#00a8ff` - Information states

#### Typography
- **Primary Font**: `Share Tech Mono` - Monospace font for the authentic terminal feel
- **Logo Font**: `Orbitron` - Futuristic font for headers and branding
- **Font Weights**: 400 (regular), 700 (bold), 900 (heavy) for logo

#### Visual Effects

**Matrix Rain Animation**
- Subtle animated background effect with falling matrix-style characters
- Low opacity overlay that doesn't interfere with content readability
- Creates atmospheric depth and reinforces the cyberpunk theme

**Glow Effects**
- Text shadows with green glow for key elements
- Box shadows on interactive elements during hover states
- Pulsing animations for active status indicators

**Scan Lines and Grid**
- Subtle scan line animation in the header
- Grid patterns in background overlays
- Linear gradient effects for depth

#### Layout Structure

**Header**
- Full-width header with brand identity
- Real-time system status indicators (System Status, App Count, Uptime)
- Animated scan line effect across bottom border

**Sidebar Navigation**
- Fixed-width navigation panel (250px)
- Menu items with hover states and active indicators
- Quick action buttons for common operations
- Collapsible on mobile devices

**Main Content Area**
- Flexible content area with multiple view panels
- Smooth transitions between different views
- Responsive grid layouts for content cards

**Modal System**
- Full-screen modal overlays for detailed views
- Consistent styling with main interface
- Smooth fade-in/fade-out animations

#### Component Specifications

**Application Cards**
- Grid-based layout with responsive columns
- Status indicators with color-coded badges
- Real-time metrics display (CPU, Memory, Uptime, Port)
- Hover effects with border highlighting and glow
- Integrated action buttons (Start, Stop, Details)

**Status Indicators**
- Running: Green background with green border
- Stopped: Red background with red border  
- Error: Yellow background with yellow border
- Online: Green for resources
- Offline: Red for resources

**Interactive Elements**
- Consistent button styling with green borders
- Hover states with background color inversion
- Uppercase text for technical aesthetic
- Smooth transition animations (0.3s ease)

**Charts and Visualizations**
- Canvas-based charts with green color scheme
- Real-time updating data visualization
- Grid lines and technical styling
- Consistent with overall theme

**Terminal Interface**
- Full terminal emulation with command processing
- Green text on dark background
- Authentic terminal prompt styling
- Command history and response formatting

#### Responsive Design
- Mobile-first approach with collapsible sidebar
- Responsive grid systems that adapt to screen size
- Touch-friendly interface elements on mobile
- Maintained theme consistency across all devices

#### Animation and Interaction
- Smooth page transitions with fade effects
- Loading spinners with Matrix-green color
- Hover animations on all interactive elements
- Real-time updates without jarring transitions

### User Experience Principles
1. **Immediate Clarity**: Status and health information is immediately visible
2. **Efficient Control**: Common operations are easily accessible
3. **Professional Aesthetic**: The Matrix theme enhances rather than distracts from functionality
4. **Real-time Updates**: Information updates seamlessly without user intervention
5. **Responsive Design**: Consistent experience across all device sizes

## Technical Architecture

### Frontend Stack
- **Vanilla JavaScript**: No framework dependencies for lightweight performance
- **Custom CSS**: Matrix-themed styling with CSS custom properties
- **WebSocket Integration**: Real-time updates from backend
- **Canvas API**: Custom chart implementations
- **Local Storage**: User preferences and session data

### Backend Integration
- **Node.js Server**: Express-based server for UI hosting
- **API Proxy**: Transparent proxy to Go API backend
- **WebSocket Server**: Real-time communication with clients
- **Docker Integration**: Container monitoring and control

### Data Flow
1. UI connects to Node.js server
2. API calls proxied to Go backend
3. Real-time updates via WebSocket
4. Mock data fallbacks for development

## Browser Compatibility
- Modern browsers with ES6+ support
- WebSocket support required
- Canvas API support for charts
- CSS Grid and Flexbox support

## Performance Requirements
- Initial page load: < 2 seconds
- Real-time update latency: < 500ms  
- Responsive interactions: < 100ms
- Memory usage: < 50MB typical

This PRD serves as the definitive specification for maintaining and enhancing the App Monitor's distinctive Matrix cyberpunk interface while ensuring robust monitoring functionality.