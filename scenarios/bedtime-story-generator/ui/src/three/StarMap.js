import * as THREE from "three";

/**
 * Procedural star map that evolves with reading history
 * Each story read adds a new star to the constellation
 * Stars are connected based on story themes and reading patterns
 */
export default class StarMap {
  constructor({ experience, readingHistory = [] }) {
    this.experience = experience;
    this.scene = experience.scene;
    this.readingHistory = readingHistory;
    
    // Create container for star map
    this.group = new THREE.Group();
    this.group.position.set(0, 3.5, -4);
    this.scene.add(this.group);
    
    // Store stars and connections
    this.stars = [];
    this.connections = [];
    this.constellations = new Map();
    
    // Theme colors for constellations
    this.themeColors = {
      Adventure: "#ff6b6b",
      Magic: "#a855f7", 
      Space: "#3b82f6",
      Ocean: "#06b6d4",
      Animals: "#84cc16",
      Friendship: "#f59e0b",
      Mystery: "#8b5cf6",
      Fantasy: "#ec4899",
      Science: "#10b981",
      Forest: "#059669"
    };
    
    // Initialize star map
    this._createBackground();
    this._generateStarsFromHistory();
    this._createConnections();
    this._setupAnimations();
  }
  
  _createBackground() {
    // Create a dark ethereal background plane for the star map
    const bgGeometry = new THREE.PlaneGeometry(6, 3);
    const bgMaterial = new THREE.MeshBasicMaterial({
      color: "#0a0e27",
      transparent: true,
      opacity: 0.8,
      side: THREE.DoubleSide
    });
    this.background = new THREE.Mesh(bgGeometry, bgMaterial);
    this.background.position.z = -0.5;
    this.group.add(this.background);
    
    // Add subtle glow effect
    const glowGeometry = new THREE.PlaneGeometry(6.2, 3.2);
    const glowMaterial = new THREE.MeshBasicMaterial({
      color: "#1e3a8a",
      transparent: true,
      opacity: 0.3,
      side: THREE.DoubleSide
    });
    const glow = new THREE.Mesh(glowGeometry, glowMaterial);
    glow.position.z = -0.51;
    this.group.add(glow);
  }
  
  _generateStarsFromHistory() {
    // Create a star for each story in reading history
    this.readingHistory.forEach((story, index) => {
      const star = this._createStar(story, index);
      this.stars.push(star);
      this.group.add(star.mesh);
      
      // Group stars by theme
      if (!this.constellations.has(story.theme)) {
        this.constellations.set(story.theme, []);
      }
      this.constellations.get(story.theme).push(star);
    });
  }
  
  _createStar(story, index) {
    // Calculate position based on reading order and theme
    const angle = (index / this.readingHistory.length) * Math.PI * 2;
    const radius = 1 + Math.random() * 1.5;
    const x = Math.cos(angle) * radius;
    const y = Math.sin(angle) * radius * 0.6;
    
    // Star geometry - size based on times read
    const size = 0.05 + (story.timesRead || 1) * 0.02;
    const starGeometry = new THREE.OctahedronGeometry(size);
    
    // Star material - color based on theme
    const color = this.themeColors[story.theme] || "#ffffff";
    const starMaterial = new THREE.MeshBasicMaterial({
      color: color,
      emissive: color,
      emissiveIntensity: 0.5
    });
    
    const starMesh = new THREE.Mesh(starGeometry, starMaterial);
    starMesh.position.set(x, y, 0);
    
    // Add glow effect
    const glowGeometry = new THREE.SphereGeometry(size * 2);
    const glowMaterial = new THREE.MeshBasicMaterial({
      color: color,
      transparent: true,
      opacity: 0.3
    });
    const glowMesh = new THREE.Mesh(glowGeometry, glowMaterial);
    starMesh.add(glowMesh);
    
    // Create point light for star
    const light = new THREE.PointLight(color, 0.5, 1);
    starMesh.add(light);
    
    return {
      mesh: starMesh,
      story: story,
      position: new THREE.Vector3(x, y, 0),
      theme: story.theme,
      color: color,
      size: size
    };
  }
  
  _createConnections() {
    // Connect stars within the same theme constellation
    this.constellations.forEach((stars, theme) => {
      if (stars.length > 1) {
        for (let i = 0; i < stars.length - 1; i++) {
          const connection = this._createConnection(stars[i], stars[i + 1]);
          this.connections.push(connection);
          this.group.add(connection);
        }
        
        // Create additional connections for larger constellations
        if (stars.length > 3) {
          const connection = this._createConnection(stars[0], stars[stars.length - 1]);
          this.connections.push(connection);
          this.group.add(connection);
        }
      }
    });
    
    // Create cross-theme connections for stories read in sequence
    for (let i = 0; i < this.stars.length - 1; i++) {
      if (this.stars[i].theme !== this.stars[i + 1].theme) {
        const connection = this._createConnection(this.stars[i], this.stars[i + 1], true);
        this.connections.push(connection);
        this.group.add(connection);
      }
    }
  }
  
  _createConnection(star1, star2, isDimmer = false) {
    const points = [star1.position, star2.position];
    const geometry = new THREE.BufferGeometry().setFromPoints(points);
    
    const material = new THREE.LineBasicMaterial({
      color: isDimmer ? "#4a5568" : "#60a5fa",
      transparent: true,
      opacity: isDimmer ? 0.2 : 0.4
    });
    
    return new THREE.Line(geometry, material);
  }
  
  _setupAnimations() {
    // Store animation state
    this.animationTime = 0;
    
    // Twinkle effect for stars
    this.stars.forEach((star, index) => {
      star.baseScale = star.mesh.scale.x;
      star.twinkleOffset = Math.random() * Math.PI * 2;
      star.twinkleSpeed = 1 + Math.random() * 2;
    });
  }
  
  update(deltaTime) {
    this.animationTime += deltaTime;
    
    // Animate stars twinkling
    this.stars.forEach((star) => {
      const twinkle = Math.sin(this.animationTime * star.twinkleSpeed + star.twinkleOffset) * 0.1 + 1;
      star.mesh.scale.setScalar(star.baseScale * twinkle);
      
      // Rotate stars slowly
      star.mesh.rotation.y += deltaTime * 0.5;
      star.mesh.rotation.z += deltaTime * 0.3;
    });
    
    // Gentle rotation of entire star map
    this.group.rotation.z = Math.sin(this.animationTime * 0.1) * 0.02;
    
    // Pulse connection opacity
    this.connections.forEach((connection, index) => {
      const pulse = Math.sin(this.animationTime * 0.5 + index * 0.5) * 0.1 + 0.3;
      connection.material.opacity = pulse;
    });
  }
  
  addNewStar(story) {
    // Add a new star when a new story is read
    const star = this._createStar(story, this.stars.length);
    this.stars.push(star);
    this.group.add(star.mesh);
    
    // Animate star appearance
    star.mesh.scale.set(0, 0, 0);
    const targetScale = star.baseScale || 1;
    
    // Simple scale animation (would use GSAP in real implementation)
    let progress = 0;
    const animate = () => {
      progress += 0.05;
      if (progress < 1) {
        star.mesh.scale.setScalar(targetScale * progress);
        requestAnimationFrame(animate);
      } else {
        star.mesh.scale.setScalar(targetScale);
      }
    };
    animate();
    
    // Update connections
    this._updateConnections(star);
  }
  
  _updateConnections(newStar) {
    // Find stars with same theme
    const sameThemeStars = this.stars.filter(s => 
      s.theme === newStar.theme && s !== newStar
    );
    
    if (sameThemeStars.length > 0) {
      // Connect to nearest star of same theme
      const nearest = sameThemeStars[sameThemeStars.length - 1];
      const connection = this._createConnection(nearest, newStar);
      this.connections.push(connection);
      this.group.add(connection);
    }
    
    // Connect to previous star in reading order
    if (this.stars.length > 1) {
      const previous = this.stars[this.stars.length - 2];
      if (previous.theme !== newStar.theme) {
        const connection = this._createConnection(previous, newStar, true);
        this.connections.push(connection);
        this.group.add(connection);
      }
    }
  }
  
  highlightConstellation(theme) {
    // Highlight all stars of a specific theme
    this.stars.forEach(star => {
      if (star.theme === theme) {
        star.mesh.material.emissiveIntensity = 1;
        star.mesh.scale.setScalar(star.baseScale * 1.5);
      } else {
        star.mesh.material.emissiveIntensity = 0.2;
        star.mesh.scale.setScalar(star.baseScale * 0.8);
      }
    });
  }
  
  resetHighlight() {
    // Reset all stars to normal state
    this.stars.forEach(star => {
      star.mesh.material.emissiveIntensity = 0.5;
      star.mesh.scale.setScalar(star.baseScale);
    });
  }
  
  dispose() {
    // Clean up resources
    this.stars.forEach(star => {
      star.mesh.geometry.dispose();
      star.mesh.material.dispose();
    });
    
    this.connections.forEach(connection => {
      connection.geometry.dispose();
      connection.material.dispose();
    });
    
    this.background.geometry.dispose();
    this.background.material.dispose();
    
    this.scene.remove(this.group);
  }
}