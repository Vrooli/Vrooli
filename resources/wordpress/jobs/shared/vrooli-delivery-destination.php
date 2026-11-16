<?php
/**
 * Plugin Name: Vrooli WordPress Delivery Destination
 * Description: Local, idempotent form and WooCommerce order destinations for synthetic workflow rehearsal.
 * Version: 1.0.0
 * Requires at least: 6.4
 * Requires PHP: 8.1
 *
 * This must be installed as an MU plugin by the WordPress resource owner.
 * It accepts only authenticated local REST writes and stores the supplied
 * canonical event plus its idempotency receipt in a dedicated MySQL table.
 */

if (!defined('ABSPATH')) {
	exit;
}

final class Vrooli_Delivery_Destination {
	private const SCHEMA_VERSION = '1';
	private const MAX_BODY_BYTES = 65536;
	private const TABLE_SUFFIX = 'vrooli_delivery_events';

	public static function boot(): void {
		add_action('rest_api_init', [self::class, 'register_routes']);
		add_action('wpcf7_mail_sent', [self::class, 'capture_contact_form_7'], 10, 1);
		add_action('wpforms_process_complete', [self::class, 'capture_wpforms_lite'], 10, 4);
		add_action('wp_enqueue_scripts', [self::class, 'enqueue_event_id_script']);
		add_filter('forminator_webhook_allowed_ports', [self::class, 'allow_local_n8n_webhook_port']);
	}

	public static function enqueue_event_id_script(): void {
		$path = __DIR__ . '/vrooli-form-event-id.js';
		if (!is_readable($path)) {
			return;
		}
		wp_enqueue_script('vrooli-form-event-id', content_url('mu-plugins/vrooli-form-event-id.js'), ['jquery'], '1.0.0', true);
	}

	public static function allow_local_n8n_webhook_port(array $ports): array {
		$ports[] = 5678;
		return array_values(array_unique(array_map('intval', $ports)));
	}

	public static function capture_contact_form_7($contact_form): void {
		if (!is_object($contact_form) || !method_exists($contact_form, 'id') || !class_exists('WPCF7_Submission')) {
			return;
		}
		$submission = WPCF7_Submission::get_instance();
		if (!is_object($submission) || !method_exists($submission, 'get_posted_data')) {
			return;
		}
		$posted_data = $submission->get_posted_data();
		if (!is_array($posted_data)) {
			return;
		}
		self::forward_form_event([
			'provider' => 'contact-form-7',
			'form_id' => (string) $contact_form->id(),
			'posted_data' => $posted_data,
		]);
	}

	public static function capture_wpforms_lite($fields, $entry, $form_data, $entry_id): void {
		if (!is_array($fields) || !is_array($form_data)) {
			return;
		}
		$normalized_fields = [];
		foreach ($fields as $field) {
			if (!is_array($field)) {
				continue;
			}
			$label = sanitize_title((string) ($field['name'] ?? $field['label'] ?? ''));
			$name = match ($label) {
				'event-id', 'event_id' => 'event_id',
				'full-name', 'name' => 'full-name',
				'email', 'email-address' => 'email',
				'message', 'comment', 'comments' => 'message',
				default => '',
			};
			if ($name !== '') {
				$normalized_fields[] = ['name' => $name, 'value' => $field['value'] ?? ''];
			}
		}
		self::forward_form_event([
			'provider' => 'wpforms-lite',
			'form_data' => ['id' => (string) ($form_data['id'] ?? '')],
			'entry_id' => is_numeric($entry_id) ? (int) $entry_id : 0,
			'fields' => $normalized_fields,
		]);
	}

	private static function forward_form_event(array $payload): void {
		$url = (string) get_option('vrooli_form_n8n_webhook', '');
		if ($url === '' || !self::is_local_n8n_webhook($url)) {
			return;
		}
		wp_remote_post($url, [
			'timeout' => 8,
			'redirection' => 0,
			'blocking' => true,
			'headers' => ['Content-Type' => 'application/json'],
			'body' => wp_json_encode($payload),
		]);
	}

	private static function is_local_n8n_webhook(string $url): bool {
		$parts = wp_parse_url($url);
		return is_array($parts)
			&& ($parts['scheme'] ?? '') === 'http'
			&& ($parts['host'] ?? '') === 'n8n'
			&& (int) ($parts['port'] ?? 5678) === 5678
			&& ($parts['path'] ?? '') === '/webhook/wordpress-form-intake'
			&& !array_key_exists('user', $parts)
			&& !array_key_exists('pass', $parts)
			&& !array_key_exists('query', $parts)
			&& !array_key_exists('fragment', $parts);
	}

	public static function register_routes(): void {
		register_rest_route('vrooli/v1', '/form-intake', [
			'methods' => WP_REST_Server::CREATABLE,
			'callback' => [self::class, 'receive_form'],
			'permission_callback' => [self::class, 'may_write'],
		]);
		register_rest_route('vrooli/v1', '/shop-order-sync', [
			'methods' => WP_REST_Server::CREATABLE,
			'callback' => [self::class, 'receive_order'],
			'permission_callback' => [self::class, 'may_write'],
		]);
	}

	public static function may_write(): bool {
		return current_user_can('manage_options');
	}

	public static function receive_form(WP_REST_Request $request) {
		$event = $request->get_json_params();
		if (!is_array($event) || !self::within_bound($request)) {
			return new WP_Error('invalid_form_event', 'The form event is empty or oversized.', ['status' => 413]);
		}
		$required = ['provider', 'formId', 'eventId', 'name', 'email', 'message'];
		foreach ($required as $field) {
			if (!isset($event[$field]) || !is_string($event[$field]) || trim($event[$field]) === '') {
				return new WP_Error('invalid_form_event', 'The form event is missing a required field.', ['status' => 422]);
			}
		}
		$providers = ['contact-form-7', 'wpforms-lite', 'forminator'];
		if (!in_array($event['provider'], $providers, true)
			|| !self::safe_identity($event['formId'])
			|| !self::safe_identity($event['eventId'])
			|| !is_email($event['email'])
			|| self::text_length($event['name']) > 200
			|| self::text_length($event['email']) > 320
			|| self::text_length($event['message']) > 4000) {
			return new WP_Error('invalid_form_event', 'The form event has an invalid email or field size.', ['status' => 422]);
		}
		$scope = $event['provider'] . "\0" . $event['formId'] . "\0" . $event['eventId'];
		return self::store('form', hash('sha256', $scope), $event);
	}

	public static function receive_order(WP_REST_Request $request) {
		$order = $request->get_json_params();
		if (!is_array($order) || !self::within_bound($request)) {
			return new WP_Error('invalid_order_event', 'The order event is empty or oversized.', ['status' => 413]);
		}
		foreach (['siteId', 'orderId', 'revision', 'status'] as $field) {
			if (!isset($order[$field]) || !is_string($order[$field]) || trim($order[$field]) === '') {
				return new WP_Error('invalid_order_event', 'The order event is missing a required field.', ['status' => 422]);
			}
		}
		if (!self::safe_identity($order['siteId']) || !self::safe_identity($order['orderId']) || !self::safe_revision($order['revision'])) {
			return new WP_Error('invalid_order_event', 'The order identity or revision is invalid.', ['status' => 422]);
		}
		$site_id = (string) get_option('vrooli_resource_site_id', '');
		if ($site_id === '' || !hash_equals($site_id, $order['siteId'])) {
			return new WP_Error('wrong_site', 'The order belongs to a different WordPress recipe site.', ['status' => 409]);
		}
		if (!in_array($order['status'], ['pending', 'processing', 'on-hold', 'completed', 'cancelled', 'refunded', 'failed'], true)) {
			return new WP_Error('invalid_order_event', 'The order status is not supported.', ['status' => 422]);
		}
		if (isset($order['customerEmail']) && (!is_string($order['customerEmail']) || !str_ends_with(strtolower($order['customerEmail']), '.invalid') || !is_email($order['customerEmail']))) {
			return new WP_Error('invalid_order_event', 'Ground orders must use reserved synthetic email addresses.', ['status' => 422]);
		}
		if (!isset($order['items']) || !is_array($order['items']) || count($order['items']) === 0 || count($order['items']) > 100) {
			return new WP_Error('invalid_order_event', 'The order event requires between one and 100 line items.', ['status' => 422]);
		}
		foreach ($order['items'] as $item) {
			if (!is_array($item) || !isset($item['sku']) || !is_string($item['sku']) || !in_array($item['sku'], ['K3-BOOK-001', 'K3-MUG-002', 'K3-SET-003'], true) || !isset($item['quantity']) || !is_int($item['quantity']) || $item['quantity'] < 1) {
				return new WP_Error('invalid_order_event', 'The order event contains an invalid line item.', ['status' => 422]);
			}
		}
		$scope = $order['siteId'] . "\0" . $order['orderId'] . "\0" . $order['revision'];
		return self::store('order', hash('sha256', $scope), $order);
	}

	private static function within_bound(WP_REST_Request $request): bool {
		$body = $request->get_body();
		return is_string($body) && strlen($body) > 0 && strlen($body) <= self::MAX_BODY_BYTES;
	}

	private static function safe_identity(string $value): bool {
		return strlen($value) <= 128 && preg_match('/^[A-Za-z0-9._:-]+$/', $value) === 1;
	}

	private static function safe_revision(string $value): bool {
		if (strlen($value) > 40 || preg_match('/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$/', $value) !== 1) {
			return false;
		}
		$date = DateTimeImmutable::createFromFormat('!Y-m-d\TH:i:s', $value);
		return $date instanceof DateTimeImmutable && $date->format('Y-m-d\TH:i:s') === $value;
	}

	private static function text_length(string $value): int {
		$length = preg_match_all('/./us', $value);
		return $length === false ? PHP_INT_MAX : $length;
	}

	private static function store(string $kind, string $key, array $payload) {
		global $wpdb;
		if (!self::ensure_schema()) {
			return new WP_Error('destination_unavailable', 'The local idempotency table is unavailable.', ['status' => 503]);
		}
		$encoded = wp_json_encode($payload, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
		if (!is_string($encoded)) {
			return new WP_Error('invalid_event_encoding', 'The event could not be encoded safely.', ['status' => 422]);
		}
		$hash = hash('sha256', $encoded);
		$table = self::table_name();
		$inserted = $wpdb->query($wpdb->prepare(
			"INSERT IGNORE INTO {$table} (idempotency_key, event_kind, payload_sha256, payload_json, created_gmt) VALUES (%s, %s, %s, %s, %s)",
			$kind . ':' . $key,
			$kind,
			$hash,
			$encoded,
			gmdate('Y-m-d H:i:s')
		));
		if ($inserted === 1) {
			return new WP_REST_Response(['status' => 'created', 'receiptId' => $kind . ':' . $key], 201);
		}
		if ($inserted === false) {
			return new WP_Error('destination_write_failed', 'The local destination could not store the event.', ['status' => 503]);
		}
		$existing = $wpdb->get_row($wpdb->prepare("SELECT payload_sha256 FROM {$table} WHERE idempotency_key = %s", $kind . ':' . $key), ARRAY_A);
		if (!is_array($existing)) {
			return new WP_Error('destination_write_failed', 'The local destination could not store the event.', ['status' => 503]);
		}
		if (hash_equals((string) $existing['payload_sha256'], $hash)) {
			return new WP_REST_Response(['status' => 'duplicate', 'receiptId' => $kind . ':' . $key], 200);
		}
		return new WP_Error('idempotency_conflict', 'The same event identity arrived with different content.', ['status' => 409]);
	}

	private static function ensure_schema(): bool {
		global $wpdb;
		$version = get_option('vrooli_delivery_destination_schema');
		$table = self::table_name();
		if ($version === self::SCHEMA_VERSION && $wpdb->get_var($wpdb->prepare('SHOW TABLES LIKE %s', $table)) === $table) {
			return true;
		}
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$charset = $wpdb->get_charset_collate();
		$sql = "CREATE TABLE {$table} (
			id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
			idempotency_key varchar(191) NOT NULL,
			event_kind varchar(16) NOT NULL,
			payload_sha256 char(64) NOT NULL,
			payload_json longtext NOT NULL,
			created_gmt datetime NOT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY idempotency_key (idempotency_key)
		) {$charset};";
		dbDelta($sql);
		$exists = $wpdb->get_var($wpdb->prepare('SHOW TABLES LIKE %s', $table));
		if ($exists !== $table) {
			return false;
		}
		update_option('vrooli_delivery_destination_schema', self::SCHEMA_VERSION, false);
		return true;
	}

	private static function table_name(): string {
		global $wpdb;
		return $wpdb->prefix . self::TABLE_SUFFIX;
	}
}

Vrooli_Delivery_Destination::boot();
