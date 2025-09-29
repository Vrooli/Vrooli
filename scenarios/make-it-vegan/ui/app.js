// API configuration - dynamically determine API base URL
const API_BASE = (() => {
    // Check if API URL is provided via environment/config
    if (window.API_URL) return window.API_URL;
    
    // Otherwise, derive from current location
    // The API port is dynamically assigned, typically in the 19xxx range
    // We need to detect it from the UI port pattern
    const currentPort = window.location.port || '35393';
    
    // Map UI port to API port based on Vrooli's dynamic assignment
    // UI ports are typically 35xxx, API ports are 19xxx
    let apiPort;
    if (currentPort.startsWith('353')) {
        // Extract last 2 digits and apply to 191xx pattern
        const suffix = currentPort.slice(-2);
        apiPort = '191' + suffix;
    } else {
        // Fallback to default pattern
        apiPort = '19124';
    }
    
    return `http://localhost:${apiPort}/api`;
})();

// Tab navigation
document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        const tabName = btn.dataset.tab;
        
        // Update active button
        document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        
        // Update active content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(`${tabName}-tab`).classList.add('active');
    });
});

// Fun facts to rotate
const funFacts = [
    "Did you know? Cashews can be made into a creamy cheese sauce!",
    "Fun fact: Aquafaba (chickpea water) can replace egg whites in meringues!",
    "Did you know? Nutritional yeast gives a cheesy flavor to any dish!",
    "Fun fact: Bananas can replace eggs in many baking recipes!",
    "Did you know? Coconut milk makes amazing vegan ice cream!",
    "Fun fact: Flax seeds mixed with water create an egg-like binding agent!",
    "Did you know? Jackfruit has a texture similar to pulled pork!",
    "Fun fact: Dates are nature's caramel when blended!",
    "Did you know? Mushrooms can provide a meaty umami flavor!",
    "Fun fact: Chia seeds can make a protein-rich pudding!",
    "Did you know? One cup of cooked spinach has more calcium than a glass of milk!",
    "Fun fact: Tempeh contains more protein per serving than most meats!",
    "Did you know? Dark chocolate is naturally vegan (check for milk though!)!",
    "Fun fact: Seaweed can give dishes a seafood-like flavor!",
    "Did you know? Maple syrup is a great honey alternative!"
];

// Achievement system
const achievements = {
    firstCheck: { icon: '🌱', title: 'First Steps', desc: 'Checked your first ingredient list' },
    veganMaster: { icon: '🏆', title: 'Vegan Master', desc: 'Found 10 vegan products' },
    swapExpert: { icon: '🔄', title: 'Swap Expert', desc: 'Found 5 vegan alternatives' },
    recipeHero: { icon: '👨‍🍳', title: 'Recipe Hero', desc: 'Veganized your first recipe' },
    explorer: { icon: '🌍', title: 'Plant Explorer', desc: 'Tried all features' },
    streak: { icon: '🔥', title: 'On Fire!', desc: '5 checks in a row were vegan' }
};

// Load achievements from localStorage
let userAchievements = JSON.parse(localStorage.getItem('veganAchievements') || '{}');
let stats = JSON.parse(localStorage.getItem('veganStats') || '{"checks":0,"vegan":0,"swaps":0,"recipes":0,"streak":0}');

// Update achievement display
function updateAchievements() {
    const achievementCount = Object.keys(userAchievements).length;
    if (achievementCount > 0) {
        const badgeContainer = document.getElementById('achievement-badges');
        if (badgeContainer) {
            badgeContainer.innerHTML = Object.entries(userAchievements)
                .map(([key, earned]) => earned ? `<span class="achievement-badge" title="${achievements[key].desc}">${achievements[key].icon}</span>` : '')
                .join('');
        }
    }
}

// Check and award achievements
function checkAchievements() {
    if (stats.checks === 1 && !userAchievements.firstCheck) {
        awardAchievement('firstCheck');
    }
    if (stats.vegan >= 10 && !userAchievements.veganMaster) {
        awardAchievement('veganMaster');
    }
    if (stats.swaps >= 5 && !userAchievements.swapExpert) {
        awardAchievement('swapExpert');
    }
    if (stats.recipes >= 1 && !userAchievements.recipeHero) {
        awardAchievement('recipeHero');
    }
    if (stats.streak >= 5 && !userAchievements.streak) {
        awardAchievement('streak');
    }
    if (stats.checks > 0 && stats.swaps > 0 && stats.recipes > 0 && !userAchievements.explorer) {
        awardAchievement('explorer');
    }
    
    // Save stats
    localStorage.setItem('veganStats', JSON.stringify(stats));
}

// Award achievement with animation
function awardAchievement(key) {
    userAchievements[key] = true;
    localStorage.setItem('veganAchievements', JSON.stringify(userAchievements));
    
    const achievement = achievements[key];
    showAchievementNotification(achievement);
    updateAchievements();
}

// Show achievement notification
function showAchievementNotification(achievement) {
    const notification = document.createElement('div');
    notification.className = 'achievement-notification';
    notification.innerHTML = `
        <span class="achievement-icon">${achievement.icon}</span>
        <div>
            <div class="achievement-title">Achievement Unlocked!</div>
            <div class="achievement-desc">${achievement.title}: ${achievement.desc}</div>
        </div>
    `;
    document.body.appendChild(notification);
    
    setTimeout(() => notification.classList.add('show'), 100);
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => notification.remove(), 500);
    }, 3000);
}

// Rotate fun facts
let factIndex = 0;
function rotateFunFact() {
    const factElement = document.getElementById('fun-fact-text');
    if (factElement) {
        factElement.style.opacity = '0';
        setTimeout(() => {
            factElement.textContent = funFacts[factIndex];
            factElement.style.opacity = '1';
            factIndex = (factIndex + 1) % funFacts.length;
        }, 300);
    }
}
setInterval(rotateFunFact, 8000);

// Check ingredients function
async function checkIngredients() {
    const input = document.getElementById('ingredients-input').value.trim();
    const resultDiv = document.getElementById('check-result');
    
    if (!input) {
        showError(resultDiv, 'Please enter some ingredients to check!');
        return;
    }
    
    showAnimatedScanning(resultDiv);
    
    try {
        const response = await fetch(`${API_BASE}/check`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ingredients: input })
        });
        
        const data = await response.json();
        
        // Update stats
        stats.checks++;
        if (data.isVegan) {
            stats.vegan++;
            stats.streak++;
        } else {
            stats.streak = 0;
        }
        checkAchievements();
        
        displayCheckResult(resultDiv, data);
    } catch (error) {
        showError(resultDiv, 'Unable to check ingredients. Please try again.');
    }
}

// Find alternatives function
async function findAlternatives() {
    const ingredient = document.getElementById('ingredient-input').value.trim();
    const context = document.getElementById('context-input').value.trim();
    const resultDiv = document.getElementById('alternatives-result');
    
    if (!ingredient) {
        showError(resultDiv, 'Please enter an ingredient to find alternatives for!');
        return;
    }
    
    showLoading(resultDiv);
    
    try {
        const response = await fetch(`${API_BASE}/substitute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ingredient, context: context || 'general use' })
        });
        
        const data = await response.json();
        
        // Update stats
        stats.swaps++;
        checkAchievements();
        
        displayAlternatives(resultDiv, data);
    } catch (error) {
        showError(resultDiv, 'Unable to find alternatives. Please try again.');
    }
}

// Quick swap function
async function quickSwap(ingredient) {
    document.getElementById('ingredient-input').value = ingredient;
    document.getElementById('context-input').value = 'general use';
    await findAlternatives();
}

// Veganize recipe function
async function veganizeRecipe() {
    const recipe = document.getElementById('recipe-input').value.trim();
    const resultDiv = document.getElementById('recipe-result');
    
    if (!recipe) {
        showError(resultDiv, 'Please enter a recipe to veganize!');
        return;
    }
    
    showLoading(resultDiv);
    
    try {
        const response = await fetch(`${API_BASE}/veganize`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ recipe })
        });
        
        const data = await response.json();
        
        // Update stats
        stats.recipes++;
        checkAchievements();
        
        displayVeganizedRecipe(resultDiv, data);
    } catch (error) {
        showError(resultDiv, 'Unable to veganize recipe. Please try again.');
    }
}

// Display functions
function displayCheckResult(container, data) {
    const isVegan = data.isVegan;
    const resultClass = isVegan ? 'result-vegan' : 'result-not-vegan';
    const icon = isVegan ? '✅' : '⚠️';
    const title = isVegan ? "It's Vegan!" : "Not Vegan";
    
    let html = `
        <div class="result-box show ${resultClass}">
            <div class="result-title">
                <span>${icon}</span>
                <span>${title}</span>
            </div>
    `;
    
    if (!isVegan && data.nonVeganIngredients && data.nonVeganIngredients.length > 0) {
        html += '<h4>Non-vegan ingredients found:</h4><ul class="ingredient-list">';
        data.nonVeganIngredients.forEach(ingredient => {
            const explanation = data.explanations && data.explanations[ingredient] 
                ? data.explanations[ingredient] 
                : 'Contains animal products';
            html += `
                <li>
                    <span class="ingredient-tag tag-not-vegan">${ingredient}</span>
                    <span>${explanation}</span>
                </li>
            `;
        });
        html += '</ul>';
        html += '<p style="margin-top: 15px;">💡 <strong>Tip:</strong> Switch to the "Find Alternatives" tab to find vegan substitutes!</p>';
    } else if (isVegan) {
        html += '<p>All ingredients are plant-based! Enjoy your vegan meal! 🌱</p>';
        
        // Parse ingredients and show them
        const ingredients = data.ingredients.split(',').map(i => i.trim());
        if (ingredients.length > 0) {
            html += '<h4>Ingredients checked:</h4><ul class="ingredient-list">';
            ingredients.forEach(ingredient => {
                html += `
                    <li>
                        <span class="ingredient-tag tag-vegan">${ingredient}</span>
                        <span>✓ Vegan</span>
                    </li>
                `;
            });
            html += '</ul>';
        }
    }
    
    html += '</div>';
    container.innerHTML = html;
}

function displayAlternatives(container, data) {
    let html = '<div class="result-box show">';
    
    if (data.alternatives && data.alternatives.length > 0) {
        html += `
            <h3>🌱 Vegan Alternatives for ${data.ingredient || 'your ingredient'}</h3>
            <p class="help-text">Context: ${data.context || 'general use'}</p>
        `;
        
        data.alternatives.forEach(alt => {
            html += `
                <div class="alternative-card">
                    <div class="alternative-name">${alt.name || alt}</div>
                    <div class="alternative-desc">${alt.description || 'Great vegan substitute!'}</div>
                    ${alt.tips ? `<p style="margin-top: 8px; font-size: 0.9rem;">💡 ${alt.tips}</p>` : ''}
                </div>
            `;
        });
    } else {
        html += '<p>No alternatives found. Try being more specific with the context!</p>';
    }
    
    html += '</div>';
    container.innerHTML = html;
}

function displayVeganizedRecipe(container, data) {
    let html = '<div class="result-box show">';
    
    if (data.veganRecipe) {
        html += '<h3>✨ Your Veganized Recipe</h3>';
        
        // Show substitutions
        if (data.substitutions && Object.keys(data.substitutions).length > 0) {
            html += '<div class="recipe-section"><h3>🔄 Substitutions Made:</h3>';
            for (const [original, substitute] of Object.entries(data.substitutions)) {
                html += `
                    <div class="substitution-item">
                        <span>${original}</span>
                        <span class="arrow">→</span>
                        <span style="color: var(--primary-green); font-weight: 600;">${substitute}</span>
                    </div>
                `;
            }
            html += '</div>';
        }
        
        // Show the veganized recipe
        html += `
            <div class="recipe-section">
                <h3>📝 Vegan Recipe:</h3>
                <p style="white-space: pre-wrap; line-height: 1.8;">${data.veganRecipe}</p>
            </div>
        `;
        
        // Show cooking tips
        if (data.cookingTips && data.cookingTips.length > 0) {
            html += '<div class="recipe-section"><h3>👨‍🍳 Cooking Tips:</h3><ul style="list-style: none;">';
            data.cookingTips.forEach(tip => {
                html += `<li style="margin: 8px 0;">✓ ${tip}</li>`;
            });
            html += '</ul></div>';
        }
        
        // Show nutrition highlights
        if (data.nutritionHighlights) {
            html += `
                <div class="recipe-section">
                    <h3>🥗 Nutrition Highlights:</h3>
                    <p>${data.nutritionHighlights}</p>
                </div>
            `;
        }
    } else {
        html += '<p>Unable to veganize this recipe. Please try with a simpler format!</p>';
    }
    
    html += '</div>';
    container.innerHTML = html;
}

function showLoading(container) {
    container.innerHTML = `
        <div class="result-box show" style="text-align: center;">
            <div class="loading"></div>
            <p style="margin-top: 10px;">Analyzing...</p>
        </div>
    `;
}

function showAnimatedScanning(container) {
    const scanningPhrases = [
        '🔍 Scanning ingredients...',
        '🌱 Checking plant-based status...',
        '📊 Analyzing nutritional data...',
        '✨ Almost done...'
    ];
    
    let phraseIndex = 0;
    container.innerHTML = `
        <div class="result-box show scanning-animation" style="text-align: center;">
            <div class="scanner-icon">🔍</div>
            <div class="loading"></div>
            <p id="scanning-text" style="margin-top: 10px;">${scanningPhrases[0]}</p>
            <div class="veggie-scanner">
                <span class="scanning-veggie">🥕</span>
                <span class="scanning-veggie">🥦</span>
                <span class="scanning-veggie">🍅</span>
                <span class="scanning-veggie">🥬</span>
            </div>
        </div>
    `;
    
    // Rotate scanning phrases
    const interval = setInterval(() => {
        phraseIndex = (phraseIndex + 1) % scanningPhrases.length;
        const textEl = document.getElementById('scanning-text');
        if (textEl) {
            textEl.style.opacity = '0';
            setTimeout(() => {
                textEl.textContent = scanningPhrases[phraseIndex];
                textEl.style.opacity = '1';
            }, 200);
        } else {
            clearInterval(interval);
        }
    }, 1500);
}

function showError(container, message) {
    container.innerHTML = `
        <div class="result-box show" style="border-left: 5px solid #f44336;">
            <p style="color: #f44336;">❌ ${message}</p>
        </div>
    `;
}

// Initialize on load
document.addEventListener('DOMContentLoaded', () => {
    // Set initial transition for fun fact
    const factElement = document.getElementById('fun-fact-text');
    if (factElement) {
        factElement.style.transition = 'opacity 0.3s ease';
    }
    
    // Load and display achievements
    updateAchievements();
    
    // Add achievement container to header if not exists
    const header = document.querySelector('header');
    if (header && !document.getElementById('achievement-badges')) {
        const achievementDiv = document.createElement('div');
        achievementDiv.id = 'achievement-badges';
        achievementDiv.className = 'achievement-container';
        header.appendChild(achievementDiv);
        updateAchievements();
    }
});