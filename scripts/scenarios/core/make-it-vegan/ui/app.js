// API configuration
const API_BASE = 'http://localhost:8080/api';

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
    "Fun fact: Chia seeds can make a protein-rich pudding!"
];

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
    
    showLoading(resultDiv);
    
    try {
        const response = await fetch(`${API_BASE}/check`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ingredients: input })
        });
        
        const data = await response.json();
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
});