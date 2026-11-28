<?php
// Offline unit harness for the MU-plugin form adapters. WordPress and external
// services are replaced by local stubs; this does not claim plugin runtime proof.
define('ABSPATH', __DIR__);
define('ARRAY_A', 'ARRAY_A');

$registered_hooks = [];
$webhook_url = 'http://n8n:5678/webhook/wordpress-form-intake';
$remote_posts = [];

function add_action($hook, $callback, $priority = 10, $accepted_args = 1): void {
	$GLOBALS['registered_hooks'][$hook] = [$callback, $priority, $accepted_args];
}
function add_filter($hook, $callback, $priority = 10, $accepted_args = 1): void {
	$GLOBALS['registered_hooks'][$hook] = [$callback, $priority, $accepted_args];
}
function sanitize_title(string $value): string {
	$value = strtolower(trim($value));
	return preg_replace('/[^a-z0-9_-]+/', '-', $value) ?? '';
}
function get_option(string $name, mixed $default = false): mixed {
	if ($name === 'vrooli_form_n8n_webhook') {
		return $GLOBALS['webhook_url'];
	}
	if ($name === 'vrooli_delivery_destination_schema') {
		return '1';
	}
	return $default;
}
function update_option(string $name, mixed $value, mixed $autoload = null): bool {
	return true;
}
function wp_parse_url(string $url): array|false {
	return parse_url($url);
}
function wp_json_encode(mixed $value, int $flags = 0): string|false {
	return json_encode($value, $flags);
}
function is_email(string $value): string|false {
	return filter_var($value, FILTER_VALIDATE_EMAIL) ? $value : false;
}
final class WP_REST_Server {
	public const CREATABLE = 'POST';
}
final class WP_Error {
	public function __construct(public string $code, public string $message, public array $data = []) {}
}
final class WP_REST_Response {
	public function __construct(public mixed $data, public int $status = 200) {}
}
final class WP_REST_Request {
	public function __construct(private array $data) {}
	public function get_json_params(): array { return $this->data; }
	public function get_body(): string { return json_encode($this->data, JSON_UNESCAPED_UNICODE) ?: ''; }
}
final class SyntheticWpdb {
	public string $prefix = 'wp_';
	private array $events = [];
	public function prepare(string $sql, mixed ...$args): array { return ['sql' => $sql, 'args' => $args]; }
	public function get_var(array $query): string { return $this->prefix . 'vrooli_delivery_events'; }
	public function get_charset_collate(): string { return ''; }
	public function query(array $query): int|false {
		$args = $query['args'];
		$key = (string) ($args[0] ?? '');
		if (array_key_exists($key, $this->events)) {
			return 0;
		}
		$this->events[$key] = (string) ($args[2] ?? '');
		return 1;
	}
	public function get_row(array $query, mixed $format = null): array|false {
		$key = (string) ($query['args'][0] ?? '');
		return array_key_exists($key, $this->events) ? ['payload_sha256' => $this->events[$key]] : false;
	}
}
function wp_remote_post(string $url, array $args): array {
	$GLOBALS['remote_posts'][] = ['url' => $url, 'args' => $args];
	return ['response' => ['code' => 200]];
}

final class WPCF7_Submission {
	public static array $posted = [];
	public static function get_instance(): self { return new self(); }
	public function get_posted_data(): array { return self::$posted; }
}
final class SyntheticContactForm {
	public function id(): int { return 73; }
}

require __DIR__ . '/vrooli-delivery-destination.php';
Vrooli_Delivery_Destination::boot();
$checks = [];
$check = static function (string $name, bool $passed) use (&$checks): void {
	$checks[] = ['name' => $name, 'passed' => $passed];
};

$textLength = new ReflectionMethod(Vrooli_Delivery_Destination::class, 'text_length');
$check('destination field limits count Unicode code points rather than UTF-8 bytes',
	$textLength->invoke(null, str_repeat('é', 101)) === 101
	&& $textLength->invoke(null, str_repeat('😀', 2001)) === 2001
	&& $textLength->invoke(null, str_repeat('😀', 4001)) === 4001);

$check('CF7 and WPForms success hooks registered with documented arity',
	isset($registered_hooks['wpcf7_mail_sent'], $registered_hooks['wpforms_process_complete'])
	&& $registered_hooks['wpcf7_mail_sent'][2] === 1
	&& $registered_hooks['wpforms_process_complete'][2] === 4);

WPCF7_Submission::$posted = ['event_id' => 'cf7-event-001', 'your-name' => 'Synthetic Reader', 'your-email' => 'reader@example.invalid', 'your-message' => 'Synthetic form fixture'];
Vrooli_Delivery_Destination::capture_contact_form_7(new SyntheticContactForm());
$cf7 = json_decode($remote_posts[0]['args']['body'] ?? '', true);
$check('CF7 forwards the successful submission with stable provider/form identity',
	($cf7['provider'] ?? '') === 'contact-form-7' && ($cf7['form_id'] ?? '') === '73'
	&& ($cf7['posted_data']['event_id'] ?? '') === 'cf7-event-001');

Vrooli_Delivery_Destination::capture_wpforms_lite([
	['name' => 'Event ID', 'value' => 'wpforms-event-001'],
	['name' => 'Full Name', 'value' => 'Synthetic Reader'],
	['name' => 'Email', 'value' => 'reader@example.invalid'],
	['name' => 'Message', 'value' => 'Synthetic form fixture'],
	['name' => 'Unmapped private field', 'value' => 'must not be forwarded'],
], [], ['id' => 29], 0);
$wpforms = json_decode($remote_posts[1]['args']['body'] ?? '', true);
$field_map = [];
foreach (($wpforms['fields'] ?? []) as $field) {
	$field_map[$field['name']] = $field['value'];
}
$check('WPForms maps documented fields and does not rely on disabled-entry ID',
	($wpforms['provider'] ?? '') === 'wpforms-lite' && ($wpforms['entry_id'] ?? -1) === 0
	&& ($wpforms['form_data']['id'] ?? '') === '29' && ($field_map['event_id'] ?? '') === 'wpforms-event-001'
	&& ($field_map['full-name'] ?? '') === 'Synthetic Reader'
	&& ($field_map['email'] ?? '') === 'reader@example.invalid'
	&& ($field_map['message'] ?? '') === 'Synthetic form fixture'
	&& !array_key_exists('Unmapped private field', $field_map));

$accepted_url = $remote_posts[0]['url'] ?? '';
$check('form adapter sends the canonical JSON payload only to the fixed local n8n endpoint',
	$accepted_url === 'http://n8n:5678/webhook/wordpress-form-intake'
	&& ($remote_posts[0]['args']['headers']['Content-Type'] ?? '') === 'application/json'
	&& ($remote_posts[0]['args']['redirection'] ?? -1) === 0);

$before_bad_url = count($remote_posts);
$blocked_urls = [
	'https://external.example.invalid/webhook/wordpress-form-intake',
	'http://n8n:5678/webhook/wordpress-form-intake?token=synthetic',
	'http://attacker@n8n:5678/webhook/wordpress-form-intake',
	'http://n8n:5679/webhook/wordpress-form-intake',
	'http://n8n:5678/foreign-path',
];
foreach ($blocked_urls as $url) {
	$GLOBALS['webhook_url'] = $url;
	Vrooli_Delivery_Destination::capture_contact_form_7(new SyntheticContactForm());
}
$check('external, credential-bearing, query-bearing, wrong-port and wrong-path webhook targets are refused', count($remote_posts) === $before_bad_url);

$ports = Vrooli_Delivery_Destination::allow_local_n8n_webhook_port([443, '5678']);
$check('Forminator local webhook filter allows only the fixed port without duplicates', $ports === [443, 5678]);

$GLOBALS['wpdb'] = new SyntheticWpdb();
$unicodeEvent = [
	'provider' => 'contact-form-7',
	'formId' => 'unicode-form',
	'eventId' => 'unicode-event',
	'name' => str_repeat('😀', 101),
	'email' => 'unicode@example.invalid',
	'message' => str_repeat('😀', 2001),
];
$created = Vrooli_Delivery_Destination::receive_form(new WP_REST_Request($unicodeEvent));
$replayed = Vrooli_Delivery_Destination::receive_form(new WP_REST_Request($unicodeEvent));
$changedEvent = $unicodeEvent;
$changedEvent['message'] .= 'changed';
$conflict = Vrooli_Delivery_Destination::receive_form(new WP_REST_Request($changedEvent));
$check('authenticated destination contract accepts, replays and conflicts Unicode events by codepoint limits',
	$created instanceof WP_REST_Response && $created->status === 201
	&& $replayed instanceof WP_REST_Response && $replayed->status === 200
	&& $conflict instanceof WP_Error && ($conflict->data['status'] ?? 0) === 409);

$oversizedName = $unicodeEvent;
$oversizedName['eventId'] = 'unicode-name-too-long';
$oversizedName['name'] = str_repeat('😀', 201);
$oversizedMessage = $unicodeEvent;
$oversizedMessage['eventId'] = 'unicode-message-too-long';
$oversizedMessage['message'] = str_repeat('😀', 4001);
$nameRejected = Vrooli_Delivery_Destination::receive_form(new WP_REST_Request($oversizedName));
$messageRejected = Vrooli_Delivery_Destination::receive_form(new WP_REST_Request($oversizedMessage));
$check('destination rejects Unicode name and message above their codepoint caps',
	$nameRejected instanceof WP_Error && ($nameRejected->data['status'] ?? 0) === 422
	&& $messageRejected instanceof WP_Error && ($messageRejected->data['status'] ?? 0) === 422);

$integrationSummary = null;
$integrationInput = trim((string) file_get_contents('php://stdin'));
if ($integrationInput !== '') {
	$integrationCases = json_decode($integrationInput, true);
	if (!is_array($integrationCases)) {
		throw new RuntimeException('integrated destination input must be a JSON array');
	}
	$integrationResults = [];
	foreach ($integrationCases as $case) {
		$id = (string) ($case['id'] ?? '');
		$event = $case['event'] ?? null;
		$wantedStatus = (int) ($case['expectedStatus'] ?? 0);
		if ($id === '' || !is_array($event) || $wantedStatus === 0) {
			throw new RuntimeException('integrated destination case is missing id, event or expectedStatus');
		}
		$response = Vrooli_Delivery_Destination::receive_form(new WP_REST_Request($event));
		$status = $response instanceof WP_REST_Response
			? $response->status
			: ($response instanceof WP_Error ? (int) ($response->data['status'] ?? 0) : 0);
		$integrationResults[] = ['id' => $id, 'expectedStatus' => $wantedStatus, 'actualStatus' => $status, 'passed' => $status === $wantedStatus];
	}
	$integrationFailed = array_values(array_filter($integrationResults, static fn(array $item): bool => !$item['passed']));
	$integrationSummary = ['total' => count($integrationResults), 'passed' => count($integrationResults) - count($integrationFailed), 'failed' => $integrationFailed, 'cases' => $integrationResults];
	$check('n8n-normalized event sequence matches destination create/replay/conflict status contract', $integrationFailed === []);
}

$failed = array_values(array_filter($checks, static fn(array $item): bool => !$item['passed']));
echo json_encode(['total' => count($checks), 'passed' => count($checks) - count($failed), 'failed' => $failed, 'integration' => $integrationSummary], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE) . PHP_EOL;
exit($failed === [] ? 0 : 1);
