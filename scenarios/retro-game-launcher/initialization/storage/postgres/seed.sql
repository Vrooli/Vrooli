-- Retro Game Launcher Seed Data
-- Initial data for testing and demonstration

-- Insert some starter prompt templates
INSERT INTO prompt_templates (category, template, example_prompt) VALUES
('platformer', 
 'Create a {difficulty} {theme} platformer game where the player {objective}. Include {features}.',
 'Create a medium space platformer game where the player collects energy cores to power their ship. Include moving platforms and enemy robots.'),

('puzzle', 
 'Design a {complexity} puzzle game involving {mechanic} with a {theme} setting. The goal is to {objective}.',
 'Design a complex puzzle game involving block matching with a mystical forest setting. The goal is to clear ancient runes from magical stones.'),

('action', 
 'Build a {pace} action game set in {setting} where the player must {challenge}. Features should include {weapons} and {enemies}.',
 'Build a fast-paced action game set in a cyberpunk city where the player must hack through security systems. Features should include laser weapons and drone enemies.'),

('arcade', 
 'Make a retro-style arcade game inspired by {classic_game} but with {twist}. The scoring system should {scoring}.',
 'Make a retro-style arcade game inspired by Pac-Man but with time manipulation powers. The scoring system should reward speed and combo multipliers.');

INSERT INTO users (username, email, subscription_tier) VALUES
('retro_arcade', 'retro.arcade@vrooli.com', 'pro'),
('demo_user', 'demo@vrooli.com', 'premium'),
('game_creator', 'creator@vrooli.com', 'pro');

INSERT INTO games (title, prompt, description, code, engine, author_id, play_count, rating, tags) VALUES
('Neon Snake', 
 'Create a neon-lit snake game with crunchy 8-bit sound effects and simple controls',
 'Guide a glowing snake through a synthwave grid, collecting energy pellets while avoiding your tail.',
 $$// Minimal neon snake implementation
const canvas = document.createElement('canvas');
canvas.width = 420;
canvas.height = 420;
document.body.appendChild(canvas);
const ctx = canvas.getContext('2d');

let snake = [{ x: 10, y: 10 }];
let direction = { x: 1, y: 0 };
let food = { x: 5, y: 5 };
let speed = 160;

function drawBlock(x, y, color) {
  ctx.fillStyle = color;
  ctx.fillRect(x * 20, y * 20, 18, 18);
}

function update() {
  const head = { x: snake[0].x + direction.x, y: snake[0].y + direction.y };
  snake.unshift(head);

  if (head.x === food.x && head.y === food.y) {
    food = { x: Math.floor(Math.random() * 21), y: Math.floor(Math.random() * 21) };
  } else {
    snake.pop();
  }

  if (head.x < 0 || head.y < 0 || head.x > 20 || head.y > 20 || snake.slice(1).some(s => s.x === head.x && s.y === head.y)) {
    snake = [{ x: 10, y: 10 }];
    direction = { x: 1, y: 0 };
  }
}

function draw() {
  ctx.fillStyle = '#080212';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.strokeStyle = '#1cf2ff';
  ctx.strokeRect(0, 0, canvas.width, canvas.height);
  drawBlock(food.x, food.y, '#ff00ff');
  snake.forEach((segment, index) => drawBlock(segment.x, segment.y, index === 0 ? '#00ff99' : '#00d16a'));
}

document.addEventListener('keydown', event => {
  const keyMap = {
    ArrowUp: { x: 0, y: -1 },
    ArrowDown: { x: 0, y: 1 },
    ArrowLeft: { x: -1, y: 0 },
    ArrowRight: { x: 1, y: 0 }
  };
  const next = keyMap[event.key];
  if (next && (next.x !== -direction.x || next.y !== -direction.y)) {
    direction = next;
  }
});

function loop() {
  update();
  draw();
  setTimeout(() => requestAnimationFrame(loop), speed);
}

loop();
$$,
 'javascript',
 (SELECT id FROM users WHERE username = 'retro_arcade'),
 384,
 4.7,
 ARRAY['arcade', 'snake', 'classic']),

('Pixel Breaker', 
 'Build a brick breaker arcade game with neon particles and responsive paddle controls',
 'Bounce energy orbs to clear progressive waves of neon bricks and chase the global high score.',
 $$const canvas = document.createElement('canvas');
canvas.width = 640;
canvas.height = 480;
document.body.appendChild(canvas);
const ctx = canvas.getContext('2d');

const paddle = { x: 280, y: 440, w: 80, h: 16, speed: 8 };
const ball = { x: 320, y: 320, vx: 4, vy: -4, r: 8 };
const bricks = [];

for (let row = 0; row < 5; row++) {
  for (let col = 0; col < 10; col++) {
    bricks.push({ x: 48 + col * 56, y: 60 + row * 28, w: 48, h: 20, alive: true });
  }
}

function update() {
  ball.x += ball.vx;
  ball.y += ball.vy;

  if (ball.x < ball.r || ball.x > canvas.width - ball.r) ball.vx *= -1;
  if (ball.y < ball.r) ball.vy *= -1;

  if (ball.y > canvas.height) {
    ball.x = 320;
    ball.y = 320;
    ball.vx = 4;
    ball.vy = -4;
  }

  if (ball.y + ball.r >= paddle.y && ball.x >= paddle.x && ball.x <= paddle.x + paddle.w) {
    ball.vy *= -1;
    ball.y = paddle.y - ball.r;
  }

  bricks.forEach(brick => {
    if (!brick.alive) return;
    if (ball.x > brick.x && ball.x < brick.x + brick.w && ball.y - ball.r < brick.y + brick.h && ball.y + ball.r > brick.y) {
      brick.alive = false;
      ball.vy *= -1;
    }
  });
}

function draw() {
  ctx.fillStyle = '#08040c';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = '#0ff';
  ctx.fillRect(paddle.x, paddle.y, paddle.w, paddle.h);
  ctx.fillStyle = '#f0f';
  ctx.beginPath();
  ctx.arc(ball.x, ball.y, ball.r, 0, Math.PI * 2);
  ctx.fill();

  bricks.forEach(brick => {
    if (!brick.alive) return;
    ctx.fillStyle = '#ff0077';
    ctx.fillRect(brick.x, brick.y, brick.w, brick.h);
  });
}

document.addEventListener('mousemove', event => {
  const rect = canvas.getBoundingClientRect();
  paddle.x = Math.max(0, Math.min(canvas.width - paddle.w, event.clientX - rect.left - paddle.w / 2));
});

function loop() {
  update();
  draw();
  requestAnimationFrame(loop);
}

loop();
$$,
 'javascript',
 (SELECT id FROM users WHERE username = 'retro_arcade'),
 256,
 4.6,
 ARRAY['arcade', 'breakout', 'neon']),

('Galactic Defender', 
 'Create a classic space shooter where players blast invading waves and collect power-ups',
 'Battle neon alien squadrons, upgrade your ship mid-flight, and survive escalating cosmic assaults.',
 $$const canvas = document.createElement('canvas');
canvas.width = 720;
canvas.height = 480;
document.body.appendChild(canvas);
const ctx = canvas.getContext('2d');

const ship = { x: 360, y: 420, w: 32, h: 32 };
const lasers = [];
const enemies = [];
let tick = 0;

function spawnWave() {
  for (let i = 0; i < 8; i++) {
    enemies.push({ x: 80 + i * 70, y: 60, w: 28, h: 18, vx: Math.sin(tick / 30 + i) * 2 });
  }
}

function update() {
  tick++;
  ship.x = Math.max(16, Math.min(canvas.width - 48, ship.x));

  lasers.forEach(laser => (laser.y -= 8));
  lasers.filter(laser => laser.y > -20);

  enemies.forEach(enemy => {
    enemy.y += 0.3;
    enemy.x += enemy.vx;
  });

  enemies.forEach(enemy => {
    lasers.forEach(laser => {
      if (!enemy.dead && !laser.dead && Math.abs(laser.x - enemy.x) < 24 && Math.abs(laser.y - enemy.y) < 18) {
        enemy.dead = true;
        laser.dead = true;
      }
    });
  });

  if (tick % 240 === 0) {
    spawnWave();
  }

  if (enemies.every(e => e.dead)) {
    enemies.length = 0;
    spawnWave();
  }
}

function draw() {
  ctx.fillStyle = '#02030b';
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  ctx.fillStyle = '#31f3ff';
  ctx.fillRect(ship.x, ship.y, ship.w, ship.h);

  ctx.fillStyle = '#ff2266';
  lasers.forEach(laser => ctx.fillRect(laser.x, laser.y, 4, 12));

  enemies.forEach(enemy => {
    if (enemy.dead) return;
    ctx.fillStyle = '#8f00ff';
    ctx.fillRect(enemy.x, enemy.y, enemy.w, enemy.h);
  });
}

document.addEventListener('mousemove', event => {
  const rect = canvas.getBoundingClientRect();
  ship.x = event.clientX - rect.left - ship.w / 2;
});

document.addEventListener('click', () => {
  lasers.push({ x: ship.x + ship.w / 2, y: ship.y });
});

function loop() {
  update();
  draw();
  requestAnimationFrame(loop);
}

spawnWave();
loop();
$$,
 'javascript',
 (SELECT id FROM users WHERE username = 'retro_arcade'),
 512,
 4.8,
 ARRAY['shooter', 'space', 'retro']),

('Turbo Tunnels', 
 'Design a top-down hover bike racer dodging tunnels and collecting boost orbs',
 'Dash through endless neon tunnels, weaving through obstacles and chaining boosts for max velocity.',
 $$const canvas = document.createElement('canvas');
canvas.width = 800;
canvas.height = 480;
document.body.appendChild(canvas);
const ctx = canvas.getContext('2d');

const bike = { x: 400, y: 380, vx: 0 };
const obstacles = [];
let distance = 0;

function spawnObstacle() {
  const width = 160 + Math.random() * 200;
  const gap = Math.random() * (canvas.width - width);
  obstacles.push({ gapStart: gap, gapWidth: width, y: -40 });
}

function update() {
  distance += 1;
  bike.x += bike.vx;
  bike.x = Math.max(40, Math.min(canvas.width - 40, bike.x));

  obstacles.forEach(obstacle => {
    obstacle.y += 6;
    if (obstacle.y > canvas.height + 40) obstacle.remove = true;
  });

  obstacles.forEach(obstacle => {
    if (obstacle.y > bike.y - 12 && obstacle.y < bike.y + 12) {
      if (bike.x < obstacle.gapStart || bike.x > obstacle.gapStart + obstacle.gapWidth) {
        distance = 0;
        bike.x = 400;
        bike.vx = 0;
        obstacles.length = 0;
      }
    }
  });

  if (distance % 45 === 0) spawnObstacle();
}

function draw() {
  ctx.fillStyle = '#05031a';
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  ctx.strokeStyle = '#ff007a';
  ctx.lineWidth = 4;
  ctx.beginPath();
  ctx.moveTo(canvas.width / 2 - 40, 0);
  ctx.lineTo(canvas.width / 2 - 80, canvas.height);
  ctx.moveTo(canvas.width / 2 + 40, 0);
  ctx.lineTo(canvas.width / 2 + 80, canvas.height);
  ctx.stroke();

  ctx.fillStyle = '#00f0ff';
  ctx.beginPath();
  ctx.arc(bike.x, bike.y, 18, 0, Math.PI * 2);
  ctx.fill();

  ctx.fillStyle = '#ffef11';
  obstacles.forEach(obstacle => {
    ctx.fillRect(0, obstacle.y, obstacle.gapStart, 24);
    ctx.fillRect(obstacle.gapStart + obstacle.gapWidth, obstacle.y, canvas.width - obstacle.gapStart - obstacle.gapWidth, 24);
  });

  ctx.fillStyle = '#0ff';
  ctx.font = '18px monospace';
  ctx.fillText('Distance: ' + distance, 24, 36);
}

document.addEventListener('keydown', event => {
  if (event.key === 'ArrowLeft') bike.vx = -6;
  if (event.key === 'ArrowRight') bike.vx = 6;
});

document.addEventListener('keyup', event => {
  if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') bike.vx = 0;
});

function loop() {
  update();
  draw();
  requestAnimationFrame(loop);
}

loop();
$$,
 'javascript',
 (SELECT id FROM users WHERE username = 'retro_arcade'),
 298,
 4.5,
 ARRAY['racing', 'endless', 'synthwave']);

-- Insert some high scores
INSERT INTO high_scores (game_id, user_id, score, play_time_seconds) VALUES
((SELECT id FROM games WHERE title = 'Neon Snake'), 
 (SELECT id FROM users WHERE username = 'retro_arcade'), 
 1420, 195),
((SELECT id FROM games WHERE title = 'Pixel Breaker'), 
 (SELECT id FROM users WHERE username = 'game_creator'), 
 8720, 360),
((SELECT id FROM games WHERE title = 'Galactic Defender'), 
 (SELECT id FROM users WHERE username = 'demo_user'), 
 42150, 512);

-- Insert some favorites
INSERT INTO favorites (user_id, game_id) VALUES
((SELECT id FROM users WHERE username = 'retro_arcade'), 
 (SELECT id FROM games WHERE title = 'Galactic Defender')),
((SELECT id FROM users WHERE username = 'demo_user'), 
 (SELECT id FROM games WHERE title = 'Neon Snake'));

-- Insert some generation history
INSERT INTO generation_history (user_id, prompt, game_id, success, generation_time_ms, model_used) VALUES
((SELECT id FROM users WHERE username = 'retro_arcade'),
 'Create a neon-lit snake game with crunchy 8-bit sound effects and simple controls',
 (SELECT id FROM games WHERE title = 'Neon Snake'),
 true, 14600, 'llama3.2'),
((SELECT id FROM users WHERE username = 'retro_arcade'),
 'Build a brick breaker arcade game with neon particles and responsive paddle controls',
 (SELECT id FROM games WHERE title = 'Pixel Breaker'),
 true, 19300, 'codellama'),
((SELECT id FROM users WHERE username = 'demo_user'),
 'Create a classic space shooter where players blast invading waves and collect power-ups',
 (SELECT id FROM games WHERE title = 'Galactic Defender'),
 true, 16500, 'codellama');
